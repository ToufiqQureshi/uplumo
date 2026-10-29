package audit

// IssueSeverity orders and groups issues in the report.
type IssueSeverity string

// The three severities, mirroring IssueSeverity.
const (
	IssueExternalChecksPending    IssueType     = "external-checks-pending"
	IssueExternalChecksIncomplete IssueType     = "external-checks-incomplete"
	SeverityCritical              IssueSeverity = "critical"
	SeverityWarning               IssueSeverity = "warning"
	SeverityInfo                  IssueSeverity = "info"
)

// IssueType identifies one issue kind.
type IssueType string

// The closed registry of issue types, mirroring AUDIT_ISSUE_TYPES.
const (
	IssueInvalidHTMLLanguage          IssueType = "invalid-html-language"
	IssueMissingViewport              IssueType = "missing-viewport"
	IssueMultipleTitles               IssueType = "multiple-titles"
	IssueMultipleDescriptions         IssueType = "multiple-descriptions"
	IssueMultipleCanonicals           IssueType = "multiple-canonicals"
	IssueRelativeCanonical            IssueType = "relative-canonical"
	IssueBadCanonicalTarget           IssueType = "bad-canonical-target"
	IssueSitemapNoindex               IssueType = "sitemap-noindex"
	IssueSitemapBlocked               IssueType = "sitemap-blocked"
	IssueSitemapNoncanonical          IssueType = "sitemap-noncanonical"
	IssueHreflangNoSelf               IssueType = "hreflang-no-self"
	IssueHreflangNoDefault            IssueType = "hreflang-no-default"
	IssueHreflangNotReciprocal        IssueType = "hreflang-not-reciprocal"
	IssueHreflangBadTarget            IssueType = "hreflang-bad-target"
	IssueHTTPLinks                    IssueType = "http-links"
	IssueInsecureForm                 IssueType = "insecure-form"
	IssueInternalNofollow             IssueType = "internal-nofollow"
	IssueImagesMissingSize            IssueType = "images-missing-size"
	IssueLargeImage                   IssueType = "large-image"
	IssueMissingHSTS                  IssueType = "missing-hsts"
	IssueBrokenExternalLink           IssueType = "broken-external-link"
	IssueBlockedPage                  IssueType = "blocked-page"
	IssueJavaScriptRenderingSuspected IssueType = "javascript-rendering-suspected"
	IssueRateLimitedPage              IssueType = "rate-limited-page"
	IssueCrawlRateLimited             IssueType = "crawl-rate-limited"
	IssueServerError                  IssueType = "server-error"
	IssueBrokenInternalLink           IssueType = "broken-internal-link"
	IssueMissingTitle                 IssueType = "missing-title"
	IssueBrokenPage                   IssueType = "broken-page"
	IssueDuplicateTitle               IssueType = "duplicate-title"
	IssueDuplicateMetaDescription     IssueType = "duplicate-meta-description"
	IssueDuplicateContent             IssueType = "duplicate-content"
	IssueMissingMetaDescription       IssueType = "missing-meta-description"
	IssueMissingH1                    IssueType = "missing-h1"
	IssueMultipleH1                   IssueType = "multiple-h1"
	IssueRedirectChain                IssueType = "redirect-chain"
	IssueRedirectLoop                 IssueType = "redirect-loop"
	IssueCanonicalConflict            IssueType = "canonical-conflict"
	IssueThinContent                  IssueType = "thin-content"
	IssueImagesMissingAlt             IssueType = "images-missing-alt"
	IssueOrphanPage                   IssueType = "orphan-page"
	IssueNoOutgoingLinks              IssueType = "no-outgoing-links"
	IssueTitleTooLong                 IssueType = "title-too-long"
	IssueTitleTooShort                IssueType = "title-too-short"
	IssueMetaDescriptionTooLong       IssueType = "meta-description-too-long"
	IssueMetaDescriptionTooShort      IssueType = "meta-description-too-short"
	IssueHeadingOrderSkip             IssueType = "heading-order-skip"
	IssueSlowResponse                 IssueType = "slow-response"
	IssueNoindexPage                  IssueType = "noindex-page"
	IssueCanonicalizedPage            IssueType = "canonicalized-page"
	IssueDeepPage                     IssueType = "deep-page"
	IssueAICrawlersBlocked            IssueType = "ai-crawlers-blocked"
	IssueAITrainingBotsBlocked        IssueType = "ai-training-bots-blocked"
	IssueMissingLLMsTxt               IssueType = "missing-llms-txt"
	IssueMissingBusinessSchema        IssueType = "missing-business-schema"
	IssueNoQuestionAnswers            IssueType = "no-question-answers"
	IssueSeveralPhoneNumbers          IssueType = "several-phone-numbers"
)

