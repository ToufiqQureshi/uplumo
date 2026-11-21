package audit

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// sectionText is 128 words in 8 sentences of 16 words: a section that the
// chunk-ready check counts (100-150 words).
var sectionText = strings.Repeat("Each cake is baked to order in our Pune kitchen with fresh butter and seasonal fruit. ", 8)

// shortText is 64 words: the filler under a direct answer, so a section stays
// within the 100-150 words the chunk-ready check counts.
var shortText = strings.Repeat("Each cake is baked to order in our Pune kitchen with fresh butter and seasonal fruit. ", 4)

// goodPage is a well-built page on the test host shop.test (served over
// plain HTTP, so only the HTTPS check fails on it).
var goodPage = fmt.Sprintf(`<!doctype html><html lang="en"><head>
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Fresh eggless cakes delivered in Pune</title>
<meta name="description" content="Order fresh eggless cakes online in Pune. Same-day delivery from Rs 499, baked every morning by Demo Bakery.">
<link rel="canonical" href="http://shop.test/">
<meta property="og:site_name" content="Demo Bakery">
<meta property="og:title" content="Demo Bakery"><meta property="og:description" content="Fresh cakes in Pune"><meta property="og:image" content="http://shop.test/og.png">
<meta name="author" content="Asha Rao"><meta property="article:modified_time" content="%[1]s">
<script type="application/ld+json">{"@context":"https://schema.org","@type":"Organization","name":"Demo Bakery","url":"http://shop.test/","logo":"http://shop.test/logo.png","sameAs":["https://www.instagram.com/demobakery","https://www.facebook.com/demobakery"],"telephone":"+91 22 1234 5678","address":{"@type":"PostalAddress","streetAddress":"12 MG Road","addressLocality":"Pune"}}</script>
</head><body><main><h1>Fresh cakes in Pune</h1>
<p>Demo Bakery delivers fresh eggless cakes across Pune the same day, starting at €9, with 98%% happy reviews from 2,000 customers since 2015. Orders before 2 pm are baked that morning and delivered within three hours.</p>
<h2>How much does a cake cost?</h2><p>Cakes start at ₹499 for half a kilo and ₹899 for one kilo. Prices include same-day delivery within Pune and eggless baking. Custom designs are quoted after you describe the cake.</p><p>%[3]s Cakes start at ₹499 for half a kilo and ₹899 for one kilo.</p>
<h2>Do you deliver on Sundays?</h2><p>Yes, we deliver every day of the week from 9 am to 9 pm, including Sundays and public holidays across Pune.</p><p>%[3]s Yes, every day from 9 am to 9 pm.</p>
<h2>Can you make eggless cakes?</h2><p>Yes, every cake is eggless by default and we bake without eggs in our Pune kitchen. Tell us about any allergy when you order.</p><p>%[3]s Every cake is eggless by default. Tell us any allergy when you order.</p>
<h2>Our process</h2><p>%[2]s Our bakers use butter, cream and seasonal fruit in every cake we make for birthdays and weddings.</p>
<ul><li>Eggless</li><li>Sugar-free</li></ul>
<p>Food safety rules are published by <a href="https://www.fssai.gov.in/">FSSAI</a>.</p>
<img src="/a.jpg" alt="Chocolate cake" width="10" height="10">
<a href="/menu">Menu</a><a href="/about">About</a><a href="/contact">Contact</a><a href="/blog">Blog</a><a href="/faq">FAQ</a>
<a href="/services">Services</a><a href="/prices">Prices</a><a href="/delivery">Delivery</a><a href="/reviews">Reviews</a><a href="/custom">Custom cakes</a><a href="/gallery">Gallery</a>
</main></body></html>`, time.Now().Format("2006-01-02"), sectionText, shortText)

