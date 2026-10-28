package citefix

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
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
	var pageGets atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			_, _ = w.Write([]byte("User-agent: *\nDisallow: /page\n"))
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

func TestRivalComparisonsUseMatchedBrandLookupPagesCapAtThreeAndCache(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool := testPool(ctx, t)
	organizationID, projectID := seedProject(ctx, t, pool)
	var pageGets atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			_, _ = w.Write([]byte("User-agent: *\nAllow: /\n"))
		case "/proof", "/brand", "/second", "/too-many":
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
		t.Fatalf("comparisons=%d page GETs=%d; want exactly three unique public pages", len(comparisons), pageGets.Load())
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
		t.Fatal("the comparison exceeded the three-page report cap")
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
		t.Fatalf("seven-day refresh page GETs=%d, want exactly three new pages", pageGets.Load())
	}

	otherOrganizationID, otherProjectID := seedProject(ctx, t, pool)
	if _, err := store.rivalComparisons(ctx, otherOrganizationID, otherProjectID, proof, mentions, nil); err != nil {
		t.Fatal(err)
	}
	if pageGets.Load() != rivalPageLimit*3 {
		t.Fatalf("another workspace reused the first workspace cache: page GETs=%d", pageGets.Load())
	}
	if !strings.Contains(seen["https://proof.example/proof"].Engine, "perplexity") {
		t.Fatalf("saved proof source was lost: %+v", seen["https://proof.example/proof"])
	}
}