// issueDescriptor is the server-side part of an issue descriptor: the severity
// (persisted with every row) and the human title. The full user-facing copy
// stays in the shared TypeScript registry, which the React pages already import.
type issueDescriptor struct {
	Severity IssueSeverity
	Title    string
}

// issueRegistry carries the severity and title for every known issue type.
var issueRegistry = map[IssueType]issueDescriptor{
	IssueExternalChecksPending:        {SeverityInfo, "External link checks are queued"},
	IssueExternalChecksIncomplete:     {SeverityInfo, "Some external links or images could not be checked"},
	IssueInvalidHTMLLanguage:          {SeverityWarning, "Missing or invalid HTML language"},
	IssueMissingViewport:              {SeverityWarning, "Missing mobile viewport"},
	IssueMultipleTitles:               {SeverityWarning, "Multiple title tags"},
	IssueMultipleDescriptions:         {SeverityWarning, "Multiple meta descriptions"},
	IssueMultipleCanonicals:           {SeverityWarning, "Multiple canonical tags"},
	IssueRelativeCanonical:            {SeverityWarning, "Relative canonical URL"},
	IssueBadCanonicalTarget:           {SeverityWarning, "Canonical points to an unavailable page"},
	IssueSitemapNoindex:               {SeverityWarning, "Sitemap includes a noindex page"},
	IssueSitemapBlocked:               {SeverityWarning, "Sitemap URL blocked by robots.txt"},
	IssueSitemapNoncanonical:          {SeverityWarning, "Sitemap includes a non-canonical URL"},
	IssueHreflangNoSelf:               {SeverityWarning, "Hreflang has no self-reference"},
	IssueHreflangNoDefault:            {SeverityInfo, "Hreflang has no x-default"},
	IssueHreflangNotReciprocal:        {SeverityWarning, "Hreflang alternate does not link back"},
	IssueHreflangBadTarget:            {SeverityWarning, "Hreflang points to a redirect or error"},
	IssueHTTPLinks:                    {SeverityWarning, "HTTPS page links to HTTP"},
	IssueInsecureForm:                 {SeverityCritical, "Form submits over HTTP"},
	IssueInternalNofollow:             {SeverityInfo, "Internal links use nofollow"},
	IssueImagesMissingSize:            {SeverityWarning, "Images lack width or height"},
	IssueLargeImage:                   {SeverityWarning, "Large image"},
	IssueMissingHSTS:                  {SeverityWarning, "HTTPS page has no active HSTS"},
	IssueBrokenExternalLink:           {SeverityWarning, "External link returns an error"},
	IssueBlockedPage:                  {SeverityCritical, "Crawler was blocked"},
	IssueJavaScriptRenderingSuspected: {SeverityWarning, "Content may require JavaScript"},
	IssueRateLimitedPage:              {SeverityWarning, "Rate limited (429)"},
	IssueCrawlRateLimited:             {SeverityWarning, "Crawl stopped early: rate limit"},
	IssueServerError:                  {SeverityCritical, "Server error (5xx)"},
	IssueBrokenInternalLink:           {SeverityCritical, "Broken internal link"},
	IssueMissingTitle:                 {SeverityCritical, "Missing title tag"},
	IssueBrokenPage:                   {SeverityWarning, "Page returns an error (4xx)"},
	IssueDuplicateTitle:               {SeverityWarning, "Duplicate title"},
	IssueDuplicateMetaDescription:     {SeverityWarning, "Duplicate meta description"},
	IssueDuplicateContent:             {SeverityWarning, "Duplicate page content"},
	IssueMissingMetaDescription:       {SeverityWarning, "Missing meta description"},
	IssueMissingH1:                    {SeverityWarning, "Missing H1 heading"},
	IssueMultipleH1:                   {SeverityWarning, "Multiple H1 headings"},
	IssueRedirectChain:                {SeverityWarning, "Redirect chain"},
	IssueRedirectLoop:                 {SeverityWarning, "Redirect loop"},
	IssueCanonicalConflict:            {SeverityWarning, "Conflicting canonical signals"},
	IssueThinContent:                  {SeverityWarning, "Thin content"},
	IssueImagesMissingAlt:             {SeverityWarning, "Images missing alt text"},
	IssueOrphanPage:                   {SeverityWarning, "Orphan page"},
	IssueNoOutgoingLinks:              {SeverityWarning, "Page has no outgoing links"},
	IssueTitleTooLong:                 {SeverityInfo, "Title too long"},
	IssueTitleTooShort:                {SeverityInfo, "Title too short"},
	IssueMetaDescriptionTooLong:       {SeverityInfo, "Meta description too long"},
	IssueMetaDescriptionTooShort:      {SeverityInfo, "Meta description too short"},
	IssueHeadingOrderSkip:             {SeverityInfo, "Heading levels skip"},
	IssueSlowResponse:                 {SeverityInfo, "Slow server response"},
	IssueNoindexPage:                  {SeverityInfo, "Page is noindex"},
	IssueCanonicalizedPage:            {SeverityInfo, "Canonicalized to another URL"},
	IssueDeepPage:                     {SeverityInfo, "Page is deep in the site structure"},
	IssueAICrawlersBlocked:            {SeverityWarning, "AI assistants are blocked from reading the site"},
	IssueAITrainingBotsBlocked:        {SeverityInfo, "AI training crawlers are blocked"},
	IssueMissingLLMsTxt:               {SeverityInfo, "No llms.txt file"},
	IssueMissingBusinessSchema:        {SeverityWarning, "AI cannot read who the business is"},
	IssueNoQuestionAnswers:            {SeverityWarning, "No page answers customer questions"},
	IssueSeveralPhoneNumbers:          {SeverityInfo, "Different phone numbers across pages"},
}