// yoastPage is a page with the usual WordPress and Yoast gaps: thin business
// schema, two question headings, only og:title, and images without sizes.
var yoastPage = func() string {
	var images strings.Builder
	for range 26 {
		images.WriteString(`<img src="/p.jpg" alt="cake">`)
	}
	return `<!doctype html><html lang="en"><head>
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Demo Bakery Pune Cakes</title>
<meta name="description" content="Order fresh cakes online in Pune. Same-day delivery and a wide range of eggless and sugar-free options for every occasion.">
<link rel="canonical" href="http://shop.test/">
<meta property="og:title" content="Demo Bakery">
<script type="application/ld+json">{"@type":"Organization","name":"Demo Bakery","url":"http://shop.test/"}</script>
</head><body><main><h1>Demo Bakery</h1>
<h2>What is a cake?</h2><p>A cake is a baked dessert made with flour, sugar, eggs and fat.</p>
<h2>Do you deliver?</h2><p>Yes, we deliver across the city every day from morning until late evening.</p>
<p>Our bakers make cakes for birthdays, weddings and office parties, with flavours for every taste and diet.</p>
` + images.String() + `<a href="/menu">Menu</a><a href="/contact">Contact</a>
</main></body></html>`
}()

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

// statuses maps each check ID of a report to its status.
func statuses(report PageReport) map[string]CheckStatus {
	got := map[string]CheckStatus{}
	for _, check := range report.Checks {
		got[check.ID] = check.Status
	}
	return got
}

// standardSite serves the page at / with the site files a complete site has.
func standardSite(page string) map[string]func(http.ResponseWriter) {
	return map[string]func(http.ResponseWriter){
		"/":            htmlPage(page),
		"/robots.txt":  plainText("User-agent: *\nAllow: /\nSitemap: http://shop.test/sitemap.xml\n"),
		"/llms.txt":    plainText("# Demo Bakery\n- /menu: cakes and prices\n"),
		"/sitemap.xml": plainText(`<?xml version="1.0"?><urlset><url><loc>http://shop.test/</loc></url></urlset>`),
	}
}

// checkBoth fetches pageURL once and scores the snapshot with both engines.
func checkBoth(t *testing.T, checker *PageChecker, pageURL string) (PageSnapshot, PageReport, PageReport) {
	t.Helper()
	snap := checker.Check(context.Background(), pageURL)
	return snap, ScoreSEO(snap), ScoreAEO(snap)
}

// withoutHTTPSFailure turns the HTTPS check of a report into a pass and
// scores again. The fixtures are served over plain HTTP by the test server, so
// this shows what the page would score on HTTPS.
func withoutHTTPSFailure(report PageReport, categories []categoryDef, severeMax int) int {
	checks := append([]PageCheck(nil), report.Checks...)
	for i := range checks {
		if checks[i].ID == "https" {
			checks[i].Status = CheckPass
		}
	}
	_, score, _ := scoreChecks(checks, categories, severeMax)
	return score
}

func TestSnapshotIsScoredByBothEnginesWithOneFetch(t *testing.T) {
	checker, requests := checkerFor(t, standardSite(goodPage))
	snap, seo, aeo := checkBoth(t, checker, "http://shop.test/")
	if !snap.Reachable || !seo.Reachable || !aeo.Reachable {
		t.Fatalf("unreachable: %q", snap.Problem)
	}
	if seo.Kind != KindSEO || aeo.Kind != KindAEO {
		t.Fatalf("kinds = %q and %q", seo.Kind, aeo.Kind)
	}
	// One page, robots.txt, llms.txt and the sitemap: nothing else. Scoring
	// both engines makes no request.
	if n := requests.Load(); n != 4 {
		t.Fatalf("requests to the site = %d, want 4", n)
	}
}

