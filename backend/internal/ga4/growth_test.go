package ga4

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/google"
	"github.com/toufiqqureshi/uplumo/backend/internal/gsc"
)

func growthFixture(t *testing.T) ([]gsc.SearchRow, []gsc.SearchRow, []gsc.SearchRow, normalizedReport) {
	t.Helper()
	current := []gsc.SearchRow{{Keys: []string{"https://example.test/winner"}, Clicks: 60, Impressions: 1000, CTR: .06, Position: 9}, {Keys: []string{"https://example.test/weak"}, Clicks: 60, Impressions: 1000, CTR: .06, Position: 9}, {Keys: []string{"https://example.test/other"}, Clicks: 30, Impressions: 300, CTR: .1, Position: 10}}
	previous := []gsc.SearchRow{{Keys: []string{"https://example.test/winner"}, Clicks: 100, Impressions: 1000, CTR: .1, Position: 9}}
	queries := []gsc.SearchRow{{Keys: []string{"Acme booking", "https://example.test/winner"}, Clicks: 3, Impressions: 100, CTR: .03, Position: 9}, {Keys: []string{"Acme booking", "https://example.test/weak"}, Clicks: 3, Impressions: 100, CTR: .03, Position: 9}}
	ga := normalizedReport{Rows: []map[string]any{
		{"hostName": "example.test", "landingPage": "/winner", "sessions": float64(200), "keyEvents": float64(20), "sessionKeyEventRate": .1, "engagementRate": .8},
		{"hostName": "example.test", "landingPage": "/weak", "sessions": float64(200), "keyEvents": float64(0), "sessionKeyEventRate": float64(0)},
		{"hostName": "example.test", "landingPage": "/other", "sessions": float64(200), "keyEvents": float64(10), "sessionKeyEventRate": .05},
	}, TotalRowCount: 3}
	return current, previous, queries, ga
}
func findGrowthPage(t *testing.T, report GrowthReport, suffix string) GrowthPage {
	t.Helper()
	for _, p := range report.Pages {
		if strings.HasSuffix(p.Page, suffix) {
			return p
		}
	}
	t.Fatalf("page %s not found", suffix)
	return GrowthPage{}
}
func findingKinds(page GrowthPage) []string {
	var out []string
	for _, f := range page.Findings {
		out = append(out, f.Kind)
	}
	return out
}