// severityOf returns the severity for an issue type, defaulting to warning so a
// future type cannot make a write fail.
func severityOf(issueType IssueType) IssueSeverity {
	if descriptor, ok := issueRegistry[issueType]; ok {
		return descriptor.Severity
	}
	return SeverityWarning
}

// IssueDescriptor describes an audit issue and its recommended action.
type IssueDescriptor struct {
	Severity IssueSeverity
	Title    string
	HowToFix string
	Why      string
}

var issueRemediation = map[IssueType]string{
	IssueAITrainingBotsBlocked:    "Review these robots.txt rules only if you want to permit training use; blocking them does not lower the score.",
	IssueExternalChecksPending:    "Refresh this audit after the queued checks finish.",
	IssueExternalChecksIncomplete: "Review unverified destinations manually; some sites block automated checks or did not respond in time.",
	IssueInvalidHTMLLanguage:      "Add a valid language tag such as lang=\"en\" or lang=\"hi\" on the html element.",
	IssueMissingViewport:          "Add a viewport meta tag with width=device-width and an appropriate initial scale.",
	IssueMultipleTitles:           "Keep one descriptive title element in the page head.",
	IssueMultipleDescriptions:     "Keep one relevant meta description in the page head.",
	IssueMultipleCanonicals:       "Keep one canonical link in the head pointing to the preferred page.",
	IssueRelativeCanonical:        "Use the full absolute HTTPS URL in the canonical link.",
	IssueBadCanonicalTarget:       "Point the canonical at an indexable, successful final URL.",
	IssueSitemapNoindex:           "Remove the URL from the sitemap or remove noindex if the page should appear in search.",
	IssueSitemapBlocked:           "Allow the URL in robots.txt or remove it from the sitemap.",
	IssueSitemapNoncanonical:      "Replace the listed URL with its canonical URL.",
	IssueHreflangNoSelf:           "Add an alternate hreflang link pointing to this page in its own language.",
	IssueHreflangNoDefault:        "Consider adding an x-default alternate for the fallback page.",
	IssueHreflangNotReciprocal:    "Add a matching return hreflang link on the alternate page.",
	IssueHreflangBadTarget:        "Point each alternate directly to its live final URL.",
	IssueHTTPLinks:                "Link directly to the HTTPS version where available.",
	IssueInsecureForm:             "Change the form action to an HTTPS endpoint and confirm it handles the submission.",
	IssueInternalNofollow:         "Remove nofollow from ordinary internal links when those pages should be crawled.",
	IssueImagesMissingSize:        "Set appropriate width and height attributes or reserve space with an aspect ratio.",
	IssueLargeImage:               "Resize or compress the image and use an efficient format appropriate for its content.",
	IssueMissingHSTS:              "Configure Strict-Transport-Security with a positive max-age after confirming HTTPS works across the intended domain.",
	IssueBrokenExternalLink:       "Check the destination and update or remove the link if it is no longer valid.",

	IssueType("blocked-page"):                   "If you own this site, allowlist the \"Uplumo-Audit\" user agent in your WAF/bot-protection settings (on Cloudflare: a WAF custom rule that skips bot protection when the user agent contains \"Uplumo-Audit\"; on some free tiers you may need to relax bot protection). Then re-run the audit. On Shopify, use Crawler access instead (Online Store → Preferences → Crawler access) and paste the signature into Uplumo under Settings → Crawler access.",
	IssueType("javascript-rendering-suspected"): "This audit checks the initial HTML; JavaScript rendering is not currently available in Uplumo. Verify the loaded page in a browser and serve important content and navigation in the initial HTML using server-side rendering or prerendering. This warning does not prove that search engines cannot index the page.",
	IssueType("rate-limited-page"):              "Raise the rate limit for crawlers, or allowlist the \"Uplumo-Audit\" user agent in your rate-limiting rules (on Cloudflare: a rate-limiting rule exception matching that user agent). Then re-run the audit. Re-running with fewer pages also helps if the limit is strict. On Shopify, use Crawler access instead (Online Store → Preferences → Crawler access) and paste the signature into Uplumo under Settings → Crawler access.",
	IssueType("crawl-rate-limited"):             "Re-run the audit after the site's rate limit resets, or ask the site owner to allow the Uplumo-Audit crawler. On Shopify, use Crawler access instead (Online Store → Preferences → Crawler access) and paste the signature into Uplumo under Settings → Crawler access.",
	IssueType("server-error"):                   "Check the server logs for this URL and fix the underlying error. If the page is gone, return a 404/410 or redirect it to a relevant page instead of erroring.",
	IssueType("broken-internal-link"):           "Update the link to point at the correct live URL, or remove it. If the target was moved, prefer linking directly to the new URL rather than relying on a redirect.",
	IssueType("missing-title"):                  "Add a unique, descriptive <title> of roughly 50–60 characters that includes the page's primary topic.",
	IssueType("broken-page"):                    "If the page should exist, restore it. If it is intentionally gone, remove it from the sitemap and internal links, and consider a 301 redirect to the closest live page.",
	IssueType("duplicate-title"):                "Write a unique title for each page describing its specific content. For templated pages, include the distinguishing attribute (name, category, location) in the template.",
	IssueType("duplicate-meta-description"):     "Write a unique meta description per page, or remove the duplicated one entirely — search engines will generate a snippet from page content, which beats a wrong duplicate.",
	IssueType("duplicate-content"):              "Consolidate duplicates: pick the canonical URL, add rel=canonical from the others, and 301-redirect duplicate URLs where possible (common causes: trailing-slash variants, URL parameters, http/https or www variants).",
	IssueType("missing-meta-description"):       "Add a meta description of roughly 70–160 characters that summarizes the page and gives a reason to click.",
	IssueType("missing-h1"):                     "Add a single H1 that states the page's main topic, consistent with the title tag.",
	IssueType("multiple-h1"):                    "Keep one H1 for the page's main heading and demote the others to H2/H3 (or unstyled elements for non-headings like logos).",
	IssueType("redirect-chain"):                 "Point the first URL (and any internal links) directly at the final destination so there is at most one redirect.",
	IssueType("redirect-loop"):                  "Trace the redirect rules for this URL and break the cycle so the chain terminates at a real 200 page.",
	IssueType("canonical-conflict"):             "Pick one canonical URL and declare it in exactly one place (HTML head is the most common); remove or align the other declaration.",
	IssueType("thin-content"):                   "Either expand the page with genuinely useful content, noindex it, or consolidate it into a stronger page. If the content exists but is rendered by JavaScript, ensure it is server-rendered or pre-rendered.",
	IssueType("images-missing-alt"):             "Add descriptive alt text to meaningful images; use an empty alt (alt=\"\") only for purely decorative ones.",
	IssueType("orphan-page"):                    "Link to this page from relevant pages (navigation, related content, hub pages), or remove it from the sitemap if it shouldn't be indexed.",
	IssueType("no-outgoing-links"):              "Add links to related pages, the parent category, or the homepage. If the page's navigation is rendered by JavaScript, make sure it also exists in the server-rendered HTML.",
	IssueType("title-too-long"):                 "Shorten the title to roughly 50–60 characters, front-loading the most important words.",
	IssueType("title-too-short"):                "Expand the title into a descriptive phrase (roughly 30–60 characters) that states what the page offers.",
	IssueType("meta-description-too-long"):      "Trim the description to roughly 70–160 characters while keeping the core message and call to action.",
	IssueType("meta-description-too-short"):     "Expand the description to roughly 70–160 characters that summarize the page and give a reason to click.",
	IssueType("heading-order-skip"):             "Adjust heading levels so they descend one step at a time (H1 → H2 → H3) without skipping.",
	IssueType("slow-response"):                  "Investigate server/database time and caching for this route; serving cached or statically generated HTML usually fixes it.",
	IssueType("noindex-page"):                   "If this page should rank, remove the noindex directive. If it's intentional (admin, thank-you, filter pages), no action is needed.",
	IssueType("canonicalized-page"):             "If this page should rank on its own, set its canonical to itself. Otherwise no action is needed.",
	IssueType("deep-page"):                      "Add links from higher-level pages (hubs, category pages, navigation) to flatten the path to this page.",
	IssueAICrawlersBlocked:                      "Remove the Disallow rules for these user agents in robots.txt (or add Allow: / for them). Blocking the search agents (OAI-SearchBot, ChatGPT-User, PerplexityBot, Claude-SearchBot) stops those assistants from linking your pages; blocking only training agents (GPTBot, ClaudeBot, Google-Extended) is a choice, not an error. Also check that your firewall or CDN bot protection does not block them.",
	IssueMissingLLMsTxt:                         "Add a plain-text /llms.txt that lists your most important pages with one line each (what the page answers). It is optional and few assistants read it yet, so do this after the AI crawler and content fixes.",
	IssueMissingBusinessSchema:                  "Add Organization or LocalBusiness JSON-LD to the home page with your exact business name, address, phone, opening hours, price range and links to your Google Business Profile and social profiles (sameAs). Use one schema.org subtype that fits, such as Hotel, Restaurant or Dentist. Check it with Google's Rich Results Test.",
	IssueNoQuestionAnswers:                      "Add a short FAQ to your main service pages: real customer questions as H2 or H3 headings, each answered in the first one or two sentences below it (price, location, timing, who it is for). Use the questions from Search Console and the AI Visibility Tracker. FAQPage schema is optional.",
	IssueSeveralPhoneNumbers:                    "If these are the same business, use one main number everywhere: website, Google Business Profile and listings such as Justdial. AI and Google trust details that match. If they are different departments, label each one clearly.",
}

