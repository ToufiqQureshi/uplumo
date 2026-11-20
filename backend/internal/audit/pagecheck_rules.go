package audit

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// The checks of the free one-page audit. Each one reads what the crawl
// already fetched; none calls a provider. Thresholds are the free audit's own
// (the full audit keeps its looser ones in pagereporters.go).

const (
	freeTitleMinChars       = 30
	freeTitleMaxChars       = 60
	freeDescriptionMinChars = 70
	freeDescriptionMaxChars = 160
	// freeThinWords is below which a page is "thin": the word-count check
	// fails and sections are not expected.
	freeThinWords = 300
	// freeFullWords is where the word-count check passes.
	freeFullWords       = 600
	freeMinInternalLink = 3
	freeGoodInternal    = 10
	freeMaxURLChars     = 100
	freeStaleMonths     = 18
	freeHTMLWarnBytes   = 500 * 1024
	// freeImageAltFailRatio is the share of images without alt text that fails.
	freeImageAltFailRatio = 0.2
	freeImageSizeFewMax   = 10
	freeReadabilityOK     = 20
	freeReadabilityWarn   = 25
	freeReadabilityMin    = 100
	freeResponseOKms      = 400
	freeResponseSlowMs    = 1000
	freeFactsGood         = 3
	freeFAQGood           = 3
	freeChunkReadyRatio   = 0.3
	freeSameAsGood        = 2
	// freeMaxSnippet bounds the evidence text shown in a check.
	freeMaxSnippet = 80
)

// pageInputs is everything the checks read for one page.
type pageInputs struct {
	page      CrawledPageResult
	analysis  PageAnalysis
	htmlBytes int
	origin    string
	robots    *string
	llms      bool
	sitemap   string
	// field is the Chrome field data for the page; nil when no source is set.
	field *StoredFieldData
	now   time.Time
}

// newCheck starts a check. Its weight comes from its severity.
func newCheck(id, category, severity, title, why string) PageCheck {
	return PageCheck{ID: id, Category: category, Severity: severity, Weight: severityWeights[severity],
		Title: title, Why: why, Status: CheckPass}
}

// set records the outcome of a check.
func (c *PageCheck) set(status CheckStatus, found, fix string) {
	c.Status, c.Found, c.Fix = status, found, fix
}

// pageChecks turns one fetched page and its site files into the report's
// checks, in the order the report shows them within each category.
func pageChecks(in pageInputs) []PageCheck {
	page := in.page
	ai := page.AI
	tech := page.Technical
	h1 := ""
	if len(in.analysis.H1s) > 0 {
		h1 = in.analysis.H1s[0]
	}
	var checks []PageCheck
	add := func(check PageCheck) { checks = append(checks, check) }

	// On-page SEO.
	add(titleCheck(page.Title, h1))
	add(descriptionCheck(page.MetaDescription))
	add(h1Check(page.H1Count, in.analysis.H1s, pageHost(page.URL)))
	add(headingOrderCheck(page))
	add(h2Check(page.H2Count, page.WordCount))
	add(canonicalCheck(page))
	add(duplicateTagsCheck(tech))
	add(indexableCheck(page))
	add(openGraphCheck(page))
	add(imageAltCheck(page))
	add(urlHygieneCheck(page.URL))

	// AI search readiness.
	add(aiCrawlerCheck(in.origin, in.robots))
	add(businessSchemaCheck(ai))
	add(faqCheck(ai))
	add(answerFirstCheck(ai))
	add(chunkReadyCheck(ai, page.WordCount))
	add(noJavaScriptCheck(page))
	add(snippetCheck(ai))
	add(hiddenInstructionsCheck(ai))
	add(factsCheck(ai))
	add(authorityCheck(ai))
	add(authorDateCheck(ai))
	add(freshnessCheck(ai, in.now))
	add(llmsCheck(in.origin, in.llms))

	// Technical.
	add(httpsCheck(page.URL))
	add(hstsCheck(ai))
	add(responseTimeCheck(page.ResponseTimeMs))
	add(viewportCheck(tech))
	add(languageCheck(tech))
	add(robotsCheck(in.origin, in.robots))
	add(sitemapCheck(in.sitemap))
	add(insecureLinksCheck(page, tech))
	add(imageSizeCheck(tech))
	add(htmlSizeCheck(in.htmlBytes))
	add(webVitalsCheck(in.field))

	// Content.
	add(wordCountCheck(page.WordCount))
	add(internalLinksCheck(page.Links))
	add(structureCheck(ai))
	add(readabilityCheck(in.analysis.BodyText))
	return checks
}

