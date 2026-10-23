package backlinks

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend/spendtest"
)

type memoryCache struct {
	mu     sync.Mutex
	values map[string][]byte
	ttls   map[string]time.Duration
	fail   bool
}

// onlyTTL is the lifetime of the single cached entry, or 0 when there is not
// exactly one.
func (c *memoryCache) onlyTTL() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.ttls) != 1 {
		return 0
	}
	for _, ttl := range c.ttls {
		return ttl
	}
	return 0
}

func (c *memoryCache) Get(_ context.Context, key string) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fail {
		return nil, errors.New("redis unavailable")
	}
	value, ok := c.values[key]
	if !ok {
		return nil, errors.New("redis: nil")
	}
	return append([]byte(nil), value...), nil
}
func (c *memoryCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fail {
		return errors.New("redis unavailable")
	}
	if c.values == nil {
		c.values = map[string][]byte{}
	}
	if c.ttls == nil {
		c.ttls = map[string]time.Duration{}
	}
	c.values[key], c.ttls[key] = append([]byte(nil), value...), ttl
	return nil
}

type usageRecorder struct {
	mu            sync.Mutex
	organizations []string
}

func (r *usageRecorder) RecordDataForSEO(_ context.Context, org string, _ dataforseo.Cost) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.organizations = append(r.organizations, org)
	return nil
}

func testClient(t *testing.T, handler http.Handler, recorder *usageRecorder) *dataforseo.Client {
	return testClientWithTimeout(t, handler, recorder, 0)
}
func testClientWithTimeout(t *testing.T, handler http.Handler, recorder *usageRecorder, timeout time.Duration) *dataforseo.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := dataforseo.NewClient(dataforseo.Options{Guard: &spendtest.Guard{}, BaseURL: server.URL, APIKey: base64.StdEncoding.EncodeToString([]byte("test-user:test-pass")), Recorder: recorder, MaxRetries: 0, Timeout: timeout})
	if err != nil {
		t.Fatal(err)
	}
	return client
}
func providerOK(w http.ResponseWriter, path string, result any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 20000, "tasks": []any{map[string]any{"status_code": 20000, "status_message": "Ok.", "path": strings.Split(strings.Trim(path, "/"), "/"), "cost": 0.01, "result": []any{result}}}})
}

func TestOverviewCacheIsOrganizationScopedAndRedisFailureIsSoft(t *testing.T) {
	t.Parallel()
	var calls int
	var mu sync.Mutex
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls++
		mu.Unlock()
		if strings.HasSuffix(r.URL.Path, "/summary/live") {
			providerOK(w, r.URL.Path, map[string]any{"rank": 52, "backlinks": 90, "referring_pages": 20, "referring_domains": 10, "info": map[string]any{"target_spam_score": 5}})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/history/live") {
			providerOK(w, r.URL.Path, map[string]any{"items": []any{map[string]any{"date": "2026-10-08T00:00:00Z", "backlinks": 90, "new_reffering_domains": 2}}})
			return
		}
		http.NotFound(w, r)
	})
	recorder := &usageRecorder{}
	cache := &memoryCache{values: map[string][]byte{}}
	svc := newService(testClient(t, server, recorder), cache, slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return time.Date(2026, 10, 9, 11, 12, 13, 0, time.UTC) })
	in := lookupInput{Target: "example.com", Scope: string(ScopeDomain)}
	first, err := svc.Overview(context.Background(), "org-one", in)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Overview(context.Background(), "org-one", in)
	if err != nil {
		t.Fatal(err)
	}
	if first.Summary.Backlinks == nil || *first.Summary.Backlinks != 90 || len(second.Trends) != 1 || second.NewLostTrends[0].NewReferringDomains == nil || *second.NewLostTrends[0].NewReferringDomains != 2 {
		t.Fatalf("unexpected overview mapping: %#v", second)
	}
	if _, err := svc.Overview(context.Background(), "org-two", in); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	gotCalls := calls
	mu.Unlock()
	if gotCalls != 4 {
		t.Fatalf("provider calls = %d, want 4 (cached within org, isolated between orgs)", gotCalls)
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if len(recorder.organizations) != 4 {
		t.Fatalf("usage rows = %d, want 4", len(recorder.organizations))
	}
	for i, org := range recorder.organizations {
		want := "org-one"
		if i >= 2 {
			want = "org-two"
		}
		if org != want {
			t.Errorf("usage organization[%d] = %q, want %q", i, org, want)
		}
	}
}

