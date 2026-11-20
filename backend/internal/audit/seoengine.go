package audit

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// The SEO engine of the free audit: how well Google can crawl, index, read
// and rank one page. It reads the snapshot only; no provider is called.
//
// Categories and weights follow the crawl-first order of current technical
// SEO guides: crawl and index (30), on-page (30), performance and mobile (20),
// content and structured data (20). Core Web Vitals come from Chrome field
// data (CrUX) when a key is configured.

// Category IDs of the SEO engine.
const (
	seoCrawl       = "crawl"
	seoOnPage      = "onpage"
	seoPerformance = "performance"
	seoContent     = "content"
)

// seoCategories are the SEO engine's categories, in the order the report
// shows them, with their weights in the score.
var seoCategories = []categoryDef{
	{ID: seoCrawl, Name: "Crawl and index", Weight: 30},
	{ID: seoOnPage, Name: "On-page SEO", Weight: 30},
	{ID: seoPerformance, Name: "Performance and mobile", Weight: 20},
	{ID: seoContent, Name: "Content and structured data", Weight: 20},
}

// seoSevereCap caps the SEO score when a critical check fails: the page
// cannot be indexed, Googlebot is blocked, or it is not served over HTTPS.
const seoSevereCap = 49

// Thresholds of the SEO checks.
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
	// freeGenericAnchorShare is the share of internal links with generic text
	// ("click here") above which the check warns.
	freeGenericAnchorShare = 0.3
	freeMaxURLChars        = 100
	freeStaleMonths        = 18
	freeHTMLWarnBytes      = 500 * 1024
	// freeImageAltFailRatio is the share of images without alt text that fails.
	freeImageAltFailRatio = 0.2
	freeImageSizeFewMax   = 10
	// freeRenderBlockWarn and freeRenderBlockFail count head scripts that block
	// rendering (no async or defer).
	freeRenderBlockWarn = 3
	freeRenderBlockFail = 8
	// freeLazyFreeImages is how many leading images may load eagerly; the
	// rest should be lazy-loaded.
	freeLazyFreeImages  = 3
	freeLazyWarn        = 10
	freeReadabilityOK   = 20
	freeReadabilityWarn = 25
	freeReadabilityMin  = 100
	freeResponseOKms    = 400
	freeResponseSlowMs  = 1000
	// freeJSONLDFail is how many unparsable JSON-LD blocks fail the check.
	freeJSONLDFail = 3
)

// seoChecks builds the SEO engine's checks in the order the report shows them.
func seoChecks(snap PageSnapshot) []PageCheck {
	page := snap.Page
	ai := page.AI
	tech := page.Technical
	h1 := ""
	if len(snap.Analysis.H1s) > 0 {
		h1 = snap.Analysis.H1s[0]
	}
	var checks []PageCheck
	add := func(check PageCheck) { checks = append(checks, check) }

	// Crawl and index.
	add(httpsCheck(page.URL))
	add(indexableCheck(page))
	add(googlebotCheck(snap.Origin, snap.Robots, page.URL))
	add(statusCheck(snap.StatusCode))
	add(redirectsCheck(snap.Redirects))
	add(canonicalCheck(page))
	add(duplicateTagsCheck(tech))
	add(hreflangCheck(tech.Hreflangs, page.URL))
	add(sitemapCheck(snap.Sitemap))
	add(robotsCheck(snap.Origin, snap.Robots))
	add(hstsCheck(ai))
	add(insecureLinksCheck(page, tech))

	// On-page SEO.
	add(titleCheck(page.Title, h1))
	add(descriptionCheck(page.MetaDescription))
	add(h1Check(page.H1Count, snap.Analysis.H1s, pageHost(page.URL)))
	add(headingOrderCheck(page))
	add(h2Check(page.H2Count, page.WordCount))
	add(urlHygieneCheck(page.URL))
	add(imageAltCheck(page))
	add(internalLinksCheck(page.Links))
	add(genericAnchorsCheck(page.Links))
	add(openGraphCheck(page))
	add(languageCheck(tech))

	// Performance and mobile.
	add(webVitalsCheck(snap.Field))
	add(responseTimeCheck(page.ResponseTimeMs))
	add(viewportCheck(tech))
	add(imageSizeCheck(tech))
	add(htmlSizeCheck(snap.HTMLBytes))
	add(renderBlockingCheck(tech))
	add(lazyImagesCheck(tech))

	// Content and structured data.
	add(wordCountCheck(page.WordCount))
	add(readabilityCheck(snap.Analysis.BodyText))
	add(structuredDataValidCheck(ai))
	add(schemaTypesCheck(ai))
	add(seoFreshnessCheck(ai, snap.CheckedAt))
	return checks
}