func titleCheck(title, h1 string) PageCheck {
	c := newCheck("title", CategorySEO, sevHigh, "Page title",
		"The title is the blue link people click in Google and the name AI assistants give your page.")
	const fix = "Write one title of 30-60 characters: what the page offers, then your brand. Use a main word from your main heading."
	length := utf8.RuneCountInString(title)
	switch {
	case title == "":
		c.set(CheckFail, "Missing.", fix)
	case length < freeTitleMinChars || length > freeTitleMaxChars:
		c.set(CheckWarn, fmt.Sprintf("%q (%d characters; aim for %d-%d).", title, length, freeTitleMinChars, freeTitleMaxChars), fix)
	case h1 != "" && !sharesWord(title, h1):
		c.set(CheckWarn, fmt.Sprintf("%q shares no word with the main heading %q.", title, h1), fix)
	default:
		c.set(CheckPass, fmt.Sprintf("%q (%d characters).", title, length), "")
	}
	return c
}

func descriptionCheck(description string) PageCheck {
	return lengthCheck("description", CategorySEO, sevMedium, "Meta description", description,
		freeDescriptionMinChars, freeDescriptionMaxChars,
		"Google often shows it under your title. A clear one gets more clicks.",
		"Write 70-160 characters that say what the visitor gets and why to choose you.")
}

// lengthCheck checks a text for presence and length in characters.
func lengthCheck(id, category, severity, title, value string, minChars, maxChars int, why, fix string) PageCheck {
	c := newCheck(id, category, severity, title, why)
	length := utf8.RuneCountInString(value)
	switch {
	case value == "":
		c.set(CheckFail, "Missing.", fix)
	case length > maxChars:
		c.set(CheckWarn, fmt.Sprintf("%q (%d characters, longer than %d; Google may cut it).", value, length, maxChars), fix)
	case length < minChars:
		c.set(CheckWarn, fmt.Sprintf("%q (%d characters, too short to describe the page).", value, length), fix)
	default:
		c.set(CheckPass, fmt.Sprintf("%q (%d characters).", value, length), "")
	}
	return c
}

func h1Check(count int, headings []string, host string) PageCheck {
	c := newCheck("h1", CategorySEO, sevHigh, "Main heading (H1)",
		"One H1 tells readers and search engines what the page is about.")
	text := ""
	if len(headings) > 0 {
		text = strings.TrimSpace(headings[0])
	}
	switch {
	case count == 0:
		c.set(CheckFail, "No H1 heading on the page.", "Add one H1 that states the page's main topic, in line with the title.")
	case count > 1:
		c.set(CheckWarn, fmt.Sprintf("%d H1 headings.", count), "Keep one H1 and turn the others into H2 or H3.")
	case text == "":
		c.set(CheckFail, "The H1 heading is empty.", "Put the page's main topic in the H1.")
	case sameName(text, host):
		c.set(CheckWarn, fmt.Sprintf("The H1 is only the site name (%q).", text),
			"Name the topic in the H1, for example what the page sells or answers, and keep the brand for the title.")
	default:
		c.set(CheckPass, fmt.Sprintf("One H1: %q.", text), "")
	}
	return c
}

func headingOrderCheck(page CrawledPageResult) PageCheck {
	c := newCheck("heading-order", CategorySEO, sevLow, "Heading structure",
		"Headings in order (H1, then H2, then H3) make the page easy to scan for people and machines.")
	c.set(CheckPass, fmt.Sprintf("%d H2 and %d H3 headings, in order.", page.H2Count, page.H3Count), "")
	if hasHeadingLevelSkip(page.HeadingOrder) {
		c.set(CheckWarn, "Heading levels skip a step (for example H1 to H3).", "Change heading levels so they go down one step at a time.")
	}
	return c
}

func h2Check(h2s, words int) PageCheck {
	c := newCheck("h2-sections", CategorySEO, sevMedium, "Sections on longer pages",
		"Longer pages need H2 sections, so each topic can be found and quoted on its own.")
	const fix = "Split the text into at least two H2 sections, each with a heading that names its topic."
	switch {
	case words < freeThinWords:
		c.set(CheckInfo, "Short page: fewer than 300 words, so sections are optional.", "")
	case h2s < 2:
		c.set(CheckWarn, fmt.Sprintf("%d H2 heading(s) in %d words.", h2s, words), fix)
	default:
		c.set(CheckPass, fmt.Sprintf("%d H2 headings in %d words.", h2s, words), "")
	}
	return c
}

func duplicateTagsCheck(tech TechnicalSignals) PageCheck {
	c := newCheck("duplicate-tags", CategorySEO, sevHigh, "Duplicate tags",
		"Two titles, descriptions or canonical tags make search engines guess which one counts.")
	var repeated []string
	if tech.Titles > 1 {
		repeated = append(repeated, "title")
	}
	if tech.Descriptions > 1 {
		repeated = append(repeated, "meta description")
	}
	if tech.Canonicals > 1 {
		repeated = append(repeated, "canonical")
	}
	if len(repeated) > 0 {
		c.set(CheckFail, "More than one tag on the page: "+strings.Join(repeated, ", ")+".",
			"Keep one of each tag. Remove the extra ones from your theme or SEO plugin.")
		return c
	}
	c.set(CheckPass, "One title, one meta description and one canonical.", "")
	return c
}

