package backlinks

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestGapBuysOnceExcludesTheCustomerAndCachesBothOrders(t *testing.T) {
	t.Parallel()
	var calls atomic.Int32
	var mu sync.Mutex
	var sent []map[string]any
	client := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body []map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		sent = append(sent, body...)
		mu.Unlock()
		providerOK(w, r.URL.Path, map[string]any{"items": []any{
			map[string]any{"domain_intersection": map[string]any{
				"1": map[string]any{"target": "lbb.in", "rank": 71.0, "backlinks": 3.0},
				"2": map[string]any{"target": "lbb.in", "rank": 71.0, "backlinks": 1.0},
			}},
			map[string]any{"domain_intersection": map[string]any{
				"1": nil,
				"2": map[string]any{"target": "blog.example", "rank": 34.0, "backlinks": 2.0},
			}},
			map[string]any{"domain_intersection": map[string]any{
				"1": map[string]any{"target": "justdial.com", "rank": 82.5, "backlinks": 9.0},
			}},
		}})
	}), &usageRecorder{})
	cache := &memoryCache{values: map[string][]byte{}}
	svc := newService(client, cache, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Now)
	ctx := t.Context()

	for _, bad := range []gapRequest{
		{Own: "", Competitors: []string{"a.com"}},
		{Own: "own.in"},
		{Own: "own.in", Competitors: []string{"a.com", "b.com", "c.com", "d.com"}},
		{Own: "own.in", Competitors: []string{"own.in"}},
		{Own: "own.in", Competitors: []string{"a.com", "A.com"}},
	} {
		if _, err := svc.Gap(ctx, "org-1", bad); err == nil {
			t.Fatalf("Gap(%+v) accepted invalid input", bad)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("invalid input reached the provider %d times; failed tasks are billed", calls.Load())
	}

	got, err := svc.Gap(ctx, "org-1", gapRequest{Own: "own.in", Competitors: []string{"brewlab.in", "lakeview.cafe"}})
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("one gap lookup made %d provider calls, want 1", calls.Load())
	}
	if len(sent) != 1 || !strings.Contains(stringify(sent[0]["exclude_targets"]), "own.in") || sent[0]["intersection_mode"] != "partial" || sent[0]["limit"] != float64(GapRows) {
		t.Fatalf("request = %+v; want own domain excluded, partial mode, %d rows", sent, GapRows)
	}
	if len(got.Rows) != 3 || got.Rows[0].Domain != "justdial.com" || got.Rows[0].Links[0] != 9 || got.Rows[0].Links[1] != 0 {
		t.Fatalf("rows = %+v; want sorted by authority with a missing side as 0", got.Rows)
	}
	if got.Rows[2].Domain != "blog.example" || got.Rows[2].Links[0] != 0 || got.Rows[2].Links[1] != 2 {
		t.Fatalf("partial row = %+v", got.Rows[2])
	}

	// Same competitors in the other order: served from cache, columns follow
	// the caller's order.
	again, err := svc.Gap(ctx, "org-1", gapRequest{Own: "own.in", Competitors: []string{"lakeview.cafe", "brewlab.in"}})
	if err != nil || calls.Load() != 1 {
		t.Fatalf("repeat lookup: err=%v calls=%d, want a free cache hit", err, calls.Load())
	}
	if again.Rows[0].Domain != "justdial.com" || again.Rows[0].Links[0] != 0 || again.Rows[0].Links[1] != 9 {
		t.Fatalf("reordered row = %+v", again.Rows[0])
	}
	if ttl := cache.onlyTTL(); ttl != gapTTL {
		t.Fatalf("cache ttl = %v, want %v", ttl, gapTTL)
	}
	// Another workspace never reads this workspace's purchase.
	if _, err := svc.Gap(ctx, "org-2", gapRequest{Own: "own.in", Competitors: []string{"brewlab.in", "lakeview.cafe"}}); err != nil || calls.Load() != 2 {
		t.Fatalf("other workspace: err=%v calls=%d, want its own purchase", err, calls.Load())
	}
}

func stringify(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
