package domain

import (
	"context"
	"sync"
	"testing"
)

// The SAM agent repeats tool calls across replies and in parallel. Each MCP
// domain tool call used to buy a fresh Labs task even when the same
// workspace had just asked the same question.
func TestMCPDomainToolsReuseOnePurchasePerWorkspace(t *testing.T) {
	t.Parallel()
	ranked := func(string) any {
		return map[string]any{"total_count": 1, "items": []any{map[string]any{
			"keyword_data":        map[string]any{"keyword": "hotel crm", "keyword_info": map[string]any{"search_volume": 90.0}},
			"ranked_serp_element": map[string]any{"serp_item": map[string]any{"type": "featured_snippet", "rank_absolute": 2.0, "url": "https://example.com/a"}},
		}}}
	}
	competitors := func(string) any {
		return map[string]any{"items": []any{map[string]any{"domain": "rival.com", "etv": 10.0}}}
	}
	for _, tc := range []struct {
		name   string
		result func(string) any
		call   func(svc *Service, org string) error
	}{
		{"ranked keywords", ranked, func(svc *Service, org string) error {
			_, err := svc.RankedKeywordsForMCP(context.Background(), org, "proj", MCPRankedKeywordsInput{Target: "Example.com", Scope: ScopeDomain, LocationCode: 2356, LanguageCode: "en"})
			return err
		}},
		{"serp competitors", competitors, func(svc *Service, org string) error {
			_, err := svc.SerpCompetitorsForMCP(context.Background(), org, MCPSerpCompetitorInput{Keywords: []string{"Hotel CRM ", "pms"}, LocationCode: 2356, LanguageCode: "en"})
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			provider := &fakeProvider{result: tc.result}
			svc := newTestService(t, provider, &memoryCache{})
			var wg sync.WaitGroup
			for range 4 {
				wg.Go(func() {
					if err := tc.call(svc, "org-a"); err != nil {
						t.Error(err)
					}
				})
			}
			wg.Wait()
			if err := tc.call(svc, "org-a"); err != nil {
				t.Fatal(err)
			}
			if got := provider.calls.Load(); got != 1 {
				t.Fatalf("provider calls = %d, want 1 for five identical calls", got)
			}
			if err := tc.call(svc, "org-b"); err != nil {
				t.Fatal(err)
			}
			if got := provider.calls.Load(); got != 2 {
				t.Fatalf("provider calls = %d, want 2: another workspace must not read org-a's cache", got)
			}
		})
	}
}

func TestMCPRankedKeywordsKeepsTheResultType(t *testing.T) {
	t.Parallel()
	provider := &fakeProvider{result: func(string) any {
		return map[string]any{"total_count": 1, "items": []any{map[string]any{
			"keyword_data":        map[string]any{"keyword": "hotel crm"},
			"ranked_serp_element": map[string]any{"serp_item": map[string]any{"type": "local_pack", "rank_absolute": 3.0}},
		}}}
	}}
	svc := newTestService(t, provider, &memoryCache{})
	got, err := svc.RankedKeywordsForMCP(context.Background(), "org-a", "proj", MCPRankedKeywordsInput{Target: "example.com", Scope: ScopeDomain, ResultTypes: []string{"organic", "local_pack"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 1 || got.Items[0].Type == nil || *got.Items[0].Type != "local_pack" {
		t.Fatalf("items = %+v, want the paid row's type kept", got.Items)
	}
}
