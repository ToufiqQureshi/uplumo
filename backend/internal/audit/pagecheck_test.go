package audit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

const goodPage = `<!doctype html><html lang="en"><head>
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Fresh cakes delivered in Pune | Demo Bakery</title>
<meta name="description" content="Order fresh eggless cakes online in Pune. Same-day delivery from 499, baked every morning by Demo Bakery.">
<link rel="canonical" href="http://shop.test/">
<meta property="og:title" content="Demo Bakery"><meta property="og:image" content="http://shop.test/og.png">
<meta name="author" content="Asha Rao"><meta property="article:modified_time" content="2026-11-01">
<script type="application/ld+json">{"@type":"Bakery","name":"Demo Bakery"}</script>
<script type="application/ld+json">{"@type":"Organization","name":"Demo Bakery"}</script>
</head><body><main><h1>Fresh cakes in Pune</h1>
<p>Demo Bakery delivers fresh eggless cakes across Pune the same day, starting at €9, with 98% happy reviews from 2,000 customers since 2015. Order before 2 pm and we bake it that morning, pack it cold and deliver it within three hours to any address inside the city limits.</p>
<h2>How much does a cake cost?</h2><p>Cakes start at 499 rupees for half a kilo and 899 rupees for one kilo.</p>
<h2>Do you deliver on Sundays?</h2><p>Yes, every day from 9 am to 9 pm.</p>
<ul><li>Eggless</li><li>Sugar-free</li></ul>
` + "<p>" + "Our bakers use butter, cream and seasonal fruit in every cake we make for birthdays and weddings. " + "</p>" +
	`<img src="/a.jpg" alt="Chocolate cake" width="10" height="10">
<a href="/menu">Menu</a><a href="/about">About</a><a href="/contact">Contact</a><a href="/blog">Blog</a><a href="/faq">FAQ</a>
</main></body></html>`

// checkerFor serves pages from routes as the public host shop.test.
func checkerFor(t *testing.T, routes map[string]func(http.ResponseWriter)) (*PageChecker, *atomic.Int32) {
	t.Helper()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if route, ok := routes[r.URL.Path]; ok {
			route(w)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)
	guard := &Guard{Resolver: fakeResolver{addrs: map[string][]string{"shop.test": {"93.184.216.34"}}}, Client: dialAllTo(server)}
	return NewPageChecker(guard), &requests
}

func htmlPage(body string) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(body))
	}
}

func plainText(body string) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) { _, _ = w.Write([]byte(body)) }
}

func statuses(report PageReport) map[string]CheckStatus {
	got := map[string]CheckStatus{}
	for _, check := range report.Checks {
		got[check.ID] = check.Status
	}
	return got
}

func TestPageCheckerScoresAWellBuiltPage(t *testing.T) {
	checker, requests := checkerFor(t, map[string]func(http.ResponseWriter){
		"/":            htmlPage(goodPage),
		"/robots.txt":  plainText("User-agent: *\nAllow: /\nSitemap: http://shop.test/sitemap.xml\n"),
		"/llms.txt":    plainText("# Demo Bakery\n- /menu: cakes and prices\n"),
		"/sitemap.xml": plainText(`<?xml version="1.0"?><urlset><url><loc>http://shop.test/</loc></url></urlset>`),
	})
	report := checker.Check(context.Background(), "http://shop.test/")
	if !report.Reachable {
		t.Fatalf("unreachable: %s", report.Problem)
	}
	got := statuses(report)
	for _, id := range []string{"title", "description", "h1", "canonical", "indexable", "open-graph", "image-alt",
		"ai-crawlers", "business-schema", "questions", "no-javascript", "snippet", "facts", "author-date", "llms-txt",
		"viewport", "language", "robots-txt", "sitemap", "image-size", "internal-links", "structure"} {
		if got[id] != CheckPass {
			t.Errorf("check %s = %q, want pass", id, got[id])
		}
	}
	// Plain HTTP is the only technical failure on this page.
	if got["https"] != CheckFail {
		t.Errorf("https = %q, want fail", got["https"])
	}
	if len(report.Categories) != 4 || report.Score <= 70 {
		t.Fatalf("categories = %+v, score = %d", report.Categories, report.Score)
	}
	// One page, robots.txt, llms.txt and the sitemap: nothing else.
	if n := requests.Load(); n != 4 {
		t.Fatalf("requests to the site = %d, want 4", n)
	}
}