func TestCacheDownFallsThroughToProvider(t *testing.T) {
	t.Parallel()
	cache := &memoryCache{fail: true}
	recorder := &usageRecorder{}
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providerOK(w, r.URL.Path, map[string]any{"backlinks": 3, "info": map[string]any{}})
	})
	svc := newService(testClient(t, server, recorder), cache, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
	got, err := svc.Overview(context.Background(), "org", lookupInput{Target: "example.com", Scope: "exact_url"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Summary.Backlinks == nil || *got.Summary.Backlinks != 3 {
		t.Fatalf("overview = %#v", got)
	}
}

func TestProviderRejectsBillingIssueWithoutRetry(t *testing.T) {
	t.Parallel()
	var calls int
	server := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 20000, "tasks": []any{map[string]any{"status_code": 40210, "status_message": "Insufficient funds", "cost": 0, "path": []string{"v3", "backlinks", "summary", "live"}}}})
	})
	svc := newService(testClient(t, server, &usageRecorder{}), nil, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
	_, err := svc.Overview(context.Background(), "org", lookupInput{Target: "example.com", Scope: "exact_url"})
	if !errors.Is(err, dataforseo.ErrBillingIssue) {
		t.Fatalf("error = %v, want billing issue", err)
	}
	if calls != 1 {
		t.Fatalf("provider calls = %d, want one non-retried request", calls)
	}
}

