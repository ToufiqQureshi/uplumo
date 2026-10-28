package sam

import (
	"encoding/json"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/mcp"
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
