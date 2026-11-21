package sitereport

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/aisearch"
	"github.com/toufiqqureshi/uplumo/backend/internal/analytics"
	"github.com/toufiqqureshi/uplumo/backend/internal/localvisibility"
	"github.com/toufiqqureshi/uplumo/backend/internal/reports"
)

type evidenceRankSource struct {
	result *Rankings
	reads  int
}

func (s *evidenceRankSource) Rankings(context.Context, string, string, Period) (*Rankings, error) {
	s.reads++
	return s.result, nil
}

func TestGenerateIncludesDashboardEvidenceInSharedReport(t *testing.T) {
	saver := &fakeSaver{}
	service := newService(&fakeSearch{data: fullSearch()}, saver)
	rankings := fullSnapshot().Rankings
	rankings.Rows[0].Volume = new(12345)
	rankings.Rows[0].Competitors = []CompetitorRank{{Rank: 2, Domain: "rival.example", URL: "https://rival.example/ranking-page"}}
	ranks := &evidenceRankSource{result: rankings}
	service.Rankings = ranks

	traffic := fullSnapshot().Traffic
	traffic.Channels = []GA4Row{{Name: "Paid Search", Sessions: new(12.0), EngagementRate: new(0.5)}}
	traffic.LandingPages = []GA4Row{{Name: "/pricing", Sessions: new(123.0)}}
	traffic.Events = []GA4Row{{Name: "purchase", KeyEvents: new(48.0), PreviousKeyEvents: new(31.0)}}
	service.Traffic = fakeTraffic{data: traffic}
	service.Analytics = func(context.Context, string, Period) (*AnalyticsSnapshot, error) {
		return &AnalyticsSnapshot{
			Visitors: 120, PreviousVisitors: 99,
			LeadActions: []analytics.EventCount{{Name: "call_button_click", Count: 4}},
			AILandings:  []analytics.AILanding{{Source: "ChatGPT", Path: "/comparison", Visitors: 8}},
		}, nil
	}
	service.AI = func(context.Context, string, string) (*AIVisibility, error) {
		return testCitations(), nil
	}
	service.Local = func(context.Context, string, string, time.Time) (*LocalVisibility, error) {
		return &LocalVisibility{
			Business: &localvisibility.Business{Title: "Example Hotel", Address: "1 Market Street"},
			CheckAt:  fixedNow, GridAt: fixedNow, ReviewsAt: fixedNow, PreviousReviewsAt: fixedNow.AddDate(0, -1, 0),
			Check: &localvisibility.CheckResult{SearchTerm: "hotel near the station", YourRank: 2,
				Rivals: []localvisibility.Contender{{Place: localvisibility.Place{Title: "Other Hotel", Rank: 1, Rating: new(4.6)}}}},
			Grid:            &localvisibility.GridResult{Size: 3, SearchTerm: "hotel near the station", Points: []localvisibility.GridPoint{{Rank: 2}, {Missing: true}, {}}},
			Reviews:         &localvisibility.ReviewsResult{Total: 35, Rating: new(4.7)},
			PreviousReviews: &localvisibility.ReviewsResult{Total: 31, Rating: new(4.5)},
		}, nil
	}
	input := validInput()
	input.WorkCompleted = "Updated the comparison page <script>\nhttps://example.com/comparison"
	result, err := service.Generate(t.Context(), "org", "proj", Author{UserID: "u1", Name: "Editor"}, input)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(result.Included, AllSections) || len(result.Skipped) != 0 || ranks.reads != 1 ||
		len(saver.saved) != 1 || !slices.Equal(saver.shared, []string{result.ReportID}) {
		t.Fatalf("result = %+v, rank reads = %d, saved = %d, shared = %v", result, ranks.reads, len(saver.saved), saver.shared)
	}
	html := saver.saved[0].HTML
	for _, want := range []string{
		"Executive summary", "Work completed", "Updated the comparison page &lt;script&gt;",
		`href="https://example.com/comparison"`, "Organic key events up 55%", "Paid Search", "purchase", "/pricing",
		"call_button_click", "Visits referred by AI assistants", "12,345", "https://rival.example/ranking-page",
		"ChatGPT", "Where should I stay?", "https://example.com/cited", "https://rival.example/cited",
		"Project position #2", "Maps rank grid", "Other Hotel", ">4.6<", "4.7 average rating", "31 reviews",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("generated report is missing %q", want)
		}
	}
	for _, banned := range []string{"<script", `href="javascript:`, "%!f", "ZgotmplZ", "credits", "billing usage"} {
		if strings.Contains(html, banned) {
			t.Errorf("generated report contains %q", banned)
		}
	}
	if len(html) > reports.MaxHTMLBytes {
		t.Errorf("generated report is %d bytes", len(html))
	}
}

