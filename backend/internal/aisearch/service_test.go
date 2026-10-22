package aisearch

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend/spendtest"
)

func lookupInput() BrandLookupInput {
	return BrandLookupInput{Query: "acme.com", LocationCode: 2826, LanguageCode: "en"}
}

func TestBrandLookupBuysTotalsThenOnlyTheMarketsPlatform(t *testing.T) {
	api := newFakeAPI(t)
	svc := newTestService(t, api)

	result, err := svc.BrandLookup(context.Background(), newOrg(), "project-1", lookupInput())
	if err != nil {
		t.Fatalf("BrandLookup() error = %v", err)
	}

	// One totals call, then Google's sample and, as 5 mentions do not fit a
	// 1-answer sample, its cited pages. ChatGPT is US-only and the market is
	// the UK, so nothing is bought for it; no competitors, no comparison.
	if got := api.total(); got != 3 {
		t.Errorf("provider calls = %d, want 3", got)
	}
	if got := len(api.requestsTo(targetMetricsPath)); got != 1 {
		t.Errorf("totals calls = %d, want 1", got)
	}
	for _, r := range api.requests {
		if strings.Contains(r.body, `"platform":"chat_gpt"`) {
			t.Errorf("bought %s for ChatGPT outside its US market", r.path)
		}
	}
	if got := api.recorder.count(); got != 3 {
		t.Errorf("recorded costs = %d, want one per billed call (3)", got)
	}
	if !result.HasData || *result.TotalMentions != 5 || *result.TotalAISearchVolume != 50 {
		// Location 2826 is not US, so ChatGPT's 10 mentions are left out.
		t.Errorf("result = hasData %v, mentions %v, volume %v, want true, 5, 50", result.HasData, result.TotalMentions, result.TotalAISearchVolume)
	}
	if result.ResolvedTarget != "acme.com" || *result.Scope != ScopeSubdomains || result.DetectedTargetType != "domain" {
		t.Errorf("target = %q %v %q, want acme.com subdomains domain", result.ResolvedTarget, *result.Scope, result.DetectedTargetType)
	}
	if len(result.TopPages) != 1 || len(result.TopQueries) != 1 {
		t.Errorf("topPages = %d, topQueries = %d, want 1 and 1 (Google only)", len(result.TopPages), len(result.TopQueries))
	}
	// The sites, search results and brands come with the totals call: shown, not re-bought.
	if len(result.SourceSites) != 2 || result.SourceSites[0].Domain != "acme.com" || result.SourceSites[1].Domain != "tripadvisor.com" {
		t.Errorf("SourceSites = %+v, want acme.com then tripadvisor.com without www.", result.SourceSites)
	}
	if len(result.RetrievedSites) != 1 || result.RetrievedSites[0].Domain != "g2.com" {
		t.Errorf("RetrievedSites = %+v, want g2.com", result.RetrievedSites)
	}
	if len(result.BrandsAlongside) != 1 || result.BrandsAlongside[0].Name != "Rival" {
		t.Errorf("BrandsAlongside = %+v, want Rival without the brand itself", result.BrandsAlongside)
	}
	if result.UnbrandedQuestions != 1 || result.BrandedQuestions != 0 || result.TopQueries[0].Branded {
		t.Errorf("questions = %d branded, %d unbranded; want the one sample prompt unbranded", result.BrandedQuestions, result.UnbrandedQuestions)
	}
}

func TestBrandLookupStopsAfterTheTotalsWhenAINeverMentionsTheBrand(t *testing.T) {
	api := newFakeAPI(t)
	api.on(targetMetricsPath, func(int, string) taskReply {
		return okReply(map[string]any{"aggregated_metrics": map[string]any{"platform": []any{}, "total": map[string]any{"mentions": 0}}})
	})
	svc := newTestService(t, api)
	in := lookupInput()
	in.LocationCode = 2840

	result, err := svc.BrandLookup(context.Background(), newOrg(), "p", in)
	if err != nil {
		t.Fatal(err)
	}
	if api.total() != 1 || result.HasData {
		t.Fatalf("calls = %d, hasData = %v; want only the totals call and no data", api.total(), result.HasData)
	}
	if result.PerPlatform[0].Status != "success" || result.PerPlatform[1].Status != "success" {
		t.Errorf("PerPlatform = %+v, want both platforms answered with no mentions", result.PerPlatform)
	}
}

