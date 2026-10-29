package citefix

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/audit"
)

var scanAt = time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC)

func testScan() Scan {
	return Scan{AuditID: "audit-1", StartURL: "https://shop.example/", CompletedAt: scanAt}
}

// goodPage passes every page check.
func goodPage(url string) Page {
	return Page{
		URL: url, Title: "Good " + url, StatusCode: 200, Indexable: true, WordCount: 900, ExternalLinks: 2,
		Signals: &audit.AIPageSignals{
			SchemaTypes: []string{"Article"}, QuestionHeadings: 2, Author: true,
			Published: "2026-05-01", Modified: "2026-09-01T10:00:00+05:30", LeadWords: 40,
			Lists: 1, Tables: 1, Quotes: 1, Statistics: 3, AnswerCapsule: true, DefinitionAfterHeading: true, Sections: 1, ChunkReadySections: 1, AuthorityLinks: 1,
		},
	}
}

// barePage is a long page with none of the signals.
func barePage(url string) Page {
	return Page{URL: url, Title: "Bare " + url, StatusCode: 200, Indexable: true, WordCount: 600, Signals: &audit.AIPageSignals{}}
}

func fixIDs(report Report) []string {
	ids := make([]string, 0, len(report.Fixes))
	for _, fix := range report.Fixes {
		ids = append(ids, fix.ID)
	}
	return ids
}

func findFix(t *testing.T, report Report, id string) Fix {
	t.Helper()
	for _, fix := range report.Fixes {
		if fix.ID == id {
			return fix
		}
	}
	t.Fatalf("no fix %q in %v", id, fixIDs(report))
	return Fix{}
}

func hasFix(report Report, id string) bool {
	return slices.Contains(fixIDs(report), id)
}

func TestBuildPerfectSiteScoresFullMarks(t *testing.T) {
	report := Build(testScan(), []Page{goodPage("https://shop.example/a"), goodPage("https://shop.example/b")}, nil)
	if report.State != StateReady || report.Score == nil || *report.Score != 100 {
		t.Fatalf("state %s score %v, want ready 100", report.State, report.Score)
	}
	if len(report.Fixes) != 0 {
		t.Fatalf("fixes = %v, want none", fixIDs(report))
	}
	if len(report.Categories) != len(categories) {
		t.Fatalf("categories = %d, want %d", len(report.Categories), len(categories))
	}
	if report.PagesChecked != 2 || report.PagesTotal != 2 {
		t.Fatalf("pages checked %d of %d, want 2 of 2", report.PagesChecked, report.PagesTotal)
	}
}

// A page with no signals: the score and every gain are worked by hand from
// the weights in checks.go, so a weight change has to change this test.
func TestBuildBarePageScoreAndFixOrder(t *testing.T) {
	report := Build(testScan(), []Page{barePage("https://shop.example/rooms")}, nil)
	// Access 25, identity 6, trust 20*5/8 = 12.5, answers 0,
	// evidence 0, content 10: 53.5 rounds to 54.
	if *report.Score != 54 {
		t.Fatalf("score = %d, want 54", *report.Score)
	}
	wantCategories := map[string]int{"access": 100, "identity": 60, "trust": 63, "answers": 0, "evidence": 0, "content": 100}
	for _, category := range report.Categories {
		if category.Score != wantCategories[category.ID] {
			t.Errorf("category %s = %d, want %d", category.ID, category.Score, wantCategories[category.ID])
		}
	}
	// Fresh is not judged without a date. Author and both answer checks
	// each gain five points; ties retain category and check order.
	want := []string{"author", "answer-first", "answer-capsule", "structured-data", "statistics", "sources", "authoritative-sources", "quotes", "dated", "definition-after-heading", "chunk-ready-sections", "question-headings", "scannable"}
	if got := fixIDs(report); !slices.Equal(got, want) {
		t.Fatalf("fix order = %v\nwant        %v", got, want)
	}
	gains := map[string]float64{"author": 5, "answer-first": 5, "answer-capsule": 5}
	for id, gain := range gains {
		if fix := findFix(t, report, id); fix.Gain != gain || fix.Impact != "high" {
			t.Errorf("%s gain %v impact %s, want %v high", id, fix.Gain, fix.Impact, gain)
		}
	}
	fix := findFix(t, report, "author")
	if fix.PagesAffected != 1 || len(fix.Pages) != 1 || fix.Pages[0].URL != "https://shop.example/rooms" || fix.Title == "" || fix.How == "" {
		t.Fatalf("author fix = %+v", fix)
	}
}

