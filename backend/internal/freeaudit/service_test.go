package freeaudit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/toufiqqureshi/uplumo/backend/internal/audit"
	"github.com/toufiqqureshi/uplumo/backend/internal/emailauth"
	"github.com/toufiqqureshi/uplumo/backend/internal/site"
)

// fakeChecker counts checks; each takes delay. The snapshot it returns has a
// title of checkerTitle, so the report's escaping can be tested.
type fakeChecker struct {
	checks atomic.Int32
	// probes counts redirect probes, the one request to the site before a check.
	probes    atomic.Int32
	delay     time.Duration
	reachable bool
	err       error
	// finalURL, when set, is the address the report says the page ended at
	// after redirects.
	finalURL string
}

// checkerTitle is the title of the page the fake checker "reads".
const checkerTitle = "<script>alert(1)</script>"

func (f *fakeChecker) ValidateURL(_ context.Context, input string) (string, error) {
	return f.normalize(input)
}

func (f *fakeChecker) NormalizeURL(_ context.Context, input string) (string, error) {
	f.probes.Add(1)
	return f.normalize(input)
}

func (f *fakeChecker) normalize(input string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	normalized := "https://" + strings.TrimPrefix(input, "https://")
	if !strings.HasSuffix(normalized, "/") {
		normalized += "/"
	}
	return normalized, nil
}

func (f *fakeChecker) Check(_ context.Context, pageURL string) audit.PageSnapshot {
	f.checks.Add(1)
	time.Sleep(f.delay)
	snap := audit.PageSnapshot{URL: pageURL, CheckedAt: time.Now().UTC(), StatusCode: 200, Reachable: f.reachable,
		Origin: "https://shop.example"}
	if f.finalURL != "" {
		snap.URL = f.finalURL
	}
	snap.Page.URL = snap.URL
	snap.Page.Title = checkerTitle
	snap.Page.WordCount = 120
	if !f.reachable {
		snap.StatusCode = 0
		snap.Problem = "We could not connect to this website."
	}
	return snap
}

func testRedis(t *testing.T) *redis.Client {
	t.Helper()
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("TEST_REDIS_URL must be set in CI")
		}
		t.Skip("TEST_REDIS_URL not set; skipping Redis integration test")
	}
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatal(err)
	}
	client := redis.NewClient(options)
	t.Cleanup(func() { _ = client.Close() })
	if err := client.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("flush test Redis: %v", err)
	}
	return client
}

