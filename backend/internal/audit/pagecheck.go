package audit

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// The one-page check behind the public free audit. It reads one page,
// robots.txt, llms.txt and the sitemap of the visitor's site with the same
// SSRF-guarded crawler and analyzer as a full audit, and calls no paid
// provider: no DataForSEO, no PageSpeed, no AI model.

// CheckStatus is how one check came out.
type CheckStatus string

// Check outcomes. Info checks explain something and are not scored.
const (
	CheckPass CheckStatus = "pass"
	CheckWarn CheckStatus = "warn"
	CheckFail CheckStatus = "fail"
	CheckInfo CheckStatus = "info"
)

// Check categories, in the order the report shows them.
const (
	CategorySEO       = "seo"
	CategoryAI        = "ai"
	CategoryTechnical = "technical"
	CategoryContent   = "content"
)

// categoryNames are what the report calls each category.
var categoryNames = map[string]string{
	CategorySEO:       "On-page SEO",
	CategoryAI:        "AI search (AEO)",
	CategoryTechnical: "Technical",
	CategoryContent:   "Content",
}

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
	// Weight is how much the check counts in its category's score.
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

// PageReport is the free audit's result for one page.
type PageReport struct {
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
	Checks     []PageCheck     `json:"checks"`
}

// PageChecker runs the free one-page audit.
type PageChecker struct {
	guard   *Guard
	crawler *Crawler
	finder  discoverer
}

// pageCheckTimeout bounds each fetch of the free audit.
const pageCheckTimeout = 12 * time.Second

// maxPageCheckRedirects is how many redirects the page fetch follows after
// the start URL probe.
const maxPageCheckRedirects = 3

