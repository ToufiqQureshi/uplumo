package sam

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/billing"
	"github.com/toufiqqureshi/uplumo/backend/internal/mcp"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/openrouter"
)

func keywordsToolset(t *testing.T, runner *fakeRunner) *toolset {
	t.Helper()
	ts := testToolset(t, runner)
	ts.restrictToKeywords()
	return ts
}

func callCount(r *fakeRunner, name string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, c := range r.calls {
		if c.name == name {
			n++
		}
	}
	return n
}

func TestKeywordsModeOffersOnlyKeywordTools(t *testing.T) {
	ts := keywordsToolset(t, newFakeRunner())
	offered := map[string]bool{}
	for _, def := range ts.Definitions() {
		offered[def.Name] = true
	}
	for _, name := range []string{"research_keywords", "get_serp_results", "get_search_console_performance", "present_keywords"} {
		if !offered[name] {
			t.Errorf("Keywords mode does not offer %s", name)
		}
	}
	for _, name := range []string{"get_domain_overview", "run_site_audit", "get_backlinks_profile", "activate_skill", "update_project_context"} {
		if offered[name] {
			t.Errorf("Keywords mode offers %s", name)
		}
		if _, isError, err := ts.Run(t.Context(), name, json.RawMessage(`{}`)); err == nil || !isError {
			t.Errorf("Keywords mode executed blocked tool %s", name)
		}
	}
}

func TestKeywordsModeCapsPaidCallsPerReply(t *testing.T) {
	runner := newFakeRunner()
	ts := keywordsToolset(t, runner)
	run := func(name, args string) bool {
		_, isError, err := ts.Run(t.Context(), name, json.RawMessage(args))
		return err == nil && !isError
	}
	if !run("research_keywords", `{"seeds":[{"seed":"a"},{"seed":"b"}]}`) {
		t.Fatal("two seeds were refused")
	}
	if run("research_keywords", `{"seeds":[{"seed":"c"},{"seed":"d"}]}`) {
		t.Error("a fourth seed was bought")
	}
	if !run("research_keywords", `{"seeds":[{"seed":"c"}]}`) {
		t.Error("the third seed was refused")
	}
	if got := callCount(runner, "research_keywords"); got != 2 {
		t.Errorf("research calls sent = %d, want 2 (refused calls never reach the provider)", got)
	}

	if run("get_serp_results", `{"queries":[{"keyword":"a"}],"depth":30}`) {
		t.Error("a SERP deeper than the limit was bought")
	}
	if run("get_serp_results", `{"queries":[{"keyword":"a"},{"keyword":"b"},{"keyword":"c"}]}`) {
		t.Error("three SERP queries were bought")
	}
	if !run("get_serp_results", `{"queries":[{"keyword":"a"},{"keyword":"b"}],"depth":10}`) {
		t.Error("two SERP queries were refused")
	}
	if got := callCount(runner, "get_serp_results"); got != 1 {
		t.Errorf("SERP calls sent = %d, want 1", got)
	}

	if !run("get_keyword_metrics", `{"keywords":["a"]}`) || run("get_keyword_metrics", `{"keywords":["b"]}`) {
		t.Error("metric lookups are not limited to one per reply")
	}
}

