package aisearch

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	visibilityHistoryRuns  = 30
	visibilitySourceLimit  = 5000
	visibilityDisplayLimit = 50
)

// VisibilityMetrics summarizes saved citation lists for one comparable scope.
// Citation rank means the source's position in the provider's saved list; it
// is not necessarily the position of a link in the generated answer text.
type VisibilityMetrics struct {
	Runs           int                `json:"runs"`
	HistoryLimited bool               `json:"historyLimited"`
	SourcesLimited bool               `json:"sourcesLimited"`
	Domains        []VisibilitySource `json:"domains"`
	Pages          []VisibilitySource `json:"pages"`
	Prompts        []VisibilityPrompt `json:"prompts"`
}

// VisibilitySource summarizes one saved cited domain or page.
type VisibilitySource struct {
	Name        string    `json:"name"`
	FirstSeen   time.Time `json:"firstSeen"`
	LastSeen    time.Time `json:"lastSeen"`
	AverageRank float64   `json:"averageRank"`
	LatestCount int       `json:"latestCount"`
	Momentum    string    `json:"momentum"` // up, down, flat or unavailable
}

// VisibilityPrompt summarizes one saved question and engine.
type VisibilityPrompt struct {
	Prompt            string   `json:"prompt"`
	Engine            string   `json:"engine"`
	Answers           int      `json:"answers"`
	CitedAnswers      int      `json:"citedAnswers"`
	AverageRank       *float64 `json:"averageRank"`
	LatestRank        *int     `json:"latestRank"`
	InlineAnswers     int      `json:"inlineAnswers"`
	AverageAnswerRank *float64 `json:"averageAnswerRank"`
	LatestAnswerRank  *int     `json:"latestAnswerRank"`
}

type visibilityAnswer struct {
	prompt, engine, status, text string
	at                           time.Time
	citations                    []Citation
}

type visibilityRun struct {
	id, status string
	answers    []visibilityAnswer
}

// LoadVisibilityMetrics reads at most the 30 latest saved runs in the selected
// run's exact scope. The bound protects shared DB/CPU capacity and is disclosed
// to callers. It has no provider client and never starts a collection.
func LoadVisibilityMetrics(ctx context.Context, db *pgxpool.Pool, org, project, runID string) (*VisibilityMetrics, error) {
	var scope, domain string
	var created time.Time
	err := db.QueryRow(ctx, `SELECT r.scope_key, COALESCE(r.configuration->>'domain', ''), r.created_at
		FROM go_ai_tracking_runs r JOIN projects p ON p.id=r.project_id AND p.organization_id=$1
		WHERE r.organization_id=$1 AND r.project_id=$2 AND r.id=$3 AND r.status IN ('completed','partial')`, org, project, runID).Scan(&scope, &domain, &created)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find visibility scope: %w", err)
	}
	rows, err := db.Query(ctx, `WITH recent AS (
		SELECT id,status,created_at,finished_at FROM go_ai_tracking_runs
		WHERE organization_id=$1 AND project_id=$2 AND scope_key=$3
		AND (created_at < $4 OR (created_at = $4 AND id <= $6))
		AND status IN ('completed','partial') ORDER BY created_at DESC, id DESC LIMIT $5
	)
	SELECT r.id,r.status,COALESCE(a.collected_at,r.finished_at,r.created_at),a.prompt,a.engine,a.status,a.answer,a.citations
	FROM recent r JOIN go_ai_tracking_answers a ON a.run_id=r.id
	ORDER BY r.created_at DESC, r.id DESC, a.prompt, a.engine`, org, project, scope, created, visibilityHistoryRuns, runID)
	if err != nil {
		return nil, fmt.Errorf("read saved visibility history: %w", err)
	}
	defer rows.Close()
	runs := []visibilityRun{}
	for rows.Next() {
		var runID, status string
		var answer visibilityAnswer
		var citations []byte
		if err := rows.Scan(&runID, &status, &answer.at, &answer.prompt, &answer.engine, &answer.status, &answer.text, &citations); err != nil {
			return nil, fmt.Errorf("scan visibility history: %w", err)
		}
		if err := json.Unmarshal(citations, &answer.citations); err != nil {
			return nil, fmt.Errorf("decode saved visibility citations: %w", err)
		}
		if len(runs) == 0 || runs[len(runs)-1].id != runID {
			runs = append(runs, visibilityRun{id: runID, status: status})
		}
		runs[len(runs)-1].answers = append(runs[len(runs)-1].answers, answer)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read saved visibility history: %w", err)
	}
	if len(runs) == 0 || runs[0].id != runID {
		return nil, nil
	}
	return buildVisibilityMetrics(domain, runs), nil
}

