package audit

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

// The one-page check behind the public free audit. It reads one page,
// robots.txt, llms.txt and the sitemap of the visitor's site with the same
// SSRF-guarded crawler and analyzer as a full audit, and calls no paid
// provider: no DataForSEO, no PageSpeed, no AI model. The only outside data
// is Chrome's field data from the free CrUX API, when a key is configured.
//
// One fetch feeds two independent engines: ScoreSEO (seoengine.go, how well
// Google can crawl, index and rank the page) and ScoreAEO (aeoengine.go, how
// ready the page is for AI assistants to read, trust and quote). Each has its
// own checks, categories, weights and caps. Neither measures rankings, links
// or AI mentions, and the report says so.

// CheckStatus is how one check came out.
type CheckStatus string

// Check outcomes. Info checks explain something and are not scored.
const (
	CheckPass CheckStatus = "pass"
	CheckWarn CheckStatus = "warn"
	CheckFail CheckStatus = "fail"
	CheckInfo CheckStatus = "info"
)

// Report kinds: the two engines of the free audit.
const (
	KindSEO = "seo"
	KindAEO = "aeo"
)

// Category IDs the upgrade offers of the report read.
const (
	CategoryOnPage  = seoOnPage
	CategoryContent = seoContent
	CategoryAnswers = aeoAnswer
	CategoryTrust   = aeoTrust
)

// KindName is the short name of a report kind: "SEO" or "AEO".
func KindName(kind string) string {
	if kind == KindAEO {
		return "AEO"
	}
	return "SEO"
}

// KindHonest is the line each report carries: what the engine checks and what
// it does not measure.
func KindHonest(kind string) string {
	if kind == KindAEO {
		return "Checks how ready this page is to be read, trusted and quoted by AI assistants. It does not measure whether ChatGPT or other assistants mention you today."
	}
	return "Checks how well this page is built for Google. It does not measure rankings or backlinks."
}

// GradeLabel names a page score in words. Only 90 and above is "Excellent",
// which a capped score cannot reach with a serious failure.
func GradeLabel(score int) string {
	switch {
	case score >= 90:
		return "Excellent"
	case score >= 80:
		return "Good"
	case score >= 50:
		return "Needs work"
	}
	return "Poor"
}

// Severities say how much a failed check hurts the page. The weight of a
// check is its severity's weight.
const (
	sevCritical = "critical"
	sevHigh     = "high"
	sevMedium   = "medium"
	sevLow      = "low"
)

var severityWeights = map[string]int{
	sevCritical: 8,
	sevHigh:     5,
	sevMedium:   3,
	sevLow:      1,
}

// Scoring rules shared by both engines. A warning earns warnCredit of its
// weight; a pass earns all of it; a fail earns nothing. A critical failure
// caps the score at the engine's own limit; the other caps are shared.
const (
	warnCredit  = 0.35
	capHigh     = 79
	capManyWarn = 89
	// manyWarns is how many warnings cap the score at capManyWarn.
	manyWarns = 6
)

// PageCheck is one check of the free audit, written for a site owner: what
// was found on their page, why it matters and what to change.
type PageCheck struct {
	ID       string      `json:"id"`
	Category string      `json:"category"`
	Status   CheckStatus `json:"status"`
	Title    string      `json:"title"`
	Found    string      `json:"found"`
	Why      string      `json:"why"`
	Fix      string      `json:"fix,omitempty"`
	// Severity is critical, high, medium or low.
	Severity string `json:"severity"`
	// Weight is the severity's weight (8, 5, 3 or 1) in its category's score.
	Weight int `json:"weight"`
}

// CategoryScore is one category's 0-100 score and its check counts.
type CategoryScore struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Score  int    `json:"score"`
	Passed int    `json:"passed"`
	Issues int    `json:"issues"`
}

// PageReport is one engine's result for one page.
type PageReport struct {
	// Kind is KindSEO or KindAEO.
	Kind      string    `json:"kind"`
	URL       string    `json:"url"`
	CheckedAt time.Time `json:"checkedAt"`
	// Reachable is false when no HTML page could be read; Problem then says
	// why in plain words and the report has no score.
	Reachable  bool            `json:"reachable"`
	Problem    string          `json:"problem,omitempty"`
	StatusCode int             `json:"statusCode"`
	ResponseMs int64           `json:"responseMs"`
	Title      string          `json:"title"`
	WordCount  int             `json:"wordCount"`
	Score      int             `json:"score"`
	Categories []CategoryScore `json:"categories"`
	// Caps lists why the score is lower than the checks alone would give, one
	// sentence each ("Score capped at 49 because ...").
	Caps   []string    `json:"caps,omitempty"`
	Checks []PageCheck `json:"checks"`
}

