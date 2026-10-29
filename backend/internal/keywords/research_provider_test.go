package keywords

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend/spendtest"
)

func TestIsAdsKeyword(t *testing.T) {
	tests := []struct {
		keyword string
		want    bool
	}{
		{"best seo tool", true},
		{"日本語 キーワード", true},
		{"café près", true},
		{"c++ tutorial", true},
		{"100% free", false},
		{"what is seo?", false},
		{"seo, tools", false},
		{"rocket 🚀 seo", false},
		{"star ⭐ seo", false},
		{"a b c d e f g h i j k", false},
		{strings.Repeat("a", 81), false},
		{strings.Repeat("a", 80), true},
	}
	for _, tt := range tests {
		if got := IsAdsKeyword(tt.keyword); got != tt.want {
			t.Errorf("IsAdsKeyword(%q) = %v, want %v", tt.keyword, got, tt.want)
		}
	}
}

type providerServer struct {
	mu     sync.Mutex
	paths  []string
	bodies []map[string]any
	reply  func(path string) map[string]any
}

func newProviderServer(t *testing.T, reply func(path string) map[string]any) (*providerServer, DataForSEOProvider) {
	t.Helper()
	ps := &providerServer{reply: reply}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body []map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		ps.mu.Lock()
		ps.paths = append(ps.paths, r.URL.Path)
		if len(body) > 0 {
			ps.bodies = append(ps.bodies, body[0])
		}
		ps.mu.Unlock()
		task := ps.reply(r.URL.Path)
		task["path"] = []string{"v3"}
		task["cost"] = 0.01
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 20000, "tasks": []any{task}})
	}))
	t.Cleanup(server.Close)
	client, err := dataforseo.NewClient(dataforseo.Options{Guard: &spendtest.Guard{}, BaseURL: server.URL, APIKey: base64.StdEncoding.EncodeToString([]byte("fixture-user:fixture-password")), Recorder: noopRecorder{}})
	if err != nil {
		t.Fatal(err)
	}
	return ps, DataForSEOProvider{Client: client}
}

func okTask(result ...any) map[string]any {
	return map[string]any{"status_code": 20000, "status_message": "Ok.", "result": result}
}

func TestSuggestionsPreserveSeparateSeedDataWithoutAnotherPaidCall(t *testing.T) {
	ps, provider := newProviderServer(t, func(_ string) map[string]any {
		return okTask(map[string]any{
			"seed_keyword_data": map[string]any{
				"keyword": "hotel in lonavala",
				"keyword_info": map[string]any{"search_volume": 0, "cpc": 0.26, "competition": 0.98,
					"low_top_of_page_bid": 0.05, "high_top_of_page_bid": 0.32,
					"monthly_searches":    []any{map[string]any{"year": 2026, "month": 9, "search_volume": 0}},
					"search_volume_trend": map[string]any{"quarterly": -33}},
				"keyword_properties": map[string]any{"keyword_difficulty": 0},
				"search_intent_info": map[string]any{"main_intent": "commercial"},
				"avg_backlinks_info": map[string]any{"referring_main_domains": 5},
			},
			"items": []any{map[string]any{"keyword": "hotel in lonavala maharashtra", "keyword_info": map[string]any{"search_volume": 18100}}},
		})
	})
	items, err := provider.LabsKeywords(t.Context(), "org", LabsRequest{Source: SourceSuggestions, Seed: "hotel in lonavala", LocationCode: 2356, LanguageCode: "en", Limit: 150})
	if err != nil {
		t.Fatal(err)
	}
	rows := labsRows(items)
	if len(rows) != 2 || rows[0].Keyword != "hotel in lonavala" || rows[0].SearchVolume == nil || *rows[0].SearchVolume != 0 {
		t.Fatalf("paid seed data was dropped or replaced with suggestion metrics: %+v", rows)
	}
	seed := rows[0]
	if seed.CPC == nil || *seed.CPC != 0.26 || seed.KeywordDifficulty == nil || *seed.KeywordDifficulty != 0 ||
		seed.TopOfPageBidHigh == nil || *seed.TopOfPageBidHigh != 0.32 || seed.ReferringDomains == nil || *seed.ReferringDomains != 5 ||
		seed.QuarterlyChange == nil || *seed.QuarterlyChange != -33 || len(seed.Trend) != 1 || seed.Intent != "commercial" {
		t.Fatalf("seed metrics did not reach the API row: %+v", seed)
	}
	if len(ps.paths) != 1 {
		t.Fatalf("seed delivery bought %d tasks; want the original suggestions task only", len(ps.paths))
	}
}