func TestProviderErrorIsNotRetried(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	server := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 20000, "tasks": []any{map[string]any{"status_code": 50000, "status_message": "provider unavailable", "cost": 0, "path": []string{"v3", "backlinks", "summary", "live"}}}})
	})
	svc := newService(testClient(t, server, &usageRecorder{}), nil, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
	_, err := svc.Overview(context.Background(), "org", lookupInput{Target: "example.com", Scope: "exact_url"})
	if !errors.Is(err, dataforseo.ErrUpstreamUnavailable) {
		t.Fatalf("error = %v, want upstream unavailable", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("provider calls = %d, want one non-retried paid request", calls.Load())
	}
}

func TestEmptyProviderListReturnsEmptyArray(t *testing.T) {
	t.Parallel()
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providerOK(w, r.URL.Path, map[string]any{"items": nil, "total_count": 0})
	})
	svc := newService(testClient(t, server, &usageRecorder{}), nil, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
	got, err := svc.Rows(context.Background(), "org", lookupInput{Target: "example.com"}, pageInput{Page: 1, PageSize: 50, Sort: "rank,desc", Mode: "as_is"}, rowsFilters{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if got.Rows == nil || len(got.Rows) != 0 || got.TotalCount == nil || *got.TotalCount != 0 || got.HasMore {
		t.Fatalf("empty list = %#v", got)
	}
}

// Everything the paid tasks return that the screen shows must survive the
// mapping: the link's context, page title, placement, status and dates.
func TestPaidFieldsReachTheResponse(t *testing.T) {
	t.Parallel()
	var row providerBacklink
	if err := json.Unmarshal([]byte(`{"text_pre":"Developed by","anchor":"revmerito","text_post":"in 2024","page_from_title":"LeVivanta","semantic_location":"footer","url_to_status_code":403,"domain_from_country":"IN","is_new":true,"is_broken":true,"last_seen":"2026-02-18 14:25:13 +00:00"}`), &row); err != nil {
		t.Fatal(err)
	}
	got := mapBacklink(row)
	if *got.TextBefore != "Developed by" || *got.TextAfter != "in 2024" || *got.SourceTitle != "LeVivanta" || *got.Placement != "footer" || *got.TargetStatus != 403 || *got.SourceCountry != "IN" || !got.IsNew || !got.IsBroken {
		t.Fatalf("row = %+v", got)
	}
	if got.LastSeen == nil || *got.LastSeen != "2026-02-18 14:25:13 +00:00" {
		t.Fatalf("lastSeen = %v, want the provider's last_seen", got.LastSeen)
	}
	var summary providerSummary
	if err := json.Unmarshal([]byte(`{"first_seen":"2023-08-31","referring_ips":2004,"referring_subnets":220,"referring_domains_nofollow":18,"referring_links_tld":{"store":349,"com":89},"referring_links_countries":{"":2154,"IN":22}}`), &summary); err != nil {
		t.Fatal(err)
	}
	profile := mapSummary(summary).Profile
	if *profile.ReferringIPs != 2004 || *profile.ReferringSubnets != 220 || *profile.ReferringDomainsNofollow != 18 || profile.TLDs["store"] != 349 || *profile.FirstSeen != "2023-08-31" {
		t.Fatalf("profile = %+v", profile)
	}
	if _, blank := profile.Countries[""]; blank || profile.Countries["IN"] != 22 || profile.PlatformTypes == nil {
		t.Fatalf("countries = %v, platforms = %v: want the blank key dropped and empty maps, not nil", profile.Countries, profile.PlatformTypes)
	}
}

// A profile that fits in one task is bought once, unfiltered; every view the
// screen offers is then cut from it without another provider call.
func TestSmallProfileBuysTheWholeListOnceForEveryView(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	rowsBodies := []map[string]any{}
	link := func(domain, url string, rank, spam float64, seen string) map[string]any {
		return map[string]any{"domain_from": domain, "url_from": url, "rank": rank, "backlink_spam_score": spam, "first_seen": seen, "dofollow": true}
	}
	items := []any{
		link("a.com", "https://a.com/1", 10, 5, "2026-01-01"),
		link("a.com", "https://a.com/2", 30, 5, "2026-03-01"),
		link("spam.store", "https://spam.store/x", 50, 90, "2026-05-01"),
		link("b.com", "https://b.com/blog", 20, 0, "2026-02-01"),
	}
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/summary/live"):
			providerOK(w, r.URL.Path, map[string]any{"backlinks": 4, "referring_domains": 3})
		case strings.HasSuffix(r.URL.Path, "/history/live"):
			providerOK(w, r.URL.Path, map[string]any{"items": []any{}})
		default:
			var tasks []map[string]any
			_ = json.NewDecoder(r.Body).Decode(&tasks)
			mu.Lock()
			rowsBodies = append(rowsBodies, tasks[0])
			mu.Unlock()
			providerOK(w, r.URL.Path, map[string]any{"items": items, "total_count": len(items)})
		}
	})
	svc := newService(testClient(t, server, &usageRecorder{}), &memoryCache{}, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
	in := lookupInput{Target: "example.com", Scope: string(ScopeDomain)}
	if _, err := svc.Overview(context.Background(), "org", in); err != nil {
		t.Fatal(err)
	}
	get := func(mode, sort string, hideSpam bool, f rowsFilters, page int) []string {
		t.Helper()
		got, err := svc.Rows(context.Background(), "org", in, pageInput{Page: page, PageSize: 50, Sort: sort, Mode: mode}, f, hideSpam)
		if err != nil {
			t.Fatal(err)
		}
		urls := []string{}
		for _, row := range got.Rows {
			urls = append(urls, *row.URLFrom)
		}
		return urls
	}
	// The screen's default: one link per domain, spam hidden, newest first.
	if got := strings.Join(get("one_per_domain", "first_seen,desc", true, rowsFilters{}, 1), " "); got != "https://a.com/2 https://b.com/blog" {
		t.Fatalf("default view = %s", got)
	}
	if got := strings.Join(get("as_is", "rank,asc", false, rowsFilters{}, 1), " "); got != "https://a.com/1 https://b.com/blog https://a.com/2 https://spam.store/x" {
		t.Fatalf("all links by rank = %s", got)
	}
	if got := strings.Join(get("as_is", "rank,desc", false, rowsFilters{DomainFrom: "a.com"}, 1), " "); got != "https://a.com/2 https://a.com/1" {
		t.Fatalf("one domain's links = %s", got)
	}
	if got := strings.Join(get("as_is", "rank,desc", false, rowsFilters{Include: "blog"}, 1), " "); got != "https://b.com/blog" {
		t.Fatalf("include filter = %s", got)
	}
	if got := get("as_is", "rank,desc", false, rowsFilters{}, 2); len(got) != 0 {
		t.Fatalf("page 2 = %v, want empty", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(rowsBodies) != 1 {
		t.Fatalf("rows provider calls = %d, want 1 for every view", len(rowsBodies))
	}
	if body := rowsBodies[0]; body["limit"] != float64(blockSize) || body["filters"] != nil || body["mode"] != "as_is" {
		t.Fatalf("whole-list request = %v, want %d unfiltered as_is rows", body, blockSize)
	}
}

// A profile longer than one task must ask the provider for the exact view.
func TestLargeProfileAsksTheProviderForTheView(t *testing.T) {
	t.Parallel()
	var bodies []map[string]any
	var mu sync.Mutex
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/summary/live"):
			providerOK(w, r.URL.Path, map[string]any{"backlinks": 5000})
		case strings.HasSuffix(r.URL.Path, "/history/live"):
			providerOK(w, r.URL.Path, map[string]any{"items": []any{}})
		default:
			var tasks []map[string]any
			_ = json.NewDecoder(r.Body).Decode(&tasks)
			mu.Lock()
			bodies = append(bodies, tasks[0])
			mu.Unlock()
			providerOK(w, r.URL.Path, map[string]any{"items": []any{}, "total_count": 5000})
		}
	})
	svc := newService(testClient(t, server, &usageRecorder{}), &memoryCache{}, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
	in := lookupInput{Target: "example.com", Scope: string(ScopeDomain)}
	if _, err := svc.Overview(context.Background(), "org", in); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Rows(context.Background(), "org", in, pageInput{Page: 1, PageSize: 50, Sort: "first_seen,desc", Mode: "one_per_domain"}, rowsFilters{}, true); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(bodies) != 1 || bodies[0]["mode"] != "one_per_domain" || bodies[0]["filters"] == nil {
		t.Fatalf("provider requests = %v, want one filtered one_per_domain request", bodies)
	}
}