func TestBrandLookupBuildsCitedPagesFromACompleteSample(t *testing.T) {
	api := newFakeAPI(t)
	api.on(targetMetricsPath, func(int, string) taskReply {
		return okReply(map[string]any{"aggregated_metrics": map[string]any{"platform": []any{
			map[string]any{"key": platformGoogle, "mentions": 2, "ai_search_volume": 3000},
		}}})
	})
	api.on(mentionsSearchPath, func(int, string) taskReply {
		return okReply(map[string]any{"items": []any{
			map[string]any{"question": "acme reviews", "ai_search_volume": 2000, "sources": []any{
				map[string]any{"url": "https://acme.com/"}, map[string]any{"url": "https://acme.com/"},
			}},
			map[string]any{"question": "best seo tool", "ai_search_volume": 1000, "sources": []any{
				map[string]any{"url": "https://acme.com/"}, map[string]any{"url": "https://review.site/acme"},
			}},
		}})
	})
	svc := newTestService(t, api)

	result, err := svc.BrandLookup(context.Background(), newOrg(), "p", lookupInput())
	if err != nil {
		t.Fatal(err)
	}
	if got := len(api.requestsTo(topPagesPath)); got != 0 || api.total() != 2 {
		t.Fatalf("top_pages calls = %d, total = %d; want the 2-answer sample to replace the top_pages purchase", got, api.total())
	}
	if len(result.TopPages) != 2 || result.TopPages[0].URL != "https://acme.com/" || *result.TopPages[0].Mentions != 2 || *result.TopPages[0].CapturedVolume != 3000 {
		t.Fatalf("TopPages = %+v, want acme.com/ cited by both answers (a page repeated in one answer counts once)", result.TopPages)
	}
	if result.BrandedQuestions != 1 || result.UnbrandedQuestions != 1 {
		t.Errorf("questions = %d branded, %d unbranded; want 1 and 1", result.BrandedQuestions, result.UnbrandedQuestions)
	}
	var mentionBody []map[string]any
	if err := json.Unmarshal([]byte(api.requestsTo(mentionsSearchPath)[0].body), &mentionBody); err != nil {
		t.Fatal(err)
	}
	if order, _ := mentionBody[0]["order_by"].([]any); len(order) != 1 || order[0] != "ai_search_volume,desc" {
		t.Errorf("mentions order_by = %v, want the most searched prompts first", mentionBody[0]["order_by"])
	}
}

func TestBrandLookupPreflightsWholeFanoutAndCacheHitHasNoHold(t *testing.T) {
	api := newFakeAPI(t)
	guard := &spendtest.Guard{}
	svc := newTestService(t, api, guard)
	org := newOrg()
	if _, err := svc.BrandLookup(context.Background(), org, "p", lookupInput()); err != nil {
		t.Fatal(err)
	}
	held := guard.Held()
	if want := 3 * dataforseo.AIMentionsCeilingUSD; len(held) != 1 || held[0].EstimateUSD != want {
		t.Fatalf("holds = %+v; want one hold of three $0.25 mentions ceilings (totals, sample, pages)", held)
	}
	if _, err := svc.BrandLookup(context.Background(), org, "p", lookupInput()); err != nil {
		t.Fatal(err)
	}
	settled := guard.Settled()
	if len(guard.Held()) != 1 || api.total() != 3 || len(settled) != 1 {
		t.Fatalf("cache hit changed paid work: holds=%d settlements=%d calls=%d", len(guard.Held()), len(settled), api.total())
	}
}