// PageSnapshot is one fetch of a page and its site's files. Both engines
// score the same snapshot, so one visitor check costs one fetch.
type PageSnapshot struct {
	// URL is the page after the site's redirects.
	URL        string
	CheckedAt  time.Time
	StatusCode int
	ResponseMs int64
	// Redirects is how many redirects the page fetch followed.
	Redirects int
	// Reachable is false when no HTML page could be read; Problem says why.
	Reachable bool
	Problem   string
	// Page and Analysis are the crawled HTML of the page (zero when unreachable).
	Page     CrawledPageResult
	Analysis PageAnalysis
	// HTMLBytes is how much HTML was read.
	HTMLBytes int
	Origin    string
	// Robots is the robots.txt text; nil when there is none.
	Robots *string
	// LLMs is true when /llms.txt exists.
	LLMs bool
	// Sitemap is the XML sitemap's URL; "" when none was found.
	Sitemap string
	// Field is Chrome's field data for the page; nil when no source is set.
	Field *StoredFieldData
}

// PageChecker runs the free one-page audit.
type PageChecker struct {
	guard     *Guard
	crawler   *Crawler
	finder    discoverer
	fieldData FieldDataProvider
	now       func() time.Time
}

// pageCheckTimeout bounds each fetch of the free audit.
const pageCheckTimeout = 12 * time.Second

// fieldDataTimeout bounds the Chrome field data lookup of one check.
const fieldDataTimeout = 8 * time.Second

// maxPageCheckRedirects is how many redirects the page fetch follows after
// the start URL probe.
const maxPageCheckRedirects = 3

// NewPageChecker builds a checker on guard. A nil guard uses the default
// SSRF guard. The checker has no field data source until WithFieldData.
func NewPageChecker(guard *Guard) *PageChecker {
	if guard == nil {
		guard = NewGuard()
	}
	client := guard.NewClient(pageCheckTimeout)
	return &PageChecker{
		guard:   guard,
		crawler: NewCrawler(CrawlerOptions{Guard: guard, Client: client}),
		finder:  discoverer{guard: guard, client: client, now: time.Now},
		now:     time.Now,
	}
}

// WithFieldData sets the Chrome field data source (CrUX). A nil provider
// leaves the Core Web Vitals check as "not measured".
func (c *PageChecker) WithFieldData(provider FieldDataProvider) *PageChecker {
	c.fieldData = provider
	return c
}

// ValidateURL validates a visitor's input and returns the start address a
// check would begin from. It makes no request to the site (DNS only), so a
// cache lookup can use it without fetching the page.
func (c *PageChecker) ValidateURL(ctx context.Context, input string) (string, error) {
	return c.guard.NormalizeAndValidateStartURL(ctx, input)
}

// NormalizeURL validates a visitor's input and returns the page URL the
// check reads after the site's own redirects. Errors are ErrStartURLInvalid
// or ErrCrawlTargetBlocked (wrapped) for input the checker refuses.
func (c *PageChecker) NormalizeURL(ctx context.Context, input string) (string, error) {
	normalized, err := c.guard.NormalizeAndValidateStartURL(ctx, input)
	if err != nil {
		return "", err
	}
	probe, err := c.guard.ResolveStartURLRedirects(ctx, normalized)
	if err != nil {
		return "", err
	}
	return probe.URL, nil
}

