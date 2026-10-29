package audit

import (
	"net/http"
	"slices"
	"strings"
	"testing"
)

func testTechnicalPage(head, body string, headers http.Header) CrawledPageResult {
	document := `<html lang="en"><head><title>Room details for visitors</title><meta name="viewport" content="width=device-width"><meta name="description" content="A useful description for visitors that explains the room and all its facilities clearly.">` + head + `</head><body><h1>Rooms</h1><p>Useful room details.</p>` + body + `</body></html>`
	analysis := AnalyzeHTML(document, "https://hotel.example/rooms", 200, 100, "")
	page := CrawledPageResult{ID: "p", URL: analysis.URL, StatusCode: 200, FetchClass: FetchOK, IsHTML: true}
	if headers == nil {
		headers = http.Header{"Strict-Transport-Security": {"max-age=31536000"}}
	}
	applyAnalysis(&page, analysis, headers)
	return page
}

func hasIssue(issues []DetectedIssue, issue IssueType) bool {
	return slices.ContainsFunc(issues, func(found DetectedIssue) bool { return found.IssueType == issue })
}

func TestTechnicalPageChecks(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*CrawledPageResult)
		issue  IssueType
	}{
		{"missing H1", func(p *CrawledPageResult) { p.H1Count = 0 }, IssueMissingH1},
		{"missing language", func(p *CrawledPageResult) { p.Technical.HTMLLanguage = "" }, IssueInvalidHTMLLanguage},
		{"invalid language", func(p *CrawledPageResult) { p.Technical.HTMLLanguage = "en_US" }, IssueInvalidHTMLLanguage},
		{"viewport", func(p *CrawledPageResult) { p.Technical.Viewport = false }, IssueMissingViewport},
		{"titles", func(p *CrawledPageResult) { p.Technical.Titles = 2 }, IssueMultipleTitles},
		{"descriptions", func(p *CrawledPageResult) { p.Technical.Descriptions = 2 }, IssueMultipleDescriptions},
		{"canonicals", func(p *CrawledPageResult) { p.Technical.Canonicals = 2 }, IssueMultipleCanonicals},
		{"relative canonical", func(p *CrawledPageResult) { p.CanonicalURL = "/rooms" }, IssueRelativeCanonical},
		{"conflicting canonicals", func(p *CrawledPageResult) { p.CanonicalURL = "/rooms"; p.HeaderCanonicalURL = "/other" }, IssueCanonicalConflict},
		{"noindex in sitemap", func(p *CrawledPageResult) { p.InSitemap = true; p.IsIndexable = false }, IssueSitemapNoindex},
		{"noncanonical in sitemap", func(p *CrawledPageResult) { p.InSitemap = true; p.CanonicalURL = "/other" }, IssueSitemapNoncanonical},
		{"images without dimensions", func(p *CrawledPageResult) { p.Technical.ImagesWithoutSize = 1 }, IssueImagesMissingSize},
		{"insecure form", func(p *CrawledPageResult) { p.Technical.InsecureForms = []string{"http://hotel.example/post"} }, IssueInsecureForm},
		{"http link", func(p *CrawledPageResult) { p.Links = []PageLink{{TargetURL: "http://other.example/"}} }, IssueHTTPLinks},
		{"internal nofollow", func(p *CrawledPageResult) {
			p.Links = []PageLink{{TargetURL: "https://hotel.example/a", IsInternal: true, IsNofollow: true}}
		}, IssueInternalNofollow},
		{"missing HSTS", func(p *CrawledPageResult) { p.AI.HSTS = new(false) }, IssueMissingHSTS},
		{"hreflang self", func(p *CrawledPageResult) {
			p.Technical.Hreflangs = []HreflangTarget{{Language: "x-default", URL: "https://hotel.example/other"}}
		}, IssueHreflangNoSelf},
		{"hreflang default", func(p *CrawledPageResult) { p.Technical.Hreflangs = []HreflangTarget{{Language: "en", URL: p.URL}} }, IssueHreflangNoDefault},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page := testTechnicalPage("", "", nil)
			if hasIssue(runPageReporters(page), tc.issue) {
				t.Fatalf("clean page unexpectedly reports %s", tc.issue)
			}
			tc.mutate(&page)
			if got := runPageReporters(page); !hasIssue(got, tc.issue) {
				t.Fatalf("issues = %+v, missing %s", got, tc.issue)
			}
			desc, ok := DescribeIssueType(string(tc.issue))
			if !ok || desc.Title == "" || desc.Why == "" || desc.HowToFix == "" || desc.Severity == "" {
				t.Fatalf("missing issue copy: %+v", desc)
			}
		})
	}
}