func TestWellBuiltPagePassesMostChecksOnBothEngines(t *testing.T) {
	checker, _ := checkerFor(t, standardSite(goodPage))
	_, seo, aeo := checkBoth(t, checker, "http://shop.test/")
	// Plain HTTP is the only failure on this test page, and it is critical.
	if got := statuses(seo)["https"]; got != CheckFail {
		t.Fatalf("SEO https = %q, want fail", got)
	}
	if _, ok := statuses(aeo)["https"]; ok {
		t.Fatalf("AEO has an https check; HTTPS is a crawl and index matter for SEO only")
	}
	for _, report := range []PageReport{seo, aeo} {
		passed := 0
		var failing []string
		for _, check := range report.Checks {
			switch check.Status {
			case CheckPass:
				passed++
			case CheckFail:
				failing = append(failing, check.ID)
			case CheckWarn, CheckInfo:
			}
		}
		if report.Kind == KindSEO && (len(failing) != 1 || failing[0] != "https") {
			t.Errorf("%s: failing checks = %v, want only https", report.Kind, failing)
		}
		if report.Kind == KindAEO && len(failing) != 0 {
			t.Errorf("%s: failing checks = %v, want none", report.Kind, failing)
		}
		if passed*100 < len(report.Checks)*60 {
			t.Errorf("%s: passed %d of %d checks, want most of them", report.Kind, passed, len(report.Checks))
		}
		if len(report.Categories) != 4 {
			t.Errorf("%s: %d categories, want 4", report.Kind, len(report.Categories))
		}
	}
	// On HTTPS the page is well built for both engines.
	if got := withoutHTTPSFailure(seo, seoCategories, 49); got < 80 {
		t.Errorf("SEO without the HTTP failure = %d, want at least 80", got)
	}
	if got := withoutHTTPSFailure(aeo, aeoCategories, 39); got < 80 {
		t.Errorf("AEO without the HTTP failure = %d, want at least 80", got)
	}
	if len(seo.Caps) != 1 || !strings.Contains(seo.Caps[0], "capped at 49") {
		t.Errorf("SEO caps = %v, want the critical HTTPS cap of 49", seo.Caps)
	}
	if len(aeo.Caps) != 0 {
		t.Errorf("AEO caps = %v, want none: HTTP does not cap AEO", aeo.Caps)
	}
}

func TestYoastLikePageScoresBelowEightyOnBothEngines(t *testing.T) {
	checker, _ := checkerFor(t, standardSite(yoastPage))
	_, seo, aeo := checkBoth(t, checker, "http://shop.test/")
	if !seo.Reachable || !aeo.Reachable {
		t.Fatalf("unreachable: %q", seo.Problem)
	}
	for _, report := range []PageReport{seo, aeo} {
		if report.Score >= 80 {
			t.Errorf("%s score = %d, want below 80", report.Kind, report.Score)
		}
		for _, check := range report.Checks {
			if check.Status != CheckPass && check.Status != CheckInfo && check.Fix == "" {
				t.Errorf("%s check %s has no fix", report.Kind, check.ID)
			}
		}
	}
	// The two engines disagree about the same page: SEO sees the image sizes
	// and the missing structure of the page, AEO sees the missing answers.
	seoCats, aeoCats := categoryScores(seo), categoryScores(aeo)
	if seoCats["onpage"] == aeoCats["answer"] {
		t.Errorf("category scores should differ between engines: seo %v, aeo %v", seoCats, aeoCats)
	}
	if got := statuses(aeo)["question-headings"]; got != CheckWarn {
		t.Errorf("AEO question-headings = %q, want warn with two questions", got)
	}
	if got := statuses(seo)["image-size"]; got != CheckFail {
		t.Errorf("SEO image-size = %q, want fail with 26 images without size", got)
	}
}

// categoryScores maps a report's category IDs to their scores.
func categoryScores(report PageReport) map[string]int {
	scores := map[string]int{}
	for _, category := range report.Categories {
		scores[category.ID] = category.Score
	}
	return scores
}

