package ga4

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

type growthBusinessGoogle struct {
	mu                                      sync.Mutex
	metadataCalls, reportCalls, funnelCalls int
	badRate                                 bool
	fail                                    bool
}

func (g *growthBusinessGoogle) DoJSON(_ context.Context, req google.APIRequest) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.fail {
		return errors.New("provider unavailable")
	}
	if strings.HasSuffix(req.URL, "/metadata") {
		g.metadataCalls++
		return json.Unmarshal([]byte(`{"metrics":[{"apiName":"sessionKeyEventRate:generate_lead"},{"apiName":"sessionKeyEventRate:purchase"},{"apiName":"sessionKeyEventRate"}]}`), req.Response)
	}
	if strings.HasSuffix(req.URL, ":runFunnelReport") {
		g.funnelCalls++
		body, err := json.Marshal(req.Body)
		if err != nil {
			return err
		}
		if !strings.Contains(string(body), `"isOpenFunnel":false`) || !strings.Contains(string(body), `"sessionSource"`) || !strings.Contains(string(body), `"google"`) {
			return errors.New("unscoped funnel")
		}
		rate := "0.25"
		if g.badRate {
			rate = "1.5"
		}
		return json.Unmarshal([]byte(fmt.Sprintf(`{"funnelTable":{"dimensionHeaders":[{"name":"funnelStepName"}],"metricHeaders":[{"name":"activeUsers"},{"name":"funnelStepCompletionRate"},{"name":"funnelStepAbandonments"},{"name":"funnelStepAbandonmentRate"}],"rows":[{"dimensionValues":[{"value":"1. page_view"}],"metricValues":[{"value":"200"},{"value":"%s"},{"value":"150"},{"value":"0.75"}]}]}}`, rate)), req.Response)
	}
	g.reportCalls++
	request := req.Body.(APIRequest)
	metrics := []string{}
	for _, metric := range request.Metrics {
		switch metric.Name {
		case "sessions":
			metrics = append(metrics, "200")
		case "engagementRate":
			metrics = append(metrics, "0.8")
		case "keyEvents":
			metrics = append(metrics, "100")
		case "sessionKeyEventRate":
			metrics = append(metrics, "0.5")
		case "sessionKeyEventRate:generate_lead":
			metrics = append(metrics, "0.01")
		default:
			metrics = append(metrics, "0")
		}
	}
	*req.Response.(*ProviderResponse) = ProviderResponse{DimensionHeaders: request.Dimensions, MetricHeaders: request.Metrics, RowCount: 1, Rows: []ProviderRow{{DimensionValues: valuesOf([]string{"example.test", "/winner"}), MetricValues: valuesOf(metrics)}}}
	return nil
}