func indexableCheck(page CrawledPageResult) PageCheck {
	c := newCheck("indexable", CategorySEO, sevCritical, "Allowed in Google",
		"A noindex rule removes the page from Google and AI search results.")
	if !page.IsIndexable {
		c.set(CheckFail, "The page says noindex: "+strings.TrimSpace(page.RobotsMeta+" "+page.XRobotsTag),
			"Remove noindex from the robots meta tag or X-Robots-Tag header if this page should be found.")
		return c
	}
	c.set(CheckPass, "No noindex rule.", "")
	return c
}

func openGraphCheck(page CrawledPageResult) PageCheck {
	c := newCheck("open-graph", CategorySEO, sevLow, "Social sharing preview",
		"Open Graph tags decide the title, text and image shown when your link is shared on WhatsApp, LinkedIn or X.")
	var missing []string
	if page.OgTitle == "" {
		missing = append(missing, "og:title")
	}
	if page.OgDescription == "" {
		missing = append(missing, "og:description")
	}
	if page.OgImage == "" {
		missing = append(missing, "og:image")
	}
	const fix = "Add og:title, og:description and og:image meta tags."
	switch len(missing) {
	case 0:
		c.set(CheckPass, "og:title, og:description and og:image are set.", "")
	case 3:
		c.set(CheckFail, "No Open Graph tags.", fix)
	default:
		c.set(CheckWarn, "Missing: "+strings.Join(missing, ", ")+".", fix)
	}
	return c
}

func imageAltCheck(page CrawledPageResult) PageCheck {
	c := newCheck("image-alt", CategorySEO, sevMedium, "Image alt text",
		"Alt text lets search engines, AI and screen readers understand your images.")
	const fix = "Describe each meaningful image in a few words; use alt=\"\" only for decoration."
	switch {
	case page.ImagesTotal == 0:
		c.set(CheckInfo, "No images on the page.", "")
	case page.ImagesMissingAlt == 0:
		c.set(CheckPass, fmt.Sprintf("All %d images have alt text.", page.ImagesTotal), "")
	case float64(page.ImagesMissingAlt)/float64(page.ImagesTotal) > freeImageAltFailRatio:
		c.set(CheckFail, fmt.Sprintf("%d of %d images have no alt text.", page.ImagesMissingAlt, page.ImagesTotal), fix)
	default:
		c.set(CheckWarn, fmt.Sprintf("%d of %d images have no alt text.", page.ImagesMissingAlt, page.ImagesTotal), fix)
	}
	return c
}

func urlHygieneCheck(pageURL string) PageCheck {
	c := newCheck("url-hygiene", CategorySEO, sevLow, "Clean web address",
		"Short, lowercase addresses without query strings are easier to share, remember and index.")
	parsed, err := url.Parse(pageURL)
	if err != nil {
		c.set(CheckInfo, "The address could not be read.", "")
		return c
	}
	var problems []string
	if parsed.Path != strings.ToLower(parsed.Path) {
		problems = append(problems, "uppercase letters")
	}
	if strings.Contains(parsed.Path, "_") {
		problems = append(problems, "underscores (use hyphens)")
	}
	if utf8.RuneCountInString(pageURL) > freeMaxURLChars {
		problems = append(problems, fmt.Sprintf("more than %d characters", freeMaxURLChars))
	}
	if parsed.RawQuery != "" {
		problems = append(problems, "query parameters")
	}
	if len(problems) > 0 {
		c.set(CheckWarn, "Address has "+strings.Join(problems, ", ")+".",
			"Use short lowercase paths with hyphens, and avoid query parameters on pages you want found.")
		return c
	}
	c.set(CheckPass, "Short, lowercase address without query parameters.", "")
	return c
}

func canonicalCheck(page CrawledPageResult) PageCheck {
	c := newCheck("canonical", CategorySEO, sevMedium, "Canonical URL",
		"The canonical tag tells Google which address is the main one, so duplicates do not compete.")
	canonical := firstNonEmpty(page.CanonicalURL, page.HeaderCanonicalURL)
	switch {
	case canonical == "":
		c.set(CheckWarn, "No canonical tag.", "Add <link rel=\"canonical\" href=\""+page.URL+"\"> to the page head.")
	case canonicalIsRelative(page.CanonicalURL):
		c.set(CheckWarn, "Relative canonical: "+canonical, "Use the full address, starting with https://.")
	case normalizedCanonical(canonical, page.URL) != normalizedCanonical(page.URL, page.URL):
		c.set(CheckWarn, "Points to another address: "+canonical,
			"If this page should rank itself, set its canonical to its own address.")
	default:
		c.set(CheckPass, "Points to itself.", "")
	}
	return c
}