// Changing the page size or moving to the next page inside the bought block
// must not buy the same rows again; the next block is one more call.
func TestRowsBuyOneBlockForEveryPageSizeInsideIt(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var bodies []map[string]any
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var tasks []map[string]any
		_ = json.NewDecoder(r.Body).Decode(&tasks)
		mu.Lock()
		bodies = append(bodies, tasks[0])
		mu.Unlock()
		offset := int(tasks[0]["offset"].(float64))
		items := []any{}
		for i := range min(int(tasks[0]["limit"].(float64)), 2077-offset) {
			items = append(items, map[string]any{"url_from": fmt.Sprintf("https://s%d.example/", offset+i)})
		}
		providerOK(w, r.URL.Path, map[string]any{"items": items, "total_count": 2077})
	})
	svc := newService(testClient(t, server, &usageRecorder{}), &memoryCache{}, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
	get := func(page, size int) Page[BacklinkRow] {
		t.Helper()
		got, err := svc.Rows(context.Background(), "org", lookupInput{Target: "example.com"}, pageInput{Page: page, PageSize: size, Sort: "rank,desc", Mode: "as_is"}, rowsFilters{}, false)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	first := get(1, 50)
	get(1, 100)
	get(1, 200)
	second := get(2, 50)
	if len(first.Rows) != 50 || *first.Rows[0].URLFrom != "https://s0.example/" || !first.HasMore {
		t.Fatalf("page 1 of 50 = %d rows, first %v, hasMore %v", len(first.Rows), *first.Rows[0].URLFrom, first.HasMore)
	}
	if *second.Rows[0].URLFrom != "https://s50.example/" || second.Page != 2 || second.PageSize != 50 {
		t.Fatalf("page 2 starts at %v (page %d size %d), want s50", *second.Rows[0].URLFrom, second.Page, second.PageSize)
	}
	mu.Lock()
	if len(bodies) != 1 || bodies[0]["limit"] != float64(blockSize) || bodies[0]["offset"] != float64(0) {
		t.Fatalf("provider requests = %d, want one block of %d at offset 0", len(bodies), blockSize)
	}
	mu.Unlock()
	get(5, 200)
	last := get(11, 200)
	if len(last.Rows) != 77 || last.HasMore || *last.Rows[0].URLFrom != "https://s2000.example/" {
		t.Fatalf("last page = %d rows, hasMore %v", len(last.Rows), last.HasMore)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(bodies) != 2 || bodies[1]["offset"] != float64(2000) {
		t.Fatalf("provider requests = %d, want one more block at offset 2000", len(bodies))
	}
}

func TestProviderTimeoutIsReturnedToCaller(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
		providerOK(w, r.URL.Path, map[string]any{})
	})
	svc := newService(testClientWithTimeout(t, server, &usageRecorder{}, time.Second), nil, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
	_, err := svc.Overview(context.Background(), "org", lookupInput{Target: "example.com", Scope: "exact_url"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want deadline exceeded", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("provider calls = %d, want 1", calls.Load())
	}
}

func TestMapBacklinkLegacyFieldFallbacks(t *testing.T) {
	t.Parallel()
	domain := "source.example"
	url := "https://source.example/a"
	lostDate := "2026-01-01"
	score := float64(12)
	row := mapBacklink(providerBacklink{DomainFrom: &domain, URLFrom: &url, LostDate: &lostDate, SpamScoreLegacy: &score, Attributes: []string{"nofollow"}})
	if row.SpamScore == nil || *row.SpamScore != score || row.LastSeen == nil || *row.LastSeen != lostDate || !row.IsLost || len(row.RelAttributes) != 1 || row.RelAttributes[0] != "nofollow" {
		t.Fatalf("mapped row = %#v", row)
	}
}

func TestEmptyLostDateDoesNotMarkLinkLost(t *testing.T) {
	t.Parallel()
	empty := ""
	if got := mapBacklink(providerBacklink{LostDate: &empty}); got.IsLost {
		t.Fatalf("empty lost_date marked row lost: %#v", got)
	}
}

func TestSubfolderOverviewCountsFilteredRowsSequentially(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var modes []string
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request []map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode provider request: %v", err)
			return
		}
		mode, _ := request[0]["mode"].(string)
		mu.Lock()
		modes = append(modes, mode)
		mu.Unlock()
		total := 120
		if mode == "one_per_domain" {
			total = 17
		}
		providerOK(w, r.URL.Path, map[string]any{"items": []any{}, "total_count": total})
	})
	recorder := &usageRecorder{}
	svc := newService(testClient(t, server, recorder), nil, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
	got, err := svc.Overview(context.Background(), "org", lookupInput{Target: "example.com/blog", Scope: "subfolder"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Summary.Backlinks == nil || *got.Summary.Backlinks != 120 || got.Summary.ReferringDomains == nil || *got.Summary.ReferringDomains != 17 || got.Summary.Rank != nil || len(got.Trends) != 0 {
		t.Fatalf("subfolder overview = %#v", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if strings.Join(modes, ",") != "as_is,one_per_domain" {
		t.Fatalf("modes = %v", modes)
	}
}

func TestCancelledRequestStopsProviderCall(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		providerOK(w, r.URL.Path, map[string]any{})
	})
	svc := newService(testClient(t, server, &usageRecorder{}), nil, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := svc.Overview(ctx, "org", lookupInput{Target: "example.com", Scope: "exact_url"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context canceled", err)
	}
	if calls.Load() != 0 {
		t.Fatalf("provider server was reached %d times", calls.Load())
	}
}

func TestConcurrentOrganizationRequestsAreRaceSafe(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		providerOK(w, r.URL.Path, map[string]any{"items": []any{}, "total_count": 0})
	})
	svc := newService(testClient(t, server, &usageRecorder{}), &memoryCache{values: map[string][]byte{}}, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
	const workers = 8
	var wg sync.WaitGroup
	for i := range workers {
		org := fmt.Sprintf("org-%d", i)
		wg.Go(func() {
			_, err := svc.Rows(context.Background(), org, lookupInput{Target: "example.com"}, pageInput{Page: 1, PageSize: 50, Sort: "rank,desc", Mode: "as_is"}, rowsFilters{}, false)
			if err != nil {
				t.Errorf("Rows() for %s: %v", org, err)
			}
		})
	}
	wg.Wait()
	if got := calls.Load(); got != workers {
		t.Fatalf("provider calls = %d, want %d distinct tenant requests", got, workers)
	}
}

func TestConcurrentEquivalentOverviewRequestsSharePaidProviderWork(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		time.Sleep(40 * time.Millisecond)
		if strings.HasSuffix(r.URL.Path, "/summary/live") {
			providerOK(w, r.URL.Path, map[string]any{"backlinks": 5, "info": map[string]any{}})
			return
		}
		providerOK(w, r.URL.Path, map[string]any{"items": []any{}})
	})
	svc := newService(testClient(t, server, &usageRecorder{}), &memoryCache{values: map[string][]byte{}}, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			<-start
			if _, err := svc.Overview(context.Background(), "same-org", lookupInput{Target: "example.com", Scope: "domain"}); err != nil {
				t.Error(err)
			}
		})
	}
	close(start)
	wg.Wait()
	if got := calls.Load(); got != 2 {
		t.Fatalf("provider calls = %d, want the summary and history once each", got)
	}
}