func monitorRepo(t *testing.T) *GrowthRepository {
	t.Helper()
	r := growthRepo(t)
	data, err := os.ReadFile("../database/migrations/00032_search_growth_monitor.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.DB.Exec(t.Context(), strings.Split(string(data), "-- +goose Down")[0]); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestGrowthGoalMountedValidationCacheAndRealOutcomeSeparation(t *testing.T) {
	repo := growthRepo(t)
	provider := &growthBusinessGoogle{}
	search := &growthSearch{}
	svc := &Service{Growth: repo, Connections: testConnections{}, Google: provider, Now: func() time.Time { return time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC) }}
	console := &gsc.Service{Connections: opportunityGSCStorage{connection: gsc.Connection{SiteURL: "https://example.test/"}}, NewClient: func(string, string) gsc.SearchClient { return search }}
	protect := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(auth.WithProjectOrganization(r.Context(), "org-a")))
		})
	}
	mux := http.NewServeMux()
	Mount(mux, Deps{Service: svc, SearchConsole: console, WithSession: protect, WithProjectAccess: func(h http.Handler) http.Handler { return h }})
	request := func(body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), "POST", "/api/v1/projects/project-a/ga4/growth/report", strings.NewReader(body)))
		return w
	}
	for _, body := range []string{`{"goalEvent":"x:y"}`, `{"comparison":"invented"}`} {
		if w := request(body); w.Code != 400 {
			t.Fatalf("invalid input: %d %s", w.Code, w.Body.String())
		}
	}
	if provider.metadataCalls != 0 || search.calls != 0 {
		t.Fatal("invalid input called provider")
	}
	if w := request(`{"goalEvent":"missing"}`); w.Code != 400 {
		t.Fatalf("unregistered goal accepted: %d", w.Code)
	}
	if search.calls != 0 || provider.reportCalls != 0 {
		t.Fatal("unregistered goal requested reports")
	}
	var wg sync.WaitGroup
	responses := make(chan *httptest.ResponseRecorder, 8)
	for range 8 {
		wg.Go(func() { responses <- request(`{"goalEvent":"generate_lead"}`) })
	}
	wg.Wait()
	close(responses)
	for w := range responses {
		if w.Code != 200 {
			t.Fatalf("report: %d %s", w.Code, w.Body.String())
		}
		var report GrowthReport
		if err := json.Unmarshal(w.Body.Bytes(), &report); err != nil {
			t.Fatal(err)
		}
		p := findGrowthPage(t, report, "winner")
		if report.GoalEvent != "generate_lead" || p.Current.GoalRate == nil || *p.Current.GoalRate != .01 || p.Current.KeyEvents == nil || *p.Current.KeyEvents != 100 {
			t.Fatalf("goal metrics mixed with all events: %+v", p.Current)
		}
		if slices.Contains(findingKinds(p), "grow_winner") {
			t.Fatal("low-success goal promoted because of unrelated key events")
		}
	}
	if provider.metadataCalls != 2 || provider.reportCalls != 2 || search.calls != 3 {
		t.Fatalf("duplicate reads: metadata=%d reports=%d search=%d", provider.metadataCalls, provider.reportCalls, search.calls)
	}
	if w := request(`{"goalEvent":"generate_lead"}`); w.Code != 200 || provider.reportCalls != 2 {
		t.Fatal("goal cache replay called provider")
	}
	if w := request(`{"comparison":"year"}`); w.Code != 200 {
		t.Fatalf("year report: %d %s", w.Code, w.Body.String())
	}
	if provider.reportCalls != 4 || search.calls != 6 {
		t.Fatal("comparison inputs did not isolate cache")
	}
}

func TestGrowthActualFunnelRejectsMissingInvalidAndLimitedEvidence(t *testing.T) {
	provider := &growthBusinessGoogle{}
	svc := &Service{Connections: testConnections{}, Google: provider}
	dates := GrowthFunnelInput{StartDate: "2026-09-01", EndDate: "2026-09-28", Steps: []string{"page_view", "form_start", "generate_lead"}}
	invalid := dates
	invalid.Steps = []string{"page_view", "page_view"}
	if _, err := svc.GetGrowthFunnel(t.Context(), "project-a", invalid); err == nil || provider.funnelCalls != 0 {
		t.Fatal("duplicate funnel steps contacted provider")
	}
	report, err := svc.GetGrowthFunnel(t.Context(), "project-a", dates)
	if err != nil {
		t.Fatal(err)
	}
	rows := report["rows"].([]map[string]any)
	if provider.funnelCalls != 1 || rows[0]["activeUsers"] != float64(200) || rows[0]["funnelStepAbandonments"] != float64(150) {
		t.Fatalf("funnel=%v", report)
	}
	provider.badRate = true
	if _, err := svc.GetGrowthFunnel(t.Context(), "project-a", dates); err == nil {
		t.Fatal("impossible funnel rate accepted")
	}
	provider.badRate = false
	provider.fail = true
	if _, err := svc.GetGrowthFunnel(t.Context(), "project-a", dates); err == nil {
		t.Fatal("provider failure hidden")
	}
}

func TestGrowthGoalFindingsAndDigestSuppressSmallRestrictedEvidence(t *testing.T) {
	current, previous, queries, ga := growthFixture(t)
	for _, row := range ga.Rows {
		row["goalRate"] = .1
	}
	report, err := buildGrowthReport(current, previous, queries, ga, ga, nil)
	if err != nil {
		t.Fatal(err)
	}
	report.GoalEvent = "generate_lead"
	p := &report.Pages[0]
	// Isolate a business loss with enough sessions, while unrelated all-event rate is unchanged.
	old := GrowthMetrics{Clicks: 100, Sessions: ptrFloat(200), GoalRate: ptrFloat(.1)}
	p.Previous = &old
	p.Current.Clicks = 60
	p.Current.Sessions = ptrFloat(200)
	p.Current.GoalRate = ptrFloat(.02)
	d := growthDigest(report)
	if len(d.Alerts) != 1 || len(d.Pages) > 3 {
		t.Fatalf("digest=%+v", d)
	}
	report.Limited = true
	if len(growthDigest(report).Alerts) != 0 {
		t.Fatal("restricted data generated business alert")
	}
	report.Limited = false
	p.Previous.Sessions = ptrFloat(20)
	if len(growthDigest(report).Alerts) != 0 {
		t.Fatal("small sample generated business alert")
	}
	dates := DateRange{StartDate: "2026-09-01", EndDate: "2026-09-28"}
	shifted, err := growthComparisonPeriod(dates, "year")
	if err != nil {
		t.Fatal(err)
	}
	start, _ := parseDate(dates.StartDate)
	prior, _ := parseDate(shifted.StartDate)
	if start.Weekday() != prior.Weekday() || start.Sub(prior) != 364*24*time.Hour {
		t.Fatalf("weekday mismatch: %+v", shifted)
	}
}
func ptrFloat(v float64) *float64 { return &v }

