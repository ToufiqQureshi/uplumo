package citefix

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/aisearch"
	"github.com/toufiqqureshi/uplumo/backend/internal/audit"
)

type publicRivalResolver struct{}

func (publicRivalResolver) LookupNetIP(context.Context, string, string) ([]netip.Addr, error) {
	return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
}

type rivalTestTransport struct {
	serverURL *url.URL
	base      http.RoundTripper
}

func (t rivalTestTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	urlCopy := *t.serverURL
	urlCopy.Path, urlCopy.RawQuery = request.URL.Path, request.URL.RawQuery
	clone.URL = &urlCopy
	return t.base.RoundTrip(clone)
}

func TestFetchRivalPageExtractsSignalsAndFlagsAppShellWithoutRendering(t *testing.T) {
	var pageGets atomic.Int32
	html := `<!doctype html><html><body><div id="root"></div><script src="/app.js"></script></body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			_, _ = w.Write([]byte("User-agent: *\nAllow: /\n"))
		case "/page":
			if r.Method == http.MethodGet {
				pageGets.Add(1)
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write([]byte(html))
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	target, _ := url.Parse(server.URL)
	client := &http.Client{Transport: rivalTestTransport{serverURL: target, base: server.Client().Transport}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	guard := &audit.Guard{Resolver: publicRivalResolver{}, Client: client}
	page, err := fetchRivalPage(t.Context(), guard, client, CitedPage{URL: "https://rival.example/page", Domain: "rival.example"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !page.AppShell || page.Signals.WordCount != 0 || pageGets.Load() != 1 {
		t.Fatalf("page = %+v, GETs = %d; app shell must be flagged and fetched once without rendering", page, pageGets.Load())
	}
}

func TestFetchRivalPageRespectsRobotsTxt(t *testing.T) {
	for _, robots := range []string{
		"User-agent: *\nDisallow: /page\n",
		// A rule for Uplumo alone must hold too, and the fetch must announce
		// the same agent the rule was judged for.
		"User-agent: Uplumo-Audit\nDisallow: /page\n\nUser-agent: *\nAllow: /\n",
	} {
		t.Run(robots, func(t *testing.T) { testRivalRobots(t, robots) })
	}
}

func testRivalRobots(t *testing.T, robots string) {
	t.Helper()
	var pageGets atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got != audit.UserAgent {
			t.Errorf("User-Agent = %q, want %q", got, audit.UserAgent)
		}
		if r.URL.Path == "/robots.txt" {
			_, _ = w.Write([]byte(robots))
			return
		}
		if r.URL.Path == "/page" && r.Method == http.MethodGet {
			pageGets.Add(1)
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<html><body>blocked</body></html>"))
		}
	}))
	defer server.Close()
	target, _ := url.Parse(server.URL)
	client := &http.Client{Transport: rivalTestTransport{serverURL: target, base: server.Client().Transport}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	guard := &audit.Guard{Resolver: publicRivalResolver{}, Client: client}
	if _, err := fetchRivalPage(t.Context(), guard, client, CitedPage{URL: "https://rival.example/page"}, time.Now()); err == nil {
		t.Fatal("disallowed page was fetched")
	}
	if pageGets.Load() != 0 {
		t.Fatalf("page GET count = %d, want 0", pageGets.Load())
	}
}

func TestRivalComparisonsCoverEveryQuestionCapAndCache(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool := testPool(ctx, t)
	organizationID, projectID := seedProject(ctx, t, pool)
	var pageGets atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			_, _ = w.Write([]byte("User-agent: *\nAllow: /\n"))
		case "/proof", "/brand", "/second", "/too-many", "/extra":
			if r.Method == http.MethodHead {
				w.Header().Set("Content-Type", "text/html")
				return
			}
			if r.Method == http.MethodGet {
				pageGets.Add(1)
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write([]byte(`<html><head><title>Useful rival page</title></head><body><h1>Booking answers</h1><p>Direct answer with specific booking guidance and useful details.</p></body></html>`))
				return
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	target, _ := url.Parse(server.URL)
	client := &http.Client{
		Transport:     rivalTestTransport{serverURL: target, base: server.Client().Transport},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	guard := &audit.Guard{Resolver: publicRivalResolver{}, Client: client}
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	store := Store{DB: pool, Guard: guard, HTTP: client, Now: func() time.Time { return now }}
	firstQuestion := "Which booking tool is best?"
	secondQuestion := "What is a good booking workflow?"
	proof := &Proof{
		Domain: "shop.example",
		Misses: []Miss{
			{Prompt: firstQuestion, Engine: "perplexity", CitedPages: []CitedPage{{URL: "https://proof.example/proof", Domain: "proof.example"}}},
			{Prompt: secondQuestion, Engine: "chatgpt", CitedPages: []CitedPage{{URL: "https://second.example/second", Domain: "second.example"}}},
		},
	}
	// More missed questions, each with one page: with the two above they fill
	// the first round but one, so the second round adds the first question's
	// Brand Lookup page and the cap leaves the second question's out.
	for i := range rivalPageLimit - 3 {
		host := fmt.Sprintf("extra%d.example", i)
		proof.Misses = append(proof.Misses, Miss{Prompt: fmt.Sprintf("extra %d", i), Engine: "gemini", CitedPages: []CitedPage{{URL: "https://" + host + "/extra", Domain: host}}})
	}
	mentionedDomain := "mentioned.example"
	mentions := &MentionsSnapshot{Result: aisearch.BrandLookupResult{TopPages: []aisearch.TopPage{
		{URL: "https://proof.example/proof", Platform: "google", Keywords: []aisearch.PromptExample{{Question: firstQuestion}}},
		{URL: "https://mentioned.example/brand", Domain: &mentionedDomain, Platform: "google", Keywords: []aisearch.PromptExample{{Question: "  which booking   tool is best? "}}},
		{URL: "https://www.shop.example/own", Platform: "google", Keywords: []aisearch.PromptExample{{Question: firstQuestion}}},
		{URL: "https://mentioned.example/too-many", Platform: "google", Keywords: []aisearch.PromptExample{{Question: secondQuestion}}},
		{URL: "https://unmatched.example/unused", Platform: "google", Keywords: []aisearch.PromptExample{{Question: "Different question"}}},
	}}}
	comparisons, err := store.rivalComparisons(ctx, organizationID, projectID, proof, mentions, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(comparisons) != rivalPageLimit || pageGets.Load() != rivalPageLimit {
		t.Fatalf("comparisons=%d page GETs=%d; want exactly rivalPageLimit unique public pages", len(comparisons), pageGets.Load())
	}
	seen := map[string]RivalComparison{}
	for _, comparison := range comparisons {
		seen[comparison.Rival.URL] = comparison
	}
	if _, ok := seen["https://mentioned.example/brand"]; !ok {
		t.Fatalf("question-matched Brand Lookup page was not compared: %+v", comparisons)
	}
	if seen["https://mentioned.example/brand"].Engine != "AI Visibility Brand Lookup · google" {
		t.Fatalf("Brand Lookup source label = %q", seen["https://mentioned.example/brand"].Engine)
	}
	if _, ok := seen["https://www.shop.example/own"]; ok {
		t.Fatal("the customer's own top page was compared as a rival")
	}
	if _, ok := seen["https://mentioned.example/too-many"]; ok {
		t.Fatal("the comparison exceeded the report cap")
	}
	if _, ok := seen["https://unmatched.example/unused"]; ok {
		t.Fatal("an unrelated Brand Lookup question was compared")
	}

	if _, err := store.rivalComparisons(ctx, organizationID, projectID, proof, mentions, nil); err != nil {
		t.Fatal(err)
	}
	if pageGets.Load() != rivalPageLimit {
		t.Fatalf("fresh cache read made %d page GETs, want 0", pageGets.Load()-rivalPageLimit)
	}
	now = now.Add(rivalRefreshInterval)
	if _, err := store.rivalComparisons(ctx, organizationID, projectID, proof, mentions, nil); err != nil {
		t.Fatal(err)
	}
	if pageGets.Load() != rivalPageLimit*2 {
		t.Fatalf("seven-day refresh page GETs=%d, want exactly rivalPageLimit new pages", pageGets.Load())
	}

	otherOrganizationID, otherProjectID := seedProject(ctx, t, pool)
	if _, err := store.rivalComparisons(ctx, otherOrganizationID, otherProjectID, proof, mentions, nil); err != nil {
		t.Fatal(err)
	}
	if pageGets.Load() != rivalPageLimit*3 {
		t.Fatalf("another workspace reused the first workspace cache: page GETs=%d", pageGets.Load())
	}
	for _, miss := range proof.Misses {
		if !slices.ContainsFunc(comparisons, func(c RivalComparison) bool { return c.Prompt == miss.Prompt }) {
			t.Errorf("missed question %q got no comparison; every question comes before a second page", miss.Prompt)
		}
	}
	if !strings.Contains(seen["https://proof.example/proof"].Engine, "perplexity") {
		t.Fatalf("saved proof source was lost: %+v", seen["https://proof.example/proof"])
	}
}

func TestRivalComparisonsRememberFailuresAndNeverWaitPastTheBudget(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pool := testPool(ctx, t)
	organizationID, projectID := seedProject(ctx, t, pool)
	var brokenGets, slowGets atomic.Int32
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			_, _ = w.Write([]byte("User-agent: *\nAllow: /\n"))
		case "/broken":
			if r.Method == http.MethodGet {
				brokenGets.Add(1)
			}
			w.WriteHeader(http.StatusInternalServerError)
		case "/slow":
			if r.Method == http.MethodGet {
				slowGets.Add(1)
				select {
				case <-release:
				case <-r.Context().Done():
				}
				return
			}
			w.Header().Set("Content-Type", "text/html")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	defer close(release)
	target, _ := url.Parse(server.URL)
	client := &http.Client{
		Transport:     rivalTestTransport{serverURL: target, base: server.Client().Transport},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	guard := &audit.Guard{Resolver: publicRivalResolver{}, Client: client}
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	store := Store{DB: pool, Guard: guard, HTTP: client, Now: func() time.Time { return now }}
	proof := &Proof{Domain: "shop.example", Misses: []Miss{
		{Prompt: "Best booking tool?", Engine: "perplexity", CitedPages: []CitedPage{
			{URL: "https://broken.example/broken", Domain: "broken.example"},
			{URL: "https://slow.example/slow", Domain: "slow.example"},
		}},
	}}

	started := time.Now()
	comparisons, err := store.rivalComparisons(ctx, organizationID, projectID, proof, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed > rivalFetchBudget+2*time.Second {
		t.Fatalf("report waited %v on rival sites; budget is %v", elapsed, rivalFetchBudget)
	}
	if len(comparisons) != 0 || brokenGets.Load() != 1 || slowGets.Load() != 1 {
		t.Fatalf("comparisons=%d broken GETs=%d slow GETs=%d; want none shown and one GET each", len(comparisons), brokenGets.Load(), slowGets.Load())
	}

	// The next report, minutes later, must not hit the failed sites again.
	now = now.Add(15 * time.Minute)
	if _, err := store.rivalComparisons(ctx, organizationID, projectID, proof, nil, nil); err != nil {
		t.Fatal(err)
	}
	if brokenGets.Load() != 1 || slowGets.Load() != 1 {
		t.Fatalf("failed rivals were fetched again within %v: broken=%d slow=%d", rivalRetryInterval, brokenGets.Load(), slowGets.Load())
	}
	now = now.Add(rivalRetryInterval)
	if _, err := store.rivalComparisons(ctx, organizationID, projectID, proof, nil, nil); err != nil {
		t.Fatal(err)
	}
	if brokenGets.Load() != 2 {
		t.Fatalf("broken rival GETs=%d after %v; want one retry", brokenGets.Load(), rivalRetryInterval)
	}
}

func TestRivalComparisonsFindListPagesThatSkipTheCustomer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool := testPool(ctx, t)
	organizationID, projectID := seedProject(ctx, t, pool)
	var pageGets atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			_, _ = w.Write([]byte("User-agent: *\nAllow: /\n"))
		case "/top-firms", "/lists-you":
			if r.Method == http.MethodHead {
				w.Header().Set("Content-Type", "text/html")
				return
			}
			pageGets.Add(1)
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			own := ""
			if r.URL.Path == "/lists-you" {
				own = `<a href="https://www.shop.example/">Shop</a>`
			}
			_, _ = w.Write([]byte(`<html><head><title>Top firms</title></head><body><h1>Top firms</h1>
				<a href="https://www.rival-one.example/">One</a> <a href="https://rival-two.example/x">Two</a>
				<a href="https://in.social.example/company/one">Profile</a> <a href="https://uncited.example/">Not cited</a>
				<a href="/about">About</a> ` + own + `</body></html>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	target, _ := url.Parse(server.URL)
	client := &http.Client{
		Transport:     rivalTestTransport{serverURL: target, base: server.Client().Transport},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	guard := &audit.Guard{Resolver: publicRivalResolver{}, Client: client}
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	store := Store{DB: pool, Guard: guard, HTTP: client, Now: func() time.Time { return now }}
	proof := &Proof{
		Domain: "shop.example",
		Rivals: []DomainCount{{Domain: "rival-one.example", Count: 2}, {Domain: "social.example", Count: 1}},
		Misses: []Miss{
			{Prompt: "best firms", Engine: "perplexity", Cited: []string{"directory.example", "rival-two.example"}, CitedPages: []CitedPage{
				{URL: "https://directory.example/top-firms", Domain: "directory.example"},
				{URL: "https://other-list.example/lists-you", Domain: "other-list.example"},
			}},
		},
	}
	comparisons, err := store.rivalComparisons(ctx, organizationID, projectID, proof, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	byURL := map[string]RivalComparison{}
	for _, c := range comparisons {
		byURL[c.Rival.URL] = c
	}
	list := byURL["https://directory.example/top-firms"]
	if want := []string{"in.social.example", "rival-one.example", "rival-two.example"}; !slices.Equal(list.ListsRivals, want) || list.LinksYou {
		t.Fatalf("list page = %v, links you %v; want %v and false", list.ListsRivals, list.LinksYou, want)
	}
	if !byURL["https://other-list.example/lists-you"].LinksYou {
		t.Fatal("a page linking to the customer's site was not marked")
	}

	// The cache keeps the links: a second read fetches nothing and agrees.
	again, err := store.rivalComparisons(ctx, organizationID, projectID, proof, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if pageGets.Load() != 2 || len(again) != 2 || !slices.Equal(again[0].ListsRivals, comparisons[0].ListsRivals) {
		t.Fatalf("cached read: GETs %d, comparisons %+v", pageGets.Load(), again)
	}

	// A page cached before links were read is fetched once more.
	if _, err := pool.Exec(ctx, `UPDATE go_citefix_rival_pages SET signals_json = signals_json - 'linksRead' - 'linkedHosts' WHERE project_id = $1`, projectID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.rivalComparisons(ctx, organizationID, projectID, proof, nil, nil); err != nil {
		t.Fatal(err)
	}
	if pageGets.Load() != 4 {
		t.Fatalf("pages cached without links: GETs %d, want 2 more", pageGets.Load())
	}
}

func TestMatchesHost(t *testing.T) {
	hosts := map[string]bool{"linkedin.com": true, "a.example": true}
	tests := map[string]bool{
		"linkedin.com": true, "in.linkedin.com": true, "a.b.linkedin.com": true,
		"notlinkedin.com": false, "com": false, "": false, "x.a.example": true, "example": false,
	}
	for host, want := range tests {
		if got := matchesHost(host, hosts); got != want {
			t.Errorf("matchesHost(%q) = %v, want %v", host, got, want)
		}
	}
}