func TestPoorPageIsLowOnBothEngines(t *testing.T) {
	shell := `<html><head><title>App</title></head><body><div id="root"></div><script src="/app.js"></script></body></html>`
	checker, _ := checkerFor(t, map[string]func(http.ResponseWriter){
		"/":           htmlPage(shell),
		"/robots.txt": plainText("User-agent: PerplexityBot\nDisallow: /\nUser-agent: GPTBot\nDisallow: /\n"),
	})
	_, seo, aeo := checkBoth(t, checker, "http://shop.test/")
	for _, report := range []PageReport{seo, aeo} {
		if report.Score >= 60 {
			t.Errorf("%s score = %d, want a low score", report.Kind, report.Score)
		}
		for _, check := range report.Checks {
			if check.Status != CheckPass && check.Status != CheckInfo && check.Fix == "" {
				t.Errorf("%s check %s has no fix", report.Kind, check.ID)
			}
		}
	}
	if got := statuses(aeo)["ai-crawlers"]; got != CheckFail {
		t.Errorf("AEO ai-crawlers = %q, want fail", got)
	}
	if got := statuses(seo)["word-count"]; got != CheckFail {
		t.Errorf("SEO word-count = %q, want fail", got)
	}
	if got := statuses(aeo)["no-javascript"]; got != CheckFail {
		t.Errorf("AEO no-javascript = %q, want fail", got)
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
			snap, seo, aeo := checkBoth(t, checker, "http://shop.test/")
			for _, report := range []PageReport{seo, aeo} {
				if report.Reachable || report.Problem == "" || report.Checks != nil || report.Score != 0 {
					t.Fatalf("%s report = %+v, want an explained failure without a score", report.Kind, report)
				}
			}
			if snap.Reachable {
				t.Fatalf("snapshot reachable for %s", name)
			}
			// The redirect loop stops after its hop limit.
			if n := requests.Load(); n > 3+maxPageCheckRedirects {
				t.Fatalf("requests = %d", n)
			}
		})
	}
}

// fakeFieldData is a Chrome field data source that answers one fixed result.
type fakeFieldData struct {
	data  StoredFieldData
	calls *atomic.Int32
}

func (f fakeFieldData) FieldData(_ context.Context, _, strategy string) StoredFieldData {
	f.calls.Add(1)
	if strategy != "phone" {
		return StoredFieldData{Status: FieldUnavailable}
	}
	return f.data
}