func testCitations() *AIVisibility {
	run := aisearch.TrackingRun{CreatedAt: fixedNow, Answered: 2, Cited: 1, Configuration: aisearch.TrackingConfig{
		Brand: "Example Hotel", Domain: "example.com", Competitors: []aisearch.TrackedBrand{{Name: "Other Hotel", Domain: "rival.example"}},
	}}
	return citationEvidence(run, []aisearch.TrackingAnswer{
		{Status: "completed", Engine: "chat_gpt", Prompt: "Where should I stay?", Answer: "Compare the hotels near the station.",
			Citations: []aisearch.Citation{{URL: "https://rival.example/cited"}, {URL: "javascript:alert(1)"}, {URL: "https://example.com/cited"}}},
		{Status: "completed", Engine: "gemini", Prompt: "Which is closest?", Answer: "The other hotel is close.", Citations: []aisearch.Citation{{URL: "https://rival.example/cited"}}},
		{Status: "failed", Engine: "gemini", Prompt: "A failed check", Answer: "A provider error."},
	})
}

func TestCitationEvidencePreservesSourceOrderAndRejectsUnsafeLinks(t *testing.T) {
	data := testCitations()
	if len(data.Evidence) != 2 {
		t.Fatalf("unexpected answer evidence: %+v", data.Evidence)
	}
	sources := data.Evidence[0].Sources
	if len(sources) != 2 || sources[0].Position != 1 || sources[0].Brand != "Other Hotel" ||
		sources[1].Position != 3 || !sources[1].Owned || sources[1].Brand != "Example Hotel" {
		t.Fatalf("unexpected citation attribution: %+v", sources)
	}
	if matchesDomain("fake-example.com", "example.com") || !matchesDomain("blog.example.com", "www.example.com") {
		t.Fatal("citation ownership must match a complete domain suffix")
	}
	snap := Snapshot{AI: data}
	steps := priorities(snap)
	if len(steps) != 1 || steps[0].Anchor != "ai-visibility" || !strings.Contains(steps[0].Detail, "Which is closest?") {
		t.Fatalf("unexpected evidence-backed priorities: %+v", steps)
	}
}

func TestLargeRankReportStaysWithinSharedHTMLLimit(t *testing.T) {
	rankings := &Rankings{Domain: "example.com", Device: "desktop", Total: 200}
	for range rankings.Total {
		row := RankRow{Keyword: "example search", Position: new(9), URL: "https://example.com/" + strings.Repeat("a", 500)}
		for i := range 5 {
			row.Competitors = append(row.Competitors, CompetitorRank{Rank: i + 1, Domain: "rival.example", URL: "https://rival.example/" + strings.Repeat("b", 500)})
		}
		rankings.Rows = append(rankings.Rows, row)
	}
	html, err := Render(Snapshot{Project: Project{Name: "Example Hotel"}, Rankings: rankings, Competitors: rankings})
	if err != nil {
		t.Fatal(err)
	}
	if len(html) > reports.MaxHTMLBytes || !strings.Contains(html, "of 200 keywords with competitor results") || !strings.Contains(html, "https://rival.example/") {
		t.Fatalf("rank report did not keep bounded, labelled evidence, got %d bytes", len(html))
	}
}