func TestPaidClickstreamZeroDoesNotFallBackToNationalVolume(t *testing.T) {
	rows := labsRows([]LabsItem{{Keyword: "hotel", Info: Info{SearchVolume: intp(100)}, Clickstream: &Info{SearchVolume: intp(0)}}})
	if len(rows) != 1 || rows[0].SearchVolume == nil || *rows[0].SearchVolume != 0 {
		t.Fatalf("measured clickstream zero replaced by national volume: %+v", rows)
	}
}

func TestLabsKeywordsRequestsAndParsing(t *testing.T) {
	item := func(keyword string, volume int) map[string]any {
		return map[string]any{"keyword": keyword,
			"keyword_info": map[string]any{"search_volume": volume, "cpc": 1.5, "competition": 0.2, "monthly_searches": []any{map[string]any{"year": 2026, "month": 9, "search_volume": volume}, map[string]any{"year": nil}},
				"low_top_of_page_bid": 0.5, "high_top_of_page_bid": 2.25, "search_volume_trend": map[string]any{"quarterly": -45}},
			"keyword_info_normalized_with_clickstream": map[string]any{"search_volume": volume * 2},
			"keyword_properties":                       map[string]any{"keyword_difficulty": 33},
			"avg_backlinks_info":                       map[string]any{"referring_main_domains": 761.3},
			"search_intent_info":                       map[string]any{"main_intent": "commercial"}}
	}
	ps, p := newProviderServer(t, func(path string) map[string]any {
		if strings.HasSuffix(path, "related_keywords/live") {
			return okTask(map[string]any{"items": []any{map[string]any{"keyword_data": item("wrapped", 7)}, map[string]any{"keyword_data": nil}}})
		}
		return okTask(map[string]any{"items": []any{item("alpha", 10), map[string]any{"keyword": nil}, map[string]any{"keyword": ""}}})
	})
	ctx := context.Background()
	req := LabsRequest{Seed: "seed", LocationCode: 2840, LanguageCode: "en", Limit: 150, Clickstream: true, IgnoreSynonyms: true}

	for _, c := range []struct {
		source LabsSource
		path   string
	}{{SourceSuggestions, labsSuggestionsPath}, {SourceIdeas, labsIdeasPath}} {
		source, wantPath := c.source, c.path
		req.Source = source
		items, err := p.LabsKeywords(ctx, "org", req)
		if err != nil || len(items) != 1 || items[0].Keyword != "alpha" {
			t.Fatalf("%s = %+v, %v", source, items, err)
		}
		got := items[0]
		if *got.Info.SearchVolume != 10 || *got.Clickstream.SearchVolume != 20 || *got.Difficulty != 33 || *got.MainIntent != "commercial" ||
			len(got.Info.Monthly) != 2 || got.Info.Monthly[1].Year != 0 {
			t.Errorf("%s item = %+v", source, got)
		}
		// The same call also returns these; they must reach the row, not be dropped.
		if *got.Info.TopBidLow != 0.5 || *got.Info.TopBidHigh != 2.25 || *got.Info.QuarterlyChange != -45 || *got.ReferringDomains != 761.3 {
			t.Errorf("%s paid fields = bid %v-%v, quarterly %v, referring domains %v", source, got.Info.TopBidLow, got.Info.TopBidHigh, got.Info.QuarterlyChange, got.ReferringDomains)
		}
		row := labsRows([]LabsItem{got})[0]
		if *row.TopOfPageBidHigh != 2.25 || *row.ReferringDomains != 761 || *row.QuarterlyChange != -45 {
			t.Errorf("%s row = %+v, want the bid range, rounded referring domains and quarterly change", source, row)
		}
		if last := ps.paths[len(ps.paths)-1]; last != wantPath {
			t.Errorf("%s path = %s, want %s", source, last, wantPath)
		}
	}
	suggestions, ideas := ps.bodies[0], ps.bodies[1]
	if suggestions["keyword"] != "seed" || suggestions["include_seed_keyword"] != true || suggestions["exact_match"] != false ||
		suggestions["include_clickstream_data"] != true || suggestions["ignore_synonyms"] != true || suggestions["limit"] != float64(150) {
		t.Errorf("suggestions request = %v", suggestions)
	}
	if kws, _ := ideas["keywords"].([]any); len(kws) != 1 || kws[0] != "seed" || ideas["closely_variants"] != false {
		t.Errorf("ideas request = %v", ideas)
	}

	req.Source = SourceRelated
	related, err := p.LabsKeywords(ctx, "org", req)
	if err != nil || len(related) != 1 || related[0].Keyword != "wrapped" {
		t.Fatalf("related = %+v, %v; want the nested keyword_data unwrapped and empty wrappers skipped", related, err)
	}
	if ps.bodies[2]["depth"] != float64(3) {
		t.Errorf("related request = %v", ps.bodies[2])
	}
	if _, err := p.LabsKeywords(ctx, "org", LabsRequest{Source: "bogus"}); err == nil {
		t.Error("an unknown source was accepted")
	}

	overview, err := p.LabsOverview(ctx, "org", []string{"alpha"}, 2840, "en", false)
	if err != nil || len(overview) != 1 || ps.paths[len(ps.paths)-1] != labsOverviewPath {
		t.Errorf("overview = %+v, %v, path %s", overview, err, ps.paths[len(ps.paths)-1])
	}
}