// ScoreSEO scores the snapshot with the SEO engine.
func ScoreSEO(snap PageSnapshot) PageReport {
	return scoreEngine(snap, KindSEO, seoCategories, seoSevereCap, seoChecks)
}

func titleCheck(title, h1 string) PageCheck {
	c := newCheck("title", seoOnPage, sevHigh, "Page title",
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
	return lengthCheck("description", seoOnPage, sevMedium, "Meta description", description,
		freeDescriptionMinChars, freeDescriptionMaxChars,
		"Google often shows it under your title. A clear one gets more clicks.",
		"Write 70-160 characters that say what the visitor gets and why to choose you.")
}

func h1Check(count int, headings []string, host string) PageCheck {
	c := newCheck("h1", seoOnPage, sevHigh, "Main heading (H1)",
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
	c := newCheck("heading-order", seoOnPage, sevLow, "Heading structure",
		"Headings in order (H1, then H2, then H3) make the page easy to scan for people and machines.")
	c.set(CheckPass, fmt.Sprintf("%d H2 and %d H3 headings, in order.", page.H2Count, page.H3Count), "")
	if hasHeadingLevelSkip(page.HeadingOrder) {
		c.set(CheckWarn, "Heading levels skip a step (for example H1 to H3).", "Change heading levels so they go down one step at a time.")
	}
	return c
}

func h2Check(h2s, words int) PageCheck {
	c := newCheck("h2-sections", seoOnPage, sevMedium, "Sections on longer pages",
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

func openGraphCheck(page CrawledPageResult) PageCheck {
	c := newCheck("open-graph", seoOnPage, sevLow, "Social sharing preview",
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
	c := newCheck("image-alt", seoOnPage, sevMedium, "Image alt text",
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
	c := newCheck("url-hygiene", seoOnPage, sevLow, "Clean web address",
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

// internalLinksCheck counts links to the site's own pages.
func internalLinksCheck(links []PageLink) PageCheck {
	internal := 0
	for _, link := range links {
		if link.IsInternal {
			internal++
		}
	}
	c := newCheck("internal-links", seoOnPage, sevMedium, "Links to your other pages",
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

// genericAnchorsCheck warns when most internal links say "click here" or
// "read more" instead of what the target is.
func genericAnchorsCheck(links []PageLink) PageCheck {
	c := newCheck("generic-anchors", seoOnPage, sevMedium, "Descriptive link text",
		"Link text such as \"click here\" tells search engines nothing about the page it points to.")
	const fix = "Write each internal link's text as the target's topic, for example \"our wedding cake prices\" instead of \"read more\"."
	internal, generic := 0, 0
	for _, link := range links {
		if !link.IsInternal {
			continue
		}
		internal++
		if isGenericAnchor(link.Anchor) {
			generic++
		}
	}
	switch {
	case internal == 0:
		c.set(CheckInfo, "No internal links to check.", "")
	case float64(generic)/float64(internal) > freeGenericAnchorShare:
		c.set(CheckWarn, fmt.Sprintf("%d of %d internal links use generic text such as \"click here\".", generic, internal), fix)
	default:
		c.set(CheckPass, fmt.Sprintf("%d of %d internal links use generic text.", generic, internal), "")
	}
	return c
}

func languageCheck(tech TechnicalSignals) PageCheck {
	c := newCheck("language", seoOnPage, sevLow, "Page language",
		"The lang attribute tells search engines and AI which language the page is in.")
	if tech.Measured && !validHTMLLanguage(tech.HTMLLanguage) {
		c.set(CheckWarn, "No valid lang attribute on <html>.", "Add a language such as lang=\"en\" to the <html> tag.")
		return c
	}
	c.set(CheckPass, "lang=\""+tech.HTMLLanguage+"\"", "")
	return c
}

// httpsCheck is critical: a page served over plain HTTP is not trusted.
func httpsCheck(pageURL string) PageCheck {
	c := newCheck("https", seoCrawl, sevCritical, "HTTPS",
		"Browsers warn visitors on sites without HTTPS, and Google prefers secure pages.")
	if strings.HasPrefix(pageURL, "https://") {
		c.set(CheckPass, "The page loads over HTTPS.", "")
		return c
	}
	c.set(CheckFail, "The page loads over plain HTTP.", "Install a TLS certificate and redirect HTTP to HTTPS.")
	return c
}

// indexableCheck is critical: noindex removes the page from search.
func indexableCheck(page CrawledPageResult) PageCheck {
	c := newCheck("indexable", seoCrawl, sevCritical, "Allowed in Google",
		"A noindex rule removes the page from Google and AI search results.")
	if !page.IsIndexable {
		c.set(CheckFail, "The page says noindex: "+strings.TrimSpace(page.RobotsMeta+" "+page.XRobotsTag),
			"Remove noindex from the robots meta tag or X-Robots-Tag header if this page should be found.")
		return c
	}
	c.set(CheckPass, "No noindex rule.", "")
	return c
}

// googlebotCheck is critical: a page robots.txt blocks for Googlebot cannot
// be crawled or ranked. It reads the rules that apply to Googlebot.
func googlebotCheck(origin string, robots *string, pageURL string) PageCheck {
	c := newCheck("googlebot", seoCrawl, sevCritical, "Googlebot may read the page",
		"robots.txt decides whether Google can crawl this page. A page it blocks cannot rank.")
	if robots != nil {
		groups, _ := parseRobotsGroups(*robots, origin)
		if !robotsAllowed(pageURL, selectRobotsRules(groups, "Googlebot")) {
			c.set(CheckFail, "robots.txt blocks Googlebot from this page.",
				"Remove the Disallow rule that matches this page for Googlebot or for *, then test the file in Search Console.")
			return c
		}
	}
	c.set(CheckPass, "robots.txt lets Googlebot read this page.", "")
	return c
}

// statusCheck passes a 200 OK page; search engines expect 200 on pages they index.
func statusCheck(status int) PageCheck {
	c := newCheck("status", seoCrawl, sevHigh, "HTTP status",
		"Search engines index a page that answers 200 OK and drop pages that answer with an error.")
	if status == 200 {
		c.set(CheckPass, "HTTP 200 OK.", "")
		return c
	}
	c.set(CheckWarn, fmt.Sprintf("HTTP %d.", status), "Serve the page with 200 OK, not an error or redirect status.")
	return c
}

// redirectsCheck counts the redirects the page fetch followed.
func redirectsCheck(hops int) PageCheck {
	c := newCheck("redirects", seoCrawl, sevMedium, "Redirects before the page",
		"Each redirect is one more request for search engines and slows the visitor down.")
	switch hops {
	case 0:
		c.set(CheckPass, "No redirects.", "")
	case 1:
		c.set(CheckPass, "One redirect.", "")
	default:
		c.set(CheckWarn, fmt.Sprintf("%d redirects before the page.", hops),
			"Link straight to the final address and update internal links and the sitemap to use it.")
	}
	return c
}

func canonicalCheck(page CrawledPageResult) PageCheck {
	c := newCheck("canonical", seoCrawl, sevMedium, "Canonical URL",
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

// duplicateTagsCheck fails when a page has two titles, descriptions or canonicals.
func duplicateTagsCheck(tech TechnicalSignals) PageCheck {
	c := newCheck("duplicate-tags", seoCrawl, sevHigh, "Duplicate tags",
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

// hreflangCheck checks language versions: when hreflang is used, the page
// must name itself and an x-default.
func hreflangCheck(targets []HreflangTarget, pageURL string) PageCheck {
	c := newCheck("hreflang", seoCrawl, sevMedium, "Language versions (hreflang)",
		"hreflang tags tell Google which language version to show each visitor. Each version must link back to the others.")
	if len(targets) == 0 {
		c.set(CheckPass, "No hreflang tags, so none are needed.", "")
		return c
	}
	self, xDefault := false, false
	for _, target := range targets {
		if normalizedCanonical(target.URL, pageURL) == normalizedCanonical(pageURL, pageURL) {
			self = true
		}
		if target.Language == "x-default" {
			xDefault = true
		}
	}
	var missing []string
	if !self {
		missing = append(missing, "a link to this page")
	}
	if !xDefault {
		missing = append(missing, "an x-default version")
	}
	if len(missing) > 0 {
		c.set(CheckWarn, fmt.Sprintf("%d language versions; missing %s.", len(targets), strings.Join(missing, " and ")),
			"Add a hreflang link to this page itself and one with hreflang=\"x-default\" in the head.")
		return c
	}
	c.set(CheckPass, fmt.Sprintf("%d language versions, including this page and x-default.", len(targets)), "")
	return c
}

func sitemapCheck(sitemap string) PageCheck {
	c := newCheck("sitemap", seoCrawl, sevMedium, "XML sitemap",
		"A sitemap helps Google and Bing find every page you want indexed.")
	if sitemap == "" {
		c.set(CheckWarn, "No XML sitemap at /sitemap.xml or in robots.txt.",
			"Create an XML sitemap (most CMSs and SEO plugins make one), link it in robots.txt and submit it in Search Console.")
		return c
	}
	c.set(CheckPass, "Sitemap found: "+sitemap, "")
	return c
}

func robotsCheck(origin string, robots *string) PageCheck {
	c := newCheck("robots-txt", seoCrawl, sevLow, "robots.txt",
		"robots.txt tells crawlers what they may read and where your sitemap is.")
	if robots == nil {
		c.set(CheckWarn, "No robots.txt found.", "Add a robots.txt that allows crawling and links your sitemap.")
		return c
	}
	c.set(CheckPass, origin+"/robots.txt exists.", "")
	return c
}

// hstsCheck reads the Strict-Transport-Security header of the page response.
func hstsCheck(ai AIPageSignals) PageCheck {
	c := newCheck("hsts", seoCrawl, sevLow, "HSTS header",
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

// insecureLinksCheck fails on forms that submit over plain HTTP and warns on
// links to http:// pages from an https page.
func insecureLinksCheck(page CrawledPageResult, tech TechnicalSignals) PageCheck {
	c := newCheck("mixed-content", seoCrawl, sevMedium, "Secure links and forms",
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

func responseTimeCheck(ms int64) PageCheck {
	c := newCheck("response-time", seoPerformance, sevMedium, "Server response time",
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
	c := newCheck("viewport", seoPerformance, sevHigh, "Mobile friendly viewport",
		"Without a viewport tag, phones show the desktop layout zoomed out.")
	if tech.Measured && !tech.Viewport {
		c.set(CheckFail, "No viewport meta tag.", "Add <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">.")
		return c
	}
	c.set(CheckPass, "Viewport meta tag is set.", "")
	return c
}

func imageSizeCheck(tech TechnicalSignals) PageCheck {
	c := newCheck("image-size", seoPerformance, sevMedium, "Image dimensions",
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
	c := newCheck("html-size", seoPerformance, sevLow, "Page weight (HTML)",
		"A very large HTML file is slow to download on mobile networks.")
	if bytes > freeHTMLWarnBytes {
		c.set(CheckWarn, fmt.Sprintf("The HTML is %d KB.", bytes/1024), "Move inline scripts and styles into files and remove unused markup.")
		return c
	}
	c.set(CheckPass, fmt.Sprintf("The HTML is %d KB.", bytes/1024), "")
	return c
}

// renderBlockingCheck counts scripts in the head that stop the page from
// showing until they load (no async or defer).
func renderBlockingCheck(tech TechnicalSignals) PageCheck {
	c := newCheck("render-blocking", seoPerformance, sevMedium, "Scripts that block the page",
		"A script in the head without async or defer makes the browser wait for it before showing anything.")
	const fix = "Add defer to scripts that do not need to run before the page shows, or move them to the end of the body."
	n := tech.RenderBlockingScripts
	switch {
	case n > freeRenderBlockFail:
		c.set(CheckFail, fmt.Sprintf("%d scripts in the head block rendering.", n), fix)
	case n > freeRenderBlockWarn:
		c.set(CheckWarn, fmt.Sprintf("%d scripts in the head block rendering.", n), fix)
	default:
		c.set(CheckPass, fmt.Sprintf("%d blocking scripts in the head.", n), "")
	}
	return c
}

// lazyImagesCheck warns when many images after the first few load eagerly.
func lazyImagesCheck(tech TechnicalSignals) PageCheck {
	c := newCheck("lazy-images", seoPerformance, sevLow, "Lazy-loaded images",
		"Images below the first screen can wait until a visitor scrolls near them, which saves data and time.")
	n := tech.EagerImagesBelowFold
	if n > freeLazyWarn {
		c.set(CheckWarn, fmt.Sprintf("%d images below the first %d load straight away.", n, freeLazyFreeImages),
			`Add loading="lazy" to images that are not in the first screen.`)
		return c
	}
	c.set(CheckPass, fmt.Sprintf("%d images below the first %d load straight away.", n, freeLazyFreeImages), "")
	return c
}

// webVitalsCheck reads Chrome's real-visitor data. It is informational when
// no data exists, because a small site has too few visitors to measure.
func webVitalsCheck(field *StoredFieldData) PageCheck {
	c := newCheck("core-web-vitals", seoPerformance, sevHigh, "Speed for real visitors (Core Web Vitals)",
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
	c := newCheck("word-count", seoContent, sevHigh, "Amount of content",
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

// readabilityCheck measures the average sentence length of the body text.
func readabilityCheck(body string) PageCheck {
	c := newCheck("readability", seoContent, sevMedium, "Sentence length",
		"Long sentences are hard to read on a phone and hard for search and AI to lift as one clear answer.")
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

// structuredDataValidCheck counts JSON-LD blocks that do not parse. A parse
// error means search engines read no structured data from that block.
func structuredDataValidCheck(ai AIPageSignals) PageCheck {
	c := newCheck("structured-data-valid", seoContent, sevMedium, "Structured data parses",
		"Google reads JSON-LD only when the block is valid JSON. A broken block is ignored silently.")
	const fix = "Validate each <script type=\"application/ld+json\"> block (a JSON validator or Google's Rich Results Test) and fix the commas, quotes and brackets."
	switch n := ai.LDJSONBroken; {
	case ai.LDJSONBlocks == 0:
		c.set(CheckInfo, "No JSON-LD structured data on the page.", "")
	case n == 0:
		c.set(CheckPass, fmt.Sprintf("All %d JSON-LD blocks parse.", ai.LDJSONBlocks), "")
	case n >= freeJSONLDFail:
		c.set(CheckFail, fmt.Sprintf("%d of %d JSON-LD blocks do not parse.", n, ai.LDJSONBlocks), fix)
	default:
		c.set(CheckWarn, fmt.Sprintf("%d of %d JSON-LD blocks do not parse.", n, ai.LDJSONBlocks), fix)
	}
	return c
}

// schemaTypesCheck passes when the page declares any structured data type.
func schemaTypesCheck(ai AIPageSignals) PageCheck {
	c := newCheck("schema-types", seoContent, sevMedium, "Structured data types",
		"Schema.org types tell Google what the page is: an organization, a product, an article or a FAQ.")
	if len(ai.SchemaTypes) == 0 {
		c.set(CheckWarn, "No schema.org structured data on the page.",
			"Add JSON-LD that matches the page, for example Organization on the home page and Article or Product elsewhere.")
		return c
	}
	types := slices.Compact(slices.Sorted(slices.Values(ai.SchemaTypes)))
	c.set(CheckPass, "Structured data: "+strings.Join(types, ", ")+".", "")
	return c
}

// seoFreshnessCheck warns when the page's stated date is more than
// freeStaleMonths old, or when no date is stated.
func seoFreshnessCheck(ai AIPageSignals, now time.Time) PageCheck {
	c := newCheck("freshness", seoContent, sevMedium, "Content is up to date",
		"A visible, recent date tells search engines and readers the page is maintained. Old dates suggest it may be out of date.")
	const fix = "Show the last updated date on the page and update it when the facts change."
	date := pageDate(ai)
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