func TestSharedOverviewSurvivesTheFirstCallerLeaving(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		entered <- struct{}{}
		<-release
		if strings.HasSuffix(r.URL.Path, "/summary/live") {
			providerOK(w, r.URL.Path, map[string]any{"backlinks": 5, "info": map[string]any{}})
			return
		}
		providerOK(w, r.URL.Path, map[string]any{"items": []any{}})
	})
	svc := newService(testClient(t, server, &usageRecorder{}), &memoryCache{values: map[string][]byte{}}, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
	in := lookupInput{Target: "example.com", Scope: "domain"}

	firstCtx, leave := context.WithCancel(t.Context())
	firstErr := make(chan error, 1)
	go func() {
		_, err := svc.Overview(firstCtx, "same-org", in)
		firstErr <- err
	}()
	<-entered
	second := make(chan error, 1)
	go func() {
		_, err := svc.Overview(t.Context(), "same-org", in)
		second <- err
	}()
	time.Sleep(20 * time.Millisecond) // let the second caller join the flight
	leave()
	if err := <-firstErr; !errors.Is(err, context.Canceled) {
		t.Fatalf("first caller error = %v, want context.Canceled", err)
	}
	close(release)
	if err := <-second; err != nil {
		t.Fatalf("second caller error = %v, want the shared result", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("provider calls = %d, want the summary and history once each", got)
	}
}

// refuseAfter is a spend guard that allows a few holds and then refuses.
type refuseAfter struct {
	left atomic.Int32
	err  error
}

func (g *refuseAfter) Hold(context.Context, spend.Request) (spend.Hold, error) {
	if g.left.Add(-1) < 0 {
		return nil, g.err
	}
	return allowedHold{}, nil
}

type allowedHold struct{}

func (allowedHold) Settle(context.Context, float64) error { return nil }

func newOverviewService(t *testing.T, summaryOK, historyStatus int, guard spend.Guard) (*Service, *memoryCache, *atomic.Int32) {
	t.Helper()
	var providerCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providerCalls.Add(1)
		if strings.HasSuffix(r.URL.Path, "/summary/live") {
			providerOK(w, r.URL.Path, map[string]any{"rank": 52, "backlinks": summaryOK, "referring_pages": 20, "referring_domains": 10})
			return
		}
		if historyStatus != http.StatusOK {
			http.Error(w, "provider down", historyStatus)
			return
		}
		providerOK(w, r.URL.Path, map[string]any{"items": []any{map[string]any{"date": "2026-10-08T00:00:00Z", "backlinks": 90}}})
	}))
	t.Cleanup(server.Close)
	client, err := dataforseo.NewClient(dataforseo.Options{Guard: guard, BaseURL: server.URL, APIKey: base64.StdEncoding.EncodeToString([]byte("test-user:test-pass")), Recorder: &usageRecorder{}})
	if err != nil {
		t.Fatal(err)
	}
	cache := &memoryCache{values: map[string][]byte{}}
	return newService(client, cache, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now), cache, &providerCalls
}

