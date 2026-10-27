// Package citefix explains why AI assistants do not cite a site and what to
// fix first. It scores the project's latest completed site audit: the crawl
// already stored every page's citation signals, so a report makes no
// provider call and costs nothing.
package citefix

import (
	"cmp"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/audit"
)

// Report states.
const (
	// StateReady: the latest completed audit was scored.
	StateReady = "ready"
	// StateNoScan: the project has no completed audit yet.
	StateNoScan = "no_scan"
	// StateNeedsRescan: the latest completed audit was crawled before the
	// crawler stored citation signals. Scoring it would report zeros the site
	// never earned, so the customer is asked to scan again instead.
	StateNeedsRescan = "needs_rescan"
)

const (
	// maxFixPages is how many affected pages each fix lists.
	maxFixPages = 20
	// maxReportPages is how many pages the page table shows, worst first.
	maxReportPages = 50
)

// Scan is the audit a report reads.
type Scan struct {
	AuditID     string    `json:"auditId"`
	StartURL    string    `json:"startUrl"`
	CompletedAt time.Time `json:"completedAt"`
}

// Running is an audit that has not finished; the page offers to wait for it.
type Running struct {
	AuditID string `json:"auditId"`
	Status  string `json:"status"`
}

// Page is one crawled page with what Cite Fix needs from it.
type Page struct {
	URL           string
	Title         string
	StatusCode    int
	Indexable     bool
	WordCount     int
	ExternalLinks int
	// Signals is nil for a page crawled before signals were stored.
	Signals *audit.AIPageSignals
}

// Issue is one stored audit issue that a check reads.
type Issue struct {
	Type    string
	PageURL string
	Details map[string]any
}

// PageRef names a page in a fix.
type PageRef struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// CategoryScore is one category's score out of 100.
type CategoryScore struct {
	Category
	Score int `json:"score"`
}

// Fix is one failing check, with the pages it affects.
type Fix struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Title    string `json:"title"`
	Why      string `json:"why"`
	How      string `json:"how"`
	// Impact is high, medium or low, from Gain.
	Impact string `json:"impact"`
	// Gain is the points the site score would rise if this check passed
	// everywhere, rounded to one decimal.
	Gain          float64   `json:"gain"`
	PagesAffected int       `json:"pagesAffected"`
	Pages         []PageRef `json:"pages"`
}

// PageScore is one content page's score and the page checks it fails.
type PageScore struct {
	URL    string   `json:"url"`
	Title  string   `json:"title"`
	Score  int      `json:"score"`
	Failed []string `json:"failed"`
}

// Report is the Cite Fix result for a project.
type Report struct {
	State   string   `json:"state"`
	Running *Running `json:"running"`
	Scan    *Scan    `json:"scan"`
	// Score is 0-100; nil unless State is ready.
	Score      *int            `json:"score"`
	Categories []CategoryScore `json:"categories"`
	Fixes      []Fix           `json:"fixes"`
	Pages      []PageScore     `json:"pages"`
	// PagesChecked counts the content pages the page checks read; PagesTotal
	// counts every crawled page.
	PagesChecked int `json:"pagesChecked"`
	PagesTotal   int `json:"pagesTotal"`
	// Proof is the latest saved AI Visibility check, in every state; nil when
	// the project has none.
	Proof *Proof `json:"proof"`
	// Agent says whether and at what price the paid agent can run; the
	// handler fills it.
	Agent AgentOffer `json:"agent"`
}

// emptyReport keeps every list a JSON array, never null.
func emptyReport(state string) Report {
	return Report{State: state, Categories: []CategoryScore{}, Fixes: []Fix{}, Pages: []PageScore{}}
}

// isContentPage reports whether a page can be quoted: fetched, indexable
// HTML with some text. Redirects, errors, noindex pages and files are not.
func isContentPage(page Page) bool {
	return page.StatusCode >= 200 && page.StatusCode < 300 && page.Indexable && page.WordCount > 0
}