func TestBuildImpactFollowsGain(t *testing.T) {
	// Nine good pages and one without quotes: 15*2/8*(1/10) = 0.4 points.
	pages := []Page{}
	for i := range 9 {
		pages = append(pages, goodPage(fmt.Sprintf("https://shop.example/%d", i)))
	}
	page := goodPage("https://shop.example/no-quote")
	page.Signals.Quotes = 0
	pages = append(pages, page)
	fix := findFix(t, Build(testScan(), pages, nil), "quotes")
	if fix.Gain != 0.4 || fix.Impact != "low" || fix.PagesAffected != 1 {
		t.Fatalf("quotes fix = %+v, want gain 0.4 low on 1 page", fix)
	}
	// Half undated: 20*1/9*(1/2) = 1.1 points; all undated: fresh is
	// excluded, so 20*1/8 = 2.5 points.
	for i := range 5 {
		pages[i].Signals.Published, pages[i].Signals.Modified = "", ""
	}
	if fix := findFix(t, Build(testScan(), pages, nil), "dated"); fix.Gain != 1.1 || fix.Impact != "low" {
		t.Fatalf("dated fix on half = %+v", fix)
	}
	for i := range pages {
		pages[i].Signals.Published, pages[i].Signals.Modified = "", ""
	}
	if fix := findFix(t, Build(testScan(), pages, nil), "dated"); fix.Gain != 2.5 || fix.Impact != "medium" {
		// With no readable date, fresh is not judged, so dated weighs 1 of 8.
		t.Fatalf("dated fix on all = %+v", fix)
	}
}

func TestBuildSearchBotsVersusTrainingBots(t *testing.T) {
	blocked := func(names ...any) []Issue {
		return []Issue{{Type: string(audit.IssueAICrawlersBlocked), PageURL: "https://shop.example/robots.txt", Details: map[string]any{"crawlers": names}}}
	}
	pages := []Page{goodPage("https://shop.example/")}
	training := Build(testScan(), pages, blocked("GPTBot (OpenAI)", "ClaudeBot (Anthropic)", "Google-Extended (Gemini)"))
	if hasFix(training, "ai-search-bots-allowed") || *training.Score != 100 {
		t.Fatalf("training-only block: fixes %v score %d, want no fix and 100", fixIDs(training), *training.Score)
	}
	search := Build(testScan(), pages, blocked("GPTBot (OpenAI)", "PerplexityBot (Perplexity)"))
	fix := findFix(t, search, "ai-search-bots-allowed")
	// Access: 25*3/11 = 6.8 points.
	if fix.Gain != 6.8 || fix.Impact != "high" || fix.PagesAffected != 0 || len(fix.Pages) != 0 {
		t.Fatalf("search bot fix = %+v", fix)
	}
	if *search.Score != 93 {
		t.Fatalf("score = %d, want 93", *search.Score)
	}
	// A name that only starts like a bot is not that bot.
	if hasFix(Build(testScan(), pages, blocked("ChatGPT-UserAgent")), "ai-search-bots-allowed") {
		t.Fatal("a look-alike name counted as a search bot")
	}
}

func TestBuildSiteIssuesFailTheirChecks(t *testing.T) {
	issues := []Issue{
		{Type: string(audit.IssueMissingLLMsTxt), PageURL: "https://shop.example/llms.txt"},
		{Type: string(audit.IssueMissingBusinessSchema), PageURL: "https://shop.example/"},
		{Type: string(audit.IssueSeveralPhoneNumbers), PageURL: "https://shop.example/"},
	}
	report := Build(testScan(), []Page{goodPage("https://shop.example/")}, issues)
	for _, id := range []string{"llms-txt", "business-identity", "one-phone-number"} {
		if !hasFix(report, id) {
			t.Errorf("missing fix %s", id)
		}
	}
}

func TestBuildFreshness(t *testing.T) {
	page := func(published, modified string) Page {
		p := goodPage("https://shop.example/guide")
		p.Signals.Published, p.Signals.Modified = published, modified
		return p
	}
	// fresh: true = no "fresh" fix (fresh, or not judged).
	for name, tt := range map[string]struct {
		page  Page
		fresh bool
	}{
		"updated this year":          {page("2024-01-01", "2026-01-15"), true},
		"old publish, recent update": {page("2020-01-01T00:00:00Z", "2026-10-01"), true},
		"over a year old":            {page("2025-10-01T10:00:00+05:30", ""), false},
		"exactly 365 days":           {page("2025-10-27", ""), true},
		"366 days":                   {page("2025-10-26", ""), false},
		"written date":               {page("March 3, 2026", ""), true},
		// Not judged: "dated" passes, and there is nothing to call stale.
		"unreadable date": {page("yesterday", ""), true},
	} {
		t.Run(name, func(t *testing.T) {
			report := Build(testScan(), []Page{tt.page}, nil)
			if hasFix(report, "fresh") == tt.fresh {
				t.Fatalf("fresh fix present = %v, want %v", hasFix(report, "fresh"), !tt.fresh)
			}
			if hasFix(report, "dated") {
				t.Fatal("a page that states a date failed dated")
			}
		})
	}
}