// The summary is billed before the history is asked for. A history that fails
// afterwards must not throw the summary away: the retry would buy it again.
func TestOverviewKeepsTheBilledSummaryWhenOnlyTheHistoryFails(t *testing.T) {
	t.Parallel()
	svc, cache, providerCalls := newOverviewService(t, 90, http.StatusInternalServerError, &spendtest.Guard{})
	in := lookupInput{Target: "example.com", Scope: string(ScopeDomain)}
	got, err := svc.Overview(t.Context(), "org-one", in)
	if err != nil {
		t.Fatalf("Overview() failed although the summary was bought: %v", err)
	}
	if got.Summary.Backlinks == nil || *got.Summary.Backlinks != 90 || len(got.Trends) != 0 {
		t.Fatalf("overview = %#v, want the summary and no trend", got)
	}
	if ttl := cache.onlyTTL(); ttl != partialCacheTTL {
		t.Fatalf("a summary-only answer is cached for %v, want %v", ttl, partialCacheTTL)
	}
	before := providerCalls.Load()
	if _, err := svc.Overview(t.Context(), "org-one", in); err != nil || providerCalls.Load() != before {
		t.Fatalf("a retry bought the report again (calls %d -> %d, err %v)", before, providerCalls.Load(), err)
	}
}

func TestOverviewKeepsTheBilledSummaryWhenCreditsCannotCoverHistory(t *testing.T) {
	t.Parallel()
	guard := &refuseAfter{err: spend.ErrNoCredits}
	guard.left.Store(1) // the summary is allowed, the history is not
	svc, cache, calls := newOverviewService(t, 90, http.StatusOK, guard)
	in := lookupInput{Target: "example.com", Scope: string(ScopeDomain)}
	got, err := svc.Overview(t.Context(), "org-one", in)
	if err != nil || got.Summary.Backlinks == nil || *got.Summary.Backlinks != 90 || len(got.Trends) != 0 {
		t.Fatalf("overview = %+v, error = %v; want the bought summary without history", got, err)
	}
	if cache.onlyTTL() != partialCacheTTL {
		t.Fatal("the bought summary must be briefly reusable")
	}
	if _, err := svc.Overview(t.Context(), "org-one", in); err != nil || calls.Load() != 1 {
		t.Fatalf("revisit bought work again: calls=%d, err=%v", calls.Load(), err)
	}
}