func TestConcurrentBrandLookupsSharePaidFanoutWithinOrganization(t *testing.T) {
	api := newFakeAPI(t)
	api.on(targetMetricsPath, func(_ int, _ string) taskReply {
		time.Sleep(30 * time.Millisecond)
		return defaultReply(targetMetricsPath)
	})
	svc := newTestService(t, api)
	org := newOrg()
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			<-start
			if _, err := svc.BrandLookup(context.Background(), org, "p", lookupInput()); err != nil {
				t.Errorf("BrandLookup: %v", err)
			}
		})
	}
	close(start)
	wg.Wait()
	if got := api.total(); got != 3 {
		t.Fatalf("provider calls = %d, want one three-call fanout", got)
	}
	if got := api.recorder.count(); got != 3 {
		t.Fatalf("recorded costs = %d, want three", got)
	}
}

func TestConcurrentBrandLookupsDoNotShareAcrossOrganizations(t *testing.T) {
	api := newFakeAPI(t)
	api.on(targetMetricsPath, func(_ int, _ string) taskReply {
		time.Sleep(30 * time.Millisecond)
		return defaultReply(targetMetricsPath)
	})
	svc := newTestService(t, api)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range 2 {
		org := newOrg()
		wg.Go(func() {
			<-start
			if _, err := svc.BrandLookup(context.Background(), org, "p", lookupInput()); err != nil {
				t.Errorf("BrandLookup: %v", err)
			}
		})
	}
	close(start)
	wg.Wait()
	if got := api.total(); got != 6 {
		t.Fatalf("provider calls = %d, want a separate three-call fanout per organization", got)
	}
}

func TestBrandLookupRecordsMonthlyVisibilityHistoryWithoutExtraProviderCalls(t *testing.T) {
	api := newFakeAPI(t)
	svc := newTestService(t, api)
	org := newOrg()
	ctx := context.Background()
	result, err := svc.BrandLookup(ctx, org, "project-1", lookupInput())
	if err != nil {
		t.Fatal(err)
	}
	providerCalls := api.total()

	history, err := svc.BrandVisibilityHistory(ctx, org, "project-1", lookupInput())
	if err != nil {
		t.Fatal(err)
	}
	if api.total() != providerCalls {
		t.Fatalf("history read made provider calls: before %d, after %d", providerCalls, api.total())
	}
	if len(history) != 1 || history[0].TotalMentions != intOrZero(result.TotalMentions) || history[0].CitedSources != len(result.TopPages) {
		t.Fatalf("history = %#v, want one snapshot from the lookup result", history)
	}

	key, err := brandLookupKey(org, "project-1", lookupInput())
	if err != nil {
		t.Fatal(err)
	}
	svc.now = func() time.Time { return time.Now().UTC().AddDate(0, 1, 0) }
	svc.saveVisibilitySnapshot(ctx, key, result)
	history, err = svc.BrandVisibilityHistory(ctx, org, "project-1", lookupInput())
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2 || history[0].Period >= history[1].Period {
		t.Fatalf("monthly history = %#v, want two ordered periods", history)
	}
	otherProject, err := svc.BrandVisibilityHistory(ctx, org, "other-project", lookupInput())
	if err != nil || len(otherProject) != 0 {
		t.Fatalf("other project history = %#v, err=%v, want empty", otherProject, err)
	}
}