func TestAdsRequestsAndParsing(t *testing.T) {
	ps, p := newProviderServer(t, func(_ string) map[string]any {
		rows := []any{}
		for i := range 5 {
			rows = append(rows, map[string]any{"keyword": fmt.Sprintf("idea %d", i), "search_volume": 100 - i, "competition_index": 40.0, "cpc": 0.5})
		}
		return okTask(rows...)
	})
	ctx := context.Background()
	ideas, err := p.AdsIdeas(ctx, "org", "seed", 2356, "hi", 3)
	if err != nil || len(ideas) != 3 || *ideas[0].SearchVolume != 100 || *ideas[0].CompetitionIndex != 40 {
		t.Fatalf("AdsIdeas() = %+v, %v; want the first 3 of the 5 returned", ideas, err)
	}
	if ps.paths[0] != adsIdeasPath || ps.bodies[0]["sort_by"] != "search_volume" {
		t.Errorf("ideas request = %s %v", ps.paths[0], ps.bodies[0])
	}

	city := "Pune,Maharashtra,India"
	if _, err := p.AdsVolume(ctx, "org", AdsVolumeRequest{Keywords: []string{"ok one", "bad, one", "ok two"}, LocationCode: 2356, LocationName: &city, LanguageCode: "en"}); err != nil {
		t.Fatal(err)
	}
	body := ps.bodies[1]
	if kws := body["keywords"].([]any); len(kws) != 2 || body["location_name"] != city || body["location_code"] != nil {
		t.Errorf("volume request = %v, want the rejected keyword left out and the city used", body)
	}
	before := len(ps.paths)
	if items, err := p.AdsVolume(ctx, "org", AdsVolumeRequest{Keywords: []string{"bad, one"}, LocationCode: 2356, LanguageCode: "en"}); err != nil || items != nil || len(ps.paths) != before {
		t.Errorf("only rejected keywords: items %v, err %v, calls %d -> %d; want no provider call", items, err, before, len(ps.paths))
	}
	if _, err := p.AdsVolume(ctx, "org", AdsVolumeRequest{Keywords: []string{"fine"}, LocationCode: 2356, LanguageCode: "en"}); err != nil {
		t.Fatal(err)
	}
	if ps.bodies[2]["location_code"] != float64(2356) || ps.bodies[2]["location_name"] != nil {
		t.Errorf("national volume request = %v", ps.bodies[2])
	}
}

