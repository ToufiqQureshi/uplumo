package citefix

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/toufiqqureshi/uplumo/backend/internal/aisearch"
	"github.com/toufiqqureshi/uplumo/backend/internal/audit"
)

const (
	rivalRefreshInterval = 7 * 24 * time.Hour
	rivalPageLimit       = 3
	rivalFetchTimeout    = 12 * time.Second
	// rivalFetchBudget caps the whole rival step of one report: the report is
	// read on page load (and every 15 s while a scan runs), so a slow
	// third-party site must never hold it up.
	rivalFetchBudget = 8 * time.Second
	// rivalRetryInterval is how long a failed fetch is remembered, so a blocked
	// or broken rival site is not hit again on every report read.
	rivalRetryInterval  = 24 * time.Hour
	rivalRobotsMaxBytes = 512 << 10
)

// AuditResults reuses the Site Audit service's scoped result reader for the
// free "Also fix for Google" list.
type AuditResults interface {
	GetResults(ctx context.Context, auditID, projectID string) (audit.ResultsPayload, error)
}

// SignalSnapshot is every citation signal shown side by side in the free rival
// comparison. FAQ is true for FAQPage schema or question-and-answer headings.
type SignalSnapshot struct {
	AnswerFirstWords int      `json:"answerFirstWords"`
	Author           bool     `json:"author"`
	Date             string   `json:"date"`
	Schema           []string `json:"schema"`
	Statistics       int      `json:"statistics"`
	FAQ              bool     `json:"faq"`
	WordCount        int      `json:"wordCount"`
}

// RivalPage is the free comparison for one page AI cited instead of the
// customer's site. An app shell is reported as such and is never rendered.
type RivalPage struct {
	URL         string         `json:"url"`
	Domain      string         `json:"domain"`
	Title       string         `json:"title"`
	AppShell    bool           `json:"appShell"`
	FetchedAt   time.Time      `json:"fetchedAt"`
	ContentHash string         `json:"contentHash"`
	Signals     SignalSnapshot `json:"signals"`
}

// RivalComparison links a missed AI question and a cited rival page with the
// strongest eligible page from the customer's most recent audit.
type RivalComparison struct {
	Prompt string            `json:"prompt"`
	Engine string            `json:"engine"`
	Rival  RivalPage         `json:"rival"`
	Yours  *RivalPageSummary `json:"yours"`
}

// RivalPageSummary is the customer's own best signal page from the latest
// audit, with no page body or sensitive headers.
type RivalPageSummary struct {
	URL     string         `json:"url"`
	Title   string         `json:"title"`
	Signals SignalSnapshot `json:"signals"`
}

type cachedRivalPage struct {
	RivalPage
}