func TestBrandLookupAsksChatGPTForUSEnglishOnly(t *testing.T) {
	bodies := func(api *fakeAPI, path string) []map[string]any {
		var sent []map[string]any
		for _, r := range api.requestsTo(path) {
			var body []map[string]any
			if err := json.Unmarshal([]byte(r.body), &body); err != nil {
				t.Fatal(err)
			}
			sent = append(sent, body[0])
		}
		return sent
	}

	api := newFakeAPI(t)
	svc := newTestService(t, api)
	in := lookupInput()
	in.LocationCode, in.LanguageCode = 2356, "hi"
	if _, err := svc.BrandLookup(context.Background(), newOrg(), "p", in); err != nil {
		t.Fatal(err)
	}
	totals := bodies(api, targetMetricsPath)
	if len(totals) != 1 || totals[0]["location_code"] != 2356.0 || totals[0]["language_code"] != "hi" || totals[0]["platform"] != nil {
		t.Fatalf("totals calls = %+v, want one for the market and every platform", totals)
	}
	for _, body := range bodies(api, mentionsSearchPath) {
		if body["platform"] != platformGoogle || body["location_code"] != 2356.0 || body["language_code"] != "hi" {
			t.Errorf("mentions call %v, want Google in the market only", body)
		}
	}

	us := newFakeAPI(t)
	svc = newTestService(t, us)
	in.LocationCode, in.LanguageCode = 2840, "en"
	if _, err := svc.BrandLookup(context.Background(), newOrg(), "p", in); err != nil {
		t.Fatal(err)
	}
	sent := bodies(us, mentionsSearchPath)
	if len(sent) != 2 || sent[0]["platform"] != platformChatGPT || sent[0]["location_code"] != 2840.0 || sent[0]["language_code"] != "en" {
		t.Fatalf("US mentions calls = %+v, want ChatGPT then Google", sent)
	}
}

func TestBrandLookupScopesTheProviderTarget(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		scope          Scope
		wantSubdomains any
		wantKeyword    string
	}{
		{name: "root domain includes subdomains", query: "acme.com", wantSubdomains: true},
		{name: "domain scope excludes subdomains", query: "acme.com", scope: ScopeDomain, wantSubdomains: false},
		{name: "subfolder is filtered after the call, not scoped by it", query: "acme.com/blog", wantSubdomains: false},
		{name: "keyword has no subdomain flag", query: "Acme Brand", wantKeyword: "Acme Brand"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := newFakeAPI(t)
			svc := newTestService(t, api)
			in := lookupInput()
			in.Query, in.Scope = tt.query, tt.scope

			if _, err := svc.BrandLookup(context.Background(), newOrg(), "p", in); err != nil {
				t.Fatal(err)
			}

			var body []struct {
				Target []map[string]any `json:"target"`
			}
			if err := json.Unmarshal([]byte(api.requestsTo(targetMetricsPath)[0].body), &body); err != nil {
				t.Fatal(err)
			}
			target := body[0].Target[0]
			if tt.wantKeyword != "" {
				if target["keyword"] != tt.wantKeyword || target["match_type"] != "word_match" {
					t.Errorf("target = %v, want keyword %q with word_match", target, tt.wantKeyword)
				}
				return
			}
			if target["include_subdomains"] != tt.wantSubdomains {
				t.Errorf("include_subdomains = %v, want %v", target["include_subdomains"], tt.wantSubdomains)
			}
		})
	}
}

func TestBrandLookupCachesCompleteResultsPerScope(t *testing.T) {
	api := newFakeAPI(t)
	svc := newTestService(t, api)
	org := newOrg()
	ctx := context.Background()

	if _, err := svc.BrandLookup(ctx, org, "p", lookupInput()); err != nil {
		t.Fatal(err)
	}
	calls := api.total()

	in := lookupInput()
	in.Query = "https://www.acme.com/" // same target, different spelling
	cached, err := svc.BrandLookup(ctx, org, "p", in)
	if err != nil {
		t.Fatal(err)
	}
	if api.total() != calls {
		t.Errorf("a repeated lookup made %d provider calls, want none", api.total()-calls)
	}
	if cached.Query != in.Query {
		t.Errorf("cached Query = %q, want the new spelling %q", cached.Query, in.Query)
	}

	in.Scope = ScopeDomain
	if _, err := svc.BrandLookup(ctx, org, "p", in); err != nil {
		t.Fatal(err)
	}
	if api.total() == calls {
		t.Error("a different scope was served from the cache, want a new lookup")
	}

	if _, err := svc.BrandLookup(ctx, newOrg(), "p", lookupInput()); err != nil {
		t.Fatal(err)
	}
	if api.total() <= calls*2 {
		t.Error("another organization was served from the cache, want its own paid lookup")
	}
}