func TestGrowthFindingsKeepEvidenceAndSuppressRestrictedOutcomes(t *testing.T) {
	current, previous, queries, ga := growthFixture(t)
	got, err := buildGrowthReport(current, previous, queries, ga, ga, []string{"acme"})
	if err != nil {
		t.Fatal(err)
	}
	winner := findGrowthPage(t, got, "winner")
	for _, kind := range []string{"traffic_loss", "grow_winner", "ctr_decline", "competing_pages"} {
		if !slices.Contains(findingKinds(winner), kind) {
			t.Fatalf("missing %s: %+v", kind, winner)
		}
	}
	if !winner.Queries[0].Branded || len(winner.Queries[0].CompetingPages) != 1 {
		t.Fatalf("query evidence: %+v", winner.Queries)
	}
	if !slices.Contains(findingKinds(findGrowthPage(t, got, "weak")), "outcome_gap") {
		t.Fatal("outcome gap not identified")
	}
	revenueGA := ga
	revenueGA.Rows = append([]map[string]any{}, ga.Rows...)
	revenueGA.Rows[0] = map[string]any{"hostName": "example.test", "landingPage": "/winner", "transactions": float64(5), "purchaseRevenue": float64(100)}
	ga.Rows[0]["purchaseRevenue"] = float64(20)
	revenueReport, err := buildGrowthReport(current, previous, queries, ga, revenueGA, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(findingKinds(findGrowthPage(t, revenueReport, "winner")), "revenue_decline") {
		t.Fatal("recorded purchase revenue decline was missed")
	}
	ga.Metadata.HasLimitedData = true
	limited, err := buildGrowthReport(current, previous, queries, ga, ga, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, page := range limited.Pages {
		for _, kind := range findingKinds(page) {
			if kind == "grow_winner" || kind == "outcome_gap" || kind == "rate_decline" || kind == "revenue_decline" {
				t.Fatalf("restricted-data business recommendation: %+v", page)
			}
		}
	}
	// Missing and ambiguous Analytics rows remain null instead of measured zero.
	ga.Rows = append(ga.Rows, ga.Rows[0])
	ambiguous, err := buildGrowthReport(current, previous, queries, ga, ga, nil)
	if err != nil {
		t.Fatal(err)
	}
	if findGrowthPage(t, ambiguous, "winner").Current.Sessions != nil {
		t.Fatal("ambiguous join manufactured a metric")
	}
	variant := current[0]
	variant.Keys = []string{"https://example.test/winner/"}
	ambiguousSearch, err := growthSearchRows(append(append([]gsc.SearchRow{}, current...), variant))
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := ambiguousSearch["example.test/winner"]; exists {
		t.Fatal("ambiguous Search Console URLs created a combined baseline")
	}
	current[0].Impressions = math.NaN()
	if _, err := buildGrowthReport(current, previous, queries, ga, ga, nil); err == nil {
		t.Fatal("non-finite provider data accepted")
	}
}

func growthRepo(t *testing.T) *GrowthRepository {
	t.Helper()
	pool := setupRepositoryDB(t)
	migration, err := os.ReadFile("../database/migrations/00031_search_growth.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), strings.Split(string(migration), "-- +goose Down")[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `INSERT INTO organization VALUES ('org-a'),('org-b'); INSERT INTO projects VALUES ('project-a','org-a'),('project-b','org-b')`); err != nil {
		t.Fatal(err)
	}
	return &GrowthRepository{DB: pool}
}
func TestGrowthRepositoryScopesAndDeduplicatesBaselines(t *testing.T) {
	repo := growthRepo(t)
	current, previous, queries, ga := growthFixture(t)
	report, err := buildGrowthReport(current, previous, queries, ga, ga, nil)
	if err != nil {
		t.Fatal(err)
	}
	report.ID = "report-a"
	report.GeneratedAt = time.Now()
	report.Range = DateRange{StartDate: "2026-09-01", EndDate: "2026-09-28"}
	if err := repo.StoreReport(t.Context(), "org-a", "project-a", report); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := repo.Cached(t.Context(), "org-b", "project-a", report.ID); err != nil || ok {
		t.Fatalf("cross-org cache: %v %v", ok, err)
	}
	var wg sync.WaitGroup
	ids := make(chan string, 8)
	errs := make(chan error, 8)
	for range 8 {
		wg.Go(func() {
			id, err := repo.SaveAction(t.Context(), "org-a", "project-a", report.ID, report.Pages[0].Page, "Review CTA")
			ids <- id
			errs <- err
		})
	}
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var first string
	for id := range ids {
		if first == "" {
			first = id
		}
		if id != first {
			t.Fatal("concurrent save made duplicate actions")
		}
	}
	if err := repo.UpdateAction(t.Context(), "org-b", "project-a", first, "applied"); err == nil {
		t.Fatal("cross-org update accepted")
	}
	if err := repo.UpdateAction(t.Context(), "org-a", "project-a", first, "applied"); err != nil {
		t.Fatal(err)
	}
	actions, err := repo.Actions(t.Context(), "org-a", "project-a")
	if err != nil || len(actions) != 1 || actions[0].AppliedAt == nil {
		t.Fatalf("actions=%+v err=%v", actions, err)
	}
	before := actions[0].AppliedAt
	if err := repo.UpdateAction(t.Context(), "org-a", "project-a", first, "applied"); err != nil {
		t.Fatal(err)
	}
	actions, err = repo.Actions(t.Context(), "org-a", "project-a")
	if err != nil {
		t.Fatal(err)
	}
	if !actions[0].AppliedAt.Equal(*before) {
		t.Fatal("replay moved measurement date")
	}
	if _, err := repo.SaveAction(t.Context(), "org-a", "project-a", report.ID, "https://attacker.test", "Other page"); err == nil {
		t.Fatal("client invented a baseline page")
	}
	if _, err := repo.DB.Exec(t.Context(), `UPDATE go_search_growth_reports SET created_at=now()-interval '11 minutes'`); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := repo.Cached(t.Context(), "org-a", "project-a", report.ID); err != nil || ok {
		t.Fatalf("expired cache: %v %v", ok, err)
	}
}

type growthGoogle struct {
	mu    sync.Mutex
	calls int
	fail  bool
}

func (g *growthGoogle) DoJSON(_ context.Context, request google.APIRequest) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.calls++
	if g.fail {
		return errors.New("offline")
	}
	req := request.Body.(APIRequest)
	encoded, err := json.Marshal(req.DimensionFilter)
	if err != nil {
		return err
	}
	if !strings.Contains(string(encoded), "sessionSource") || !strings.Contains(string(encoded), "google") {
		return errors.New("Google source filter missing")
	}
	response := ProviderResponse{DimensionHeaders: req.Dimensions, MetricHeaders: req.Metrics, RowCount: 1, Rows: []ProviderRow{{DimensionValues: valuesOf([]string{"example.test", "/winner"}), MetricValues: valuesOf([]string{"200", "180", "160", "0.8", "20", "0.1", "1", "30"})}}}
	*request.Response.(*ProviderResponse) = response
	return nil
}

type growthSearch struct {
	mu    sync.Mutex
	calls int
}

func (g *growthSearch) QuerySearchAnalytics(_ context.Context, _ string, request gsc.SearchRequest) ([]gsc.SearchRow, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.calls++
	keys := []string{"https://example.test/winner"}
	if len(request.Dimensions) == 2 {
		keys = []string{"booking software", "https://example.test/winner"}
	}
	return []gsc.SearchRow{{Keys: keys, Clicks: 60, Impressions: 1000, CTR: .06, Position: 9}}, nil
}
func TestGrowthMountedReportCachesConcurrentRequestsAndRejectsBadInput(t *testing.T) {
	repo := growthRepo(t)
	provider := &growthGoogle{}
	search := &growthSearch{}
	service := &Service{Growth: repo, Connections: testConnections{}, Google: provider, Now: func() time.Time { return time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC) }}
	console := &gsc.Service{Connections: opportunityGSCStorage{connection: gsc.Connection{SiteURL: "https://example.test/"}}, NewClient: func(string, string) gsc.SearchClient { return search }}
	protect := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := auth.WithUser(r.Context(), auth.User{ID: "user-a"})
			ctx = auth.WithProjectOrganization(ctx, "org-a")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
	mux := http.NewServeMux()
	Mount(mux, Deps{Logger: slog.Default(), Service: service, SearchConsole: console, WithSession: protect, WithProjectAccess: func(h http.Handler) http.Handler { return h }})
	request := func(body string) *httptest.ResponseRecorder {
		r := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/projects/project-a/ga4/growth/report", strings.NewReader(body))
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	for _, body := range []string{`{"unknown":true}`, `{"brandTerms":null}`, `{"startDate":"2026-10-20","endDate":"2026-10-21"}`, `{"brandTerms":[""]}`} {
		if w := request(body); w.Code != 400 {
			t.Fatalf("bad input %s: %d %s", body, w.Code, w.Body.String())
		}
	}
	if search.calls != 0 || provider.calls != 0 {
		t.Fatal("invalid request contacted Google")
	}
	var wg sync.WaitGroup
	results := make(chan *httptest.ResponseRecorder, 8)
	for range 8 {
		wg.Go(func() { results <- request(`{}`) })
	}
	wg.Wait()
	close(results)
	var first GrowthReport
	for w := range results {
		if w.Code != 200 {
			t.Fatalf("report: %d %s", w.Code, w.Body.String())
		}
		if err := json.Unmarshal(w.Body.Bytes(), &first); err != nil {
			t.Fatal(err)
		}
	}
	if search.calls != 3 || provider.calls != 2 {
		t.Fatalf("calls GSC=%d GA4=%d; want 3+2 for all concurrent requests", search.calls, provider.calls)
	}
	if w := request(`{}`); w.Code != 200 || search.calls != 3 || provider.calls != 2 {
		t.Fatal("cache replay called provider")
	}
	id, err := repo.SaveAction(t.Context(), "org-a", "project-a", first.ID, first.Pages[0].Page, "Improve this page")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateAction(t.Context(), "org-a", "project-a", id, "applied"); err != nil {
		t.Fatal(err)
	}
	comparison, err := service.CompareGrowthAction(t.Context(), console, "org-a", "project-a", id, 14)
	if err != nil || comparison["status"] != "waiting" {
		t.Fatalf("comparison=%v err=%v", comparison, err)
	}
	if provider.calls != 2 || search.calls != 3 {
		t.Fatal("waiting comparison wasted provider requests")
	}
}

func TestGrowthComparisonUsesExactPageAndKeepsMissingDataUnknown(t *testing.T) {
	repo := growthRepo(t)
	current, previous, queries, ga := growthFixture(t)
	report, err := buildGrowthReport(current, previous, queries, ga, ga, nil)
	if err != nil {
		t.Fatal(err)
	}
	report.ID = "baseline"
	report.Range = DateRange{StartDate: "2026-08-01", EndDate: "2026-08-28"}
	report.PropertyID = "properties/123"
	report.SiteURL = "https://example.test/"
	report.TimeZone = "America/Los_Angeles"
	report.Currency = "USD"
	if err := repo.StoreReport(t.Context(), "org-a", "project-a", report); err != nil {
		t.Fatal(err)
	}
	id, err := repo.SaveAction(t.Context(), "org-a", "project-a", report.ID, "https://example.test/winner", "Review intent")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.DB.Exec(t.Context(), `UPDATE go_search_growth_actions SET status='applied',applied_at='2026-09-01T12:00:00Z' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	googleClient := &growthGoogle{}
	search := &growthSearch{}
	service := &Service{Growth: repo, Connections: testConnections{}, Google: googleClient, Now: func() time.Time { return time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC) }}
	console := &gsc.Service{Connections: opportunityGSCStorage{connection: gsc.Connection{SiteURL: report.SiteURL}}, NewClient: func(string, string) gsc.SearchClient { return search }}
	result, err := service.CompareGrowthAction(t.Context(), console, "org-a", "project-a", id, 14)
	if err != nil {
		t.Fatal(err)
	}
	if result["status"] != "ready" || result["after"] == nil || result["baselineClicksPerDay"] != float64(60)/28 || result["afterClicksPerDay"] != float64(60)/14 {
		t.Fatalf("comparison=%+v", result)
	}
	if googleClient.calls != 1 || search.calls != 1 {
		t.Fatalf("comparison multiplied reads: GA=%d GSC=%d", googleClient.calls, search.calls)
	}
	if _, err := repo.DB.Exec(t.Context(), `UPDATE go_search_growth_actions SET source=jsonb_set(source,'{propertyId}','"properties/999"') WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CompareGrowthAction(t.Context(), console, "org-a", "project-a", id, 14); err == nil {
		t.Fatal("changed property reused a baseline")
	}
	if _, err := service.CompareGrowthAction(t.Context(), console, "org-b", "project-a", id, 14); err == nil {
		t.Fatal("cross-org comparison was allowed")
	}
	if googleClient.calls != 1 || search.calls != 1 {
		t.Fatal("invalid comparison contacted Google")
	}
}