func aiCrawlerCheck(origin string, robots *string) PageCheck {
	c := newCheck("ai-crawlers", CategoryAI, sevCritical, "AI crawlers allowed",
		"ChatGPT, Perplexity, Claude and Copilot can only cite pages their crawlers may read.")
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
		c.set(CheckFail, "robots.txt blocks: "+strings.Join(search, ", ")+".",
			"Remove the Disallow rules for these user agents in robots.txt, and check your firewall does not block them.")
	case len(training) > 0:
		c.set(CheckPass, "AI search crawlers may read the site. Training-only crawlers are blocked ("+strings.Join(training, ", ")+"); that is a choice, not an error.", "")
	default:
		c.set(CheckPass, "robots.txt lets the AI search crawlers in.", "")
	}
	return c
}

func businessSchemaCheck(ai AIPageSignals) PageCheck {
	c := newCheck("business-schema", CategoryAI, sevHigh, "Who you are (schema)",
		"Organization or LocalBusiness schema tells Google and AI your exact name, logo, contact details and profiles.")
	const fix = "Add Organization or LocalBusiness JSON-LD with name, logo, address, phone and at least two sameAs profile links."
	business := slices.ContainsFunc(ai.SchemaTypes, func(t string) bool { return businessTypes[strings.ToLower(t)] })
	if !business {
		found := "No Organization or LocalBusiness schema."
		if len(ai.SchemaTypes) > 0 {
			found = "Structured data found (" + strings.Join(ai.SchemaTypes, ", ") + "), but no Organization or LocalBusiness."
		}
		c.set(CheckFail, found, fix)
		return c
	}
	have := map[string]bool{}
	for _, field := range ai.BusinessFields {
		have[field] = true
	}
	var missing []string
	if !have["name"] {
		missing = append(missing, "name")
	}
	if !have["logo"] {
		missing = append(missing, "logo")
	}
	if ai.SameAs < freeSameAsGood {
		missing = append(missing, fmt.Sprintf("sameAs profile links (%d of %d)", ai.SameAs, freeSameAsGood))
	}
	if !have["telephone"] && !have["contactPoint"] {
		missing = append(missing, "phone or contactPoint")
	}
	if !have["address"] {
		missing = append(missing, "address")
	}
	if len(missing) > 0 {
		c.set(CheckWarn, "Business schema is missing: "+strings.Join(missing, ", ")+".", fix)
		return c
	}
	c.set(CheckPass, "Business schema has name, logo, contact, address and profile links.", "")
	return c
}

func faqCheck(ai AIPageSignals) PageCheck {
	c := newCheck("faq", CategoryAI, sevHigh, "Answers to customer questions",
		"AI assistants quote pages that ask a real question and answer it right below.")
	const fix = "Add a short FAQ: the questions customers ask as H2 or H3 headings, each answered in the first sentence below it."
	faq := slices.ContainsFunc(ai.SchemaTypes, func(t string) bool { return strings.EqualFold(t, "FAQPage") })
	switch {
	case faq:
		c.set(CheckPass, fmt.Sprintf("FAQ schema and %d question headings.", ai.QuestionHeadings), "")
	case ai.QuestionHeadings >= freeFAQGood:
		c.set(CheckPass, fmt.Sprintf("%d question headings.", ai.QuestionHeadings), "")
	case ai.QuestionHeadings > 0:
		c.set(CheckWarn, fmt.Sprintf("%d question heading(s); three or more answer more of the questions buyers ask.", ai.QuestionHeadings), fix)
	default:
		c.set(CheckFail, "No headings phrased as a question and no FAQ schema.", fix)
	}
	return c
}

func answerFirstCheck(ai AIPageSignals) PageCheck {
	c := newCheck("answer-first", CategoryAI, sevMedium, "Direct answer up front",
		"AI answers lift short, factual paragraphs (30-120 words) that start with a statement.")
	if ai.AnswerCapsule {
		c.set(CheckPass, "At least one paragraph gives a direct, factual answer.", "")
		return c
	}
	c.set(CheckWarn, "No short paragraph that answers a question with a concrete fact.",
		"Open the page with 2-3 sentences that say what you offer, for whom, where and at what price.")
	return c
}

func chunkReadyCheck(ai AIPageSignals, words int) PageCheck {
	c := newCheck("chunk-ready", CategoryAI, sevMedium, "Sections that stand alone",
		"AI assistants lift one section at a time. Short sections that start with a clear statement are the ones they quote.")
	const fix = "Give each H2 section a short, self-contained first paragraph that answers its heading."
	ratio := 0.0
	if ai.Sections > 0 {
		ratio = float64(ai.ChunkReadySections) / float64(ai.Sections)
	}
	switch {
	case ai.Sections == 0 && words < freeThinWords:
		c.set(CheckInfo, "Too short to split into sections.", "")
	case ai.Sections == 0 || ai.ChunkReadySections == 0:
		c.set(CheckFail, fmt.Sprintf("None of the %d sections is short and self-contained enough to quote.", ai.Sections), fix)
	case ratio >= freeChunkReadyRatio:
		c.set(CheckPass, fmt.Sprintf("%d of %d sections stand alone.", ai.ChunkReadySections, ai.Sections), "")
	default:
		c.set(CheckWarn, fmt.Sprintf("Only %d of %d sections stand alone.", ai.ChunkReadySections, ai.Sections), fix)
	}
	return c
}