func TestBrandLookupDoesNotCachePartialResults(t *testing.T) {
	api := newFakeAPI(t)
	api.on(topPagesPath, func(int, string) taskReply { return failReply(40501, "Invalid Field: 'target'.") })
	svc := newTestService(t, api)
	org := newOrg()

	first, err := svc.BrandLookup(context.Background(), org, "p", lookupInput())
	if err != nil {
		t.Fatal(err)
	}
	if !first.HasData || len(first.TopPages) != 0 {
		t.Errorf("partial result = hasData %v, topPages %d, want data without pages", first.HasData, len(first.TopPages))
	}
	calls := api.total()

	if _, err := svc.BrandLookup(context.Background(), org, "p", lookupInput()); err != nil {
		t.Fatal(err)
	}
	if api.total() == calls {
		t.Error("a result with a failed sub-call was cached, want it fetched again")
	}
}

func TestBrandLookupReportsAFailedPlatformWithoutLosingTheOther(t *testing.T) {
	api := newFakeAPI(t)
	svc := newTestService(t, api)
	api.on(topPagesPath, func(_ int, body string) taskReply {
		if strings.Contains(body, `"platform":"google"`) {
			return failReply(50000, "Internal Error.")
		}
		return defaultReply(topPagesPath)
	})
	api.on(mentionsSearchPath, func(_ int, body string) taskReply {
		if strings.Contains(body, `"platform":"google"`) {
			return failReply(50000, "Internal Error.")
		}
		return defaultReply(mentionsSearchPath)
	})

	result, err := svc.BrandLookup(context.Background(), newOrg(), "p", BrandLookupInput{Query: "acme.com", LocationCode: 2840, LanguageCode: "en"})
	if err != nil {
		t.Fatalf("BrandLookup() error = %v, want the surviving platform's data", err)
	}

	if got := result.PerPlatform[1]; got.Platform != platformGoogle || got.Status != "error" {
		t.Errorf("google row = %+v, want status error", got)
	}
	if got := result.PerPlatform[0]; got.Status != "success" || *got.Mentions != 10 {
		t.Errorf("chat_gpt row = %+v, want success with 10 mentions", got)
	}
}

func TestBrandLookupBillingIssueFailsTheLookup(t *testing.T) {
	api := newFakeAPI(t)
	api.on(targetMetricsPath, func(int, string) taskReply { return failReply(40200, "Payment Required.") })
	svc := newTestService(t, api)

	_, err := svc.BrandLookup(context.Background(), newOrg(), "p", lookupInput())

	if !errors.Is(err, dataforseo.ErrBillingIssue) {
		t.Fatalf("BrandLookup() error = %v, want ErrBillingIssue", err)
	}
	if got := api.total(); got != 1 {
		t.Errorf("provider calls after a billing failure = %d, want 1 (stop at the first)", got)
	}
}

func TestBrandLookupComparesCompetitors(t *testing.T) {
	api := newFakeAPI(t)
	svc := newTestService(t, api)
	in := lookupInput()
	in.Competitors = []string{"rival.com", "RIVAL.com", "acme.com"} // a duplicate and the target itself

	result, err := svc.BrandLookup(context.Background(), newOrg(), "p", in)
	if err != nil {
		t.Fatal(err)
	}

	requests := api.requestsTo(crossAggregatedPath)
	if len(requests) != 1 {
		t.Fatalf("cross-aggregated calls = %d, want one: Google only outside ChatGPT's US market", len(requests))
	}
	var body []struct {
		Targets []struct {
			Key string `json:"aggregation_key"`
		} `json:"targets"`
	}
	if err := json.Unmarshal([]byte(requests[0].body), &body); err != nil {
		t.Fatal(err)
	}
	if keys := body[0].Targets; len(keys) != 2 || keys[0].Key != "acme.com" || keys[1].Key != "rival.com" {
		t.Errorf("compared keys = %+v, want acme.com and rival.com once each", keys)
	}
	if result.ShareOfVoice == nil || len(result.ShareOfVoice.Entries) != 2 {
		t.Fatalf("ShareOfVoice = %+v, want 2 entries", result.ShareOfVoice)
	}
}