var issueExplanations = map[IssueType]string{
	IssueMissingH1:                "A clear H1 helps readers and search engines identify the page's main topic.",
	IssueCanonicalConflict:        "Conflicting HTML and HTTP canonical declarations make the preferred URL ambiguous.",
	IssueAICrawlersBlocked:        "Search assistants may be unable to retrieve and cite pages when their search crawler is blocked.",
	IssueAITrainingBotsBlocked:    "These bots are used for training; allowing them is optional and independent of citation readiness.",
	IssueExternalChecksPending:    "External links and image sizes are checked separately so they do not delay other site scans.",
	IssueExternalChecksIncomplete: "The audit could not verify every external link or image within its safety limits.",
	IssueInvalidHTMLLanguage:      "Without a valid language, browsers and search engines cannot reliably identify the language of the page.",
	IssueMissingViewport:          "Mobile browsers may render the page at desktop width, making content hard to read.",
	IssueMultipleTitles:           "Search engines may select an unintended title when a page declares more than one.",
	IssueMultipleDescriptions:     "Multiple descriptions provide conflicting snippets to search engines.",
	IssueMultipleCanonicals:       "Different canonical tags can make the preferred URL ambiguous.",
	IssueRelativeCanonical:        "A relative canonical can resolve differently when the page moves or a base URL changes.",
	IssueBadCanonicalTarget:       "A canonical target that redirects, errors or is noindex may not be a suitable page to index.",
	IssueSitemapNoindex:           "The sitemap asks search engines to discover a page that also asks not to be indexed.",
	IssueSitemapBlocked:           "Search engines cannot crawl a listed URL when the site blocks it in robots.txt.",
	IssueSitemapNoncanonical:      "A sitemap should list the preferred URL rather than another URL that points away.",
	IssueHreflangNoSelf:           "A language group without a self-reference can be harder for search engines to interpret.",
	IssueHreflangNoDefault:        "A default version helps visitors whose language does not match another alternate.",
	IssueHreflangNotReciprocal:    "Language alternates need reciprocal links so search engines can verify the relationship.",
	IssueHreflangBadTarget:        "A language alternate that redirects or errors may not be usable as the intended version.",
	IssueHTTPLinks:                "Visitors following an insecure link can lose HTTPS protection.",
	IssueInsecureForm:             "Information submitted to an HTTP endpoint can be exposed in transit.",
	IssueInternalNofollow:         "Nofollow can prevent crawlers from following links to useful pages.",
	IssueImagesMissingSize:        "Without dimensions the browser may shift content while images load.",
	IssueLargeImage:               "Large image files can slow page loading, especially on mobile connections.",
	IssueMissingHSTS:              "Without HSTS, a browser may allow an insecure HTTP connection to the site.",
	IssueBrokenExternalLink:       "A link to an unavailable external page can frustrate readers.",
}

// DescribeIssueType returns the descriptor for a known audit issue type.
func DescribeIssueType(issueType string) (IssueDescriptor, bool) {
	key := IssueType(issueType)
	descriptor, ok := issueRegistry[key]
	if !ok {
		return IssueDescriptor{}, false
	}
	return IssueDescriptor{Severity: descriptor.Severity, Title: descriptor.Title, HowToFix: issueRemediation[key], Why: issueExplanations[key]}, true
}
