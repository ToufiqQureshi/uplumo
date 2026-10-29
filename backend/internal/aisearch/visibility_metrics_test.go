package aisearch

import (
	"context"
	"crypto/rand"
	"slices"
	"testing"
	"time"
)

func TestVisibilityHistoryReadsSavedScopedRuns(t *testing.T) {
	db := trackingDB(t)
	ctx := t.Context()
	org, project := "visibility-org-"+rand.Text(), "visibility-project-"+rand.Text()
	if _, err := db.Exec(ctx, `INSERT INTO organization(id,name,slug,created_at) VALUES($1,'Visibility',$1,now())`, org); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec(context.Background(), `DELETE FROM organization WHERE id=$1`, org); err != nil {
			t.Error(err)
		}
	})
	if _, err := db.Exec(ctx, `INSERT INTO projects(id,organization_id,name) VALUES($1,$2,'Visibility')`, project, org); err != nil {
		t.Fatal(err)
	}
	config := `{"domain":"shop.example"}`
	if _, err := db.Exec(ctx, `INSERT INTO go_ai_trackers(project_id,organization_id,configuration) VALUES($1,$2,$3)`, project, org, config); err != nil {
		t.Fatal(err)
	}
	old := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, scope := range []string{"same", "same", "other"} {
		id := project + string(rune('a'+i))
		at := old.Add(time.Duration(i) * time.Hour)
		if _, err := db.Exec(ctx, `INSERT INTO go_ai_tracking_runs(id,project_id,organization_id,configuration,scope_key,trigger,status,created_at,request_key) VALUES($1,$2,$3,$4,$5,'manual','completed',$6,$1)`, id, project, org, config, scope, at); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(ctx, `INSERT INTO go_ai_tracking_answers(id,run_id,prompt,topic,engine,status,answer,citations,collected_at) VALUES($1,$2,'कहाँ?','general','chat_gpt','completed','See https://shop.example/a',$3,$4)`, id+"-answer", id, `[{"url":"https://shop.example/a"}]`, at); err != nil {
			t.Fatal(err)
		}
	}
	latest := project + "b"
	metrics, err := LoadVisibilityMetrics(ctx, db, org, project, latest)
	if err != nil || metrics == nil || metrics.Runs != 2 || len(metrics.Pages) != 1 || !metrics.Pages[0].FirstSeen.Equal(old) || !metrics.Pages[0].LastSeen.Equal(old.Add(time.Hour)) {
		t.Fatalf("scoped metrics = %+v, %v", metrics, err)
	}
	if metrics.Prompts[0].AverageAnswerRank == nil || *metrics.Prompts[0].AverageAnswerRank != 1 {
		t.Fatalf("answer rank = %+v", metrics.Prompts)
	}
	if foreign, err := LoadVisibilityMetrics(ctx, db, "foreign-workspace", project, latest); err != nil || foreign != nil {
		t.Fatalf("foreign workspace saw %+v, %v", foreign, err)
	}
}

func visibilityAnswerFor(prompt string, when time.Time, urls ...string) visibilityAnswer {
	out := visibilityAnswer{prompt: prompt, engine: "chat_gpt", status: "completed", text: "An answer", at: when}
	for _, u := range urls {
		out.citations = append(out.citations, Citation{URL: u})
	}
	return out
}

func TestVisibilityMetricsFromSavedComparableAnswers(t *testing.T) {
	old := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	latest := old.Add(24 * time.Hour)
	runs := []visibilityRun{
		{id: "new", status: "completed", answers: []visibilityAnswer{
			visibilityAnswerFor("Where?", latest, "https://shop.example/a", "https://rival.example/"),
			visibilityAnswerFor("कीमत क्या है?", latest, "https://blog.shop.example/price"),
		}},
		{id: "old", status: "completed", answers: []visibilityAnswer{
			visibilityAnswerFor("Where?", old, "https://rival.example/", "https://shop.example/a"),
			visibilityAnswerFor("कीमत क्या है?", old, "https://rival.example/"),
		}},
	}
	got := buildVisibilityMetrics("shop.example", runs)
	if got.Runs != 2 || got.HistoryLimited || len(got.Prompts) != 2 {
		t.Fatalf("metrics = %+v", got)
	}
	domainIndex := slices.IndexFunc(got.Domains, func(d VisibilitySource) bool { return d.Name == "shop.example" })
	if domainIndex < 0 {
		t.Fatalf("domains = %+v", got.Domains)
	}
	domain := got.Domains[domainIndex]
	if !domain.FirstSeen.Equal(old) || !domain.LastSeen.Equal(latest) || domain.AverageRank != 1.5 || domain.Momentum != "flat" || domain.LatestCount != 1 {
		t.Fatalf("domain = %+v", domain)
	}
	pageIndex := slices.IndexFunc(got.Pages, func(p VisibilitySource) bool { return p.Name == "https://blog.shop.example/price" })
	if pageIndex < 0 || got.Pages[pageIndex].Momentum != "up" {
		t.Fatalf("pages = %+v", got.Pages)
	}
	promptIndex := slices.IndexFunc(got.Prompts, func(p VisibilityPrompt) bool { return p.Prompt == "Where?" })
	if promptIndex < 0 {
		t.Fatalf("prompts = %+v", got.Prompts)
	}
	prompt := got.Prompts[promptIndex]
	if prompt.Answers != 2 || prompt.CitedAnswers != 2 || prompt.LatestRank == nil || *prompt.LatestRank != 1 || prompt.AverageRank == nil || *prompt.AverageRank != 1.5 {
		t.Fatalf("prompt = %+v", prompt)
	}
}