func TestBrandLookupRejectsAScopeThatDoesNotFitTheQuery(t *testing.T) {
	api := newFakeAPI(t)
	svc := newTestService(t, api)
	in := lookupInput()
	in.Scope = ScopeSubfolder // acme.com has no path

	_, err := svc.BrandLookup(context.Background(), newOrg(), "p", in)

	var input inputError
	if !errors.As(err, &input) {
		t.Fatalf("BrandLookup() error = %v, want an inputError", err)
	}
	if api.total() != 0 {
		t.Errorf("provider calls = %d, want none for an invalid request", api.total())
	}
}

func TestBrandLookupStopsWhenTheCallerGoesAway(t *testing.T) {
	api := newFakeAPI(t)
	svc := newTestService(t, api)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := svc.BrandLookup(ctx, newOrg(), "p", lookupInput()); !errors.Is(err, context.Canceled) {
		t.Fatalf("BrandLookup() error = %v, want context.Canceled", err)
	}
}

func TestBrandLookupDoesNotCacheAnEmptyResult(t *testing.T) {
	api := newFakeAPI(t)
	api.on(targetMetricsPath, func(int, string) taskReply {
		return okReply(map[string]any{"aggregated_metrics": map[string]any{"platform": []any{}}})
	})
	svc := newTestService(t, api)
	org := newOrg()

	first, err := svc.BrandLookup(context.Background(), org, "p", lookupInput())
	if err != nil {
		t.Fatal(err)
	}
	if first.HasData {
		t.Fatal("HasData = true for an empty provider answer")
	}
	calls := api.total()

	// The provider may simply not have data yet, so the user can look again later.
	if _, err := svc.BrandLookup(context.Background(), org, "p", lookupInput()); err != nil {
		t.Fatal(err)
	}
	if api.total() == calls {
		t.Error("an empty result was cached, want it fetched again")
	}
}

func TestBrandLookupCompetitorOrderAndCaseShareOneCacheEntry(t *testing.T) {
	api := newFakeAPI(t)
	svc := newTestService(t, api)
	org := newOrg()
	first := lookupInput()
	first.Competitors = []string{"rival.com", "other.com"}
	if _, err := svc.BrandLookup(context.Background(), org, "p", first); err != nil {
		t.Fatal(err)
	}
	calls := api.total()

	second := lookupInput()
	second.Competitors = []string{"OTHER.com", "Rival.com"}
	if _, err := svc.BrandLookup(context.Background(), org, "p", second); err != nil {
		t.Fatal(err)
	}
	if api.total() != calls {
		t.Errorf("a reordered competitor list made %d new provider calls, want none", api.total()-calls)
	}
}

func TestCacheKeys(t *testing.T) {
	key := cacheKey("brand-lookup", "org-1", "a", "b")
	if !strings.HasPrefix(key, "ai-search:brand-lookup:org-1:") {
		t.Errorf("key = %q, want the kind and organization in the clear, for purging by prefix", key)
	}
	for name, other := range map[string]string{
		"another organization":         cacheKey("brand-lookup", "org-2", "a", "b"),
		"another kind":                 cacheKey("prompt-response", "org-1", "a", "b"),
		"a part moved across boundary": cacheKey("brand-lookup", "org-1", "ab", ""),
		"a part moved the other way":   cacheKey("brand-lookup", "org-1", "", "ab"),
		"a part appended":              cacheKey("brand-lookup", "org-1", "a", "b", ""),
	} {
		if other == key {
			t.Errorf("%s gives the same key %q", name, key)
		}
	}
	if again := cacheKey("brand-lookup", "org-1", "a", "b"); again != key {
		t.Errorf("same input gives %q then %q, want a stable key", key, again)
	}
}