type sourceAccumulator struct {
	first, last       time.Time
	rankSum, ranks    int
	current, previous int
}

type promptAccumulator struct {
	row                        VisibilityPrompt
	rankSum, ranks             int
	answerRankSum, answerRanks int
}

// answerCitationRank uses only URLs actually present in saved answer text. It
// never assumes that a provider's source-list order is its prose order.
func answerCitationRank(text, domain string, citations []Citation) int {
	type match struct {
		offset int
		url    string
	}
	var matches []match
	seen := map[string]bool{}
	for _, citation := range citations {
		if visibilityHost(citation.URL) == "" || seen[citation.URL] {
			continue
		}
		seen[citation.URL] = true
		start := 0
		for start < len(text) {
			index := strings.Index(text[start:], citation.URL)
			if index < 0 {
				break
			}
			index += start
			end := index + len(citation.URL)
			const continuation = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~/%?&=#"
			before := index > 0 && strings.ContainsRune(continuation, rune(text[index-1]))
			after := end < len(text) && strings.ContainsRune(continuation, rune(text[end]))
			if !before && !after {
				matches = append(matches, match{offset: index, url: citation.URL})
				break
			}
			start = index + len(citation.URL)
		}
	}
	slices.SortFunc(matches, func(a, b match) int { return cmp.Compare(a.offset, b.offset) })
	for i, found := range matches {
		if visibilityOwnHost(visibilityHost(found.url), domain) {
			return i + 1
		}
	}
	return 0
}

func completeVisibilityRun(run visibilityRun) bool {
	if run.status != "completed" || len(run.answers) == 0 {
		return false
	}
	for _, answer := range run.answers {
		if answer.status != "completed" || strings.TrimSpace(answer.text) == "" {
			return false
		}
	}
	return true
}

func comparableVisibilityRuns(a, b visibilityRun) bool {
	if !completeVisibilityRun(a) || !completeVisibilityRun(b) || len(a.answers) != len(b.answers) {
		return false
	}
	keys := make(map[string]bool, len(a.answers))
	for _, answer := range a.answers {
		keys[answer.prompt+"\x00"+answer.engine] = true
	}
	for _, answer := range b.answers {
		if !keys[answer.prompt+"\x00"+answer.engine] {
			return false
		}
	}
	return true
}

func visibilityHost(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(strings.TrimSuffix(u.Hostname(), ".")), "www.")
}

func visibilityOwnHost(host, domain string) bool {
	domain = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(domain)), "www.")
	return domain != "" && (host == domain || strings.HasSuffix(host, "."+domain))
}