func TestVisibilityMetricsDoNotCompareFailedOrDifferentAnswers(t *testing.T) {
	today := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	current := visibilityRun{id: "new", status: "completed", answers: []visibilityAnswer{visibilityAnswerFor("A", today, "https://shop.example/")}}
	for _, tc := range []struct {
		name     string
		previous visibilityRun
		want     string
	}{
		{"fewer citations", visibilityRun{id: "old", status: "completed", answers: []visibilityAnswer{visibilityAnswerFor("A", today.Add(-24*time.Hour))}}, "up"},
		{"partial run", visibilityRun{id: "old", status: "partial", answers: []visibilityAnswer{visibilityAnswerFor("A", today.Add(-24*time.Hour))}}, "unavailable"},
		{"different prompt", visibilityRun{id: "old", status: "completed", answers: []visibilityAnswer{visibilityAnswerFor("B", today.Add(-24*time.Hour))}}, "unavailable"},
		{"empty answer", visibilityRun{id: "old", status: "completed", answers: []visibilityAnswer{{prompt: "A", engine: "chat_gpt", status: "completed"}}}, "unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := buildVisibilityMetrics("shop.example", []visibilityRun{current, tc.previous})
			if len(got.Domains) != 1 || got.Domains[0].Momentum != tc.want {
				t.Fatalf("domains = %+v, want %s", got.Domains, tc.want)
			}
		})
	}
}

func TestVisibilityMetricsRejectUnsafeURLsAndDoNotGuessAbsentRanks(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	answer := visibilityAnswerFor("A", now, "javascript:alert(1)", "https://evil.example/", "https://notshop.example/")
	got := buildVisibilityMetrics("shop.example", []visibilityRun{{id: "a", status: "completed", answers: []visibilityAnswer{answer}}})
	if len(got.Domains) != 2 || got.Prompts[0].CitedAnswers != 0 || got.Prompts[0].AverageRank != nil || got.Prompts[0].LatestRank != nil {
		t.Fatalf("metrics = %+v", got)
	}
}

func TestAnswerCitationRankUsesAnswerOrder(t *testing.T) {
	urls := []Citation{{URL: "https://shop.example/a"}, {URL: "https://rival.example/b"}}
	for _, tc := range []struct {
		name, text string
		want       int
	}{
		{"answer order differs", "[other](https://rival.example/b) then [ours](https://shop.example/a)", 2},
		{"our source first", "https://shop.example/a and https://rival.example/b", 1},
		{"source not in text", "No URLs in this answer.", 0},
		{"longer URL is not a match", "https://shop.example/another", 0},
		{"embedded in a query is not a match", "https://else.example/?redirect=https://shop.example/a", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := answerCitationRank(tc.text, "shop.example", urls); got != tc.want {
				t.Fatalf("rank = %d, want %d", got, tc.want)
			}
		})
	}
	old := visibilityAnswerFor("Where?", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), urls[0].URL, urls[1].URL)
	old.text = "https://shop.example/a first"
	latest := visibilityAnswerFor("Where?", old.at.Add(time.Hour), urls[0].URL, urls[1].URL)
	latest.text = "[other](https://rival.example/b) then [ours](https://shop.example/a)"
	got := buildVisibilityMetrics("shop.example", []visibilityRun{{id: "new", status: "completed", answers: []visibilityAnswer{latest}}, {id: "old", status: "completed", answers: []visibilityAnswer{old}}})
	prompt := got.Prompts[0]
	if prompt.LatestAnswerRank == nil || *prompt.LatestAnswerRank != 2 || prompt.AverageAnswerRank == nil || *prompt.AverageAnswerRank != 1.5 || prompt.InlineAnswers != 2 {
		t.Fatalf("inline ranks = %+v", prompt)
	}
}
