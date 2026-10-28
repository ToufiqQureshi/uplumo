package audit

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend/spendtest"
)

type rendererCostRecorder struct {
	mu    sync.Mutex
	costs []dataforseo.Cost
}

func (r *rendererCostRecorder) RecordDataForSEO(_ context.Context, _ string, cost dataforseo.Cost) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.costs = append(r.costs, cost)
	return nil
}

func (r *rendererCostRecorder) snapshot() []dataforseo.Cost {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]dataforseo.Cost(nil), r.costs...)
}

func newRendererClient(t *testing.T, server *httptest.Server, recorder dataforseo.CostRecorder, guard *spendtest.Guard, timeout time.Duration) *dataforseo.Client {
	t.Helper()
	client, err := dataforseo.NewClient(dataforseo.Options{
		BaseURL: server.URL, APIKey: base64.StdEncoding.EncodeToString([]byte("fixture-user:fixture-password")),
		HTTPClient: server.Client(), Recorder: recorder, Guard: guard, Timeout: timeout, MaxRetries: 4, RetryBackoff: time.Nanosecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

const renderedProviderHTML = `<!doctype html><html><head><title>Rendered page for answers</title></head><body><h1>Useful answer</h1><p>This rendered HTML contains a direct answer, useful supporting context and enough normal page text for analysis.</p></body></html>`

func TestDataForSEORendererUsesExactPaidCallsAndChargesDeliveredPages(t *testing.T) {
	var instantCalls, rawCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case onPageInstantPagesPath:
			call := instantCalls.Add(1)
			var tasks []renderTaskInput
			if err := json.NewDecoder(r.Body).Decode(&tasks); err != nil || len(tasks) == 0 || len(tasks) > onPageMaxTasksPerDomain {
				t.Errorf("instant_pages request has %d tasks, error=%v; want 1 to 5 same-domain tasks", len(tasks), err)
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			entries := make([]any, 0, len(tasks))
			for i, task := range tasks {
				if !task.EnableJavaScript || !task.StoreRawHTML {
					t.Errorf("instant_pages task = %+v; want JavaScript rendering with raw HTML", task)
				}
				id := fmt.Sprintf("render-%d-%d", call, i)
				entries = append(entries, map[string]any{"id": id, "status_code": 20000, "cost": 0.0015, "path": []string{"v3", "on_page", "instant_pages"},
					"data": map[string]string{"url": task.URL}, "result": []any{map[string]any{"items": []any{map[string]any{"resource_type": "html", "status_code": 200, "url": task.URL}}}}})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 20000, "tasks": entries})
		case onPageRawHTMLPath:
			rawCalls.Add(1)
			var tasks []rawHTMLInput
			if err := json.NewDecoder(r.Body).Decode(&tasks); err != nil || len(tasks) == 0 || len(tasks) > onPageMaxTasksPerDomain {
				t.Errorf("raw_html request = %+v, error=%v; want the 1 to 5 task ids of one batch", tasks, err)
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}
			results := make([]any, 0, len(tasks))
			for _, task := range tasks {
				results = append(results, map[string]any{"id": task.ID, "status_code": 20000, "cost": 0, "path": []string{"v3", "on_page", "raw_html"},
					"result": []any{map[string]any{"items": map[string]string{"html": renderedProviderHTML}}}})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 20000, "tasks": results})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	recorder, guard := &rendererCostRecorder{}, &spendtest.Guard{}
	renderer := DataForSEORenderer{Client: newRendererClient(t, server, recorder, guard, 2*time.Second)}
	urls := make([]string, MaxRenderedPagesPerAudit)
	for i := range urls {
		urls[i] = fmt.Sprintf("https://example.test/page-%d", i)
	}
	pages, err := renderer.Render(t.Context(), "org-render", "audit-render", urls)
	if err != nil {
		t.Fatal(err)
	}
	if instantCalls.Load() != 4 || rawCalls.Load() != 4 || len(pages) != MaxRenderedPagesPerAudit {
		t.Fatalf("paid instant_pages calls=%d free raw_html calls=%d pages=%d; want 4 (5 tasks each), 4, 20", instantCalls.Load(), rawCalls.Load(), len(pages))
	}
	if len(guard.Held()) != 1 || guard.Held()[0].EstimateUSD != 0.03 || len(guard.Settled()) != 1 || guard.Settled()[0] != 0.03 {
		t.Fatalf("holds=%+v settlements=%v; want one $0.03 hold and delivered-cost settlement", guard.Held(), guard.Settled())
	}
	var paidUSD float64
	paidTasks := 0
	for _, cost := range recorder.snapshot() {
		if strings.Contains(strings.Join(cost.Path, "/"), "instant_pages") {
			paidTasks++
			paidUSD += cost.USD
		}
	}
	if paidTasks != MaxRenderedPagesPerAudit || paidUSD < 0.029999 || paidUSD > 0.030001 {
		t.Fatalf("recorded paid tasks=%d cost=$%v; want 20 tasks at $0.03", paidTasks, paidUSD)
	}
}

func TestDataForSEORendererDeduplicatesURLAndRejectsOverTwentyBeforeCalls(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == onPageInstantPagesPath {
			var tasks []renderTaskInput
			_ = json.NewDecoder(r.Body).Decode(&tasks)
			_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 20000, "tasks": []any{map[string]any{
				"id": "one", "status_code": 20000, "cost": 0.0015, "path": []string{"v3", "on_page", "instant_pages"},
				"data": map[string]string{"url": tasks[0].URL}, "result": []any{map[string]any{"items": []any{map[string]any{"resource_type": "html", "status_code": 200}}}},
			}}})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 20000, "tasks": []any{map[string]any{
			"id": "one", "status_code": 20000, "cost": 0, "path": []string{"v3", "on_page", "raw_html"},
			"result": []any{map[string]any{"items": map[string]string{"html": renderedProviderHTML}}},
		}}})
	}))
	defer server.Close()
	guard := &spendtest.Guard{}
	renderer := DataForSEORenderer{Client: newRendererClient(t, server, &rendererCostRecorder{}, guard, 2*time.Second)}
	pages, err := renderer.Render(t.Context(), "org-render", "audit-render", []string{"https://example.test/same", "https://example.test/same"})
	if err != nil || len(pages) != 1 || calls.Load() != 2 {
		t.Fatalf("duplicate URL result pages=%d calls=%d err=%v; want one paid render and one raw retrieval", len(pages), calls.Load(), err)
	}
	tooMany := make([]string, MaxRenderedPagesPerAudit+1)
	for i := range tooMany {
		tooMany[i] = fmt.Sprintf("https://example.test/%d", i)
	}
	if _, err := renderer.Render(t.Context(), "org-render", "audit-too-many", tooMany); err == nil || calls.Load() != 2 {
		t.Fatalf("over-cap call err=%v provider calls=%d; want rejection before any provider call", err, calls.Load())
	}
}