func TestBrandLookupTreatsACorruptCacheEntryAsAMiss(t *testing.T) {
	api := newFakeAPI(t)
	svc := newTestService(t, api)
	org := newOrg()
	ctx := context.Background()
	if _, err := svc.BrandLookup(ctx, org, "p", lookupInput()); err != nil {
		t.Fatal(err)
	}
	keys, err := svc.redis.Keys(ctx, "ai-search:brand-lookup:"+org+":*").Result()
	if err != nil || len(keys) != 1 {
		t.Fatalf("cache keys = %v, %v; want exactly one for the organization", keys, err)
	}
	if err := svc.redis.Set(ctx, keys[0], "{not json", 0).Err(); err != nil {
		t.Fatal(err)
	}
	calls := api.total()

	result, err := svc.BrandLookup(ctx, org, "p", lookupInput())

	if err != nil || !result.HasData {
		t.Fatalf("BrandLookup() = hasData %v, error %v; want a fresh lookup", result.HasData, err)
	}
	if api.total() == calls {
		t.Error("a corrupt cache entry was served, want it ignored and fetched again")
	}
}

func TestBrandLookupWorksWhenRedisIsDown(t *testing.T) {
	api := newFakeAPI(t)
	svc := newTestService(t, api)
	down := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", MaxRetries: -1})
	t.Cleanup(func() {
		if err := down.Close(); err != nil {
			t.Errorf("close redis: %v", err)
		}
	})
	svc.redis = down

	result, err := svc.BrandLookup(context.Background(), newOrg(), "p", lookupInput())

	if err != nil || !result.HasData {
		t.Fatalf("BrandLookup() = hasData %v, error %v; want data without the cache", result.HasData, err)
	}
}

func TestBrandLookupShareOfVoiceSurvivesOnePlatformFailing(t *testing.T) {
	api := newFakeAPI(t)
	api.on(crossAggregatedPath, func(_ int, body string) taskReply {
		if strings.Contains(body, `"platform":"chat_gpt"`) {
			return failReply(50000, "Internal Error.")
		}
		return defaultReply(crossAggregatedPath)
	})
	svc := newTestService(t, api)
	org := newOrg()
	in := lookupInput()
	in.LocationCode = 2840
	in.Competitors = []string{"rival.com"}

	result, err := svc.BrandLookup(context.Background(), org, "p", in)
	if err != nil {
		t.Fatal(err)
	}

	if sov := result.ShareOfVoice; sov == nil || len(sov.Platforms) != 1 || sov.Platforms[0] != platformGoogle {
		t.Errorf("ShareOfVoice = %+v, want google only", sov)
	}
	calls := api.total()
	if _, err := svc.BrandLookup(context.Background(), org, "p", in); err != nil {
		t.Fatal(err)
	}
	if api.total() == calls {
		t.Error("a result with a failed share-of-voice call was cached, want it fetched again")
	}
}

func TestBrandLookupBillingIssueDuringShareOfVoiceFailsTheLookup(t *testing.T) {
	api := newFakeAPI(t)
	api.on(crossAggregatedPath, func(int, string) taskReply { return failReply(40200, "Payment Required.") })
	svc := newTestService(t, api)
	in := lookupInput()
	in.Competitors = []string{"rival.com"}

	if _, err := svc.BrandLookup(context.Background(), newOrg(), "p", in); !errors.Is(err, dataforseo.ErrBillingIssue) {
		t.Fatalf("BrandLookup() error = %v, want ErrBillingIssue", err)
	}
}

func TestBrandLookupAllSubCallsFailingMarksThePlatformFailed(t *testing.T) {
	api := newFakeAPI(t)
	for _, path := range []string{topPagesPath, mentionsSearchPath} {
		api.on(path, func(_ int, body string) taskReply {
			if strings.Contains(body, `"platform":"chat_gpt"`) {
				return failReply(50000, "Internal Error.")
			}
			return defaultReply(path)
		})
	}
	svc := newTestService(t, api)
	in := lookupInput()
	in.LocationCode = 2840

	result, err := svc.BrandLookup(context.Background(), newOrg(), "p", in)
	if err != nil {
		t.Fatal(err)
	}

	if got := result.PerPlatform[0]; got.Platform != platformChatGPT || got.Status != "error" {
		t.Errorf("chat_gpt row = %+v, want status error", got)
	}
	if got := result.PerPlatform[1]; got.Status != "success" {
		t.Errorf("google row = %+v, want success", got)
	}
}