// Check reads pageURL (from NormalizeURL) and its site's robots.txt,
// llms.txt and sitemap. It makes at most maxPageCheckRedirects+4 requests to
// the visitor's site, and at most one field data lookup. It scores nothing:
// ScoreSEO and ScoreAEO score the snapshot.
func (c *PageChecker) Check(ctx context.Context, pageURL string) PageSnapshot {
	snap := PageSnapshot{URL: pageURL, CheckedAt: c.now().UTC()}
	origin, err := originOf(pageURL)
	if err != nil {
		snap.Problem = "That address is not a website we can check."
		return snap
	}
	snap.Origin = origin

	var (
		wg        sync.WaitGroup
		page      CrawlResult
		analysis  *PageAnalysis
		redirects int
		robots    *string
		llms      bool
	)
	wg.Go(func() { page, analysis, redirects = c.fetchPage(ctx, pageURL) })
	wg.Go(func() { robots = c.finder.fetchRobotsTxtText(ctx, origin) })
	wg.Go(func() { llms = c.finder.hasLLMsTxt(ctx, origin) })
	wg.Wait()

	snap.URL = page.URL
	snap.StatusCode = page.StatusCode
	snap.ResponseMs = page.ResponseTimeMs
	snap.Redirects = redirects
	if problem := unreachable(page); problem != "" || analysis == nil {
		if problem == "" {
			problem = "We could not read this page's HTML."
		}
		snap.Problem = problem
		return snap
	}
	snap.Reachable = true
	snap.Page = page.CrawledPageResult
	snap.Analysis = *analysis
	snap.HTMLBytes = page.HTMLBytesRead
	snap.Robots = robots
	snap.LLMs = llms
	snap.Sitemap = c.findSitemap(ctx, origin, robots)
	snap.Field = c.fieldDataFor(ctx, page.URL)
	return snap
}

// fieldDataFor asks the field data source for the page's phone metrics, or
// returns nil when no source is set up.
func (c *PageChecker) fieldDataFor(ctx context.Context, pageURL string) *StoredFieldData {
	if c.fieldData == nil {
		return nil
	}
	lookupCtx, cancel := context.WithTimeout(ctx, fieldDataTimeout)
	defer cancel()
	data := c.fieldData.FieldData(lookupCtx, pageURL, "phone")
	return &data
}

// fetchPage fetches the page, following up to maxPageCheckRedirects
// redirects, each revalidated against the SSRF policy. The analysis is from
// the last response and nil when that response is not HTML. It returns how
// many redirects were followed.
func (c *PageChecker) fetchPage(ctx context.Context, pageURL string) (CrawlResult, *PageAnalysis, int) {
	page, analysis := c.crawler.fetchAnalyzed(ctx, pageURL)
	hops := 0
	for range maxPageCheckRedirects {
		if page.StatusCode < 300 || page.StatusCode >= 400 || page.RedirectURL == "" {
			break
		}
		next, err := c.guard.NormalizeAndValidateStartURL(ctx, page.RedirectURL)
		if err != nil {
			break
		}
		hops++
		page, analysis = c.crawler.fetchAnalyzed(ctx, next)
	}
	return page, analysis, hops
}

// findSitemap returns the first sitemap robots.txt names, or /sitemap.xml,
// when it is a real XML sitemap; "" when there is none.
func (c *PageChecker) findSitemap(ctx context.Context, origin string, robots *string) string {
	candidate := origin + "/sitemap.xml"
	if declared := parseRobotsTxt(origin, robots).SitemapURLs; len(declared) > 0 {
		candidate = declared[0]
	}
	if !isCrawlableURL(candidate) {
		return ""
	}
	fetched, err := c.finder.fetchFollowingRedirects(ctx, candidate, pageCheckTimeout)
	if err != nil || fetched == nil {
		return ""
	}
	defer func() { _ = fetched.resp.Body.Close() }()
	if fetched.resp.StatusCode < 200 || fetched.resp.StatusCode >= 300 {
		return ""
	}
	body, err := ReadTextUpTo(fetched.resp, 16*1024)
	if err != nil && body == "" {
		return ""
	}
	if !strings.Contains(body, "<urlset") && !strings.Contains(body, "<sitemapindex") {
		return ""
	}
	return fetched.finalURL
}

func originOf(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return "", errors.Join(ErrStartURLInvalid, err)
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}

// unreachable explains why a fetched page cannot be checked, or "".
func unreachable(page CrawlResult) string {
	switch {
	case page.FetchClass == FetchBlocked:
		return "Your website's firewall or bot protection blocked our checker. Search engines and AI assistants may be blocked too. " +
			"If you use Cloudflare or a similar service, allow the \"" + UserAgent + "\" user agent and run the check again."
	case page.FetchClass == FetchRateLimited:
		return "Your website asked us to slow down (HTTP 429). Wait a minute and run the check again."
	case page.StatusCode == 0:
		return "We could not connect to this website. Check the address, or try again if the site is slow or down."
	case page.StatusCode >= 300 && page.StatusCode < 400:
		return "This address keeps redirecting. Enter the final address of the page you want checked."
	case page.StatusCode >= 500:
		return fmt.Sprintf("The website answered with a server error (HTTP %d). Visitors and search engines see this error too.", page.StatusCode)
	case page.StatusCode >= 400:
		return fmt.Sprintf("The page answered with HTTP %d, so there is nothing to check. Enter a page that loads in a browser.", page.StatusCode)
	case !page.IsHTML:
		return "This address is not a web page (it returned a file or data). Enter your website's home page."
	}
	return ""
}