// NewPageChecker builds a checker on guard. A nil guard uses the default
// SSRF guard.
func NewPageChecker(guard *Guard) *PageChecker {
	if guard == nil {
		guard = NewGuard()
	}
	client := guard.NewClient(pageCheckTimeout)
	return &PageChecker{
		guard:   guard,
		crawler: NewCrawler(CrawlerOptions{Guard: guard, Client: client}),
		finder:  discoverer{guard: guard, client: client, now: time.Now},
	}
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
// llms.txt and sitemap, then scores the page. It makes at most
// maxPageCheckRedirects+4 requests to the visitor's site.
func (c *PageChecker) Check(ctx context.Context, pageURL string) PageReport {
	report := PageReport{URL: pageURL, CheckedAt: time.Now().UTC()}
	origin, err := originOf(pageURL)
	if err != nil {
		report.Problem = "That address is not a website we can check."
		return report
	}

	var (
		wg     sync.WaitGroup
		page   CrawlResult
		robots *string
		llms   bool
	)
	wg.Go(func() { page = c.fetchPage(ctx, pageURL) })
	wg.Go(func() { robots = c.finder.fetchRobotsTxtText(ctx, origin) })
	wg.Go(func() { llms = c.finder.hasLLMsTxt(ctx, origin) })
	wg.Wait()

	report.URL = page.URL
	report.StatusCode = page.StatusCode
	report.ResponseMs = page.ResponseTimeMs
	if problem := unreachable(page); problem != "" {
		report.Problem = problem
		return report
	}
	sitemap := c.findSitemap(ctx, origin, robots)

	report.Reachable = true
	report.Title = page.Title
	report.WordCount = page.WordCount
	report.Checks = pageChecks(page.CrawledPageResult, origin, robots, llms, sitemap)
	report.Categories, report.Score = scoreChecks(report.Checks)
	return report
}

// fetchPage fetches the page, following up to maxPageCheckRedirects
// redirects, each revalidated against the SSRF policy.
func (c *PageChecker) fetchPage(ctx context.Context, pageURL string) CrawlResult {
	page := c.crawler.FetchPage(ctx, pageURL)
	for range maxPageCheckRedirects {
		if page.StatusCode < 300 || page.StatusCode >= 400 || page.RedirectURL == "" {
			break
		}
		next, err := c.guard.NormalizeAndValidateStartURL(ctx, page.RedirectURL)
		if err != nil {
			break
		}
		page = c.crawler.FetchPage(ctx, next)
	}
	return page
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
			"If you use Cloudflare or a similar service, allow the \"" + auditUserAgent + "\" user agent and run the check again."
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

// pageChecks turns one fetched page and its site files into the report's
// checks. The thresholds are the full audit's, so both say the same thing.
func pageChecks(page CrawledPageResult, origin string, robots *string, llms bool, sitemap string) []PageCheck {
	var checks []PageCheck
	add := func(check PageCheck) { checks = append(checks, check) }
	ai := page.AI
	tech := page.Technical

	// On-page SEO.
	add(lengthCheck("title", CategorySEO, 3, "Page title", page.Title, titleMinChars, titleMaxChars,
		"The title is the blue link people click in Google and the name AI assistants give your page.",
		"Write one title of about 50-60 characters: what the page offers, then your brand."))
	add(lengthCheck("description", CategorySEO, 2, "Meta description", page.MetaDescription, metaDescriptionMinChars, metaDescriptionMaxChars,
		"Google often shows it under your title. A clear one gets more clicks.",
		"Write 70-160 characters that say what the visitor gets and why to choose you."))
	h1 := PageCheck{ID: "h1", Category: CategorySEO, Weight: 2, Title: "Main heading (H1)",
		Why: "One H1 tells readers and search engines what the page is about."}
	switch page.H1Count {
	case 0:
		h1.Status, h1.Found, h1.Fix = CheckFail, "No H1 heading on the page.", "Add one H1 that states the page's main topic, in line with the title."
	case 1:
		h1.Status, h1.Found = CheckPass, "One H1 heading."
	default:
		h1.Status, h1.Found, h1.Fix = CheckWarn, fmt.Sprintf("%d H1 headings.", page.H1Count), "Keep one H1 and turn the others into H2 or H3."
	}
	add(h1)
	headings := PageCheck{ID: "heading-order", Category: CategorySEO, Weight: 1, Title: "Heading structure",
		Why:    "Headings in order (H1, then H2, then H3) make the page easy to scan for people and machines.",
		Status: CheckPass, Found: fmt.Sprintf("%d H2 and %d H3 headings, in order.", page.H2Count, page.H3Count)}
	if hasHeadingLevelSkip(page.HeadingOrder) {
		headings.Status, headings.Found, headings.Fix = CheckWarn, "Heading levels skip a step (for example H1 to H3).", "Change heading levels so they go down one step at a time."
	}
	add(headings)
	add(canonicalCheck(page))
	index := PageCheck{ID: "indexable", Category: CategorySEO, Weight: 3, Title: "Allowed in Google",
		Why: "A noindex rule removes the page from Google and AI search results.", Status: CheckPass, Found: "No noindex rule."}
	if !page.IsIndexable {
		index.Status, index.Found = CheckFail, "The page says noindex: "+strings.TrimSpace(page.RobotsMeta+" "+page.XRobotsTag)
		index.Fix = "Remove noindex from the robots meta tag or X-Robots-Tag header if this page should be found."
	}
	add(index)
	social := PageCheck{ID: "open-graph", Category: CategorySEO, Weight: 1, Title: "Social sharing preview",
		Why: "Open Graph tags decide the title and image shown when your link is shared on WhatsApp, LinkedIn or X.", Status: CheckPass,
		Found: "Open Graph title and image are set."}
	if page.OgTitle == "" || page.OgImage == "" {
		social.Status, social.Found, social.Fix = CheckWarn, "Open Graph title or image is missing.", "Add og:title, og:description and og:image meta tags."
	}
	add(social)
	alt := PageCheck{ID: "image-alt", Category: CategorySEO, Weight: 1, Title: "Image alt text",
		Why: "Alt text lets search engines, AI and screen readers understand your images."}
	switch {
	case page.ImagesTotal == 0:
		alt.Status, alt.Found = CheckInfo, "No images on the page."
	case page.ImagesMissingAlt == 0:
		alt.Status, alt.Found = CheckPass, fmt.Sprintf("All %d images have alt text.", page.ImagesTotal)
	default:
		alt.Status, alt.Found = CheckWarn, fmt.Sprintf("%d of %d images have no alt text.", page.ImagesMissingAlt, page.ImagesTotal)
		alt.Fix = "Describe each meaningful image in a few words; use alt=\"\" only for decoration."
	}
	add(alt)

	// AI search readiness.
	add(aiCrawlerCheck(origin, robots))
	add(schemaCheck(ai.SchemaTypes))
	faq := slices.ContainsFunc(ai.SchemaTypes, func(t string) bool { return strings.EqualFold(t, "FAQPage") })
	questions := PageCheck{ID: "questions", Category: CategoryAI, Weight: 2, Title: "Answers to customer questions",
		Why: "AI assistants quote pages that ask a real question and answer it right below."}
	switch {
	case faq || ai.QuestionHeadings >= 2:
		questions.Status, questions.Found = CheckPass, fmt.Sprintf("%d question headings.", ai.QuestionHeadings)
		if faq {
			questions.Found = fmt.Sprintf("%d question headings and FAQ schema.", ai.QuestionHeadings)
		}
	case ai.QuestionHeadings == 1:
		questions.Status, questions.Found = CheckWarn, "One question heading."
		questions.Fix = "Add a short FAQ: the questions customers ask as H2 or H3 headings, each answered in the first sentence below it."
	default:
		questions.Status, questions.Found = CheckFail, "No headings phrased as a question."
		questions.Fix = "Add a short FAQ: the questions customers ask as H2 or H3 headings, each answered in the first sentence below it."
	}
	add(questions)
	answer := PageCheck{ID: "answer-first", Category: CategoryAI, Weight: 2, Title: "Direct answer up front",
		Why: "AI answers lift short, factual paragraphs (30-120 words) that start with a statement.", Status: CheckPass,
		Found: "At least one paragraph gives a direct, factual answer."}
	if !ai.AnswerCapsule {
		answer.Status, answer.Found = CheckWarn, "No short paragraph that answers a question with a concrete fact."
		answer.Fix = "Open the page with 2-3 sentences that say what you offer, for whom, where and at what price."
	}
	add(answer)
	readable := PageCheck{ID: "no-javascript", Category: CategoryAI, Weight: 3, Title: "Readable without JavaScript",
		Why: "Most AI crawlers do not run JavaScript; they only see the HTML your server sends.", Status: CheckPass,
		Found: fmt.Sprintf("%d words are in the HTML the server sends.", page.WordCount)}
	if page.JavaScriptShell {
		readable.Status, readable.Found = CheckFail, "The page is an empty shell until JavaScript runs."
		readable.Fix = "Use server-side rendering or prerendering so the text is in the first HTML response."
	}
	add(readable)
	snippet := PageCheck{ID: "snippet", Category: CategoryAI, Weight: 2, Title: "Allowed to be quoted",
		Why: "nosnippet or max-snippet:0 stops Google and AI answers from quoting your text.", Status: CheckPass, Found: "No rule blocks quoting."}
	if ai.SnippetBlocked {
		snippet.Status, snippet.Found, snippet.Fix = CheckFail, "A nosnippet rule blocks quoting.", "Remove nosnippet / max-snippet:0 unless you mean to block it."
	}
	add(snippet)
	facts := PageCheck{ID: "facts", Category: CategoryAI, Weight: 1, Title: "Facts and numbers",
		Why: "Prices, percentages and figures give AI something specific to quote.", Status: CheckPass,
		Found: fmt.Sprintf("%d figures (prices, percentages, amounts).", ai.Statistics)}
	if ai.Statistics == 0 {
		facts.Status, facts.Found, facts.Fix = CheckWarn, "No prices, percentages or figures.", "Add concrete facts: prices, years in business, customers served, delivery times."
	}
	add(facts)
	trust := PageCheck{ID: "author-date", Category: CategoryAI, Weight: 1, Title: "Author or date",
		Why: "A named author or an updated date tells AI and readers the content is real and current.", Status: CheckPass}
	switch {
	case ai.Author && (ai.Modified != "" || ai.Published != ""):
		trust.Found = "Author and date are stated."
	case ai.Author:
		trust.Found = "Author is stated."
	case ai.Modified != "" || ai.Published != "":
		trust.Found = "Date is stated: " + firstNonEmpty(ai.Modified, ai.Published) + "."
	default:
		trust.Status, trust.Found = CheckWarn, "No author or date found."
		trust.Fix = "Show who is behind the content and when it was last updated (and add them to your schema)."
	}
	add(trust)
	llmsCheck := PageCheck{ID: "llms-txt", Category: CategoryAI, Weight: 1, Title: "llms.txt file",
		Why: "llms.txt is an optional, plain-text map of your key pages for AI tools. Few read it yet.", Status: CheckPass, Found: origin + "/llms.txt exists."}
	if !llms {
		llmsCheck.Status, llmsCheck.Found = CheckWarn, "No /llms.txt."
		llmsCheck.Fix = "Optional: add /llms.txt listing your main pages, one line each on what the page answers."
	}
	add(llmsCheck)

	// Technical.
	https := PageCheck{ID: "https", Category: CategoryTechnical, Weight: 3, Title: "HTTPS",
		Why: "Browsers warn visitors on sites without HTTPS, and Google prefers secure pages.", Status: CheckPass, Found: "The page loads over HTTPS."}
	if !strings.HasPrefix(page.URL, "https://") {
		https.Status, https.Found, https.Fix = CheckFail, "The page loads over plain HTTP.", "Install a TLS certificate and redirect HTTP to HTTPS."
	}
	add(https)
	speed := PageCheck{ID: "response-time", Category: CategoryTechnical, Weight: 2, Title: "Server response time",
		Why:   "A slow server delays everything else and can limit how much crawlers fetch.",
		Found: fmt.Sprintf("The HTML arrived in %s ms (from our server).", strconv.FormatInt(page.ResponseTimeMs, 10))}
	switch {
	case page.ResponseTimeMs <= 600:
		speed.Status = CheckPass
	case page.ResponseTimeMs <= slowResponseReportMs:
		speed.Status, speed.Fix = CheckWarn, "Cache the page HTML or use a CDN so it is served faster."
	default:
		speed.Status, speed.Fix = CheckFail, "Cache the page HTML or use a CDN; check slow database queries and plugins."
	}
	add(speed)
	viewport := PageCheck{ID: "viewport", Category: CategoryTechnical, Weight: 2, Title: "Mobile friendly viewport",
		Why: "Without a viewport tag, phones show the desktop layout zoomed out.", Status: CheckPass, Found: "Viewport meta tag is set."}
	if tech.Measured && !tech.Viewport {
		viewport.Status, viewport.Found, viewport.Fix = CheckFail, "No viewport meta tag.", "Add <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">."
	}
	add(viewport)
	lang := PageCheck{ID: "language", Category: CategoryTechnical, Weight: 1, Title: "Page language",
		Why: "The lang attribute tells search engines and AI which language the page is in.", Status: CheckPass, Found: "lang=\"" + tech.HTMLLanguage + "\""}
	if tech.Measured && !validHTMLLanguage(tech.HTMLLanguage) {
		lang.Status, lang.Found, lang.Fix = CheckWarn, "No valid lang attribute on <html>.", "Add a language such as lang=\"en\" to the <html> tag."
	}
	add(lang)
	robotsCheck := PageCheck{ID: "robots-txt", Category: CategoryTechnical, Weight: 1, Title: "robots.txt",
		Why: "robots.txt tells crawlers what they may read and where your sitemap is.", Status: CheckPass, Found: origin + "/robots.txt exists."}
	if robots == nil {
		robotsCheck.Status, robotsCheck.Found, robotsCheck.Fix = CheckWarn, "No robots.txt found.", "Add a robots.txt that allows crawling and links your sitemap."
	}
	add(robotsCheck)
	sitemapCheck := PageCheck{ID: "sitemap", Category: CategoryTechnical, Weight: 2, Title: "XML sitemap",
		Why: "A sitemap helps Google and Bing find every page you want indexed.", Status: CheckPass, Found: "Sitemap found: " + sitemap}
	if sitemap == "" {
		sitemapCheck.Status, sitemapCheck.Found = CheckWarn, "No XML sitemap at /sitemap.xml or in robots.txt."
		sitemapCheck.Fix = "Create an XML sitemap (most CMSs and SEO plugins make one), link it in robots.txt and submit it in Search Console."
	}
	add(sitemapCheck)
	secure := PageCheck{ID: "mixed-content", Category: CategoryTechnical, Weight: 1, Title: "Secure links and forms",
		Why: "Forms or links to http:// pages can expose visitors' data.", Status: CheckPass, Found: "No insecure forms or http:// links."}
	insecureLinks := 0
	for _, link := range page.Links {
		if strings.HasPrefix(strings.ToLower(link.TargetURL), "http://") {
			insecureLinks++
		}
	}
	switch {
	case len(tech.InsecureForms) > 0:
		secure.Status, secure.Found, secure.Fix = CheckFail, fmt.Sprintf("%d forms submit over plain HTTP.", len(tech.InsecureForms)), "Send every form to an https:// address."
	case insecureLinks > 0 && strings.HasPrefix(page.URL, "https://"):
		secure.Status, secure.Found, secure.Fix = CheckWarn, fmt.Sprintf("%d links point to http:// addresses.", insecureLinks), "Link to the https:// version of each page."
	}
	add(secure)
	sizes := PageCheck{ID: "image-size", Category: CategoryTechnical, Weight: 1, Title: "Image dimensions",
		Why: "Images without width and height make the layout jump while the page loads.", Status: CheckPass, Found: "Images declare their size."}
	if tech.ImagesWithoutSize > 0 {
		sizes.Status, sizes.Found, sizes.Fix = CheckWarn, fmt.Sprintf("%d images have no width or height.", tech.ImagesWithoutSize), "Set width and height on each <img>."
	}
	add(sizes)

	// Content.
	words := PageCheck{ID: "word-count", Category: CategoryContent, Weight: 3, Title: "Amount of content",
		Why:   "Pages with very little text rarely rank or get quoted; there is not enough to understand.",
		Found: fmt.Sprintf("%d words of text.", page.WordCount)}
	switch {
	case page.WordCount < thinContentWords:
		words.Status, words.Fix = CheckFail, "Explain what you offer, who it is for, prices, process and common questions in at least 300 words."
	case page.WordCount < 2*thinContentWords:
		words.Status, words.Fix = CheckWarn, "Add more useful detail: services, prices, process, proof and FAQs."
	default:
		words.Status = CheckPass
	}
	add(words)
	internal := 0
	for _, link := range page.Links {
		if link.IsInternal {
			internal++
		}
	}
	links := PageCheck{ID: "internal-links", Category: CategoryContent, Weight: 2, Title: "Links to your other pages",
		Why: "Internal links help visitors and crawlers reach your important pages.", Found: fmt.Sprintf("%d links to your own pages.", internal)}
	switch {
	case internal == 0:
		links.Status, links.Fix = CheckFail, "Link to your main services, products, about and contact pages."
	case internal < 5:
		links.Status, links.Fix = CheckWarn, "Link to more of your key pages from this one."
	default:
		links.Status = CheckPass
	}
	add(links)
	structure := PageCheck{ID: "structure", Category: CategoryContent, Weight: 1, Title: "Lists and tables",
		Why: "Lists and tables are easy to skim and easy for AI to turn into answers.", Status: CheckPass,
		Found: fmt.Sprintf("%d lists and %d tables.", ai.Lists, ai.Tables)}
	if ai.Lists+ai.Tables == 0 {
		structure.Status, structure.Fix = CheckWarn, "Use a bulleted list or table for features, prices or steps."
	}
	add(structure)
	return checks
}

// lengthCheck checks a title-like text for presence and length in characters.
func lengthCheck(id, category string, weight int, title, value string, minChars, maxChars int, why, fix string) PageCheck {
	check := PageCheck{ID: id, Category: category, Weight: weight, Title: title, Why: why}
	length := utf8.RuneCountInString(value)
	switch {
	case value == "":
		check.Status, check.Found, check.Fix = CheckFail, "Missing.", fix
	case length > maxChars:
		check.Status, check.Found, check.Fix = CheckWarn, fmt.Sprintf("%q (%d characters, longer than %d; Google may cut it).", value, length, maxChars), fix
	case length < minChars:
		check.Status, check.Found, check.Fix = CheckWarn, fmt.Sprintf("%q (%d characters, too short to describe the page).", value, length), fix
	default:
		check.Status, check.Found = CheckPass, fmt.Sprintf("%q (%d characters).", value, length)
	}
	return check
}

func canonicalCheck(page CrawledPageResult) PageCheck {
	check := PageCheck{ID: "canonical", Category: CategorySEO, Weight: 1, Title: "Canonical URL",
		Why: "The canonical tag tells Google which address is the main one, so duplicates do not compete."}
	canonical := firstNonEmpty(page.CanonicalURL, page.HeaderCanonicalURL)
	switch {
	case canonical == "":
		check.Status, check.Found = CheckWarn, "No canonical tag."
		check.Fix = "Add <link rel=\"canonical\" href=\"" + page.URL + "\"> to the page head."
	case canonicalIsRelative(page.CanonicalURL):
		check.Status, check.Found = CheckWarn, "Relative canonical: "+canonical
		check.Fix = "Use the full address, starting with https://."
	case normalizedCanonical(canonical, page.URL) != normalizedCanonical(page.URL, page.URL):
		check.Status, check.Found = CheckWarn, "Points to another address: "+canonical
		check.Fix = "If this page should rank itself, set its canonical to its own address."
	default:
		check.Status, check.Found = CheckPass, "Points to itself."
	}
	return check
}

func aiCrawlerCheck(origin string, robots *string) PageCheck {
	check := PageCheck{ID: "ai-crawlers", Category: CategoryAI, Weight: 3, Title: "AI crawlers allowed",
		Why: "ChatGPT, Perplexity, Claude and Copilot can only cite pages their crawlers may read.", Status: CheckPass,
		Found: "robots.txt lets the AI search crawlers in."}
	var search, training []string
	for _, crawler := range blockedAICrawlers(origin, robots) {
		if IsAISearchBot(crawler) {
			search = append(search, crawler)
		} else {
			training = append(training, crawler)
		}
	}
	switch {
	case len(search) > 0:
		check.Status, check.Found = CheckFail, "robots.txt blocks: "+strings.Join(search, ", ")
		check.Fix = "Remove the Disallow rules for these user agents in robots.txt, and check your firewall does not block them."
	case len(training) > 0:
		check.Found = "AI search crawlers may read the site. Training-only crawlers are blocked (" + strings.Join(training, ", ") + "); that is a choice, not an error."
	}
	return check
}

func schemaCheck(types []string) PageCheck {
	check := PageCheck{ID: "business-schema", Category: CategoryAI, Weight: 3, Title: "Who you are (schema)",
		Why: "Organization or LocalBusiness schema tells Google and AI your exact name, address, phone and profiles."}
	business := slices.ContainsFunc(types, func(t string) bool { return businessTypes[strings.ToLower(t)] })
	found := "No structured data."
	if len(types) > 0 {
		found = "Structured data found: " + strings.Join(types, ", ") + "."
	}
	check.Found = found
	switch {
	case business:
		check.Status = CheckPass
	case len(types) > 0:
		check.Status, check.Fix = CheckWarn, "Add Organization or LocalBusiness JSON-LD with your name, logo, address, phone and social profiles (sameAs)."
	default:
		check.Status, check.Fix = CheckFail, "Add Organization or LocalBusiness JSON-LD with your name, logo, address, phone and social profiles (sameAs)."
	}
	return check
}

// scoreChecks scores each category from its weighted checks (pass counts
// fully, a warning half) and the page as the average of its categories.
func scoreChecks(checks []PageCheck) ([]CategoryScore, int) {
	var categories []CategoryScore
	total := 0
	for _, id := range []string{CategorySEO, CategoryAI, CategoryTechnical, CategoryContent} {
		category := CategoryScore{ID: id, Name: categoryNames[id]}
		earned, possible := 0, 0
		for _, check := range checks {
			if check.Category != id || check.Status == CheckInfo {
				continue
			}
			possible += 2 * check.Weight
			switch check.Status {
			case CheckPass:
				earned += 2 * check.Weight
				category.Passed++
			case CheckWarn:
				earned += check.Weight
				category.Issues++
			case CheckFail:
				category.Issues++
			case CheckInfo:
			}
		}
		if possible > 0 {
			category.Score = (100*earned + possible/2) / possible
		}
		total += category.Score
		categories = append(categories, category)
	}
	return categories, (total + len(categories)/2) / len(categories)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
