package dataforseo

import (
	"math"
	"testing"
)

func TestEstimateUSDIsTheMostARequestCanCost(t *testing.T) {
	for _, tc := range []struct {
		name, path, body string
		want             float64
	}{
		{"free appendix", "/v3/appendix/status", "", 0},
		{"free locations", "/v3/serp/google/locations/us", "", 0},
		{"free ai models", "/v3/ai_optimization/chat_gpt/llm_responses/models", "", 0},
		{"live serp top 10", "/v3/serp/google/organic/live/advanced", `[{"keyword":"shoes","depth":10}]`, 0.002},
		{"live serp default depth is 10", "/v3/serp/google/organic/live/advanced", `[{"keyword":"shoes"}]`, 0.002},
		{"live serp with operator", "/v3/serp/google/organic/live/advanced", `[{"keyword":"site:example.com shoes","depth":10}]`, 0.01},
		{"live serp with pixel rectangles adds one base price", "/v3/serp/google/organic/live/advanced", `[{"keyword":"shoes","depth":10,"calculate_rectangles":true}]`, 0.004},
		{"queued serp per task", "/v3/serp/google/organic/task_post", `[{"keyword":"a","depth":20},{"keyword":"b","depth":20}]`, 2 * (0.0006 + 0.0006)},
		{"queued serp priority", "/v3/serp/google/organic/task_post", `[{"keyword":"a","depth":10,"priority":2}]`, 0.0012},
		{"labs with limit at July 2026 rates", "/v3/dataforseo_labs/google/keyword_ideas/live", `[{"limit":100}]`, 0.024},
		{"clickstream doubles the complete Labs task", "/v3/dataforseo_labs/google/keyword_suggestions/live", `[{"limit":75,"include_clickstream_data":true}]`, 0.042},
		{"disabled clickstream keeps the ordinary price", "/v3/dataforseo_labs/google/keyword_suggestions/live", `[{"limit":75,"include_clickstream_data":false}]`, 0.021},
		{"mixed batch prices each clickstream task separately", "/v3/dataforseo_labs/google/keyword_ideas/live", `[{"limit":75,"include_clickstream_data":true},{"limit":75}]`, 0.063},
		{"labs without limit holds the maximum rows at July 2026 rates", "/v3/dataforseo_labs/google/keyword_overview/live", `[{}]`, 0.132},
		{"backlinks at July 2026 rates", "/v3/backlinks/referring_domains/live", `[{"limit":1000}]`, 0.06},
		{"google ads live", "/v3/keywords_data/google_ads/search_volume/live", `[{}]`, 0.09},
		{"ai mentions", "/v3/ai_optimization/llm_mentions/search/live", `[{}]`, 0.25},
		{"ai model answer", "/v3/ai_optimization/chat_gpt/llm_responses/live", `[{}]`, 0.1},
		{"lighthouse", "/v3/on_page/lighthouse/live/json", `[{}]`, 0.005},
		{"unknown endpoint is held high", "/v3/content_analysis/search/live", `[{}]`, 1},
		{"unknown business endpoint is held high", "/v3/business_data/unknown/live", `[{}]`, 1},
		{"body that is not a task list is one task", "/v3/on_page/lighthouse/live/json", `{"url":"x"}`, 0.005},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := EstimateUSD(tc.path, []byte(tc.body)); math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("EstimateUSD() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEstimateUSDForApprovedBriefCalls(t *testing.T) {
	serp := EstimateUSD("/v3/serp/google/organic/live/advanced", []byte(`[{"depth":30}]`))
	parsing := EstimateUSD("/v3/on_page/content_parsing/live", []byte(`[{},{},{},{},{},{},{},{}]`))
	if math.Abs(serp-0.006) > 1e-9 || math.Abs(parsing-0.0012) > 1e-9 || math.Abs(serp+parsing-0.0072) > 1e-9 {
		t.Fatalf("brief ceilings SERP=%v parse=%v total=%v, want 0.006, 0.0012, 0.0072", serp, parsing, serp+parsing)
	}
}