// categoryDef is one category of an engine: its report name and its share of
// the score.
type categoryDef struct {
	ID     string
	Name   string
	Weight int
}

// scoreEngine builds one engine's report from a snapshot. An unreachable page
// gets no checks and no score, only the problem.
func scoreEngine(snap PageSnapshot, kind string, categories []categoryDef, severeMax int, checks func(PageSnapshot) []PageCheck) PageReport {
	report := PageReport{Kind: kind, URL: snap.URL, CheckedAt: snap.CheckedAt, Problem: snap.Problem,
		StatusCode: snap.StatusCode, ResponseMs: snap.ResponseMs}
	if !snap.Reachable {
		return report
	}
	report.Reachable = true
	report.Title = snap.Page.Title
	report.WordCount = snap.Page.WordCount
	report.Checks = checks(snap)
	report.Categories, report.Score, report.Caps = scoreChecks(report.Checks, categories, severeMax)
	return report
}

// scoreChecks scores each category from its checks (a pass earns the full
// weight, a warning warnCredit of it, a fail nothing; info is not scored),
// averages the categories by their weights, then applies the caps. It returns
// the categories, the final score and the caps that lowered it.
func scoreChecks(checks []PageCheck, categories []categoryDef, severeMax int) ([]CategoryScore, int, []string) {
	var scored []CategoryScore
	weighted, weightSum := 0.0, 0
	for _, def := range categories {
		category := CategoryScore{ID: def.ID, Name: def.Name}
		earned, possible := 0.0, 0
		for _, check := range checks {
			if check.Category != def.ID || check.Status == CheckInfo {
				continue
			}
			possible += check.Weight
			switch check.Status {
			case CheckPass:
				earned += float64(check.Weight)
				category.Passed++
			case CheckWarn:
				earned += warnCredit * float64(check.Weight)
				category.Issues++
			case CheckFail:
				category.Issues++
			case CheckInfo:
			}
		}
		if possible > 0 {
			category.Score = int(100*earned/float64(possible) + 0.5)
			weighted += float64(category.Score * def.Weight)
			weightSum += def.Weight
		}
		scored = append(scored, category)
	}
	raw := 0
	if weightSum > 0 {
		raw = int(weighted/float64(weightSum) + 0.5)
	}
	score, caps := applyCaps(raw, checks, severeMax)
	return scored, score, caps
}

// applyCaps lowers raw by every cap that applies and returns the reasons for
// the caps that lowered it, strictest first. A failed critical check caps the
// score at severeMax (the engine's own limit).
func applyCaps(raw int, checks []PageCheck, severeMax int) (int, []string) {
	var critical, high, warns []string
	for _, check := range checks {
		switch {
		case check.Status == CheckFail && check.Severity == sevCritical:
			critical = append(critical, check.Title)
		case check.Status == CheckFail && check.Severity == sevHigh:
			high = append(high, check.Title)
		case check.Status == CheckWarn:
			warns = append(warns, check.Title)
		}
	}
	type capRule struct {
		max    int
		reason string
	}
	var rules []capRule
	if len(critical) > 0 {
		rules = append(rules, capRule{severeMax, "a critical check failed (" + strings.Join(critical, ", ") + ")"})
	}
	if len(high) > 0 {
		rules = append(rules, capRule{capHigh, "a high-impact check failed (" + strings.Join(high, ", ") + ")"})
	}
	if len(warns) >= manyWarns {
		rules = append(rules, capRule{capManyWarn, fmt.Sprintf("%d checks need improvement", len(warns))})
	}
	score := raw
	var caps []string
	for _, rule := range rules {
		if rule.max < raw {
			caps = append(caps, fmt.Sprintf("Score capped at %d because %s.", rule.max, rule.reason))
		}
		score = min(score, rule.max)
	}
	return score, caps
}