func noJavaScriptCheck(page CrawledPageResult) PageCheck {
	c := newCheck("no-javascript", CategoryAI, sevHigh, "Readable without JavaScript",
		"Most AI crawlers do not run JavaScript; they only see the HTML your server sends.")
	if page.JavaScriptShell {
		c.set(CheckFail, "The page is an empty shell until JavaScript runs.",
			"Use server-side rendering or prerendering so the text is in the first HTML response.")
		return c
	}
	c.set(CheckPass, fmt.Sprintf("%d words are in the HTML the server sends.", page.WordCount), "")
	return c
}

func snippetCheck(ai AIPageSignals) PageCheck {
	c := newCheck("snippet", CategoryAI, sevHigh, "Allowed to be quoted",
		"nosnippet or max-snippet:0 stops Google and AI answers from quoting your text.")
	if ai.SnippetBlocked {
		c.set(CheckFail, "A nosnippet rule blocks quoting.", "Remove nosnippet / max-snippet:0 unless you mean to block it.")
		return c
	}
	c.set(CheckPass, "No rule blocks quoting.", "")
	return c
}

func hiddenInstructionsCheck(ai AIPageSignals) PageCheck {
	c := newCheck("hidden-instructions", CategoryAI, sevCritical, "No hidden instructions to AI",
		"Text visitors cannot see that tells AI assistants what to say is manipulation. Search engines and AI can distrust or drop such pages.")
	if len(ai.InjectionEvidence) == 0 {
		c.set(CheckPass, "No hidden text aimed at AI assistants.", "")
		return c
	}
	sample := []rune(ai.InjectionEvidence[0].Text)
	if len(sample) > freeMaxSnippet {
		sample = sample[:freeMaxSnippet]
	}
	c.set(CheckFail, fmt.Sprintf("%d hidden instruction(s) aimed at AI assistants, for example: %q.", len(ai.InjectionEvidence), string(sample)),
		"Remove the hidden text. Write only what visitors can read.")
	return c
}

func factsCheck(ai AIPageSignals) PageCheck {
	c := newCheck("facts", CategoryAI, sevMedium, "Facts and numbers",
		"Prices, percentages and figures give AI something specific to quote.")
	const fix = "Add concrete facts: prices, years in business, customers served, delivery times."
	switch {
	case ai.Statistics >= freeFactsGood:
		c.set(CheckPass, fmt.Sprintf("%d figures (prices, percentages, amounts).", ai.Statistics), "")
	case ai.Statistics > 0:
		c.set(CheckWarn, fmt.Sprintf("Only %d figure(s); three or more give AI something specific to quote.", ai.Statistics), fix)
	default:
		c.set(CheckFail, "No prices, percentages or figures.", fix)
	}
	return c
}

func authorityCheck(ai AIPageSignals) PageCheck {
	c := newCheck("authority-links", CategoryAI, sevMedium, "Links to sources",
		"Pages that link to the sources behind their facts look more trustworthy to AI and readers.")
	if ai.AuthorityLinks >= 1 {
		c.set(CheckPass, fmt.Sprintf("%d link(s) to authoritative sources.", ai.AuthorityLinks), "")
		return c
	}
	c.set(CheckWarn, "No links to government, academic or other authoritative sources.",
		"Link the facts you state to their source, for example a government page, a standard or a study.")
	return c
}

func authorDateCheck(ai AIPageSignals) PageCheck {
	c := newCheck("author-date", CategoryAI, sevMedium, "Author and date",
		"A named author and a date tell AI and readers that the content is real and current.")
	const fix = "Show who wrote the page and when it was published or last updated, and add both to your schema."
	date := firstNonEmpty(ai.Modified, ai.Published) != ""
	switch {
	case ai.Author && date:
		c.set(CheckPass, "Author and date are stated.", "")
	case ai.Author:
		c.set(CheckWarn, "Author is stated, but no date.", fix)
	case date:
		c.set(CheckWarn, "Date is stated, but no author.", fix)
	default:
		c.set(CheckFail, "No author and no date.", fix)
	}
	return c
}

func freshnessCheck(ai AIPageSignals, now time.Time) PageCheck {
	c := newCheck("freshness", CategoryAI, sevMedium, "Content is up to date",
		"Visible, recent dates tell AI and readers the answer is current. Old dates suggest it may be out of date.")
	const fix = "Show the last updated date on the page and update it when the facts change."
	date := firstNonEmpty(ai.Modified, ai.Published)
	if date == "" {
		c.set(CheckWarn, "No date found on the page.", fix)
		return c
	}
	parsed, ok := parseLooseDate(date)
	if !ok {
		c.set(CheckWarn, fmt.Sprintf("The date %q could not be read.", date), "Write the date as YYYY-MM-DD.")
		return c
	}
	if parsed.Before(now.AddDate(0, -freeStaleMonths, 0)) {
		c.set(CheckWarn, fmt.Sprintf("Last dated %s, more than %d months ago.", date, freeStaleMonths), fix)
		return c
	}
	c.set(CheckPass, "Last dated "+date+".", "")
	return c
}