func newService(t *testing.T, checker *fakeChecker) *Service {
	t.Helper()
	rdb := testRedis(t)
	return New(checker, rdb, emailauth.RedisLimiter{Client: rdb}, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func TestRunCachesAPageForEveryVisitor(t *testing.T) {
	checker := &fakeChecker{reachable: true}
	svc := newService(t, checker)
	for i := range 3 {
		report, err := svc.Run(context.Background(), "shop.example", fmt.Sprintf("10.0.0.%d", i), audit.KindSEO)
		if err != nil || !report.Reachable || report.Kind != audit.KindSEO {
			t.Fatalf("run %d: report = %+v, err = %v", i, report, err)
		}
	}
	if n := checker.checks.Load(); n != 1 {
		t.Fatalf("checks = %d, want 1 (cached for everyone)", n)
	}
	ttl := svc.redis.TTL(context.Background(), cacheKey("https://shop.example/")).Val()
	if ttl < cacheTTL-time.Minute || ttl > cacheTTL {
		t.Fatalf("cache ttl = %s, want about %s", ttl, cacheTTL)
	}
}

func TestRunKeepsUnreachableResultsBriefly(t *testing.T) {
	svc := newService(t, &fakeChecker{})
	if _, err := svc.Run(context.Background(), "down.example", "10.0.0.1", audit.KindSEO); err != nil {
		t.Fatal(err)
	}
	if ttl := svc.redis.TTL(context.Background(), cacheKey("https://down.example/")).Val(); ttl > failedCacheTTL || ttl <= 0 {
		t.Fatalf("ttl = %s, want at most %s", ttl, failedCacheTTL)
	}
}

func TestRunChecksOncePerPageUnderConcurrentRequests(t *testing.T) {
	checker := &fakeChecker{reachable: true, delay: 100 * time.Millisecond}
	svc := newService(t, checker)
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Go(func() {
			if _, err := svc.Run(context.Background(), "busy.example", fmt.Sprintf("10.0.1.%d", i), audit.KindSEO); err != nil {
				t.Errorf("run: %v", err)
			}
		})
	}
	wg.Wait()
	if n := checker.checks.Load(); n != 1 {
		t.Fatalf("checks = %d, want 1", n)
	}
}

func TestRunLimitsFreshChecksPerVisitor(t *testing.T) {
	checker := &fakeChecker{reachable: true}
	svc := newService(t, checker)
	ctx := context.Background()
	for i := range freshPerDay {
		if _, err := svc.Run(ctx, fmt.Sprintf("site%d.example", i), "10.0.2.1", audit.KindSEO); err != nil {
			t.Fatalf("fresh check %d: %v", i, err)
		}
	}
	if _, err := svc.Run(ctx, "one-more.example", "10.0.2.1", audit.KindSEO); !errors.Is(err, ErrTooMany) {
		t.Fatalf("err = %v, want ErrTooMany", err)
	}
	// A cached page still opens, and another visitor is not limited.
	if _, err := svc.Run(ctx, "site0.example", "10.0.2.1", audit.KindSEO); err != nil {
		t.Fatalf("cached page: %v", err)
	}
	if _, err := svc.Run(ctx, "one-more.example", "10.0.2.2", audit.KindSEO); err != nil {
		t.Fatalf("other visitor: %v", err)
	}
	if n := checker.checks.Load(); n != freshPerDay+1 {
		t.Fatalf("checks = %d, want %d", n, freshPerDay+1)
	}
}

func TestRunRefusesBadAddressesWithoutChecking(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want error
	}{
		{audit.ErrStartURLInvalid, ErrInvalidURL},
		{fmt.Errorf("%w: 10.0.0.1", audit.ErrCrawlTargetBlocked), ErrBlocked},
	} {
		checker := &fakeChecker{err: tc.err}
		svc := newService(t, checker)
		if _, err := svc.Run(context.Background(), "localhost", "10.0.3.1", audit.KindSEO); !errors.Is(err, tc.want) {
			t.Fatalf("err = %v, want %v", err, tc.want)
		}
		if checker.checks.Load() != 0 {
			t.Fatal("a refused address was checked")
		}
	}
}

func TestRunReturnsBusyWhenEverySlotIsTaken(t *testing.T) {
	svc := newService(t, &fakeChecker{reachable: true})
	for range concurrentChecks {
		svc.slots <- struct{}{}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := svc.Run(ctx, "slow.example", "10.0.4.1", audit.KindSEO); !errors.Is(err, ErrBusy) {
		t.Fatalf("err = %v, want ErrBusy", err)
	}
}

func TestReportRouteRendersEscapedHTML(t *testing.T) {
	pages, err := site.New(&url.URL{Scheme: "https", Host: "uplumo.com"})
	if err != nil {
		t.Fatal(err)
	}
	checker := &fakeChecker{reachable: true}
	mux := http.NewServeMux()
	Mount(mux, Deps{Service: newService(t, checker), Pages: pages, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		ClientIP: func(*http.Request) string { return "10.0.5.1" }})

	cases := []struct {
		query  string
		status int
		want   []string
	}{
		{"?url=shop.example", http.StatusOK, []string{"SEO score", "shop.example", "&lt;script&gt;", "Track where you rank on Google"}},
		{"?url=shop.example&type=aeo", http.StatusOK, []string{"AEO score", "shop.example", "Checks how ready this page is to be read"}},
		{"?url=shop.example&type=bogus", http.StatusBadRequest, []string{"Choose the SEO audit or the AEO audit"}},
		{"", http.StatusBadRequest, []string{"Enter a website address"}},
		{"?url=" + strings.Repeat("a", maxInputBytes+1), http.StatusBadRequest, []string{"Enter a website address"}},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, ReportPath+tc.query, nil))
		body := rec.Body.String()
		if rec.Code != tc.status {
			t.Fatalf("%s: status = %d, body = %s", tc.query, rec.Code, body)
		}
		for _, want := range tc.want {
			if !strings.Contains(body, want) {
				t.Errorf("%.40s: body lacks %q", tc.query, want)
			}
		}
		if strings.Contains(body, "<script>alert") {
			t.Fatal("page text was not escaped")
		}
		if rec.Header().Get("Cache-Control") != "private, no-store" {
			t.Errorf("cache-control = %q", rec.Header().Get("Cache-Control"))
		}
	}
	// The SEO and AEO requests of one page are one fresh check.
	if checker.checks.Load() != 1 {
		t.Fatalf("checks = %d, want 1", checker.checks.Load())
	}
}