// result is one check's outcome: passed of eligible (1 of 1 for a site check).
type result struct {
	passed, eligible int
	failing          []PageRef
}

// outcome is one page's result on one page check.
type outcome struct {
	id   string
	pass bool
}

func (r result) ratio() float64 { return float64(r.passed) / float64(r.eligible) }

// Build scores one completed scan. It assumes every page carries signals; the
// caller reports StateNeedsRescan otherwise.
func Build(scan Scan, pages []Page, issues []Issue) Report {
	report := emptyReport(StateReady)
	report.Scan = &scan
	report.PagesTotal = len(pages)

	byType := map[string][]Issue{}
	for _, issue := range issues {
		byType[issue.Type] = append(byType[issue.Type], issue)
	}
	shells := pageSet(byType[string(audit.IssueJavaScriptRenderingSuspected)])
	duplicates := pageSet(byType[string(audit.IssueDuplicateContent)])

	results := map[string]result{
		"ai-search-bots-allowed": siteResult(!blocksSearchBots(byType[string(audit.IssueAICrawlersBlocked)])),
		"llms-txt":               siteResult(len(byType[string(audit.IssueMissingLLMsTxt)]) == 0),
		"business-identity":      siteResult(len(byType[string(audit.IssueMissingBusinessSchema)]) == 0),
		"one-phone-number":       siteResult(len(byType[string(audit.IssueSeveralPhoneNumbers)]) == 0),
	}
	// Dates are read without a time of day, so freshness is counted in days.
	completed := scan.CompletedAt.UTC()
	scanDay := time.Date(completed.Year(), completed.Month(), completed.Day(), 0, 0, 0, 0, time.UTC)
	pageScores := []PageScore{}
	for _, page := range pages {
		_, shell := shells[page.URL]
		if page.StatusCode >= 200 && page.StatusCode < 300 && page.Indexable {
			record(results, "pages-readable", page, !shell)
		}
		// A JavaScript shell fails "pages-readable" alone: its missing text is
		// one problem, not one per page check.
		if shell || !isContentPage(page) || page.Signals == nil {
			continue
		}
		report.PagesChecked++
		signals := page.Signals
		_, duplicate := duplicates[page.URL]
		outcomes := []outcome{
			{"structured-data", len(signals.SchemaTypes) > 0},
			{"author", signals.Author},
			{"dated", signals.Published != "" || signals.Modified != ""},
			{"answer-first", signals.LeadWords >= minLeadWords && signals.LeadWords <= maxLeadWords},
			{"question-headings", signals.QuestionHeadings > 0},
			{"scannable", signals.Lists+signals.Tables > 0},
			{"statistics", signals.Statistics > 0},
			{"sources", page.ExternalLinks > 0},
			{"quotes", signals.Quotes > 0},
			{"not-thin", page.WordCount >= minWords},
			{"unique", !duplicate},
		}
		// Freshness is judged only on pages that state a date it can read;
		// an undated page already fails "dated".
		if updated, ok := latestDate(signals.Published, signals.Modified); ok {
			outcomes = append(outcomes, outcome{"fresh", !scanDay.After(updated.AddDate(0, 0, freshDays))})
		}
		var passedWeight, totalWeight int
		failed := []string{}
		for _, outcome := range outcomes {
			record(results, outcome.id, page, outcome.pass)
			weight := checkByID(outcome.id).weight
			totalWeight += weight
			if outcome.pass {
				passedWeight += weight
			} else {
				failed = append(failed, outcome.id)
			}
		}
		pageScores = append(pageScores, PageScore{
			URL: page.URL, Title: page.Title, Failed: failed,
			Score: int(math.Round(100 * float64(passedWeight) / float64(totalWeight))),
		})
	}

	var siteScore, scoredWeight float64
	for _, category := range categories {
		var earned, possible float64
		for _, c := range checks {
			r := results[c.id]
			if c.category != category.ID || r.eligible == 0 {
				continue
			}
			earned += float64(c.weight) * r.ratio()
			possible += float64(c.weight)
		}
		if possible == 0 {
			continue
		}
		score := 100 * earned / possible
		report.Categories = append(report.Categories, CategoryScore{Category: category, Score: int(math.Round(score))})
		siteScore += float64(category.Weight) * score
		scoredWeight += float64(category.Weight)
		for _, c := range checks {
			r := results[c.id]
			if c.category != category.ID || r.eligible == 0 || r.passed == r.eligible {
				continue
			}
			report.Fixes = append(report.Fixes, newFix(c, r, float64(category.Weight)*float64(c.weight)/possible*(1-r.ratio())))
		}
	}
	// Normalised by the categories that had something to judge, so a site
	// with no content pages is scored on access and identity alone.
	score := int(math.Round(siteScore / scoredWeight))
	report.Score = &score

	slices.SortStableFunc(report.Fixes, func(a, b Fix) int {
		return cmp.Or(cmp.Compare(b.Gain, a.Gain), cmp.Compare(b.PagesAffected, a.PagesAffected))
	})
	slices.SortStableFunc(pageScores, func(a, b PageScore) int {
		return cmp.Or(cmp.Compare(a.Score, b.Score), strings.Compare(a.URL, b.URL))
	})
	report.Pages = pageScores[:min(len(pageScores), maxReportPages)]
	return report
}