// parseLooseDate reads the date formats pages state (ISO 8601 and the common
// written forms); ok is false for anything else.
func parseLooseDate(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	layouts := []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02", "2 January 2006", "January 2, 2006", "2 Jan 2006", "Jan 2, 2006"}
	for _, candidate := range []string{value, prefixRunes(value, 10)} {
		for _, layout := range layouts {
			if parsed, err := time.Parse(layout, candidate); err == nil {
				return parsed, true
			}
		}
	}
	return time.Time{}, false
}

func llmsCheck(origin string, present bool) PageCheck {
	c := newCheck("llms-txt", CategoryAI, sevLow, "llms.txt file",
		"llms.txt is an optional, plain-text map of your key pages for AI tools. Few read it yet.")
	if present {
		c.set(CheckPass, origin+"/llms.txt exists.", "")
		return c
	}
	c.set(CheckWarn, "No /llms.txt.", "Optional: add /llms.txt listing your main pages, one line each on what the page answers.")
	return c
}

func httpsCheck(pageURL string) PageCheck {
	c := newCheck("https", CategoryTechnical, sevCritical, "HTTPS",
		"Browsers warn visitors on sites without HTTPS, and Google prefers secure pages.")
	if strings.HasPrefix(pageURL, "https://") {
		c.set(CheckPass, "The page loads over HTTPS.", "")
		return c
	}
	c.set(CheckFail, "The page loads over plain HTTP.", "Install a TLS certificate and redirect HTTP to HTTPS.")
	return c
}

func hstsCheck(ai AIPageSignals) PageCheck {
	c := newCheck("hsts", CategoryTechnical, sevLow, "HSTS header",
		"HSTS tells browsers to use HTTPS only, so visitors cannot be sent to a plain-HTTP copy of the site.")
	switch {
	case ai.HSTS == nil:
		c.set(CheckInfo, "Not measured.", "")
	case *ai.HSTS:
		c.set(CheckPass, "Strict-Transport-Security is set.", "")
	default:
		c.set(CheckWarn, "No Strict-Transport-Security header.",
			"Send Strict-Transport-Security: max-age=31536000 once HTTPS works on every page.")
	}
	return c
}

func responseTimeCheck(ms int64) PageCheck {
	c := newCheck("response-time", CategoryTechnical, sevMedium, "Server response time",
		"A slow server delays everything else and can limit how much crawlers fetch.")
	found := fmt.Sprintf("The HTML arrived in %d ms (from our server).", ms)
	switch {
	case ms <= freeResponseOKms:
		c.set(CheckPass, found, "")
	case ms <= freeResponseSlowMs:
		c.set(CheckWarn, found, "Cache the page HTML or use a CDN so it is served faster.")
	default:
		c.set(CheckFail, found, "Cache the page HTML or use a CDN; check slow database queries and plugins.")
	}
	return c
}

