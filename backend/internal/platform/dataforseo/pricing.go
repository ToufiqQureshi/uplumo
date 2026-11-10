package dataforseo

import (
	"encoding/json"
	"math"
	"regexp"
	"strings"
)

// Price ceilings per task in USD, from DataForSEO's published prices. They are
// what a task can cost at most, because they are held as credit before the
// request is sent; the real price is charged afterwards and the rest is
// released. A path this table does not know is held at unknownTaskUSD, so a
// new endpoint costs a customer a large hold rather than costing Uplumo money.
const (
	serpLivePageUSD     = 0.002
	serpLiveExtraUSD    = 0.002
	contentParsePageUSD = 0.00015
	serpQueuedPageUSD   = 0.0006
	serpQueuedExtraUSD  = 0.0006
	serpQueuedAIOUSD    = 0.0006
	onPageBasicPageUSD  = 0.00015
	onPageJSPageUSD     = 0.0015
	serpPriorityFactor  = 2
	serpOperatorFactor  = 5
	// DataForSEO Labs and Backlinks rates reflect their July 1, 2026 updates.
	labsTaskUSD          = 0.012
	labsRowUSD           = 0.00012
	labsDefaultRows      = 1000
	backlinksTaskUSD     = 0.024
	backlinksRowUSD      = 0.000036
	backlinksDefaultRows = 1000
	googleAdsTaskUSD     = 0.09
	// AI optimization tasks were held at a flat $0.50, 10-17 times what they
	// bill (live, Oct 2026: a ChatGPT answer with web search $0.03, the dearest
	// answer seen $0.05; a mentions search $0.151, aggregated and top pages
	// $0.101, the dearest seen $0.173). The ceilings below sit above those.
	AIResponseCeilingUSD = 0.10
	AIMentionsCeilingUSD = 0.25
	lighthouseTaskUSD    = 0.005
	// Google Maps SERP is billed per 100 results; Business Data per task and,
	// for updates and reviews, per 10 items. Priority (2) doubles a queued
	// price. Published prices, Oct 2026.
	mapsQueuedPer100USD      = 0.0006
	mapsLivePer100USD        = 0.002
	businessInfoQueuedUSD    = 0.0015
	businessInfoLiveUSD      = 0.0054
	businessUpdatesQueuedUSD = 0.0015
	businessUpdatesPer10USD  = 0.00075
	businessReviewsPer10USD  = 0.00075
	businessDefaultDepth     = 10
	businessPriorityFactor   = 2
	unknownTaskUSD           = 1.0
	defaultSerpDepth         = 10
)

// serpOperator matches advanced search operators, which Google SERP endpoints
// bill at five times the price.
var serpOperator = regexp.MustCompile(`(?i)(allinanchor|allintext|allintitle|allinurl|cache|define|definition|filetype|id|inanchor|info|intext|intitle|inurl|link|site):`)

// freePaths are lookups DataForSEO does not charge for.
var freePaths = []string{"/v3/appendix/", "/v3/serp/google/locations", "/v3/on_page/raw_html"}

// EstimateUSD returns the most a request to path with body can cost.
func EstimateUSD(path string, body []byte) float64 {
	for _, free := range freePaths {
		if strings.HasPrefix(path, free) {
			return 0
		}
	}
	if strings.HasSuffix(path, "/models") {
		return 0
	}
	tasks := taskBodies(body)
	total := 0.0
	for _, task := range tasks {
		total += taskCeiling(path, task)
	}
	return math.Round(total*1e6) / 1e6
}

// BrandLookupCeilingUSD is the most one Brand Lookup can buy, and so the most
// it can charge: the totals call, then per platform a mentions sample and its
// cited pages, plus one comparison per platform when competitors are set.
func BrandLookupCeilingUSD(platforms int, comparisons bool) float64 {
	calls := 1 + 2*platforms
	if comparisons {
		calls += platforms
	}
	return float64(calls) * AIMentionsCeilingUSD
}

// PromptExplorerCeilingUSD is the most one Prompt Explorer run can charge: one
// answer per model, and with web search one bounded retry per model.
func PromptExplorerCeilingUSD(models int, webSearch bool) float64 {
	calls := models
	if webSearch {
		calls *= 2
	}
	return float64(calls) * AIResponseCeilingUSD
}

// SerpCostUSD returns the published cost of one Google Organic SERP task at
// the requested depth, using Live or Standard queue pricing.
func SerpCostUSD(depth int, queued bool) float64 {
	path := "/v3/serp/google/organic/live/advanced"
	if queued {
		path = "/v3/serp/google/organic/task_post"
	}
	return serpCeiling(path, map[string]any{"depth": float64(depth)})
}

// ContentParsingCostUSD returns the published cost of parsing the requested
// number of pages.
func ContentParsingCostUSD(pages int) float64 {
	if pages <= 0 {
		return 0
	}
	return math.Round(float64(pages)*contentParsePageUSD*1e6) / 1e6
}