func TestGrowthWeeklyMonitorClaimIsolationDisableAndConnectionChanges(t *testing.T) {
	repo := monitorRepo(t)
	provider := &growthBusinessGoogle{}
	search := &growthSearch{}
	svc := &Service{Growth: repo, Connections: testConnections{}, Google: provider, Now: func() time.Time { return time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC) }}
	console := &gsc.Service{Connections: opportunityGSCStorage{connection: gsc.Connection{SiteURL: "https://example.test/"}}, NewClient: func(string, string) gsc.SearchClient { return search }}
	state, err := svc.SaveGrowthMonitor(t.Context(), console, "org-a", "project-a", GrowthMonitorInput{Enabled: true, GoalEvent: "generate_lead"})
	if err != nil || !state.Enabled {
		t.Fatalf("save: %+v %v", state, err)
	}
	other, err := repo.Monitor(t.Context(), "org-b", "project-a")
	if err != nil || other.Enabled || other.Digest != nil {
		t.Fatalf("cross-org monitor leak: %+v %v", other, err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 6)
	for range 6 {
		wg.Go(func() { errs <- svc.CheckGrowthMonitors(t.Context(), console) })
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	state, err = repo.Monitor(t.Context(), "org-a", "project-a")
	if err != nil || state.Digest == nil || state.CheckedAt == nil {
		t.Fatalf("no digest: %+v %v", state, err)
	}
	if provider.reportCalls != 2 || search.calls != 3 {
		t.Fatalf("replicas multiplied checks: %d %d", provider.reportCalls, search.calls)
	}
	if _, err := repo.DB.Exec(t.Context(), `UPDATE go_search_growth_monitors SET next_check_at=now()-interval '1 minute',property_id='properties/999'`); err != nil {
		t.Fatal(err)
	}
	if err := svc.CheckGrowthMonitors(t.Context(), console); err == nil {
		t.Fatal("changed property silently reused schedule")
	}
	state, err = repo.Monitor(t.Context(), "org-a", "project-a")
	if err != nil || state.LastError == nil {
		t.Fatalf("missing failure: %+v %v", state, err)
	}
	if provider.reportCalls != 2 {
		t.Fatal("changed property requested Google data")
	}
	state, err = svc.SaveGrowthMonitor(t.Context(), nil, "org-a", "project-a", GrowthMonitorInput{Enabled: false})
	if err != nil || state.Enabled {
		t.Fatalf("cannot disable without Google: %+v %v", state, err)
	}
}

func TestClickDiagnosisDistinguishesEvidenceFromMissingQueries(t *testing.T) {
	old := GrowthMetrics{Clicks: 100, Impressions: 1000, CTR: .1, Position: 4}
	page := GrowthPage{Current: GrowthMetrics{Clicks: 30, Impressions: 600, CTR: .05, Position: 7}, Previous: &old}
	findings := clickFindings(page)
	kinds := []string{}
	for _, f := range findings {
		kinds = append(kinds, f.Kind)
	}
	for _, kind := range []string{"visibility_drop", "position_drop", "ctr_drop"} {
		if !slices.Contains(kinds, kind) {
			t.Fatalf("missing %s: %+v", kind, findings)
		}
	}
	rows, err := growthClickQueries([]gsc.SearchRow{{Keys: []string{"new query"}, Clicks: 0, Impressions: 10, Position: 20}}, []gsc.SearchRow{{Keys: []string{"old query"}, Clicks: 20, Impressions: 100, CTR: .2, Position: 4}})
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range rows {
		if q.Query == "old query" && q.Current != nil {
			t.Fatal("withheld query manufactured as zero current clicks")
		}
	}
	low := clickFindings(GrowthPage{Current: GrowthMetrics{Impressions: 10, Position: 20}})
	if len(low) != 1 || low[0].Kind != "low_visibility" {
		t.Fatalf("low-data diagnosis: %+v", low)
	}
}

type growthReadFunc func(context.Context, google.APIRequest) error

func (f growthReadFunc) DoJSON(ctx context.Context, r google.APIRequest) error { return f(ctx, r) }

func TestGrowthAnalyticsExpandedCoverageStopsAtFivePages(t *testing.T) {
	calls := 0
	svc := &Service{Google: growthReadFunc(func(_ context.Context, r google.APIRequest) error {
		calls++
		request := r.Body.(APIRequest)
		if request.Offset != fmt.Sprint((calls-1)*1000) || request.Limit != "1000" {
			t.Fatalf("wrong page request: %+v", request)
		}
		rows := make([]ProviderRow, 1000)
		for i := range rows {
			rows[i] = ProviderRow{DimensionValues: valuesOf([]string{"example.test", fmt.Sprintf("/page-%d", (calls-1)*1000+i)}), MetricValues: valuesOf([]string{"200", "180", "160", "0.8", "20", "0.1", "1", "30"})}
		}
		*r.Response.(*ProviderResponse) = ProviderResponse{DimensionHeaders: request.Dimensions, MetricHeaders: request.Metrics, RowCount: 10000, Rows: rows}
		return nil
	})}
	report, err := svc.growthLandingReport(t.Context(), Connection{PropertyID: "properties/123"}, DateRange{StartDate: "2026-09-01", EndDate: "2026-09-28"}, "", 5000)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 5 || len(report.Rows) != 5000 || report.TotalRowCount != 10000 {
		t.Fatalf("calls=%d rows=%d count=%d", calls, len(report.Rows), report.TotalRowCount)
	}
}

func TestGrowthComparisonPageBaselineIsImmutableAndScoped(t *testing.T) {
	repo := growthRepo(t)
	current, previous, queries, ga := growthFixture(t)
	report, err := buildGrowthReport(current, previous, queries, ga, ga, nil)
	if err != nil {
		t.Fatal(err)
	}
	report.ID = "control-baseline"
	report.Range = DateRange{StartDate: "2026-08-01", EndDate: "2026-08-28"}
	if err := repo.StoreReport(t.Context(), "org-a", "project-a", report); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SaveAction(t.Context(), "org-a", "project-a", report.ID, "https://example.test/winner", "Review CTA", "https://elsewhere.test/page"); err == nil {
		t.Fatal("untrusted comparison page accepted")
	}
	if _, err := repo.SaveAction(t.Context(), "org-b", "project-a", report.ID, "https://example.test/winner", "Review CTA", "https://example.test/weak"); err == nil {
		t.Fatal("other org reused baseline")
	}
	id, err := repo.SaveAction(t.Context(), "org-a", "project-a", report.ID, "https://example.test/winner", "Review CTA", "https://example.test/weak")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.SaveAction(t.Context(), "org-a", "project-a", report.ID, "https://example.test/winner", "Other note", "https://example.test/other"); err != nil {
		t.Fatal(err)
	}
	actions, err := repo.Actions(t.Context(), "org-a", "project-a")
	if err != nil {
		t.Fatal(err)
	}
	if len(actions) != 1 || actions[0].ID != id || actions[0].Source.Control == nil || actions[0].Source.Control.Page != "https://example.test/weak" {
		t.Fatalf("comparison baseline moved: %+v", actions)
	}
}

func TestGrowthCompetingPageExamplesStayBounded(t *testing.T) {
	pages := []gsc.SearchRow{}
	queries := []gsc.SearchRow{}
	for i := range 101 {
		page := fmt.Sprintf("https://example.test/page-%03d", i)
		pages = append(pages, gsc.SearchRow{Keys: []string{page}, Clicks: 10, Impressions: 100, CTR: .1, Position: 10})
		queries = append(queries, gsc.SearchRow{Keys: []string{"shared query", page}, Clicks: 10, Impressions: 100, CTR: .1, Position: 10})
	}
	report, err := buildGrowthReport(pages, pages, queries, normalizedReport{}, normalizedReport{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, page := range report.Pages {
		if len(page.Queries) != 1 || len(page.Queries[0].CompetingPages) != 5 || page.Queries[0].CompetingPageCount != 100 {
			t.Fatalf("quadratic competing-page response: %+v", page.Queries)
		}
	}
}

func TestGrowthMonitorDisableFencesAnInFlightDigest(t *testing.T) {
	repo := monitorRepo(t)
	provider := &growthBusinessGoogle{}
	search := &growthSearch{}
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	var once sync.Once
	svc := &Service{Growth: repo, Connections: testConnections{}, Google: growthReadFunc(func(ctx context.Context, r google.APIRequest) error {
		if strings.HasSuffix(r.URL, ":runReport") {
			once.Do(func() {
				close(started)
				select {
				case <-release:
				case <-ctx.Done():
				}
			})
		}
		return provider.DoJSON(ctx, r)
	}), Now: func() time.Time { return time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC) }}
	console := &gsc.Service{Connections: opportunityGSCStorage{connection: gsc.Connection{SiteURL: "https://example.test/"}}, NewClient: func(string, string) gsc.SearchClient { return search }}
	if _, err := svc.SaveGrowthMonitor(t.Context(), console, "org-a", "project-a", GrowthMonitorInput{Enabled: true, GoalEvent: "generate_lead"}); err != nil {
		t.Fatal(err)
	}
	work, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	go func() { done <- svc.CheckGrowthMonitors(work, console) }()
	select {
	case <-started:
	case <-work.Done():
		t.Fatal("worker did not start")
	}
	if _, err := svc.SaveGrowthMonitor(t.Context(), nil, "org-a", "project-a", GrowthMonitorInput{Enabled: false}); err != nil {
		close(release)
		t.Fatal(err)
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	state, err := repo.Monitor(t.Context(), "org-a", "project-a")
	if err != nil {
		t.Fatal(err)
	}
	if state.Enabled || state.Digest != nil || state.CheckedAt != nil {
		t.Fatalf("disabled monitor received stale work: %+v", state)
	}
}

func TestGrowthMonitorEntitlementAndFailureCooldown(t *testing.T) {
	repo := monitorRepo(t)
	provider := &growthBusinessGoogle{}
	search := &growthSearch{}
	svc := &Service{Growth: repo, Connections: testConnections{}, Google: provider, GrowthMonitorAllowed: func(context.Context, string) (bool, error) { return false, nil }}
	console := &gsc.Service{Connections: opportunityGSCStorage{connection: gsc.Connection{SiteURL: "https://example.test/"}}, NewClient: func(string, string) gsc.SearchClient { return search }}
	if _, err := svc.SaveGrowthMonitor(t.Context(), console, "org-a", "project-a", GrowthMonitorInput{Enabled: true, GoalEvent: "generate_lead"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.CheckGrowthMonitors(t.Context(), console); err == nil {
		t.Fatal("monitor bypassed entitlement")
	}
	if provider.reportCalls != 0 || search.calls != 0 {
		t.Fatal("denied schedule contacted report providers")
	}
	state, err := repo.Monitor(t.Context(), "org-a", "project-a")
	if err != nil {
		t.Fatal(err)
	}
	if state.LastError == nil || state.NextCheckAt == nil || state.NextCheckAt.Before(time.Now().Add(23*time.Hour)) {
		t.Fatalf("failure did not back off: %+v", state)
	}
	svc.GrowthMonitorAllowed = nil
	if err := svc.CheckGrowthMonitors(t.Context(), console); err != nil {
		t.Fatal(err)
	}
	if provider.reportCalls != 0 || search.calls != 0 {
		t.Fatal("failure retried before cooldown")
	}
}

func TestGrowthDrilldownsReportDisconnectedState(t *testing.T) {
	svc := &Service{Connections: testConnections{err: ErrConnectionNotFound}, Google: &growthBusinessGoogle{}}
	_, err := svc.GetGrowthGoals(t.Context(), "project-a")
	if reportCode(t, err) != "ga4_not_connected" {
		t.Fatalf("goal state: %v", err)
	}
	_, err = svc.GetGrowthFunnel(t.Context(), "project-a", GrowthFunnelInput{Steps: []string{"page_view", "generate_lead"}})
	if reportCode(t, err) != "ga4_not_connected" {
		t.Fatalf("funnel state: %v", err)
	}
	_, err = svc.GetGrowthTracking(t.Context(), "project-a", GrowthInput{})
	if reportCode(t, err) != "ga4_not_connected" {
		t.Fatalf("tracking state: %v", err)
	}
}