// rivalComparisons selects up to three unique cited pages for one report,
// including Brand Lookup pages tied to the same missed question. It reuses a
// seven-day cache, fetches only guarded public HTML, and never renders rivals.
func (s Store) rivalComparisons(ctx context.Context, organizationID, projectID string, proof *Proof, mentions *MentionsSnapshot, pages []Page) ([]RivalComparison, error) {
	if proof == nil || len(proof.Misses) == 0 {
		return []RivalComparison{}, nil
	}
	guard := s.Guard
	if guard == nil {
		guard = audit.NewGuard()
	}
	client := s.HTTP
	if client == nil {
		client = guard.NewClient(rivalFetchTimeout)
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	yours := bestOwnPage(pages)
	type pick struct {
		prompt, engine string
		cited          CitedPage
		cached         *cachedRivalPage
		fetched        *RivalPage
	}
	picks := make([]*pick, 0, rivalPageLimit)
	seen := map[string]bool{}
	for _, miss := range proof.Misses {
		for _, candidate := range rivalCandidates(miss, proof.Domain, mentions) {
			if len(picks) >= rivalPageLimit {
				break
			}
			if seen[candidate.page.URL] || !isPublicHTTPURL(candidate.page.URL) {
				continue
			}
			seen[candidate.page.URL] = true
			cached, err := s.rivalCache(ctx, organizationID, projectID, candidate.page.URL)
			if err != nil {
				return nil, err
			}
			picks = append(picks, &pick{prompt: miss.Prompt, engine: candidate.engine, cited: candidate.page, cached: cached})
		}
	}

	// Stale or missing pages are fetched together under one budget; a page
	// that does not answer in time is left out of this report only.
	fetchCtx, cancel := context.WithTimeout(ctx, rivalFetchBudget)
	var wg sync.WaitGroup
	for _, p := range picks {
		if !rivalNeedsFetch(p.cached, now()) {
			continue
		}
		wg.Go(func() {
			fetched, err := fetchRivalPage(fetchCtx, guard, client, p.cited, now())
			if err != nil {
				// Remember the failure so the next report does not try again
				// before rivalRetryInterval.
				fetched = RivalPage{URL: p.cited.URL, Domain: p.cited.Domain, FetchedAt: now()}
			}
			p.fetched = &fetched
		})
	}
	wg.Wait()
	cancel()

	comparisons := make([]RivalComparison, 0, len(picks))
	for _, p := range picks {
		page := p.cached
		if p.fetched != nil {
			if err := s.saveRivalCache(ctx, organizationID, projectID, *p.fetched); err != nil {
				return nil, err
			}
			page = &cachedRivalPage{RivalPage: *p.fetched}
		}
		// An empty content hash marks a failed fetch: nothing to compare.
		if page == nil || page.ContentHash == "" {
			continue
		}
		comparisons = append(comparisons, RivalComparison{Prompt: p.prompt, Engine: p.engine, Rival: page.RivalPage, Yours: yours})
	}
	return comparisons, nil
}

// rivalNeedsFetch says whether a rival page must be fetched for this report:
// it was never fetched, its snapshot is older than rivalRefreshInterval, or
// its last fetch failed more than rivalRetryInterval ago.
func rivalNeedsFetch(cached *cachedRivalPage, now time.Time) bool {
	if cached == nil {
		return true
	}
	if cached.ContentHash == "" {
		return now.Sub(cached.FetchedAt) >= rivalRetryInterval
	}
	return now.Sub(cached.FetchedAt) >= rivalRefreshInterval
}

type rivalCandidate struct {
	page   CitedPage
	engine string
}

func rivalCandidates(miss Miss, ownDomain string, mentions *MentionsSnapshot) []rivalCandidate {
	candidates := make([]rivalCandidate, 0, len(miss.CitedPages))
	for _, page := range miss.CitedPages {
		candidates = append(candidates, rivalCandidate{page: page, engine: miss.Engine})
	}
	if mentions == nil {
		return candidates
	}
	question := normalizeRivalQuestion(miss.Prompt)
	for _, page := range mentions.Result.TopPages {
		if page.URL == "" || isOwnHost(candidateHost(page.URL), ownDomain) || !topPageMatchesQuestion(page, question) {
			continue
		}
		candidates = append(candidates, rivalCandidate{
			page:   CitedPage{URL: page.URL, Domain: candidateHost(page.URL)},
			engine: "AI Visibility Brand Lookup · " + clip(page.Platform, 32),
		})
	}
	return candidates
}

func topPageMatchesQuestion(page aisearch.TopPage, question string) bool {
	for _, keyword := range page.Keywords {
		if normalizeRivalQuestion(keyword.Question) == question {
			return true
		}
	}
	return false
}

func normalizeRivalQuestion(question string) string {
	return strings.ToLower(strings.Join(strings.Fields(question), " "))
}

func candidateHost(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return bareHost(parsed.Hostname())
}

func (s Store) rivalCache(ctx context.Context, organizationID, projectID, pageURL string) (*cachedRivalPage, error) {
	var cached cachedRivalPage
	var signals []byte
	err := s.DB.QueryRow(ctx, `SELECT c.page_url, c.domain, c.title, c.app_shell, c.fetched_at, c.content_hash, c.signals_json
		FROM go_citefix_rival_pages c JOIN projects p ON p.id=c.project_id AND p.organization_id=$1
		WHERE c.project_id=$2 AND c.organization_id=$1 AND c.page_url=$3
		ORDER BY c.fetched_at DESC LIMIT 1`, organizationID, projectID, pageURL).Scan(
		&cached.URL, &cached.Domain, &cached.Title, &cached.AppShell, &cached.FetchedAt, &cached.ContentHash, &signals)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read Cite Fix rival cache: %w", err)
	}
	if err := json.Unmarshal(signals, &cached.Signals); err != nil {
		return nil, fmt.Errorf("decode Cite Fix rival signals: %w", err)
	}
	return &cached, nil
}

func (s Store) saveRivalCache(ctx context.Context, organizationID, projectID string, page RivalPage) error {
	signals, err := json.Marshal(page.Signals)
	if err != nil {
		return fmt.Errorf("encode Cite Fix rival signals: %w", err)
	}
	_, err = s.DB.Exec(ctx, `INSERT INTO go_citefix_rival_pages
		(organization_id, project_id, page_url, domain, title, app_shell, fetched_at, content_hash, signals_json)
		SELECT $1, p.id, $3, $4, $5, $6, $7, $8, $9 FROM projects p
		WHERE p.id=$2 AND p.organization_id=$1 AND p.archived_at IS NULL
		ON CONFLICT(project_id,page_url,content_hash) DO UPDATE SET
			domain=EXCLUDED.domain,title=EXCLUDED.title,app_shell=EXCLUDED.app_shell,
			fetched_at=EXCLUDED.fetched_at,signals_json=EXCLUDED.signals_json`,
		organizationID, projectID, page.URL, page.Domain, page.Title, page.AppShell, page.FetchedAt, page.ContentHash, signals)
	if err != nil {
		return fmt.Errorf("save Cite Fix rival cache: %w", err)
	}
	return nil
}