func TestCachedFindsTheReportUnderItsFinalURL(t *testing.T) {
	// The probe saw example.com; the page redirected on to /final/, so the
	// report's URL is the final address. The Word link uses that address.
	checker := &fakeChecker{reachable: true, finalURL: "https://example.com/final/"}
	svc := newService(t, checker)
	ctx := context.Background()
	report, err := svc.Run(ctx, "example.com", "203.0.113.9", audit.KindSEO)
	if err != nil {
		t.Fatal(err)
	}
	if report.URL != "https://example.com/final/" {
		t.Fatalf("report URL = %q", report.URL)
	}
	found, ok, err := svc.Cached(ctx, report.URL, "203.0.113.9", audit.KindSEO)
	if err != nil || !ok {
		t.Fatalf("Cached(final URL) ok=%v err=%v, want the cached report", ok, err)
	}
	if found.URL != report.URL {
		t.Fatalf("cached URL = %q", found.URL)
	}
	if checker.checks.Load() != 1 {
		t.Fatalf("checks = %d, want 1", checker.checks.Load())
	}
}

func TestOneFreshCheckServesBothEngines(t *testing.T) {
	checker := &fakeChecker{reachable: true}
	svc := newService(t, checker)
	ctx := context.Background()
	seo, err := svc.Run(ctx, "shop.example", "10.0.6.1", audit.KindSEO)
	if err != nil {
		t.Fatal(err)
	}
	aeo, err := svc.Run(ctx, "shop.example", "10.0.6.1", audit.KindAEO)
	if err != nil {
		t.Fatalf("AEO after SEO: %v", err)
	}
	if n := checker.checks.Load(); n != 1 {
		t.Fatalf("checks = %d, want 1: the second engine is served from the cache", n)
	}
	if n := checker.probes.Load(); n != 1 {
		t.Fatalf("site probes = %d, want 1: the switch reads the cache without a request to the site", n)
	}
	if seo.Kind != audit.KindSEO || aeo.Kind != audit.KindAEO {
		t.Fatalf("kinds = %q and %q", seo.Kind, aeo.Kind)
	}
	if seo.Score == 0 && aeo.Score == 0 {
		t.Fatalf("both reports have no score")
	}
	// The toggle reads the cache only.
	found, ok, err := svc.Cached(ctx, "shop.example", "10.0.6.1", audit.KindSEO)
	if err != nil || !ok || found.Kind != audit.KindSEO {
		t.Fatalf("Cached(seo) = %+v ok=%v err=%v", found, ok, err)
	}
	if checker.checks.Load() != 1 || checker.probes.Load() != 1 {
		t.Fatalf("checks = %d, probes = %d after Cached: want 1 and 1", checker.checks.Load(), checker.probes.Load())
	}
}

func TestCachedNeverRunsACheck(t *testing.T) {
	checker := &fakeChecker{reachable: true}
	svc := newService(t, checker)
	_, ok, err := svc.Cached(context.Background(), "never.example", "10.0.6.2", audit.KindAEO)
	if err != nil || ok {
		t.Fatalf("Cached on an unchecked page: ok=%v err=%v", ok, err)
	}
	if checker.checks.Load() != 0 {
		t.Fatalf("Cached ran %d checks, want 0", checker.checks.Load())
	}
}

func TestParseKindDefaultsToSEO(t *testing.T) {
	cases := map[string]struct {
		kind string
		ok   bool
	}{
		"":        {audit.KindSEO, true},
		"seo":     {audit.KindSEO, true},
		"aeo":     {audit.KindAEO, true},
		"AEO":     {"", false},
		"both":    {"", false},
		"seo aeo": {"", false},
	}
	for raw, want := range cases {
		kind, ok := ParseKind(raw)
		if kind != want.kind || ok != want.ok {
			t.Errorf("ParseKind(%q) = %q, %v; want %q, %v", raw, kind, ok, want.kind, want.ok)
		}
	}
}