func buildVisibilityMetrics(domain string, runs []visibilityRun) *VisibilityMetrics {
	out := &VisibilityMetrics{Runs: len(runs), HistoryLimited: len(runs) == visibilityHistoryRuns, Domains: []VisibilitySource{}, Pages: []VisibilitySource{}, Prompts: []VisibilityPrompt{}}
	if len(runs) == 0 {
		return out
	}
	runsComparable := len(runs) > 1 && comparableVisibilityRuns(runs[0], runs[1])
	domains := map[string]*sourceAccumulator{}
	pages := map[string]*sourceAccumulator{}
	prompts := map[string]*promptAccumulator{}
	add := func(items map[string]*sourceAccumulator, key string, rank, runIndex int, at time.Time) {
		item := items[key]
		if item == nil {
			if len(items) >= visibilitySourceLimit {
				out.SourcesLimited = true
				return
			}
			item = &sourceAccumulator{}
			items[key] = item
		}
		if item.first.IsZero() || at.Before(item.first) {
			item.first = at
		}
		if item.last.IsZero() || at.After(item.last) {
			item.last = at
		}
		item.rankSum += rank
		item.ranks++
		if runIndex == 0 {
			item.current++
		}
		if runIndex == 1 {
			item.previous++
		}
	}
	for runIndex, run := range runs {
		for _, answer := range run.answers {
			if answer.status != "completed" || strings.TrimSpace(answer.text) == "" {
				continue
			}
			key := answer.prompt + "\x00" + answer.engine
			prompt := prompts[key]
			if prompt == nil {
				prompt = &promptAccumulator{row: VisibilityPrompt{Prompt: answer.prompt, Engine: answer.engine}}
				prompts[key] = prompt
			}
			prompt.row.Answers++
			seenDomains, seenPages := map[string]bool{}, map[string]bool{}
			ownRank := 0
			for i, citation := range answer.citations {
				host := visibilityHost(citation.URL)
				if host == "" {
					continue
				}
				rank := i + 1
				if !seenDomains[host] {
					add(domains, host, rank, runIndex, answer.at)
					seenDomains[host] = true
				}
				if !seenPages[citation.URL] {
					add(pages, citation.URL, rank, runIndex, answer.at)
					seenPages[citation.URL] = true
				}
				if ownRank == 0 && visibilityOwnHost(host, domain) {
					ownRank = rank
				}
			}
			if ownRank > 0 {
				prompt.row.CitedAnswers++
				prompt.rankSum += ownRank
				prompt.ranks++
				if runIndex == 0 {
					rank := ownRank
					prompt.row.LatestRank = &rank
				}
			}
			if answerRank := answerCitationRank(answer.text, domain, answer.citations); answerRank > 0 {
				prompt.row.InlineAnswers++
				prompt.answerRankSum += answerRank
				prompt.answerRanks++
				if runIndex == 0 {
					rank := answerRank
					prompt.row.LatestAnswerRank = &rank
				}
			}
		}
	}
	convert := func(items map[string]*sourceAccumulator) []VisibilitySource {
		values := make([]VisibilitySource, 0, len(items))
		for name, item := range items {
			momentum := "unavailable"
			if runsComparable {
				momentum = "flat"
				if item.current > item.previous {
					momentum = "up"
				} else if item.current < item.previous {
					momentum = "down"
				}
			}
			values = append(values, VisibilitySource{Name: name, FirstSeen: item.first, LastSeen: item.last, AverageRank: float64(item.rankSum) / float64(item.ranks), LatestCount: item.current, Momentum: momentum})
		}
		slices.SortFunc(values, func(a, b VisibilitySource) int {
			return cmp.Or(cmp.Compare(b.LatestCount, a.LatestCount), strings.Compare(a.Name, b.Name))
		})
		if len(values) > visibilityDisplayLimit {
			out.SourcesLimited = true
			values = values[:visibilityDisplayLimit]
		}
		return values
	}
	out.Domains, out.Pages = convert(domains), convert(pages)
	for _, item := range prompts {
		if item.ranks > 0 {
			average := float64(item.rankSum) / float64(item.ranks)
			item.row.AverageRank = &average
		}
		if item.answerRanks > 0 {
			average := float64(item.answerRankSum) / float64(item.answerRanks)
			item.row.AverageAnswerRank = &average
		}
		out.Prompts = append(out.Prompts, item.row)
	}
	slices.SortFunc(out.Prompts, func(a, b VisibilityPrompt) int {
		return cmp.Or(strings.Compare(a.Prompt, b.Prompt), strings.Compare(a.Engine, b.Engine))
	})
	return out
}