func newFix(c check, r result, gain float64) Fix {
	gain = math.Round(gain*10) / 10
	impact := "low"
	switch {
	case gain >= 5:
		impact = "high"
	case gain >= 2:
		impact = "medium"
	}
	affected := r.eligible - r.passed
	pages := r.failing
	if pages == nil {
		pages = []PageRef{}
	}
	if !c.page {
		affected = 0
	}
	return Fix{
		ID: c.id, Category: c.category, Title: c.title, Why: c.why, How: c.how,
		Impact: impact, Gain: gain, PagesAffected: affected, Pages: pages,
	}
}

func siteResult(pass bool) result {
	if pass {
		return result{passed: 1, eligible: 1}
	}
	return result{eligible: 1}
}

// record adds one page's outcome; the first maxFixPages failures are named.
func record(results map[string]result, id string, page Page, pass bool) {
	r := results[id]
	r.eligible++
	if pass {
		r.passed++
	} else if len(r.failing) < maxFixPages {
		r.failing = append(r.failing, PageRef{URL: page.URL, Title: page.Title})
	}
	results[id] = r
}

func checkByID(id string) check {
	index := slices.IndexFunc(checks, func(c check) bool { return c.id == id })
	return checks[index]
}

func pageSet(issues []Issue) map[string]struct{} {
	set := make(map[string]struct{}, len(issues))
	for _, issue := range issues {
		set[issue.PageURL] = struct{}{}
	}
	return set
}

// blocksSearchBots reports whether the audit's blocked-crawler issue names a
// search bot. Its details list names such as "PerplexityBot (Perplexity)".
func blocksSearchBots(issues []Issue) bool {
	for _, issue := range issues {
		crawlers, _ := issue.Details["crawlers"].([]any)
		for _, crawler := range crawlers {
			name, _ := crawler.(string)
			for _, bot := range searchBots {
				if name == bot || strings.HasPrefix(name, bot+" ") {
					return true
				}
			}
		}
	}
	return false
}

// dateLayouts are the date forms pages commonly state, ISO first.
var dateLayouts = []string{"January 2, 2006", "Jan 2, 2006", "2 January 2006", "2 Jan 2006"}

// latestDate returns the later of the dates it can read.
func latestDate(values ...string) (time.Time, bool) {
	var latest time.Time
	for _, value := range values {
		if parsed, ok := parseDate(value); ok && parsed.After(latest) {
			latest = parsed
		}
	}
	return latest, !latest.IsZero()
}

// parseDate reads the date part of an ISO timestamp or a written date. Time
// of day is ignored: freshness is counted in days.
func parseDate(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if len(value) >= 10 {
		if parsed, err := time.Parse(time.DateOnly, value[:10]); err == nil {
			return parsed, true
		}
	}
	for _, layout := range dateLayouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}