func TestPresentKeywordsUsesBoughtNumbersOnly(t *testing.T) {
	runner := newFakeRunner()
	runner.results["research_keywords"] = []mcp.ToolResult{{Text: "ok", Structured: map[string]any{"results": []any{
		map[string]any{"rows": []any{map[string]any{"keyword": "Hotel Revenue Management", "searchVolume": 390}}},
	}}}}
	ts := keywordsToolset(t, runner)
	if _, isError, err := ts.Run(t.Context(), "research_keywords", json.RawMessage(`{"seeds":[{"seed":"hotel revenue management"}]}`)); err != nil || isError {
		t.Fatalf("research failed: %v", err)
	}
	out, isError, err := ts.Run(t.Context(), "present_keywords", json.RawMessage(`{"understood":[{"label":"Market","value":"India"}],"summary":"s",
		"keywords":[{"keyword":"hotel revenue management","chance":"high"},{"keyword":"made up keyword","chance":"low"}]}`))
	if err != nil || isError {
		t.Fatalf("present failed: %v", err)
	}
	got := out.(map[string]any)
	rows := got["keywords"].([]map[string]any)
	metrics, ok := rows[0]["metrics"].(map[string]any)
	if !ok || metrics["searchVolume"] != float64(390) {
		t.Errorf("first row metrics = %v, want the researched volume", rows[0]["metrics"])
	}
	if rows[1]["metrics"] != nil {
		t.Errorf("an unresearched keyword got metrics %v", rows[1]["metrics"])
	}
	if missing := got["missingMetrics"].([]string); len(missing) != 1 || missing[0] != "made up keyword" {
		t.Errorf("missingMetrics = %v", missing)
	}
	if _, isError, err := ts.Run(t.Context(), "present_keywords", json.RawMessage(`{"understood":[],"summary":"s","keywords":[]}`)); err == nil && !isError {
		t.Error("an empty presentation was accepted")
	}
}

func TestKeywordsModeRefusesDearerResearchOptions(t *testing.T) {
	runner := newFakeRunner()
	ts := keywordsToolset(t, runner)
	for _, args := range []string{
		`{"seeds":[{"seed":"a"}],"resultLimit":500}`,
		`{"seeds":[{"seed":"a"}],"includeClickstreamData":true}`,
	} {
		if _, isError, err := ts.Run(t.Context(), "research_keywords", json.RawMessage(args)); err == nil && !isError {
			t.Errorf("%s was bought; it costs more than the shown limit allows", args)
		}
	}
	if _, isError, err := ts.Run(t.Context(), "get_keyword_metrics", json.RawMessage(`{"keywords":["a"],"includeClickstreamData":true}`)); err == nil && !isError {
		t.Error("clickstream metrics were bought")
	}
	if got := callCount(runner, "research_keywords") + callCount(runner, "get_keyword_metrics"); got != 0 {
		t.Errorf("provider calls = %d, want 0: refused options must never be sent", got)
	}
	if ts.keywords.seeds != 0 || ts.keywords.dataCredits != 0 {
		t.Errorf("refused calls were counted: %+v", ts.keywords)
	}
	if !runOK(t, ts, "research_keywords", `{"seeds":[{"seed":"a"}],"resultLimit":150}`) {
		t.Error("the default research size was refused")
	}
}

func TestKeywordsReplyNeverPassesItsShownCreditLimit(t *testing.T) {
	// Every paid call at its worst case still leaves room for a full model
	// step, so a normal reply can finish inside the limit.
	data := maxLocalSeedsPerTurn*localSeedMaxCredits + (maxKeywordSeedsPerTurn-maxLocalSeedsPerTurn)*seedMaxCredits + maxSerpQueriesPerTurn*serpQueryMaxCredits + metricsMaxCredits
	step, err := openrouter.MaxCostUSD(openrouter.Request{MaxTokens: keywordsMaxOutputTokens,
		Messages: []openrouter.Message{{Content: strings.Repeat("x", 4*maxToolOutputChars)}}})
	if err != nil || data+billing.CreditsForUSD(step) >= KeywordsReplyMaxCredits {
		t.Fatalf("worst-case data %d + one step %d credits does not fit the %d-credit limit (err %v)", data, billing.CreditsForUSD(step), KeywordsReplyMaxCredits, err)
	}

	runner := newFakeRunner()
	ts := keywordsToolset(t, runner)
	// The model steps already charged leave no room for another paid lookup.
	ts.keywords.aiCredits = KeywordsReplyMaxCredits - seedMaxCredits + 1
	if runOK(t, ts, "research_keywords", `{"seeds":[{"seed":"a"}]}`) || callCount(runner, "research_keywords") != 0 {
		t.Fatal("a lookup that would pass the shown limit was sent")
	}
	ts.keywords.aiCredits = KeywordsReplyMaxCredits - seedMaxCredits
	if !runOK(t, ts, "research_keywords", `{"seeds":[{"seed":"a"}]}`) {
		t.Fatal("a lookup that fits exactly was refused")
	}
	if total := ts.keywords.aiCredits + ts.keywords.dataCredits; total != KeywordsReplyMaxCredits {
		t.Fatalf("reserved %d credits, want exactly the limit %d", total, KeywordsReplyMaxCredits)
	}
}