func viewportCheck(tech TechnicalSignals) PageCheck {
	c := newCheck("viewport", CategoryTechnical, sevHigh, "Mobile friendly viewport",
		"Without a viewport tag, phones show the desktop layout zoomed out.")
	if tech.Measured && !tech.Viewport {
		c.set(CheckFail, "No viewport meta tag.", "Add <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">.")
		return c
	}
	c.set(CheckPass, "Viewport meta tag is set.", "")
	return c
}

func languageCheck(tech TechnicalSignals) PageCheck {
	c := newCheck("language", CategoryTechnical, sevLow, "Page language",
		"The lang attribute tells search engines and AI which language the page is in.")
	if tech.Measured && !validHTMLLanguage(tech.HTMLLanguage) {
		c.set(CheckWarn, "No valid lang attribute on <html>.", "Add a language such as lang=\"en\" to the <html> tag.")
		return c
	}
	c.set(CheckPass, "lang=\""+tech.HTMLLanguage+"\"", "")
	return c
}

func robotsCheck(origin string, robots *string) PageCheck {
	c := newCheck("robots-txt", CategoryTechnical, sevLow, "robots.txt",
		"robots.txt tells crawlers what they may read and where your sitemap is.")
	if robots == nil {
		c.set(CheckWarn, "No robots.txt found.", "Add a robots.txt that allows crawling and links your sitemap.")
		return c
	}
	c.set(CheckPass, origin+"/robots.txt exists.", "")
	return c
}

func sitemapCheck(sitemap string) PageCheck {
	c := newCheck("sitemap", CategoryTechnical, sevMedium, "XML sitemap",
		"A sitemap helps Google and Bing find every page you want indexed.")
	if sitemap == "" {
		c.set(CheckWarn, "No XML sitemap at /sitemap.xml or in robots.txt.",
			"Create an XML sitemap (most CMSs and SEO plugins make one), link it in robots.txt and submit it in Search Console.")
		return c
	}
	c.set(CheckPass, "Sitemap found: "+sitemap, "")
	return c
}

// insecureLinksCheck fails on forms that submit over plain HTTP and warns on
// links to http:// pages from an https page.
func insecureLinksCheck(page CrawledPageResult, tech TechnicalSignals) PageCheck {
	c := newCheck("mixed-content", CategoryTechnical, sevMedium, "Secure links and forms",
		"Forms or links to http:// pages can expose visitors' data.")
	insecureLinks := 0
	for _, link := range page.Links {
		if strings.HasPrefix(strings.ToLower(link.TargetURL), "http://") {
			insecureLinks++
		}
	}
	switch {
	case len(tech.InsecureForms) > 0:
		c.set(CheckFail, fmt.Sprintf("%d forms submit over plain HTTP.", len(tech.InsecureForms)), "Send every form to an https:// address.")
	case insecureLinks > 0 && strings.HasPrefix(page.URL, "https://"):
		c.set(CheckWarn, fmt.Sprintf("%d links point to http:// addresses.", insecureLinks), "Link to the https:// version of each page.")
	default:
		c.set(CheckPass, "No insecure forms or http:// links.", "")
	}
	return c
}

func imageSizeCheck(tech TechnicalSignals) PageCheck {
	c := newCheck("image-size", CategoryTechnical, sevMedium, "Image dimensions",
		"Images without width and height make the layout jump while the page loads.")
	const fix = "Set width and height on each <img>."
	switch n := tech.ImagesWithoutSize; {
	case n == 0:
		c.set(CheckPass, "Images declare their size.", "")
	case n > freeImageSizeFewMax:
		c.set(CheckFail, fmt.Sprintf("%d images have no width or height.", n), fix)
	default:
		c.set(CheckWarn, fmt.Sprintf("%d images have no width or height.", n), fix)
	}
	return c
}

func htmlSizeCheck(bytes int) PageCheck {
	c := newCheck("html-size", CategoryTechnical, sevLow, "Page weight (HTML)",
		"A very large HTML file is slow to download on mobile networks.")
	if bytes > freeHTMLWarnBytes {
		c.set(CheckWarn, fmt.Sprintf("The HTML is %d KB.", bytes/1024), "Move inline scripts and styles into files and remove unused markup.")
		return c
	}
	c.set(CheckPass, fmt.Sprintf("The HTML is %d KB.", bytes/1024), "")
	return c
}

// webVitalsCheck reads Chrome's real-visitor data. It is informational when
// no data exists, because a small site has too few visitors to measure.
func webVitalsCheck(field *StoredFieldData) PageCheck {
	c := newCheck("core-web-vitals", CategoryTechnical, sevHigh, "Speed for real visitors (Core Web Vitals)",
		"Google uses real Chrome visitors' measurements when it ranks pages. These show how the page felt to people who used it.")
	const fix = "Make the main image or text appear sooner, reserve space for images and banners so nothing jumps, and cut slow scripts."
	switch {
	case field == nil:
		c.set(CheckInfo, "Not measured: no real-visitor data source is set up.", "")
	case field.Status == FieldInsufficientData:
		c.set(CheckInfo, "Not enough real Chrome visitors to measure this page.", "")
	case field.Status != FieldAvailable:
		c.set(CheckInfo, "Real-visitor data could not be read right now.", "")
	case field.CoreWebVitals == CoreWebVitalsPassed:
		c.set(CheckPass, "Passed for real Chrome visitors at the 75th percentile"+scopeNote(field)+".", "")
	case field.CoreWebVitals == CoreWebVitalsFailed:
		c.set(CheckFail, "Failed for real Chrome visitors"+scopeNote(field)+": "+slowMetrics(field.Metrics)+".", fix)
	default:
		c.set(CheckInfo, "Not enough metrics to assess Core Web Vitals.", "")
	}
	return c
}

func scopeNote(field *StoredFieldData) string {
	if field.Scope == "origin" {
		return " (site-wide data)"
	}
	return ""
}

// slowMetrics names the Core Web Vitals metrics that are not "good".
func slowMetrics(metrics []StoredFieldMetric) string {
	labels := map[string]string{"lcp": "LCP", "inp": "INP", "cls": "CLS"}
	var slow []string
	for _, metric := range metrics {
		if label, ok := labels[metric.Key]; ok && metric.Category != fieldGood {
			slow = append(slow, label)
		}
	}
	if len(slow) == 0 {
		return "a metric is not good"
	}
	return strings.Join(slow, ", ") + " not good"
}

func wordCountCheck(words int) PageCheck {
	c := newCheck("word-count", CategoryContent, sevHigh, "Amount of content",
		"Pages with very little text rarely rank or get quoted; there is not enough to understand.")
	found := fmt.Sprintf("%d words of text.", words)
	switch {
	case words < freeThinWords:
		c.set(CheckFail, found, "Explain what you offer, who it is for, prices, process and common questions in at least 300 words.")
	case words < freeFullWords:
		c.set(CheckWarn, found, "Add more useful detail: services, prices, process, proof and FAQs.")
	default:
		c.set(CheckPass, found, "")
	}
	return c
}

func internalLinksCheck(links []PageLink) PageCheck {
	internal := 0
	for _, link := range links {
		if link.IsInternal {
			internal++
		}
	}
	c := newCheck("internal-links", CategoryContent, sevMedium, "Links to your other pages",
		"Internal links help visitors and crawlers reach your important pages.")
	found := fmt.Sprintf("%d links to your own pages.", internal)
	switch {
	case internal < freeMinInternalLink:
		c.set(CheckFail, found, fmt.Sprintf("Link to your main services, products, about and contact pages (at least %d).", freeMinInternalLink))
	case internal < freeGoodInternal:
		c.set(CheckWarn, found, "Link to more of your key pages from this one.")
	default:
		c.set(CheckPass, found, "")
	}
	return c
}

func structureCheck(ai AIPageSignals) PageCheck {
	c := newCheck("structure", CategoryContent, sevLow, "Lists and tables",
		"Lists and tables are easy to skim and easy for AI to turn into answers.")
	found := fmt.Sprintf("%d lists and %d tables.", ai.Lists, ai.Tables)
	if ai.Lists+ai.Tables == 0 {
		c.set(CheckWarn, found, "Use a bulleted list or table for features, prices or steps.")
		return c
	}
	c.set(CheckPass, found, "")
	return c
}

// readabilityCheck measures the average sentence length of the body text.
func readabilityCheck(body string) PageCheck {
	c := newCheck("readability", CategoryContent, sevMedium, "Sentence length",
		"Long sentences are hard to read on a phone and hard for AI to lift as one clear answer.")
	const fix = "Split long sentences. Aim for about 20 words or fewer per sentence."
	words := len(strings.Fields(body))
	if words < freeReadabilityMin {
		c.set(CheckInfo, "Too little text to measure sentence length.", "")
		return c
	}
	sentences := 0
	for part := range strings.FieldsFuncSeq(body, func(r rune) bool { return r == '.' || r == '!' || r == '?' || r == '।' }) {
		if strings.TrimSpace(part) != "" {
			sentences++
		}
	}
	average := float64(words)
	if sentences > 0 {
		average = float64(words) / float64(sentences)
	}
	found := fmt.Sprintf("About %.0f words per sentence.", average)
	switch {
	case average <= freeReadabilityOK:
		c.set(CheckPass, found, "")
	case average <= freeReadabilityWarn:
		c.set(CheckWarn, found, fix)
	default:
		c.set(CheckFail, found, fix)
	}
	return c
}

// titleStopWords are words that do not make a title and a heading related.
var titleStopWords = map[string]bool{
	"the": true, "and": true, "for": true, "with": true, "from": true, "your": true,
	"our": true, "you": true, "are": true, "this": true, "that": true, "all": true,
	"new": true, "home": true, "page": true,
}

// meaningfulWords returns the distinct words of text that can tie a title to
// a heading: at least three characters, not a stop word. Letters, digits and
// combining marks (Devanagari vowel signs) make up a word.
func meaningfulWords(text string) map[string]bool {
	words := map[string]bool{}
	spaced := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r) {
			return unicode.ToLower(r)
		}
		return ' '
	}, text)
	for word := range strings.FieldsSeq(spaced) {
		if utf8.RuneCountInString(word) >= 3 && !titleStopWords[word] {
			words[word] = true
		}
	}
	return words
}

// sharesWord reports whether a and b have a meaningful word in common.
func sharesWord(a, b string) bool {
	words := meaningfulWords(a)
	for word := range meaningfulWords(b) {
		if words[word] {
			return true
		}
	}
	return false
}

// sameName reports whether text is only the site's name: the host without
// "www." or its first label, compared letters and digits only.
func sameName(text, host string) bool {
	name := strings.TrimPrefix(host, "www.")
	label, _, _ := strings.Cut(name, ".")
	key := func(s string) string {
		return strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return unicode.ToLower(r)
			}
			return -1
		}, s)
	}
	normalized := key(text)
	return normalized != "" && (normalized == key(label) || normalized == key(name))
}

// pageHost returns the host of pageURL without a port, or "".
func pageHost(pageURL string) string {
	parsed, err := url.Parse(pageURL)
	if err != nil {
		return ""
	}
	return parsed.Hostname()
}

// firstNonEmpty returns the first non-empty value.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// prefixRunes returns at most n runes of s.
func prefixRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}