func fetchRivalPage(ctx context.Context, guard *audit.Guard, client *http.Client, cited CitedPage, fetchedAt time.Time) (RivalPage, error) {
	ctx, cancel := context.WithTimeout(ctx, rivalFetchTimeout)
	defer cancel()
	validated, err := guard.NormalizeAndValidateStartURL(ctx, cited.URL)
	if err != nil {
		return RivalPage{}, err
	}
	probe, err := guard.ResolveStartURLRedirects(ctx, validated)
	if err != nil {
		return RivalPage{}, err
	}
	target := probe.URL
	parsed, err := url.Parse(target)
	if err != nil {
		return RivalPage{}, err
	}
	origin := parsed.Scheme + "://" + parsed.Host
	robotsURL := origin + "/robots.txt"
	robotsRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, robotsURL, nil)
	if err != nil {
		return RivalPage{}, err
	}
	robotsRequest.Header.Set("User-Agent", audit.UserAgent)
	robotsResponse, err := client.Do(robotsRequest)
	if err != nil {
		return RivalPage{}, err
	}
	var robotsText string
	switch robotsResponse.StatusCode {
	case http.StatusNotFound:
		robotsText = ""
	case http.StatusOK:
		robotsText, err = readBounded(robotsResponse.Body, rivalRobotsMaxBytes)
	default:
		_ = robotsResponse.Body.Close()
		return RivalPage{}, fmt.Errorf("robots.txt returned status %d", robotsResponse.StatusCode)
	}
	_ = robotsResponse.Body.Close()
	if err != nil || len(robotsText) >= rivalRobotsMaxBytes || !audit.RobotsAllows(origin, target, robotsText) {
		return RivalPage{}, fmt.Errorf("rival page is disallowed by robots.txt or robots.txt could not be read")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return RivalPage{}, err
	}
	request.Header.Set("User-Agent", audit.UserAgent)
	request.Header.Set("Accept", "text/html")
	response, err := client.Do(request)
	if err != nil {
		return RivalPage{}, err
	}
	if response.StatusCode != http.StatusOK || !strings.Contains(strings.ToLower(response.Header.Get("Content-Type")), "text/html") {
		if closeErr := response.Body.Close(); closeErr != nil {
			return RivalPage{}, fmt.Errorf("close rival page response: %w", closeErr)
		}
		return RivalPage{}, fmt.Errorf("rival page returned status %d or non-HTML content", response.StatusCode)
	}
	html, err := readBounded(response.Body, audit.MaxHTMLBytes)
	closeErr := response.Body.Close()
	if err != nil {
		return RivalPage{}, fmt.Errorf("rival page HTML could not be read: %w", err)
	}
	if closeErr != nil {
		return RivalPage{}, fmt.Errorf("close rival page response: %w", closeErr)
	}
	if strings.TrimSpace(html) == "" {
		return RivalPage{}, errors.New("rival page HTML is empty")
	}
	sum := sha256.Sum256([]byte(html))
	analysis := audit.AnalyzeHTML(html, target, response.StatusCode, 0, "")
	return RivalPage{
		// Keep the AI-cited URL as the cache key; target is the redirect
		// destination used for fetching and signal extraction.
		URL: cited.URL, Domain: strings.TrimPrefix(strings.ToLower(parsed.Hostname()), "www."),
		Title: analysis.Title, AppShell: analysis.JavaScriptShell, FetchedAt: fetchedAt.UTC(),
		ContentHash: hex.EncodeToString(sum[:]),
		Signals:     snapshotSignals(analysis.AI, analysis.WordCount),
	}, nil
}

func readBounded(body io.Reader, limit int) (string, error) {
	data, err := io.ReadAll(io.LimitReader(body, int64(limit)+1))
	if err != nil {
		return "", err
	}
	if len(data) > limit {
		data = data[:limit]
	}
	return string(data), nil
}

func snapshotSignals(signals audit.AIPageSignals, wordCount int) SignalSnapshot {
	faq := signals.QuestionHeadings > 0
	for _, schema := range signals.SchemaTypes {
		if strings.EqualFold(schema, "FAQPage") {
			faq = true
			break
		}
	}
	date := signals.Modified
	if date == "" {
		date = signals.Published
	}
	return SignalSnapshot{
		AnswerFirstWords: signals.LeadWords, Author: signals.Author, Date: date,
		Schema: append([]string{}, signals.SchemaTypes...), Statistics: signals.Statistics,
		FAQ: faq, WordCount: wordCount,
	}
}

func bestOwnPage(pages []Page) *RivalPageSummary {
	var best *RivalPageSummary
	bestScore := -1
	for _, page := range pages {
		if page.Signals == nil || !page.Indexable || page.WordCount == 0 {
			continue
		}
		snapshot := snapshotSignals(*page.Signals, page.WordCount)
		score := 0
		if snapshot.AnswerFirstWords > 0 {
			score += 2
		}
		if snapshot.Author {
			score++
		}
		if snapshot.Date != "" {
			score++
		}
		if len(snapshot.Schema) > 0 {
			score++
		}
		if snapshot.Statistics > 0 {
			score++
		}
		if snapshot.FAQ {
			score++
		}
		if score > bestScore {
			bestScore = score
			best = &RivalPageSummary{URL: page.URL, Title: page.Title, Signals: snapshot}
		}
	}
	return best
}

func isPublicHTTPURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != "" && u.User == nil
}
