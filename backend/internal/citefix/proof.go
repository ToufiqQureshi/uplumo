package citefix

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

	"github.com/toufiqqureshi/uplumo/backend/internal/aisearch"
)

// Proof is what the project's latest saved AI Visibility check shows: which
// sites AI answers cite for the customer's own questions, and whether the
// customer's site is one of them. It reads saved answers only; it never buys
// a new check.
type Proof struct {
	CheckedAt time.Time `json:"checkedAt"`
	Domain    string    `json:"domain"`
	// Answers is how many saved answers cite at least one page; CitingYou is
	// how many of them cite the customer's domain.
	Answers   int           `json:"answers"`
	CitingYou int           `json:"citingYou"`
	Rivals    []DomainCount `json:"rivals"`
	YourPages []URLCount    `json:"yourPages"`
	// Misses are answers that cite other sites and not the customer's.
	Misses     []Miss                      `json:"misses"`
	Visibility *aisearch.VisibilityMetrics `json:"visibility"`
	// Said is what answers that name the customer say, in their own words.
	Said []Said `json:"said"`
}

// Said is the part of one saved AI answer that names the customer.
type Said struct {
	Prompt  string `json:"prompt"`
	Engine  string `json:"engine"`
	Excerpt string `json:"excerpt"`
}

// DomainCount is a cited domain and how many answers cite it.
type DomainCount struct {
	Domain string `json:"domain"`
	Count  int    `json:"count"`
}

// URLCount is a cited page and how many answers cite it.
type URLCount struct {
	URL   string `json:"url"`
	Count int    `json:"count"`
}

// Miss is one question where the AI cited others instead of the customer.
type Miss struct {
	Prompt     string      `json:"prompt"`
	Engine     string      `json:"engine"`
	Cited      []string    `json:"cited"`
	CitedPages []CitedPage `json:"citedPages"`
}

// CitedPage is one rival page saved by AI Visibility for a missed question.
type CitedPage struct {
	URL    string `json:"url"`
	Domain string `json:"domain"`
	Title  string `json:"title,omitempty"`
}

const (
	maxProofRivals = 5
	maxProofPages  = 5
	// maxProofMisses keeps every missed question and engine of a normal run
	// (a few prompts times six engines), so each one can show who AI cited.
	maxProofMisses = 60
	// maxProofAnswers bounds the answers read; a run holds at most a few
	// hundred (prompts times engines).
	maxProofAnswers = 2_000
	maxMissCited    = 3
	maxSaid         = 6
)

// storedCitation is the part of a saved citation the proof reads.
type storedCitation struct {
	URL    string  `json:"url"`
	Domain *string `json:"domain"`
	Title  *string `json:"title"`
}

// proofAnswer is one saved answer's citations.
type proofAnswer struct {
	prompt, engine, answer string
	citations              []storedCitation
}

// proof loads the latest finished AI Visibility run of the project. nil means
// the project has never finished one: the page then offers to set it up.
func (s Store) proof(ctx context.Context, organizationID, projectID string) (*Proof, error) {
	var runID string
	var checkedAt time.Time
	var configJSON []byte
	err := s.DB.QueryRow(ctx, `SELECT r.id, COALESCE(r.finished_at, r.created_at), r.configuration
		FROM go_ai_tracking_runs r JOIN projects p ON p.id = r.project_id AND p.organization_id = $1
		WHERE r.project_id = $2 AND r.organization_id = $1 AND r.status IN ('completed', 'partial')
		ORDER BY r.created_at DESC LIMIT 1`, organizationID, projectID).Scan(&runID, &checkedAt, &configJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find latest AI visibility run: %w", err)
	}
	var config struct {
		Brand  string `json:"brand"`
		Domain string `json:"domain"`
	}
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, fmt.Errorf("decode AI visibility configuration: %w", err)
	}
	rows, err := s.DB.Query(ctx, `SELECT prompt, engine, answer, citations FROM go_ai_tracking_answers
		WHERE run_id = $1 AND status = 'completed' ORDER BY prompt, engine LIMIT $2`, runID, maxProofAnswers)
	if err != nil {
		return nil, fmt.Errorf("list AI visibility answers: %w", err)
	}
	defer rows.Close()
	answers := []proofAnswer{}
	for rows.Next() {
		var answer proofAnswer
		var citationsJSON []byte
		if err := rows.Scan(&answer.prompt, &answer.engine, &answer.answer, &citationsJSON); err != nil {
			return nil, fmt.Errorf("scan AI visibility answer: %w", err)
		}
		if err := json.Unmarshal(citationsJSON, &answer.citations); err != nil {
			return nil, fmt.Errorf("decode citations of %q: %w", answer.prompt, err)
		}
		answers = append(answers, answer)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read AI visibility answers: %w", err)
	}
	rows.Close()
	proof := buildProof(checkedAt, config.Domain, answers)
	proof.Said = said(config.Brand, proof.Domain, answers)
	proof.Visibility, err = aisearch.LoadVisibilityMetrics(ctx, s.DB, organizationID, projectID, runID)
	if err != nil {
		return nil, err
	}
	return proof, nil
}