func TestCoreWebVitalsFromFieldDataReachBothReportsAsOneCheck(t *testing.T) {
	cases := map[string]struct {
		provider FieldDataProvider
		want     CheckStatus
		calls    int32
	}{
		"passed": {fakeFieldData{data: StoredFieldData{Status: FieldAvailable, CoreWebVitals: CoreWebVitalsPassed}}, CheckPass, 1},
		"failed": {fakeFieldData{data: StoredFieldData{Status: FieldAvailable, CoreWebVitals: CoreWebVitalsFailed,
			Metrics: []StoredFieldMetric{{Key: "lcp", Category: fieldPoor}, {Key: "cls", Category: fieldGood}}}}, CheckFail, 1},
		"insufficient": {fakeFieldData{data: StoredFieldData{Status: FieldInsufficientData}}, CheckInfo, 1},
		"unavailable":  {fakeFieldData{data: StoredFieldData{Status: FieldUnavailable}}, CheckInfo, 1},
		"not set":      {nil, CheckInfo, 0},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			checker, _ := checkerFor(t, standardSite(goodPage))
			var calls atomic.Int32
			if fake, ok := tc.provider.(fakeFieldData); ok {
				fake.calls = &calls
				tc.provider = fake
			}
			checker.WithFieldData(tc.provider)
			snap, seo, _ := checkBoth(t, checker, "http://shop.test/")
			if calls.Load() != tc.calls {
				t.Fatalf("field data lookups = %d, want %d", calls.Load(), tc.calls)
			}
			if snap.Field == nil && tc.calls > 0 {
				t.Fatalf("snapshot has no field data")
			}
			if got := statuses(seo)["core-web-vitals"]; got != tc.want {
				t.Fatalf("core-web-vitals = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPageChecksSkipFieldDataForUnreachablePages(t *testing.T) {
	checker, _ := checkerFor(t, map[string]func(http.ResponseWriter){"/": func(w http.ResponseWriter) { w.WriteHeader(http.StatusBadGateway) }})
	var calls atomic.Int32
	checker.WithFieldData(fakeFieldData{data: StoredFieldData{Status: FieldAvailable}, calls: &calls})
	checker.Check(context.Background(), "http://shop.test/")
	if calls.Load() != 0 {
		t.Fatalf("field data lookups for an unreachable page = %d, want 0", calls.Load())
	}
}

func TestScoreChecksWeighsSeveritiesAndWarnings(t *testing.T) {
	categories := []categoryDef{{ID: "a", Name: "A", Weight: 30}, {ID: "b", Name: "B", Weight: 70}}
	categoriesScored, score, caps := scoreChecks([]PageCheck{
		{Category: "a", Status: CheckPass, Severity: sevCritical, Weight: 8},
		{Category: "a", Status: CheckWarn, Severity: sevLow, Weight: 1},
		{Category: "a", Status: CheckInfo, Severity: sevHigh, Weight: 5},
		{Category: "b", Status: CheckWarn, Severity: sevMedium, Weight: 3},
		{Category: "b", Status: CheckPass, Severity: sevMedium, Weight: 3},
		{Category: "b", Status: CheckFail, Severity: sevLow, Weight: 1},
	}, categories, 49)
	// a: 8 + 0.35 of 9 points = 93. b: 3.35... of 7 points (warn 3 at 35%,
	// pass 3, fail 1) = 58. Weighted 30/70: 93*30 + 58*70 = 6850, rounded 69.
	want := map[string]int{"a": 93, "b": 58}
	for _, category := range categoriesScored {
		if got := category.Score; got != want[category.ID] {
			t.Errorf("%s = %d, want %d", category.ID, got, want[category.ID])
		}
	}
	if score != 69 || len(caps) != 0 {
		t.Fatalf("score = %d caps = %v, want 69 and no caps", score, caps)
	}
}

func TestApplyCapsLimitsSeriousProblems(t *testing.T) {
	pass := func(sev string, w int) PageCheck { return PageCheck{Status: CheckPass, Severity: sev, Weight: w} }
	fail := func(sev string, title string) PageCheck {
		return PageCheck{Status: CheckFail, Severity: sev, Title: title, Weight: severityWeights[sev]}
	}
	warn := func() PageCheck { return PageCheck{Status: CheckWarn, Severity: sevLow, Weight: 1, Title: "w"} }
	cases := map[string]struct {
		raw     int
		checks  []PageCheck
		want    int
		capsLen int
	}{
		"no problems":        {96, []PageCheck{pass(sevHigh, 5)}, 96, 0},
		"critical fail":      {96, []PageCheck{fail(sevCritical, "Allowed in Google")}, 49, 1},
		"high fail":          {96, []PageCheck{fail(sevHigh, "Main heading (H1)")}, 79, 1},
		"five warnings":      {96, []PageCheck{warn(), warn(), warn(), warn(), warn()}, 96, 0},
		"six warnings":       {96, []PageCheck{warn(), warn(), warn(), warn(), warn(), warn()}, 89, 1},
		"critical and high":  {96, []PageCheck{fail(sevCritical, "HTTPS"), fail(sevHigh, "Viewport")}, 49, 2},
		"cap above raw only": {45, []PageCheck{fail(sevCritical, "HTTPS")}, 45, 0},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, caps := applyCaps(tc.raw, tc.checks, 49)
			if got != tc.want || len(caps) != tc.capsLen {
				t.Fatalf("score = %d caps = %v, want %d with %d caps", got, caps, tc.want, tc.capsLen)
			}
		})
	}
}

func TestGradeLabelBands(t *testing.T) {
	cases := map[int]string{100: "Excellent", 90: "Excellent", 89: "Good", 80: "Good", 79: "Needs work", 50: "Needs work", 49: "Poor", 0: "Poor"}
	for score, want := range cases {
		if got := GradeLabel(score); got != want {
			t.Errorf("GradeLabel(%d) = %q, want %q", score, got, want)
		}
	}
}

func TestLengthCheckCountsCharactersNotBytes(t *testing.T) {
	// 40 Devanagari characters are 120 bytes: inside the 30-60 title range.
	title := strings.Repeat("क", 40)
	if got := titleCheck(title, ""); got.Status != CheckPass {
		t.Fatalf("status = %q (%s), want pass", got.Status, got.Found)
	}
	if got := titleCheck(strings.Repeat("क", 61), ""); got.Status != CheckWarn {
		t.Fatalf("61 characters = %q, want warn", got.Status)
	}
}
