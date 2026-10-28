package citefix

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toufiqqureshi/uplumo/backend/internal/aisearch"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend/spendtest"
)

type googleOverviewPlan struct {
	allowed bool
	err     error
}

func (p googleOverviewPlan) HasPaidPlan(context.Context, string) (bool, error) {
	return p.allowed, p.err
}

func seedGoogleOverviewQuestions(ctx context.Context, t *testing.T, pool *pgxpool.Pool, projectID, organizationID string, count int) {
	t.Helper()
	prompts := make([]aisearch.TrackedPrompt, count)
	for i := range prompts {
		prompts[i] = aisearch.TrackedPrompt{Text: fmt.Sprintf("Which booking tool is best, option %d?", i), Topic: "Booking"}
	}
	config, err := json.Marshal(map[string]any{
		"domain": "example.com", "prompts": prompts, "locationCode": 2840, "languageCode": "en",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO go_ai_trackers(project_id,organization_id,configuration) VALUES($1,$2,$3)`, projectID, organizationID, config); err != nil {
		t.Fatalf("save tracked questions: %v", err)
	}
}

func newGoogleOverviewClient(t *testing.T, server *httptest.Server, pool dataforseo.CostRecorder, guard *spendtest.Guard, timeout time.Duration) *dataforseo.Client {
	t.Helper()
	client, err := dataforseo.NewClient(dataforseo.Options{
		BaseURL: server.URL, APIKey: base64.StdEncoding.EncodeToString([]byte("fixture-user:fixture-password")),
		HTTPClient: server.Client(), Recorder: pool, Guard: guard, Timeout: timeout, MaxRetries: 4, RetryBackoff: time.Nanosecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestGoogleAIOverviewDoubleClickConcurrentRequestsAndCacheHitUseOnePaidRequest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool := testPool(ctx, t)
	organizationID, projectID := seedProject(ctx, t, pool)
	seedGoogleOverviewQuestions(ctx, t, pool, projectID, organizationID, 12)
	var posts, gets atomic.Int32
	var postTaskCount atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && r.URL.Path == googleAIOPath {
			posts.Add(1)
			var tasks []struct {
				Keyword             string `json:"keyword"`
				Depth               int    `json:"depth"`
				LoadAsyncAIOverview bool   `json:"load_async_ai_overview"`
			}
			if err := json.NewDecoder(r.Body).Decode(&tasks); err != nil {
				t.Errorf("decode queued request: %v", err)
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			postTaskCount.Store(int64(len(tasks)))
			entries := make([]any, 0, len(tasks))
			for i, task := range tasks {
				if task.Depth != 10 || !task.LoadAsyncAIOverview {
					t.Errorf("queued task = %+v; want depth 10 with async AI Overview", task)
				}
				entries = append(entries, map[string]any{
					"id": fmt.Sprintf("aio-%02d", i), "status_code": 20100, "cost": 0.0012,
					"path": []string{"v3", "serp", "google", "organic", "task_post"}, "data": map[string]string{"keyword": task.Keyword},
				})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 20000, "tasks": entries})
			return
		}
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, googleAIOGet) {
			gets.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 20000, "tasks": []any{map[string]any{
				"status_code": 20000, "result": []any{map[string]any{"items": []any{map[string]any{
					"type": "ai_overview", "markdown": "A direct AI Overview answer.", "references": []any{map[string]any{
						"url": "https://example.com/answer", "domain": "example.com", "title": "Customer answer", "source": "Example Journal", "text": "This passage supports the AI Overview answer.", "publication_date": "2026-10-20",
					}},
				}}}},
			}}})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	guard := &spendtest.Guard{}
	var id atomic.Int32
	service := GoogleAIOverview{
		DB: pool, Client: newGoogleOverviewClient(t, server, dataforseo.NewUsageRecorder(pool), guard, 3*time.Second),
		Plans: googleOverviewPlan{allowed: true}, NewID: func() string { return fmt.Sprintf("aio-%d", id.Add(1)) },
	}
	keys := []string{"double-click-key", "double-click-key", "parallel-key"}
	start := make(chan struct{})
	var wg sync.WaitGroup
	errCh := make(chan error, len(keys))
	for _, key := range keys {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			<-start
			state, err := service.Refresh(ctx, organizationID, projectID, key)
			if err == nil && len(state.Results) != 10 {
				err = fmt.Errorf("returned %d questions, want ten", len(state.Results))
			}
			errCh <- err
		}(key)
	}
	close(start)
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	if posts.Load() != 1 || postTaskCount.Load() != 10 || gets.Load() != 10 {
		t.Fatalf("paid task_post calls=%d queued tasks=%d free task_get calls=%d; want 1, 10, 10", posts.Load(), postTaskCount.Load(), gets.Load())
	}
	if len(guard.Held()) != 1 || math.Abs(guard.Held()[0].EstimateUSD-0.012) > 1e-9 || len(guard.Settled()) != 1 || math.Abs(guard.Settled()[0]-0.012) > 1e-9 {
		t.Fatalf("holds=%+v settlements=%v; want one $0.012 maximum and actual settlement", guard.Held(), guard.Settled())
	}
	var completed int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM go_citefix_ai_overviews WHERE project_id=$1 AND status='completed'`, projectID).Scan(&completed); err != nil || completed != 10 {
		t.Fatalf("completed Google AI Overview rows=%d err=%v; want ten", completed, err)
	}
	var usageRows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM go_dataforseo_usage WHERE organization_id=$1 AND path='/v3/serp/google/organic/task_post'`, organizationID).Scan(&usageRows); err != nil || usageRows != 10 {
		t.Fatalf("recorded queued DataForSEO task rows=%d err=%v; want ten", usageRows, err)
	}
	var citations []GoogleAIOverviewCitation
	var raw []byte
	if err := pool.QueryRow(ctx, `SELECT citations_json FROM go_citefix_ai_overviews WHERE project_id=$1 ORDER BY prompt LIMIT 1`, projectID).Scan(&raw); err != nil || json.Unmarshal(raw, &citations) != nil {
		t.Fatalf("stored citations=%s err=%v", raw, err)
	}
	state, err := service.Refresh(ctx, organizationID, projectID, "fresh-cache-key")
	if err != nil || len(state.Results) != 10 || state.Results[0].Status != "completed" {
		t.Fatalf("cached state=%+v err=%v", state, err)
	}
	if posts.Load() != 1 || gets.Load() != 10 || len(guard.Held()) != 1 {
		t.Fatalf("seven-day cache hit made work: paid POSTs=%d free GETs=%d holds=%d", posts.Load(), gets.Load(), len(guard.Held()))
	}
	if len(citations) != 1 || citations[0].URL != "https://example.com/answer" || citations[0].Title == nil || *citations[0].Title != "Customer answer" || citations[0].Domain == nil || *citations[0].Domain != "example.com" || citations[0].Source == nil || *citations[0].Source != "Example Journal" || citations[0].PublishedAt == nil || *citations[0].PublishedAt != "2026-10-20" || citations[0].Text == nil || *citations[0].Text != "This passage supports the AI Overview answer." {
		t.Fatalf("saved paid citation fields = %+v; URL, domain, title, source, snippet and publication date must be retained", citations)
	}
}

func TestGoogleAIOverviewProviderFailureTimeoutAndGarbageAreNeverRetried(t *testing.T) {
	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"provider 500", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) }},
		{"timeout", func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(1200 * time.Millisecond)
			_, _ = io.WriteString(w, `{}`)
		}},
		{"garbage", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "not json") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			pool := testPool(ctx, t)
			organizationID, projectID := seedProject(ctx, t, pool)
			seedGoogleOverviewQuestions(ctx, t, pool, projectID, organizationID, 1)
			var posts, gets atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost && r.URL.Path == googleAIOPath {
					posts.Add(1)
					tc.handler(w, r)
					return
				}
				if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, googleAIOGet) {
					gets.Add(1)
				}
				http.NotFound(w, r)
			}))
			defer server.Close()
			guard := &spendtest.Guard{}
			service := GoogleAIOverview{
				DB: pool, Client: newGoogleOverviewClient(t, server, dataforseo.NewUsageRecorder(pool), guard, time.Second),
				Plans: googleOverviewPlan{allowed: true},
			}
			_, err := service.Refresh(ctx, organizationID, projectID, "provider-error-key")
			if err == nil || !errors.Is(err, dataforseo.ErrOutcomeUnknown) {
				t.Fatalf("refresh error=%v; want an unknown provider outcome", err)
			}
			if posts.Load() != 1 || gets.Load() != 0 || len(guard.Held()) != 1 || len(guard.Settled()) != 0 {
				t.Fatalf("provider POSTs=%d GETs=%d holds=%d settlements=%v; want one dispatch, no collect/retry and the hold retained", posts.Load(), gets.Load(), len(guard.Held()), guard.Settled())
			}
			var status string
			if queryErr := pool.QueryRow(ctx, `SELECT status FROM go_citefix_ai_overviews WHERE project_id=$1`, projectID).Scan(&status); queryErr != nil || status != "unknown" {
				t.Fatalf("saved ambiguous task status=%q err=%v; want unknown to block automatic retry", status, queryErr)
			}
		})
	}
}

func TestGoogleAIOverviewPreflightGuardsMakeNoProviderCalls(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool := testPool(ctx, t)
	organizationID, projectID := seedProject(ctx, t, pool)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()
	client := newGoogleOverviewClient(t, server, dataforseo.NewUsageRecorder(pool), &spendtest.Guard{}, 2*time.Second)
	service := GoogleAIOverview{DB: pool, Client: client, Plans: googleOverviewPlan{allowed: false}}
	if _, err := service.Refresh(ctx, organizationID, projectID, "unapproved-plan-key"); !errors.Is(err, ErrGoogleAIOPlanRequired) {
		t.Fatalf("plan guard error=%v; want ErrGoogleAIOPlanRequired", err)
	}
	service.Plans = googleOverviewPlan{allowed: true}
	if _, err := service.Refresh(ctx, organizationID, projectID, "missing-questions-key"); !errors.Is(err, ErrGoogleAIONotConfigured) {
		t.Fatalf("question guard error=%v; want ErrGoogleAIONotConfigured", err)
	}
	if _, err := service.Refresh(ctx, organizationID, projectID, "short"); err == nil {
		t.Fatal("invalid idempotency key was accepted")
	}
	if calls.Load() != 0 {
		t.Fatalf("preflight guards sent %d provider requests; want zero", calls.Load())
	}
}

func TestGoogleAIOverviewReturnsTheSavedMarketAndCustomerCitation(t *testing.T) {
	if got := normalizeCitationDomain("www.example.com"); got != "example.com" {
		t.Fatalf("normal domain = %q", got)
	}
	if !citationsContainDomain([]GoogleAIOverviewCitation{{Domain: stringPointer("www.example.com")}}, "example.com") {
		t.Fatal("customer domain was not matched to its saved citation")
	}
}

func stringPointer(value string) *string { return &value }
