package gsc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/google"
)

func TestGrowthRowsPagingHasFixedBoundAndRetainsFinalWebScope(t *testing.T) {
	client := &performanceClientStub{rows: make([]SearchRow, 1000)}
	service := &Service{Connections: performanceConnectionStub{connection: Connection{SiteURL: "https://example.test/"}}, NewClient: func(string, string) SearchClient { return client }}
	if _, err := service.GetGrowthRows(t.Context(), "org", "project", "2026-09-01", "2026-09-28", false, 100000); err == nil {
		t.Fatal("unbounded coverage accepted")
	}
	if len(client.requestSnapshot()) != 0 {
		t.Fatal("invalid coverage contacted provider")
	}
	rows, err := service.GetGrowthRows(t.Context(), "org", "project", "2026-09-01", "2026-09-28", true, 5000)
	if err != nil {
		t.Fatal(err)
	}
	requests := client.requestSnapshot()
	if len(rows) != 5000 || len(requests) != 5 {
		t.Fatalf("rows=%d calls=%d", len(rows), len(requests))
	}
	for i, request := range requests {
		if request.StartRow != i*1000 || request.RowLimit != 1000 || request.DataState != "final" || request.Type != "web" || !slices.Equal(request.Dimensions, []string{"query", "page"}) {
			t.Fatalf("unscoped pagination: %+v", request)
		}
	}
}

func TestSubmittedSitemapsAndExactQueryReadsUseRealClientValidation(t *testing.T) {
	calls := 0
	invalidCount := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method == http.MethodGet {
			errorsCount := "2"
			if invalidCount {
				errorsCount = "-1"
			}
			if err := json.NewEncoder(w).Encode(map[string]any{"sitemap": []any{map[string]any{"path": "https://example.test/sitemap.xml", "isSitemapsIndex": true, "lastDownloaded": "2026-10-01T00:00:00Z", "errors": errorsCount, "warnings": "1", "contents": []any{map[string]any{"type": "web", "submitted": "42"}}}}}); err != nil {
				t.Error(err)
			}
			return
		}
		var req SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if len(req.Dimensions) != 1 || req.Dimensions[0] != "query" || len(req.DimensionFilterGroups) != 1 || req.DimensionFilterGroups[0].Filters[0].Expression != "https://example.test/page" {
			t.Errorf("wrong page query scope: %+v", req)
		}
		if err := json.NewEncoder(w).Encode(map[string]any{"rows": []any{map[string]any{"keys": []string{"booking"}, "clicks": 10, "impressions": 100, "ctr": .1, "position": 5}}}); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	client := &Client{API: &google.APIClient{Tokens: staticToken{}, Client: server.Client()}, BaseURL: server.URL, UserID: "user"}
	service := &Service{Connections: performanceConnectionStub{connection: Connection{SiteURL: "https://example.test/"}}, NewClient: func(string, string) SearchClient { return client }}
	result, err := service.GetSitemaps(t.Context(), "org", "project")
	if err != nil {
		t.Fatal(err)
	}
	rows := result["sitemaps"].([]SitemapStatus)
	if calls != 1 || len(rows) != 1 || rows[0].Errors != 2 || rows[0].Submitted != 42 || !rows[0].IsIndex || len(rows[0].Contents) != 1 || rows[0].Contents[0].Type != "web" {
		t.Fatalf("sitemaps=%v calls=%d", result, calls)
	}
	invalidCount = true
	if _, err := service.GetSitemaps(t.Context(), "org", "project"); err == nil {
		t.Fatal("negative sitemap errors accepted")
	}
	queries, err := service.GetPageQueryRows(t.Context(), "org", "project", "2026-09-01", "2026-09-28", "https://example.test/page")
	if err != nil || len(queries) != 1 || queries[0].Clicks != 10 {
		t.Fatalf("queries=%+v err=%v", queries, err)
	}
	if calls != 3 {
		t.Fatalf("unexpected provider calls=%d", calls)
	}
}