func TestDataForSEORendererFailuresNeverRetryAndKnownFailureCostsZeroCredits(t *testing.T) {
	t.Run("failed provider task", func(t *testing.T) {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"status_code":20000,"tasks":[{"id":"bad","status_code":40501,"cost":0.0015,"path":["v3","on_page","instant_pages"]}]}`)
		}))
		defer server.Close()
		guard := &spendtest.Guard{}
		renderer := DataForSEORenderer{Client: newRendererClient(t, server, &rendererCostRecorder{}, guard, 2*time.Second)}
		pages, err := renderer.Render(t.Context(), "org-render", "audit-failed", []string{"https://example.test/page"})
		if err == nil || len(pages) != 0 || calls.Load() != 1 || len(guard.Settled()) != 1 || guard.Settled()[0] != 0 {
			t.Fatalf("rendered=%d err=%v calls=%d settlements=%v; want one failed task and zero customer credits", len(pages), err, calls.Load(), guard.Settled())
		}
	})

	for _, tc := range []struct {
		name   string
		handle http.HandlerFunc
	}{
		{"provider 500", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) }},
		{"garbage", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "not json") }},
		{"timeout", func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(1100 * time.Millisecond)
			_, _ = io.WriteString(w, `{}`)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				tc.handle(w, r)
			}))
			defer server.Close()
			guard := &spendtest.Guard{}
			renderer := DataForSEORenderer{Client: newRendererClient(t, server, &rendererCostRecorder{}, guard, time.Second)}
			if _, err := renderer.Render(t.Context(), "org-render", "audit-uncertain", []string{"https://example.test/page"}); err == nil {
				t.Fatal("uncertain provider response succeeded")
			}
			if calls.Load() != 1 || len(guard.Held()) != 1 || len(guard.Settled()) != 0 {
				t.Fatalf("provider calls=%d holds=%d settlements=%v; want one dispatch, one held reservation, and no guessed refund", calls.Load(), len(guard.Held()), guard.Settled())
			}
		})
	}
}

func TestDataForSEORendererKeepsPaidPagesWhenALaterBatchFails(t *testing.T) {
	for _, tc := range []struct {
		name          string
		laterStatus   int
		wantSettledUS float64
		wantSettled   bool
	}{
		// DataForSEO answered 400: the outcome is known, batch 1 is charged.
		{"definite rejection", http.StatusBadRequest, 0.0075, true},
		// A 500 leaves the spend unreconciled, but the paid pages are still kept.
		{"uncertain outcome", http.StatusInternalServerError, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) { testLaterBatchFails(t, tc.laterStatus, tc.wantSettled, tc.wantSettledUS) })
	}
}

func testLaterBatchFails(t *testing.T, laterStatus int, wantSettled bool, wantSettledUSD float64) {
	t.Helper()
	var instantCalls, rawCalls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case onPageInstantPagesPath:
			if instantCalls.Add(1) > 1 {
				w.WriteHeader(laterStatus)
				return
			}
			var tasks []renderTaskInput
			_ = json.NewDecoder(r.Body).Decode(&tasks)
			entries := make([]any, 0, len(tasks))
			for i, task := range tasks {
				entries = append(entries, map[string]any{"id": fmt.Sprintf("ok-%d", i), "status_code": 20000, "cost": 0.0015, "path": []string{"v3", "on_page", "instant_pages"},
					"data": map[string]string{"url": task.URL}, "result": []any{map[string]any{"items": []any{map[string]any{"resource_type": "html", "status_code": 200}}}}})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 20000, "tasks": entries})
		case onPageRawHTMLPath:
			rawCalls.Add(1)
			var tasks []rawHTMLInput
			_ = json.NewDecoder(r.Body).Decode(&tasks)
			results := make([]any, 0, len(tasks))
			for _, task := range tasks {
				results = append(results, map[string]any{"id": task.ID, "status_code": 20000, "cost": 0, "path": []string{"v3", "on_page", "raw_html"},
					"result": []any{map[string]any{"items": map[string]string{"html": renderedProviderHTML}}}})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 20000, "tasks": results})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	guard := &spendtest.Guard{}
	renderer := DataForSEORenderer{Client: newRendererClient(t, server, &rendererCostRecorder{}, guard, 2*time.Second)}
	urls := make([]string, 10)
	for i := range urls {
		urls[i] = fmt.Sprintf("https://example.test/page-%d", i)
	}
	pages, err := renderer.Render(t.Context(), "org-render", "audit-partial", urls)
	if err == nil {
		t.Fatal("a failed batch must be reported")
	}
	// Batch 1 (5 pages) was paid for: its pages are kept and only they are charged.
	if len(pages) != 5 || instantCalls.Load() != 2 || rawCalls.Load() != 1 {
		t.Fatalf("pages=%d instant_pages=%d raw_html=%d; want 5 kept pages, 2 paid requests, 1 free retrieval", len(pages), instantCalls.Load(), rawCalls.Load())
	}
	settled := guard.Settled()
	if wantSettled != (len(settled) == 1) || (wantSettled && (settled[0] < wantSettledUSD-1e-9 || settled[0] > wantSettledUSD+1e-9)) {
		t.Fatalf("settlements=%v; want settled=%v at $%v", settled, wantSettled, wantSettledUSD)
	}
}