func TestSerpLive(t *testing.T) {
	status := 20000
	ps, p := newProviderServer(t, func(string) map[string]any {
		task := okTask(map[string]any{"items": []any{
			map[string]any{"type": "organic", "rank_group": 1, "rank_absolute": 2, "domain": "a.com", "title": "A", "url": "https://a.com", "description": "d", "etv": 12.5,
				"estimated_paid_traffic_cost": 3.0, "backlinks_info": map[string]any{"referring_domains": 10, "backlinks": 40}},
			map[string]any{"type": "people_also_ask"},
		}})
		task["status_code"] = status
		return task
	})
	ctx := context.Background()
	city := "Pune,Maharashtra,India"
	items, err := p.SerpLive(ctx, "org", SerpRequest{Keyword: "seo", LocationCode: 2356, LanguageCode: "en", LocationName: &city, Depth: 20})
	if err != nil || len(items) != 2 {
		t.Fatalf("SerpLive() = %+v, %v", items, err)
	}
	organic := items[0]
	if organic.Type != "organic" || *organic.RankGroup != 1 || organic.Domain != "a.com" || *organic.Etv != 12.5 || *organic.Backlinks != 40 || *organic.ReferringDomains != 10 {
		t.Errorf("organic item = %+v", organic)
	}
	if b := ps.bodies[0]; b["location_name"] != city || b["location_code"] != nil || b["depth"] != float64(20) || b["device"] != "desktop" {
		t.Errorf("request = %v", b)
	}
	status = 40102
	if items, err := p.SerpLive(ctx, "org", SerpRequest{Keyword: "seo", LocationCode: 2840, LanguageCode: "en", Depth: 20}); err != nil || len(items) != 0 {
		t.Errorf("no-results SERP = %+v, %v; want an empty list, not an error", items, err)
	}
	// "Invalid Field" (40501) is our own bad request: it must surface as an error,
	// never be treated as an empty SERP.
	status = 40501
	if _, err := p.SerpLive(ctx, "org", SerpRequest{Keyword: "seo", LocationCode: 2840, LanguageCode: "en", Depth: 20}); err == nil {
		t.Error("an invalid-field response was accepted as an empty SERP")
	}
	before := len(ps.paths)
	for _, depth := range []int{0, 15, 110} {
		if _, err := p.SerpLive(ctx, "org", SerpRequest{Keyword: "seo", LocationCode: 2840, LanguageCode: "en", Depth: depth}); err == nil {
			t.Errorf("depth %d was accepted", depth)
		}
	}
	if len(ps.paths) != before {
		t.Error("an invalid depth reached the provider, where it would be billed")
	}
}