func TestFilterConditionsCountNestedGroups(t *testing.T) {
	t.Parallel()
	filters := prependInclude(buildRowsFilters(rowsFilters{Exclude: "a,b", MinLinkAuthority: new(float64(1))}), "url_from", "x,y")
	if got := countFilters(filters); got != 5 {
		t.Fatalf("filter count = %d, want 5: %#v", got, filters)
	}
}

func providerFailure(w http.ResponseWriter, code int) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 20000, "tasks": []any{map[string]any{"status_code": code, "status_message": "task failed", "cost": 0, "path": []string{"v3", "backlinks", "summary", "live"}}}})
}

// A task-level failure (HTTP 200 carrying status 50000) is treated like a
// failed request: the billed summary is kept, the trend is left out.
func TestOverviewAnswersWithTheSummaryWhenTheHistoryTaskFails(t *testing.T) {
	t.Parallel()
	server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/summary/live") {
			providerOK(w, r.URL.Path, map[string]any{"backlinks": 90})
			return
		}
		providerFailure(w, 50000)
	})
	svc := newService(testClient(t, server, &usageRecorder{}), nil, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
	got, err := svc.Overview(context.Background(), "org", lookupInput{Target: "example.com", Scope: string(ScopeDomain)})
	if err != nil {
		t.Fatalf("Overview() failed although the summary was bought: %v", err)
	}
	if got.Summary.Backlinks == nil || *got.Summary.Backlinks != 90 || len(got.Trends) != 0 {
		t.Fatalf("overview = %#v, want the summary and no trend", got)
	}
}

func TestOverviewStopsAfterBillingIssue(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	server := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		providerFailure(w, 40210)
	})
	svc := newService(testClient(t, server, &usageRecorder{}), nil, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
	_, err := svc.Overview(context.Background(), "org", lookupInput{Target: "example.com", Scope: string(ScopeDomain)})
	if !errors.Is(err, dataforseo.ErrBillingIssue) {
		t.Fatalf("error = %v, want billing issue", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("provider calls = %d, want 1: a billing failure must not send (and pay for) the history task", got)
	}
}

func TestRowsHideSpamControlsProviderFilter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		hideSpam bool
	}{{"hidden", true}, {"shown", false}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var body atomic.Value
			server := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				raw, _ := io.ReadAll(r.Body)
				body.Store(string(raw))
				providerOK(w, r.URL.Path, map[string]any{"items": []any{}, "total_count": 0})
			})
			svc := newService(testClient(t, server, &usageRecorder{}), nil, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
			page := pageInput{Page: 1, PageSize: 50, Sort: "rank,desc", Mode: "as_is"}
			if _, err := svc.Rows(context.Background(), "org", lookupInput{Target: "example.com", Scope: string(ScopeDomain)}, page, rowsFilters{}, tc.hideSpam); err != nil {
				t.Fatal(err)
			}
			got, _ := body.Load().(string)
			if has := strings.Contains(got, "backlink_spam_score"); has != tc.hideSpam {
				t.Fatalf("spam filter present = %v, want %v; request body: %s", has, tc.hideSpam, got)
			}
		})
	}
}