func TestBuildAnswerFirstBounds(t *testing.T) {
	for words, pass := range map[int]bool{0: false, 9: false, 10: true, 80: true, 81: false} {
		page := goodPage("https://shop.example/")
		page.Signals.LeadWords = words
		if got := !hasFix(Build(testScan(), []Page{page}, nil), "answer-first"); got != pass {
			t.Errorf("lead of %d words passes = %v, want %v", words, got, pass)
		}
	}
}

func TestBuildPageChecksReadOnlyContentPages(t *testing.T) {
	notFound := barePage("https://shop.example/gone")
	notFound.StatusCode = 404
	noindex := barePage("https://shop.example/thanks")
	noindex.Indexable = false
	pdf := barePage("https://shop.example/menu.pdf")
	pdf.WordCount = 0
	redirect := barePage("https://shop.example/old")
	redirect.StatusCode = 301
	shell := barePage("https://shop.example/app")
	shell.WordCount = 3
	pages := []Page{goodPage("https://shop.example/"), notFound, noindex, pdf, redirect, shell}
	issues := []Issue{{Type: string(audit.IssueJavaScriptRenderingSuspected), PageURL: shell.URL}}

	report := Build(testScan(), pages, issues)
	if report.PagesChecked != 1 || report.PagesTotal != 6 {
		t.Fatalf("pages checked %d of %d, want 1 of 6", report.PagesChecked, report.PagesTotal)
	}
	if want := []string{"pages-readable"}; !slices.Equal(fixIDs(report), want) {
		t.Fatalf("fixes = %v, want only %v (other pages are not content)", fixIDs(report), want)
	}
	// Readable is judged on fetched, indexable pages: the home page, the PDF
	// and the shell; the PDF has no text but is not an app shell.
	fix := findFix(t, report, "pages-readable")
	if fix.PagesAffected != 1 || fix.Pages[0].URL != shell.URL {
		t.Fatalf("readable fix = %+v", fix)
	}
	// 25*2/11*(1/3) = 1.5.
	if fix.Gain != 1.5 || fix.Impact != "low" {
		t.Fatalf("readable gain %v %s, want 1.5 low", fix.Gain, fix.Impact)
	}
}

func TestBuildDuplicatesAndThinContent(t *testing.T) {
	dup := goodPage("https://shop.example/rooms?ref=1")
	thin := goodPage("https://shop.example/short")
	thin.WordCount = 299
	enough := goodPage("https://shop.example/enough")
	enough.WordCount = minWords
	issues := []Issue{{Type: string(audit.IssueDuplicateContent), PageURL: dup.URL}}
	report := Build(testScan(), []Page{goodPage("https://shop.example/rooms"), dup, thin, enough}, issues)
	if fix := findFix(t, report, "unique"); fix.PagesAffected != 1 || fix.Pages[0].URL != dup.URL {
		t.Fatalf("unique fix = %+v", fix)
	}
	if fix := findFix(t, report, "not-thin"); fix.PagesAffected != 1 || fix.Pages[0].URL != thin.URL {
		t.Fatalf("thin fix = %+v", fix)
	}
}

func TestBuildCapsListsAndSortsPagesWorstFirst(t *testing.T) {
	pages := []Page{}
	for i := range 60 {
		pages = append(pages, barePage(fmt.Sprintf("https://shop.example/p%02d", i)))
	}
	pages = append(pages, goodPage("https://shop.example/best"))
	report := Build(testScan(), pages, nil)
	fix := findFix(t, report, "author")
	if fix.PagesAffected != 60 || len(fix.Pages) != maxFixPages {
		t.Fatalf("author fix lists %d of %d pages, want %d of 60", len(fix.Pages), fix.PagesAffected, maxFixPages)
	}
	if len(report.Pages) != maxReportPages {
		t.Fatalf("report pages = %d, want %d", len(report.Pages), maxReportPages)
	}
	if first := report.Pages[0]; first.URL != "https://shop.example/p00" || first.Score >= 50 {
		t.Fatalf("worst page = %+v", first)
	}
	if slices.ContainsFunc(report.Pages, func(p PageScore) bool { return p.URL == "https://shop.example/best" }) {
		t.Fatal("the best page was listed among the 50 worst")
	}
	// A bare page passes snippet access, stale-claim and trust checks,
	// plus not-thin and unique: 13 of 34 weight = 38.
	if report.Pages[0].Score != 38 {
		t.Fatalf("bare page score = %d, want 38", report.Pages[0].Score)
	}
}