func TestLabsPlaceFilterAndSerpFeaturesReachTheProvider(t *testing.T) {
	ps, provider := newProviderServer(t, func(path string) map[string]any {
		row := map[string]any{"keyword": "resorts in lonavala", "keyword_info": map[string]any{"search_volume": 5400},
			"serp_info": map[string]any{"serp_item_types": []any{"ai_overview", "hotels_pack", "organic"}}}
		if strings.Contains(path, "related_keywords") {
			row = map[string]any{"keyword_data": row}
		}
		return okTask(map[string]any{"items": []any{row}})
	})
	for _, tt := range []struct {
		source LabsSource
		prefix string
		place  bool // suggestions are phrase matches and never place-filtered
	}{
		{SourceIdeas, "", true},
		{SourceRelated, "keyword_data.", true},
		{SourceSuggestions, "", false},
	} {
		items, err := provider.LabsKeywords(t.Context(), "org", LabsRequest{Source: tt.source, Seed: "best resort in lonavala", LocationCode: 2356, LanguageCode: "en", Limit: 75, MustContain: "lonavala"})
		if err != nil {
			t.Fatal(err)
		}
		body := ps.bodies[len(ps.bodies)-1]
		if body["include_serp_info"] != true {
			t.Errorf("%s: include_serp_info = %v, want true (free with the task)", tt.source, body["include_serp_info"])
		}
		volume := fmt.Sprint([]any{tt.prefix + "keyword_info.search_volume", ">", 0})
		want := volume
		if tt.place {
			want = fmt.Sprint([]any{[]any{tt.prefix + "keyword", "ilike", "%lonavala%"}, "and", []any{tt.prefix + "keyword_info.search_volume", ">", 0}})
		}
		if got := fmt.Sprint(body["filters"]); got != want {
			t.Errorf("%s: filters = %s, want %s", tt.source, got, want)
		}
		rows := labsRows(items)
		if len(rows) != 1 || !slices.Equal(rows[0].SerpFeatures, []string{"ai_overview", "hotels_pack", "organic"}) {
			t.Errorf("%s: rows = %+v, want the SERP features the task returned", tt.source, rows)
		}
	}
	if len(ps.paths) != 3 {
		t.Errorf("provider calls = %d, want exactly one per request", len(ps.paths))
	}

	// No place: no filter at all, and a row without serp_info still has a list.
	if _, err := provider.LabsKeywords(t.Context(), "org", LabsRequest{Source: SourceIdeas, Seed: "seo tool", LocationCode: 2840, LanguageCode: "en", Limit: 75}); err != nil {
		t.Fatal(err)
	}
	if got, want := fmt.Sprint(ps.bodies[len(ps.bodies)-1]["filters"]), fmt.Sprint([]any{"keyword_info.search_volume", ">", 0}); got != want {
		t.Errorf("a seed without a place filters = %s, want only the zero-volume filter %s", got, want)
	}
	if rows := labsRows([]LabsItem{{Keyword: "x"}}); rows[0].SerpFeatures == nil {
		t.Error("SerpFeatures must be an empty list, not null, when Labs sent none")
	}
}

func TestLabsItemPreservesPaidBenchmarkAndContextFields(t *testing.T) {
	ps, provider := newProviderServer(t, func(_ string) map[string]any {
		return okTask(map[string]any{
			"items": []any{map[string]any{
				"keyword": "best resort in lonavala",
				"keyword_info": map[string]any{
					"search_volume": 18100, "cpc": 0.26, "competition": 0.98, "competition_level": "LOW",
					"low_top_of_page_bid": 0.05, "high_top_of_page_bid": 0.32,
					"search_volume_trend": map[string]any{"monthly": -33, "quarterly": -33, "yearly": 22},
					"last_updated_time":   "2026-10-17 20:42:27 +00:00",
				},
				"keyword_properties": map[string]any{"keyword_difficulty": 51, "detected_language": "en", "words_count": 3},
				"search_intent_info": map[string]any{"main_intent": "transactional", "foreign_intent": []any{"informational", "commercial"}},
				"avg_backlinks_info": map[string]any{
					"referring_main_domains": 159.4, "backlinks": 641.4, "dofollow": 448.3, "referring_pages": 561.3,
				},
			}},
		})
	})
	items, err := provider.LabsKeywords(t.Context(), "org", LabsRequest{Source: SourceIdeas, Seed: "resort", LocationCode: 2840, LanguageCode: "en", Limit: 150})
	if err != nil || len(items) != 1 {
		t.Fatalf("LabsKeywords = %+v, %v", items, err)
	}
	row := labsRows(items)[0]
	if row.Backlinks == nil || *row.Backlinks != 641 || row.DofollowBacklinks == nil || *row.DofollowBacklinks != 448 ||
		row.ReferringPages == nil || *row.ReferringPages != 561 || row.MonthlyChange == nil || *row.MonthlyChange != -33 ||
		row.YearlyChange == nil || *row.YearlyChange != 22 || row.CompetitionLevel == nil || *row.CompetitionLevel != "LOW" ||
		row.LastUpdatedTime == nil || *row.LastUpdatedTime != "2026-10-17 20:42:27 +00:00" ||
		row.DetectedLanguage == nil || *row.DetectedLanguage != "en" || row.WordsCount == nil || *row.WordsCount != 3 ||
		len(row.SecondaryIntents) != 2 || row.SecondaryIntents[0] != "informational" || row.SecondaryIntents[1] != "commercial" {
		t.Fatalf("paid benchmark/context fields dropped or misread: %+v", row)
	}
	if len(ps.paths) != 1 {
		t.Fatalf("bought %d tasks; want one", len(ps.paths))
	}
}