func TestTechnicalHTMLExtraction(t *testing.T) {
	page := testTechnicalPage(`<title>Second</title><meta name="DESCRIPTION" content="second"><link rel="canonical alternate" href="/rooms"><link rel="alternate" hreflang="en" href="/rooms">`, `<img src="/a.png" width="10"><img src="/b.png" width="10" height="20"><form action="http://hotel.example/post"></form><a rel="nofollow" href="/about">About</a><a href="http://else.example/x">Outside</a>`, nil)
	if page.Technical.Titles != 2 || page.Technical.Descriptions != 2 || page.Technical.Canonicals != 1 || page.Technical.ImagesWithoutSize != 1 || len(page.Technical.InsecureForms) != 1 || len(page.Technical.Hreflangs) != 1 {
		t.Fatalf("technical = %+v", page.Technical)
	}
	for _, issue := range []IssueType{IssueMultipleTitles, IssueMultipleDescriptions, IssueRelativeCanonical, IssueImagesMissingSize, IssueInsecureForm, IssueHTTPLinks, IssueInternalNofollow, IssueHreflangNoDefault} {
		if !hasIssue(runPageReporters(page), issue) {
			t.Errorf("missing %s", issue)
		}
	}
}

func TestTechnicalLanguageAndHSTS(t *testing.T) {
	for _, tc := range []struct {
		value string
		valid bool
	}{{"en-US", true}, {"hi", true}, {"", false}, {"en_US", false}, {"no such language", false}} {
		if got := validHTMLLanguage(tc.value); got != tc.valid {
			t.Errorf("language %q = %v", tc.value, got)
		}
	}
	for _, tc := range []struct {
		value string
		valid bool
	}{{"max-age=31536000; includeSubDomains", true}, {"MAX-AGE=1", true}, {"max-age=0", false}, {"includeSubDomains", false}, {"max-age=invalid", false}} {
		if got := hstsEnabled(tc.value); got != tc.valid {
			t.Errorf("HSTS %q = %v", tc.value, got)
		}
	}
}

func TestCanonicalAndHreflangCompareOnlyCrawledTargets(t *testing.T) {
	status := func(n int) *int { return &n }
	page := SlimPage{ID: "a", URL: "https://hotel.example/en", StatusCode: status(200), FetchClass: FetchOK, IsIndexable: true, SignalsVersion: CitationSignalsVersion, CanonicalURL: "/bad", HreflangTargets: []HreflangTarget{{Language: "hi", URL: "https://hotel.example/hi"}}}
	bad := SlimPage{ID: "bad", URL: "https://hotel.example/bad", StatusCode: status(302), FetchClass: FetchOK}
	alt := SlimPage{ID: "hi", URL: "https://hotel.example/hi", StatusCode: status(200), FetchClass: FetchOK, IsIndexable: true, SignalsVersion: CitationSignalsVersion}
	got := findCanonicalAndHreflangIssues([]SlimPage{page, bad, alt})
	if !hasIssue(got, IssueBadCanonicalTarget) || !hasIssue(got, IssueHreflangNotReciprocal) {
		t.Fatalf("issues = %+v", got)
	}
	alt.HreflangTargets = []HreflangTarget{{Language: "en", URL: page.URL}}
	bad.StatusCode = status(200)
	bad.IsIndexable = false
	if got := findCanonicalAndHreflangIssues([]SlimPage{page, bad, alt}); !hasIssue(got, IssueBadCanonicalTarget) {
		t.Fatalf("noindex canonical target = %+v", got)
	}
	bad.IsIndexable = true
	if got := findCanonicalAndHreflangIssues([]SlimPage{page, bad, alt}); len(got) != 0 {
		t.Fatalf("clean = %+v", got)
	}
	for _, code := range []int{301, 404} {
		alt.StatusCode = status(code)
		if got := findCanonicalAndHreflangIssues([]SlimPage{page, alt}); !hasIssue(got, IssueHreflangBadTarget) || hasIssue(got, IssueBadCanonicalTarget) {
			t.Fatalf("observed %d = %+v", code, got)
		}
	}
	alt.StatusCode = status(200)
	alt.HreflangTargets = nil
	alt.SignalsVersion = 0
	if got := findCanonicalAndHreflangIssues([]SlimPage{page, alt}); len(got) != 0 {
		t.Fatalf("legacy target = %+v", got)
	}
}

func TestSnippetMetaUsesMultipleRobotsTags(t *testing.T) {
	page := testTechnicalPage(`<meta name="robots" content="index"><meta name="googlebot" content="max-snippet:0">`, "", nil)
	if !page.AI.SnippetBlocked || !strings.Contains(page.RobotsMeta, "index") {
		t.Fatalf("page = %+v", page.AI)
	}
}