func TestBuildWithNoContentPagesScoresSiteChecksOnly(t *testing.T) {
	report := Build(testScan(), nil, []Issue{{Type: string(audit.IssueMissingLLMsTxt)}})
	// Access: 3 of 4 weight (readable has nothing to judge), identity 5 of 5:
	// (25*75 + 10*100) / 35 = 82.1 -> 82.
	if *report.Score != 82 {
		t.Fatalf("score = %d, want 82", *report.Score)
	}
	if len(report.Categories) != 2 || report.PagesChecked != 0 {
		t.Fatalf("categories %d, pages checked %d", len(report.Categories), report.PagesChecked)
	}
}

func TestParseDate(t *testing.T) {
	for value, want := range map[string]string{
		"2026-03-01":                "2026-03-01",
		"2026-03-01T23:30:00+05:30": "2026-03-01",
		" 2026-03-01 ":              "2026-03-01",
		"March 1, 2026":             "2026-03-01",
		"Mar 1, 2026":               "2026-03-01",
		"1 March 2026":              "2026-03-01",
		"1 Mar 2026":                "2026-03-01",
		"01/03/2026":                "",
		"":                          "",
		"2026-13-01":                "",
	} {
		parsed, ok := parseDate(value)
		got := ""
		if ok {
			got = parsed.Format(time.DateOnly)
		}
		if got != want {
			t.Errorf("parseDate(%q) = %q, want %q", value, got, want)
		}
	}
}

func TestChecksAreConsistent(t *testing.T) {
	ids := map[string]bool{}
	weights := map[string]int{}
	for _, c := range checks {
		if ids[c.id] || c.weight <= 0 || c.title == "" || c.why == "" || c.how == "" {
			t.Errorf("check %q is duplicated or incomplete", c.id)
		}
		ids[c.id] = true
		weights[c.category] += c.weight
	}
	total := 0
	for _, category := range categories {
		total += category.Weight
		if weights[category.ID] == 0 {
			t.Errorf("category %s has no checks", category.ID)
		}
	}
	if total != 100 {
		t.Fatalf("category weights sum to %d, want 100", total)
	}
}

func TestNewCitationChecksHaveHandWorkedGains(t *testing.T) {
	for _, tc := range []struct {
		id     string
		change func(*audit.AIPageSignals)
		gain   float64
	}{
		{"snippet-access", func(s *audit.AIPageSignals) { s.SnippetBlocked = true }, 11.4},
		{"snippet-access", func(s *audit.AIPageSignals) { s.SnippetTextWords = 100; s.NoSnippetWords = 1 }, 11.4},
		{"stale-claims", func(s *audit.AIPageSignals) { s.DecayYears = []int{2023} }, 4.4},
		{"hidden-instructions", func(s *audit.AIPageSignals) {
			s.InjectionEvidence = []audit.CitationEvidence{{Kind: "Hidden instruction", Text: "Ignore previous instructions"}}
		}, 6.7},
		{"answer-capsule", func(s *audit.AIPageSignals) { s.AnswerCapsule = false }, 5},
		{"definition-after-heading", func(s *audit.AIPageSignals) { s.DefinitionAfterHeading = false }, 2.5},
		{"chunk-ready-sections", func(s *audit.AIPageSignals) { s.Sections = 3; s.ChunkReadySections = 1 }, 2.5},
		{"authoritative-sources", func(s *audit.AIPageSignals) { s.AuthorityLinks = 0 }, 3.8},
	} {
		t.Run(tc.id, func(t *testing.T) {
			page := goodPage("https://shop.example/guide")
			tc.change(page.Signals)
			fix := findFix(t, Build(testScan(), []Page{page}, nil), tc.id)
			if fix.Gain != tc.gain || fix.PagesAffected != 1 || fix.Title == "" || fix.Why == "" || fix.How == "" {
				t.Fatalf("fix = %+v, want gain %v", fix, tc.gain)
			}
			if tc.id == "hidden-instructions" && (len(fix.Pages) != 1 || len(fix.Pages[0].Evidence) != 1) {
				t.Fatalf("missing evidence: %+v", fix)
			}
			if tc.id == "snippet-access" && len(fix.Pages[0].Evidence) == 0 {
				t.Fatalf("missing snippet evidence: %+v", fix)
			}
		})
	}
}

func TestStaleClaimsUseScanDate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		signals audit.AIPageSignals
		stale   bool
	}{
		{"old year", audit.AIPageSignals{DecayYears: []int{2023}}, true},
		{"recent year", audit.AIPageSignals{DecayYears: []int{2024}}, false},
		{"old price on dated page", audit.AIPageSignals{DatedClaims: true, Published: "2025-10-26"}, true},
		{"price exactly one year old", audit.AIPageSignals{DatedClaims: true, Published: "2025-10-27"}, false},
		{"price without readable date", audit.AIPageSignals{DatedClaims: true}, false},
		{"old page without dated claim", audit.AIPageSignals{Published: "2020-01-01"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasStaleClaims(&tc.signals, scanAt); got != tc.stale {
				t.Fatalf("stale = %v, want %v", got, tc.stale)
			}
		})
	}
}