func TestForeignIntentsReadsStringAndList(t *testing.T) {
	if got := foreignIntents(json.RawMessage(`"commercial"`)); len(got) != 1 || got[0] != "commercial" {
		t.Fatalf("single foreign intent = %v", got)
	}
	if got := foreignIntents(json.RawMessage(`["informational","commercial"]`)); len(got) != 2 || got[1] != "commercial" {
		t.Fatalf("list foreign intent = %v", got)
	}
	if got := foreignIntents(json.RawMessage(`null`)); got != nil {
		t.Fatalf("null foreign intent = %v", got)
	}
}

func TestAdsRowsPreserveTopBidsAndCompetitionLevel(t *testing.T) {
	rows := adsRows([]AdsItem{{Keyword: "hotel", SearchVolume: intp(100), Competition: strp("LOW"),
		TopBidLow: floatp(0.1), TopBidHigh: floatp(0.9), CompetitionIndex: floatp(40)}})
	if len(rows) != 1 || rows[0].TopOfPageBidLow == nil || *rows[0].TopOfPageBidLow != 0.1 ||
		rows[0].TopOfPageBidHigh == nil || *rows[0].TopOfPageBidHigh != 0.9 ||
		rows[0].CompetitionLevel == nil || *rows[0].CompetitionLevel != "LOW" {
		t.Fatalf("Ads bids or competition level dropped: %+v", rows[0])
	}
}

func TestLabsCoreKeywordAndLanguageReachRows(t *testing.T) {
	_, provider := newProviderServer(t, func(string) map[string]any {
		return okTask(map[string]any{"items": []any{
			map[string]any{"keyword": "revenue management hotel", "keyword_info": map[string]any{"search_volume": 390},
				"keyword_properties": map[string]any{"core_keyword": "hotel revenue management"},
				"avg_backlinks_info": map[string]any{"main_domain_rank": 402.7}},
			map[string]any{"keyword": "deloitte revenue", "keyword_info": map[string]any{"search_volume": 1000},
				"keyword_properties": map[string]any{"is_another_language": true}},
		}})
	})
	items, err := provider.LabsKeywords(t.Context(), "org", LabsRequest{Source: SourceIdeas, Seed: "hotel revenue management", LocationCode: 2356, LanguageCode: "en", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	rows := labsRows(items)
	if len(rows) != 1 || rows[0].Keyword != "revenue management hotel" {
		t.Fatalf("rows = %+v, want only the keyword in the requested language", rows)
	}
	if rows[0].CoreKeyword == nil || *rows[0].CoreKeyword != "hotel revenue management" {
		t.Errorf("CoreKeyword = %v, want the Labs variant group", rows[0].CoreKeyword)
	}
	if rows[0].TopPagesDomainRank == nil || *rows[0].TopPagesDomainRank != 403 {
		t.Errorf("TopPagesDomainRank = %v, want 403", rows[0].TopPagesDomainRank)
	}
}