// buildProof counts citations per domain and page. A domain is counted once
// per answer, however many of its pages the answer cites.
func buildProof(checkedAt time.Time, domain string, answers []proofAnswer) *Proof {
	own := bareHost(domain)
	proof := &Proof{CheckedAt: checkedAt, Domain: own, Rivals: []DomainCount{}, YourPages: []URLCount{}, Misses: []Miss{}, Said: []Said{}}
	rivals := map[string]int{}
	pages := map[string]int{}
	for _, answer := range answers {
		seenURL, seenHost := map[string]bool{}, map[string]bool{}
		cited := []string{}
		citedPages := []CitedPage{}
		seenPage := map[string]bool{}
		citesYou := false
		for _, citation := range answer.citations {
			host := citationHost(citation)
			if host == "" || seenURL[citation.URL] {
				continue
			}
			seenURL[citation.URL] = true
			if isOwnHost(host, own) {
				citesYou = true
				pages[canonicalCitedURL(citation.URL)]++
			} else if !seenHost[host] {
				seenHost[host] = true
				rivals[host]++
				cited = append(cited, host)
			}
			pageURL := canonicalCitedURL(citation.URL)
			if !isOwnHost(host, own) && !seenPage[pageURL] && len(citedPages) < maxMissCited {
				seenPage[pageURL] = true
				page := CitedPage{URL: pageURL, Domain: host}
				if citation.Title != nil {
					page.Title = *citation.Title
				}
				citedPages = append(citedPages, page)
			}
		}
		if len(seenURL) == 0 {
			continue
		}
		proof.Answers++
		if citesYou {
			proof.CitingYou++
		} else if len(proof.Misses) < maxProofMisses {
			proof.Misses = append(proof.Misses, Miss{Prompt: answer.prompt, Engine: answer.engine, Cited: cited[:min(len(cited), maxMissCited)], CitedPages: citedPages})
		}
	}
	for host, count := range rivals {
		proof.Rivals = append(proof.Rivals, DomainCount{Domain: host, Count: count})
	}
	slices.SortFunc(proof.Rivals, func(a, b DomainCount) int {
		return cmp.Or(cmp.Compare(b.Count, a.Count), strings.Compare(a.Domain, b.Domain))
	})
	proof.Rivals = proof.Rivals[:min(len(proof.Rivals), maxProofRivals)]
	for page, count := range pages {
		proof.YourPages = append(proof.YourPages, URLCount{URL: page, Count: count})
	}
	slices.SortFunc(proof.YourPages, func(a, b URLCount) int {
		return cmp.Or(cmp.Compare(b.Count, a.Count), strings.Compare(a.URL, b.URL))
	})
	proof.YourPages = proof.YourPages[:min(len(proof.YourPages), maxProofPages)]
	return proof
}

// said quotes the answers that name the customer, by brand or domain, in
// prompt and engine order, at most maxSaid.
func said(brand, domain string, answers []proofAnswer) []Said {
	quotes := []Said{}
	for _, answer := range answers {
		if len(quotes) == maxSaid {
			break
		}
		if excerpt := plainText(aisearch.BrandExcerpt(answer.answer, brand, domain)); excerpt != "" {
			quotes = append(quotes, Said{Prompt: answer.prompt, Engine: answer.engine, Excerpt: excerpt})
		}
	}
	return quotes
}

// markdownMarks are the emphasis and heading marks AI answers use; quoted
// as plain text they read like noise.
var markdownMarks = strings.NewReplacer("**", "", "__", "", "`", "", "### ", "", "## ", "", "# ", "")

// plainText is an answer excerpt without markdown marks or extra spaces.
func plainText(excerpt string) string {
	return strings.Join(strings.Fields(markdownMarks.Replace(excerpt)), " ")
}

// canonicalCitedURL drops what makes one page look like several: the
// fragment (AI engines add "#:~:text=" highlights) and utm_ tracking
// parameters (some engines tag their own links). A URL that does not parse
// is kept as sent.
func canonicalCitedURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	parsed.Fragment, parsed.RawFragment = "", ""
	query := parsed.Query()
	changed := false
	for key := range query {
		if strings.HasPrefix(strings.ToLower(key), "utm_") {
			query.Del(key)
			changed = true
		}
	}
	if changed {
		parsed.RawQuery = query.Encode()
	}
	return parsed.String()
}

// citationHost is the citation's domain, or its URL's host when the provider
// sent none, without "www.".
func citationHost(citation storedCitation) string {
	if citation.Domain != nil && strings.TrimSpace(*citation.Domain) != "" {
		return bareHost(*citation.Domain)
	}
	parsed, err := url.Parse(citation.URL)
	if err != nil {
		return ""
	}
	return bareHost(parsed.Hostname())
}

func bareHost(host string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(host)), "www.")
}

// isOwnHost counts the customer's subdomains (blog.example.com) as theirs.
func isOwnHost(host, own string) bool {
	return own != "" && (host == own || strings.HasSuffix(host, "."+own))
}
