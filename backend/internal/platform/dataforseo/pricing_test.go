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
		{"free On-Page raw HTML", "/v3/on_page/raw_html", `[{"id":"task","url":"https://example.com"}]`, 0},
		{"free ai models", "/v3/ai_optimization/chat_gpt/llm_responses/models", "", 0},
		{"live serp top 10", "/v3/serp/google/organic/live/advanced", `[{"keyword":"shoes","depth":10}]`, 0.002},
		{"live serp default depth is 10", "/v3/serp/google/organic/live/advanced", `[{"keyword":"shoes"}]`, 0.002},
		{"live serp with operator", "/v3/serp/google/organic/live/advanced", `[{"keyword":"site:example.com shoes","depth":10}]`, 0.01},
		{"queued serp per task", "/v3/serp/google/organic/task_post", `[{"keyword":"a","depth":20},{"keyword":"b","depth":20}]`, 2 * (0.0006 + 0.0006)},
		{"queued serp priority", "/v3/serp/google/organic/task_post", `[{"keyword":"a","depth":10,"priority":2}]`, 0.0012},
		{"queued Google AI Overview citations", "/v3/serp/google/organic/task_post", `[{"keyword":"a","depth":10,"load_async_ai_overview":true}]`, 0.0012},
		{"JavaScript-rendered On-Page task", "/v3/on_page/instant_pages", `[{"url":"https://example.com","enable_javascript":true,"store_raw_html":true}]`, 0.0015},
		{"ordinary On-Page instant task", "/v3/on_page/instant_pages", `[{"url":"https://example.com"}]`, 0.00015},
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
		{"queued maps priority, top 20", "/v3/serp/google/maps/task_post", `[{"keyword":"dentist","depth":20,"priority":2}]`, 0.0012},
		{"queued maps standard, 101 results is two blocks", "/v3/serp/google/maps/task_post", `[{"keyword":"dentist","depth":101}]`, 0.0012},
		{"live maps search", "/v3/serp/google/maps/live/advanced", `[{"keyword":"smile dental pune","depth":20}]`, 0.002},
		{"business info priority", "/v3/business_data/google/my_business_info/task_post", `[{"keyword":"cid:1","priority":2}]`, 0.003},
		{"business info live", "/v3/business_data/google/my_business_info/live", `[{"keyword":"cid:1"}]`, 0.0054},
		{"business updates priority, 10 posts", "/v3/business_data/google/my_business_updates/task_post", `[{"keyword":"cid:1","depth":10,"priority":2}]`, 0.0045},
		{"reviews priority, 100 reviews", "/v3/business_data/google/reviews/task_post", `[{"cid":"1","depth":100,"priority":2}]`, 0.015},
		{"reviews standard, default 10", "/v3/business_data/google/reviews/task_post", `[{"cid":"1"}]`, 0.00075},
		{"body that is not a task list is one task", "/v3/on_page/lighthouse/live/json", `{"url":"x"}`, 0.005},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := EstimateUSD(tc.path, []byte(tc.body)); math.Abs(got-tc.want) > 1e-9 {
				t.Fatalf("EstimateUSD() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestApprovedDataForSEOCostScenarios(t *testing.T) {
	if got := OnPageJavaScriptCostUSD(20); got != 0.03 {
		t.Fatalf("20 JavaScript pages cost $%v, want $0.03", got)
	}
}

func TestEstimateUSDForApprovedBriefCalls(t *testing.T) {
	serpEstimate := EstimateUSD("/v3/serp/google/organic/live/advanced", []byte(`[{"depth":30}]`))
	serp := SerpCostUSD(30, false)
	parsingEstimate := EstimateUSD("/v3/on_page/content_parsing/live", []byte(`[{},{},{},{},{},{},{},{}]`))
	parsing := ContentParsingCostUSD(8)
	if math.Abs(serp-serpEstimate) > 1e-9 || math.Abs(parsing-parsingEstimate) > 1e-9 ||
		math.Abs(serp-0.006) > 1e-9 || math.Abs(parsing-0.0012) > 1e-9 || math.Abs(serp+parsing-0.0072) > 1e-9 {
		t.Fatalf("brief ceilings SERP=%v/%v parse=%v/%v total=%v, want provider estimates 0.006, 0.0012, 0.0072", serp, serpEstimate, parsing, parsingEstimate, serp+parsing)
	}
	if ContentParsingCostUSD(0) != 0 || ContentParsingCostUSD(-1) != 0 {
		t.Fatal("content parsing cost must be zero when no pages are requested")
	}
}