func TestKeywordsLocalSeedReservesItsGoogleAdsVolumeCall(t *testing.T) {
	// A city or region seed buys the national research and then one Google
	// Ads search volume call for the place; both must be inside the reserve.
	ads := billing.CreditsForUSD(dataforseo.EstimateUSD("/v3/keywords_data/google_ads/search_volume/live", []byte(`[{}]`)))
	if localSeedMaxCredits < seedMaxCredits+ads {
		t.Fatalf("a local seed reserves %d credits, want at least %d research + %d Ads volume", localSeedMaxCredits, seedMaxCredits, ads)
	}
	runner := newFakeRunner()
	ts := keywordsToolset(t, runner)
	if !runOK(t, ts, "research_keywords", `{"seeds":[{"seed":"a"},{"seed":"b","locationName":"Mumbai,Maharashtra,India"}]}`) {
		t.Fatal("a national and a local seed were refused")
	}
	if want := seedMaxCredits + localSeedMaxCredits; ts.keywords.dataCredits != want {
		t.Fatalf("reserved %d credits, want %d", ts.keywords.dataCredits, want)
	}
	if runOK(t, ts, "research_keywords", `{"seeds":[{"seed":"c","locationName":"Pune,Maharashtra,India"}]}`) || callCount(runner, "research_keywords") != 1 {
		t.Fatal("a second local seed was bought")
	}
}

func TestKeywordsOperatorQueryReservesItsFiveTimesPrice(t *testing.T) {
	// Google SERP bills a query with an operator such as site: at five times
	// the plain price; the reserve must book what DataForSEO will charge.
	runner := newFakeRunner()
	ts := keywordsToolset(t, runner)
	plain := billing.CreditsForUSD(dataforseo.SerpCostUSD(maxSerpDepthPerTurn, false))
	operator := billing.CreditsForUSD(dataforseo.EstimateUSD("/v3/serp/google/organic/live/advanced", []byte(`[{"keyword":"site:uplumo.com hotel","depth":20}]`)))
	if operator <= plain {
		t.Fatalf("operator query priced %d, plain %d: the price list lost its operator factor", operator, plain)
	}
	if !runOK(t, ts, "get_serp_results", `{"queries":[{"keyword":"hotel"},{"keyword":"site:uplumo.com hotel"}],"depth":20}`) {
		t.Fatal("two SERP queries were refused")
	}
	if want := plain + operator; ts.keywords.dataCredits != want {
		t.Fatalf("reserved %d credits, want %d", ts.keywords.dataCredits, want)
	}
	if serpQueryMaxCredits < operator {
		t.Fatalf("the worst-case SERP query is %d credits, below an operator query's %d", serpQueryMaxCredits, operator)
	}
}

func TestKeywordsPageShowsTheBackendCreditLimit(t *testing.T) {
	page, err := os.ReadFile("../../../frontend/src/client/features/keywords/agent/KeywordAgentPage.tsx")
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("const MAX_REPLY_CREDITS = %d;", KeywordsReplyMaxCredits)
	if !strings.Contains(string(page), want) {
		t.Fatalf("the Keywords page must show the limit the server enforces: want %q in KeywordAgentPage.tsx", want)
	}
}

func runOK(t *testing.T, ts *toolset, name, args string) bool {
	t.Helper()
	_, isError, err := ts.Run(t.Context(), name, json.RawMessage(args))
	return err == nil && !isError
}
