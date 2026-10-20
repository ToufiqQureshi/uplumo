package gsc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/google"
)

func TestExactPageComparisonPassesRealClientValidation(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var input SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			t.Error(err)
		}
		if len(input.DimensionFilterGroups) != 1 || input.DimensionFilterGroups[0].GroupType != "and" || input.DimensionFilterGroups[0].Filters[0].Expression != "https://example.test/guide" {
			t.Errorf("wrong exact-page filter: %+v", input)
		}
		_, _ = w.Write([]byte(`{"rows":[{"keys":["https://example.test/guide"],"clicks":1,"impressions":10,"ctr":0.1,"position":9}]}`))
	}))
	defer server.Close()
	client := &Client{API: &google.APIClient{Tokens: staticToken{}, Client: server.Client()}, UserID: "u", BaseURL: server.URL}
	service := &Service{Connections: performanceConnectionStub{connection: Connection{SiteURL: "https://example.test/"}}, NewClient: func(string, string) SearchClient { return client }}
	rows, err := service.GetFilteredPageRows(t.Context(), "org", "project", "2026-09-01", "2026-09-28", "https://example.test/guide")
	if err != nil || len(rows) != 1 || calls != 1 {
		t.Fatalf("exact-page report: rows=%v err=%v calls=%d", rows, err, calls)
	}
}