func TestPageCheckerFlagsAPoorPage(t *testing.T) {
	shell := `<html><head><title>App</title></head><body><div id="root"></div><script src="/app.js"></script></body></html>`
	checker, _ := checkerFor(t, map[string]func(http.ResponseWriter){
		"/":           htmlPage(shell),
		"/robots.txt": plainText("User-agent: PerplexityBot\nDisallow: /\nUser-agent: GPTBot\nDisallow: /\n"),
	})
	report := checker.Check(context.Background(), "http://shop.test/")
	got := statuses(report)
	for id, want := range map[string]CheckStatus{
		"ai-crawlers": CheckFail, "description": CheckFail, "business-schema": CheckFail, "sitemap": CheckWarn,
		"llms-txt": CheckWarn, "title": CheckWarn, "no-javascript": CheckFail, "word-count": CheckFail, "image-alt": CheckInfo,
	} {
		if got[id] != want {
			t.Errorf("check %s = %q, want %q", id, got[id], want)
		}
	}
	for _, check := range report.Checks {
		if check.ID == "ai-crawlers" && (!strings.Contains(check.Found, "PerplexityBot") || strings.Contains(check.Found, "GPTBot")) {
			t.Errorf("ai-crawlers found = %q: should name the search bot, not the training bot", check.Found)
		}
		if check.Status != CheckPass && check.Status != CheckInfo && check.Fix == "" {
			t.Errorf("check %s has no fix", check.ID)
		}
	}
	if report.Score >= 60 {
		t.Fatalf("score = %d, want a low score", report.Score)
	}
}

func TestPageCheckerExplainsUnreachablePages(t *testing.T) {
	cases := map[string]func(http.ResponseWriter){
		"firewall": func(w http.ResponseWriter) {
			w.Header().Set("Cf-Mitigated", "challenge")
			w.WriteHeader(http.StatusForbidden)
		},
		"server error": func(w http.ResponseWriter) { w.WriteHeader(http.StatusBadGateway) },
		"not html": func(w http.ResponseWriter) {
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = w.Write([]byte("%PDF-1.7"))
		},
		"redirect loop": func(w http.ResponseWriter) {
			w.Header().Set("Location", "http://shop.test/")
			w.WriteHeader(http.StatusMovedPermanently)
		},
	}
	for name, route := range cases {
		t.Run(name, func(t *testing.T) {
			checker, requests := checkerFor(t, map[string]func(http.ResponseWriter){"/": route})
			report := checker.Check(context.Background(), "http://shop.test/")
			if report.Reachable || report.Problem == "" || report.Checks != nil || report.Score != 0 {
				t.Fatalf("report = %+v, want an explained failure without a score", report)
			}
			// The redirect loop stops after its hop limit.
			if n := requests.Load(); n > 3+maxPageCheckRedirects {
				t.Fatalf("requests = %d", n)
			}
		})
	}
}

func TestLengthCheckCountsCharactersNotBytes(t *testing.T) {
	// 40 Devanagari characters are 120 bytes: not "too long".
	title := strings.Repeat("क", 40)
	if got := lengthCheck("title", CategorySEO, 1, "Title", title, titleMinChars, titleMaxChars, "", "fix"); got.Status != CheckPass {
		t.Fatalf("status = %q (%s), want pass", got.Status, got.Found)
	}
}

func TestScoreChecksWeighsWarningsHalfAndSkipsInfo(t *testing.T) {
	categories, score := scoreChecks([]PageCheck{
		{Category: CategorySEO, Status: CheckPass, Weight: 3},
		{Category: CategorySEO, Status: CheckWarn, Weight: 1},
		{Category: CategorySEO, Status: CheckInfo, Weight: 5},
		{Category: CategoryAI, Status: CheckFail, Weight: 2},
		{Category: CategoryTechnical, Status: CheckPass, Weight: 1},
		{Category: CategoryContent, Status: CheckWarn, Weight: 1},
	})
	want := map[string]int{CategorySEO: 88, CategoryAI: 0, CategoryTechnical: 100, CategoryContent: 50}
	for _, category := range categories {
		if category.Score != want[category.ID] {
			t.Errorf("%s = %d, want %d", category.ID, category.Score, want[category.ID])
		}
	}
	if score != 60 {
		t.Fatalf("overall = %d, want 60", score)
	}
}