// OnPageJavaScriptCostUSD returns the published JavaScript-rendered On-Page
// price for the requested pages.
func OnPageJavaScriptCostUSD(pages int) float64 {
	if pages <= 0 {
		return 0
	}
	return math.Round(float64(pages)*onPageJSPageUSD*1e6) / 1e6
}

// taskBodies splits a request body into its tasks. DataForSEO takes a JSON
// array of tasks; anything else is one task.
func taskBodies(body []byte) []map[string]any {
	var tasks []map[string]any
	if err := json.Unmarshal(body, &tasks); err == nil && len(tasks) > 0 {
		return tasks
	}
	return []map[string]any{{}}
}

func taskCeiling(path string, task map[string]any) float64 {
	switch {
	case strings.HasPrefix(path, "/v3/serp/google/maps/"):
		return mapsCeiling(path, task)
	case strings.HasPrefix(path, "/v3/business_data/google/"):
		return businessCeiling(path, task)
	case strings.HasPrefix(path, "/v3/serp/"):
		return serpCeiling(path, task)
	case strings.HasPrefix(path, "/v3/dataforseo_labs/"):
		price := labsTaskUSD + labsRowUSD*rows(task, labsDefaultRows)
		if clickstream, _ := task["include_clickstream_data"].(bool); clickstream {
			price *= 2
		}
		return price
	case strings.HasPrefix(path, "/v3/backlinks/"):
		return backlinksTaskUSD + backlinksRowUSD*rows(task, backlinksDefaultRows)
	case strings.HasPrefix(path, "/v3/keywords_data/"):
		return googleAdsTaskUSD
	case strings.HasPrefix(path, "/v3/ai_optimization/") && strings.Contains(path, "/llm_responses/"):
		return AIResponseCeilingUSD
	case strings.HasPrefix(path, "/v3/ai_optimization/"):
		return AIMentionsCeilingUSD
	case strings.HasPrefix(path, "/v3/on_page/instant_pages"):
		if javascript, _ := task["enable_javascript"].(bool); javascript {
			return onPageJSPageUSD
		}
		return onPageBasicPageUSD
	case strings.HasPrefix(path, "/v3/on_page/content_parsing/live"):
		return contentParsePageUSD
	case strings.HasPrefix(path, "/v3/on_page/lighthouse"):
		return lighthouseTaskUSD
	}
	return unknownTaskUSD
}

func serpCeiling(path string, task map[string]any) float64 {
	pages := math.Max(1, math.Ceil(number(task, "depth", defaultSerpDepth)/10))
	price := serpLivePageUSD + (pages-1)*serpLiveExtraUSD
	if strings.Contains(path, "/task_post") {
		price = serpQueuedPageUSD + (pages-1)*serpQueuedExtraUSD
		if number(task, "priority", 1) >= 2 {
			price *= serpPriorityFactor
		}
		if async, _ := task["load_async_ai_overview"].(bool); async {
			price += serpQueuedAIOUSD
		}
	}
	if keyword, _ := task["keyword"].(string); serpOperator.MatchString(keyword) {
		price *= serpOperatorFactor
	}
	return price
}

func mapsCeiling(path string, task map[string]any) float64 {
	blocks := math.Max(1, math.Ceil(number(task, "depth", 100)/100))
	if !strings.Contains(path, "/task_post") {
		return blocks * mapsLivePer100USD
	}
	price := blocks * mapsQueuedPer100USD
	if number(task, "priority", 1) >= 2 {
		price *= businessPriorityFactor
	}
	return price
}

// businessCeiling prices Google Business Data tasks. Only the endpoints
// Local Visibility posts are known; any other Business Data path is held at
// unknownTaskUSD like every unknown endpoint.
func businessCeiling(path string, task map[string]any) float64 {
	tens := math.Ceil(number(task, "depth", businessDefaultDepth) / 10)
	var price float64
	switch path {
	case "/v3/business_data/google/my_business_info/live":
		return businessInfoLiveUSD
	case "/v3/business_data/google/my_business_info/task_post":
		price = businessInfoQueuedUSD
	case "/v3/business_data/google/my_business_updates/task_post":
		price = businessUpdatesQueuedUSD + tens*businessUpdatesPer10USD
	case "/v3/business_data/google/reviews/task_post":
		price = tens * businessReviewsPer10USD
	default:
		return unknownTaskUSD
	}
	if number(task, "priority", 1) >= 2 {
		price *= businessPriorityFactor
	}
	return price
}

// rows is the result limit a task asked for, or fallback when it set none.
func rows(task map[string]any, fallback float64) float64 {
	return number(task, "limit", fallback)
}

func number(task map[string]any, key string, fallback float64) float64 {
	value, ok := task[key].(float64)
	if !ok || value <= 0 || math.IsInf(value, 0) {
		return fallback
	}
	return value
}
