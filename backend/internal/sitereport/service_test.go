package sitereport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/branding"
	"github.com/toufiqqureshi/uplumo/backend/internal/reports"
)

var fixedNow = time.Date(2026, 11, 6, 9, 30, 0, 0, time.UTC)

func ptr[T any](v T) *T { return &v }

func fullSearch() *Search {
	s := &Search{Current: Totals{Clicks: 1840, Impressions: 96_400, CTR: 0.0191, Position: 14.2}, Previous: Totals{Clicks: 1502, Impressions: 88_100, CTR: 0.017, Position: 16.1}, HasPrevious: true}
	for i := range 28 {
		d := time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i)
		s.Daily = append(s.Daily, SearchDay{Date: d.Format(day), Clicks: float64(50 + i*2 + (i%7)*5), Impressions: float64(3000 + i*40)})
	}
	s.TopQueries = []Row{{Key: "medical billing services", Clicks: 320, Impressions: 9800, CTR: 0.0327, Position: 4.1}, {Key: "revenue cycle management <script>", Clicks: 210, Impressions: 12000, CTR: 0.0175, Position: 7.9}}
	s.TopPages = []Row{{Key: "https://example.com/services/billing", Clicks: 540, Impressions: 21000, CTR: 0.0257, Position: 6.2}, {Key: "javascript:alert(1)", Clicks: 1, Impressions: 2}}
	s.Gains = []Change{{Key: "denial management", Clicks: 40, PrevClicks: 12, Position: 5.2, PrevPosition: 9.4}, {Key: "new query", Clicks: 9}}
	s.Losses = []Change{{Key: "ar follow up", Clicks: 8, PrevClicks: 30, Position: 12, PrevPosition: 6}}
	s.Opportunities = []Opportunity{{Query: "rcm company", Page: "https://example.com/rcm", Impressions: 5400, Clicks: 22, Position: 11.3}}
	s.Countries = []Row{{Key: "usa", Clicks: 1500, Impressions: 80000, CTR: 0.019, Position: 13}}
	return s
}

func fullSnapshot() Snapshot {
	t := &Traffic{Currency: "USD", Sessions: Metric{ptr(2410.0), ptr(2005.0)}, Users: Metric{ptr(1980.0), ptr(1700.0)}, EngagedSessions: Metric{ptr(1500.0), ptr(1200.0)}, EngagementRate: Metric{ptr(0.62), ptr(0.6)}, KeyEvents: Metric{ptr(48.0), ptr(31.0)}, Revenue: Metric{ptr(0.0), ptr(0.0)}}
	for i := range 28 {
		t.Daily = append(t.Daily, TrafficDay{Date: time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC).AddDate(0, 0, i).Format(day), Sessions: float64(70 + i*3)})
	}
	r := &Rankings{Domain: "example.com", Location: "United States", Device: "desktop", CheckedAt: ptr(fixedNow.AddDate(0, 0, -1)), Rows: []RankRow{
		{Keyword: "medical billing", Position: ptr(2), Previous: ptr(5), URL: "https://example.com/", Volume: ptr(12100)},
		{Keyword: "rcm services", Position: ptr(9), Previous: ptr(8), URL: "https://example.com/rcm", Volume: ptr(880)},
		{Keyword: "billing audit", Position: ptr(15), Previous: nil},
		{Keyword: "coding company", Position: nil, Previous: ptr(18)},
	}}
	summarizeRanks(r)
	a := &Audit{CompletedAt: fixedNow.AddDate(0, 0, -2), PagesCrawled: 182, Critical: 4, Warning: 37, Info: 60, Issues: []Issue{
		{Type: "broken-internal-link", Severity: "critical", Pages: 4, Examples: []string{"https://example.com/a", "https://example.com/b"}},
		{Type: "missing-meta-description", Severity: "warning", Pages: 37, Examples: []string{"https://example.com/c"}},
	}, Lighthouse: &Lighthouse{Pages: 5, Performance: ptr(72), Accessibility: ptr(94), BestPractices: ptr(100), SEO: ptr(45), LCPMillis: ptr(2900.0), CLS: ptr(0.04), INPMillis: ptr(180.0)}}
	b := &Backlinks{CapturedAt: fixedNow.AddDate(0, 0, -3), Rank: ptr[int64](312), Backlinks: ptr[int64](4210), ReferringDomains: ptr[int64](286), NewReferringDomains: ptr[int64](14), LostReferringDomains: ptr[int64](3)}
	return Snapshot{
		Project: Project{Name: "Acme", Domain: "example.com"}, Period: periodFor(fixedNow, 28), GeneratedAt: fixedNow,
		Note: "Hi team,\n\nGreat month: billing pages are climbing.", NextSteps: "Fix the 4 broken links.\n\nRefresh the RCM page.",
		Search: fullSearch(), Traffic: t, Rankings: r, Audit: a, Backlinks: b,
	}
}

func TestRenderFullSnapshotIsSafeAndSelfContained(t *testing.T) {
	html, err := Render(fullSnapshot())
	if err != nil {
		t.Fatal(err)
	}
	if len(html) > reports.MaxHTMLBytes || !strings.HasSuffix(strings.TrimSpace(html), "</html>") {
		t.Fatalf("report is %d bytes or incomplete", len(html))
	}
	for _, banned := range []string{"<script", `href="javascript`, "http://", "src=\"http"} {
		if strings.Contains(html, banned) {
			t.Errorf("report contains %q", banned)
		}
	}
	for _, want := range []string{
		`id="search"`, `id="traffic"`, `id="rankings"`, `id="audit"`, `id="backlinks"`, `id="next"`, `id="opportunities"`,
		"Search clicks up 23%", "revenue cycle management &lt;script&gt;", "/services/billing", "Broken internal link",
		"Missing meta description", "and 2 more", "Lost", "New", "+14", "Great month", "Refresh the RCM page", "#65a30d",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("report is missing %q", want)
		}
	}
	if strings.Contains(html, "Revenue USD") {
		t.Error("zero revenue must not render a card")
	}
}

func TestRenderOmitsSectionsWithoutData(t *testing.T) {
	snap := Snapshot{Project: Project{Name: "Only audit"}, Period: periodFor(fixedNow, 7), GeneratedAt: fixedNow, Audit: &Audit{CompletedAt: fixedNow, PagesCrawled: 1}}
	html, err := Render(snap)
	if err != nil {
		t.Fatal(err)
	}
	for _, absent := range []string{`id="search"`, `id="traffic"`, `id="rankings"`, `id="backlinks"`, `id="next"`, `class="note"`} {
		if strings.Contains(html, absent) {
			t.Errorf("empty section rendered: %s", absent)
		}
	}
	if strings.Contains(html, "compared with") || !strings.Contains(html, "Prepared <b>6 Nov 2026</b>") {
		t.Error("a report without period data must not claim a comparison window")
	}
	if !strings.Contains(html, "The audit found no issues.") || !strings.Contains(html, "<h1>Only audit</h1>") {
		t.Error("audit-only report is missing its content")
	}
}

func TestPeriodEndsOnSearchConsoleFinalDay(t *testing.T) {
	tests := []struct {
		days                           int
		start, end, prevStart, prevEnd string
	}{
		{7, "2026-10-28", "2026-11-03", "2026-10-21", "2026-10-27"},
		{28, "2026-10-07", "2026-11-03", "2026-09-09", "2026-10-06"},
		{90, "2026-08-06", "2026-11-03", "2026-05-08", "2026-08-05"},
	}
	for _, tt := range tests {
		p := periodFor(fixedNow, tt.days)
		got := []string{p.Start.Format(day), p.End.Format(day), p.PrevStart.Format(day), p.PrevEnd.Format(day)}
		want := []string{tt.start, tt.end, tt.prevStart, tt.prevEnd}
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%d days = %v, want %v", tt.days, got, want)
		}
	}
}

func TestFormatting(t *testing.T) {
	tests := map[string]string{
		Number(0): "0", Number(1234567): "1,234,567", Number(-1500): "-1,500", Compact(9999): "9,999",
		Compact(12_345): "12.3K", Compact(2_000_000): "2M", Percent(4.26): "4.3%", Percent(23.4): "23%",
		Position(12): "12", issueLabel("missing-h1"): "Missing H1", issueLabel("http-links"): "HTTP links",
		shortURL("https://example.com/a?b=1"): "/a?b=1", shortURL("https://example.com"): "/", shortURL("not a url"): "not a url",
		string(safeHref("javascript:alert(1)")): "", string(safeHref("https://u:p@example.com")): "",
	}
	for got, want := range tests {
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}

// --- Generate ---

type fakeStore struct {
	project  Project
	err      error
	audit    *Audit
	links    *Backlinks
	auditErr error
}

func (f fakeStore) Project(context.Context, string, string) (Project, error) { return f.project, f.err }
func (f fakeStore) LatestAudit(context.Context, string) (*Audit, error)      { return f.audit, f.auditErr }
func (f fakeStore) LatestBacklinks(context.Context, string, string) (*Backlinks, error) {
	return f.links, nil
}

type fakeSearch struct {
	data  *Search
	err   error
	calls atomic.Int32
	gate  chan struct{}
}

func (f *fakeSearch) Search(context.Context, string, string, Period) (*Search, error) {
	f.calls.Add(1)
	if f.gate != nil {
		<-f.gate
	}
	return f.data, f.err
}

type fakeTraffic struct {
	data *Traffic
	err  error
}

func (f fakeTraffic) Traffic(context.Context, string, Period) (*Traffic, error) { return f.data, f.err }

type fakeRanks struct{ data *Rankings }

func (f fakeRanks) Rankings(context.Context, string, string, Period) (*Rankings, error) {
	return f.data, nil
}

type fakeSaver struct {
	mu      sync.Mutex
	saved   []reports.SaveInput
	shared  []string
	saveErr error
}

func (f *fakeSaver) Save(_ context.Context, in reports.SaveInput) (reports.SaveResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.saveErr != nil {
		return reports.SaveResult{}, f.saveErr
	}
	f.saved = append(f.saved, in)
	return reports.SaveResult{ReportID: fmt.Sprintf("r%d", len(f.saved)), Title: in.Title, Created: true}, nil
}

func (f *fakeSaver) Share(_ context.Context, _, id string) (reports.Metadata, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.shared = append(f.shared, id)
	return reports.Metadata{ID: id, ShareToken: ptr("token-" + id)}, nil
}

func newService(search *fakeSearch, saver *fakeSaver) *Service {
	snap := fullSnapshot()
	return &Service{
		Store:  fakeStore{project: snap.Project, audit: snap.Audit, links: snap.Backlinks},
		Search: search, Traffic: fakeTraffic{err: ErrNotConnected}, Rankings: fakeRanks{data: snap.Rankings},
		Reports: saver, Now: func() time.Time { return fixedNow },
		Branding: func(context.Context, string) (branding.Branding, bool, error) {
			return branding.Branding{BrandName: "Agency", AccentColor: "#7c3aed"}, true, nil
		},
	}
}

func validInput() Input {
	return Input{Title: "Acme · October", PeriodDays: 28, Sections: AllSections, Note: "", NextSteps: "Ship it", Share: true}
}

func TestGenerateSavesSharesAndExplainsSkippedSections(t *testing.T) {
	saver := &fakeSaver{}
	svc := newService(&fakeSearch{data: fullSearch()}, saver)
	got, err := svc.Generate(t.Context(), "org", "proj", Author{UserID: "u1", Name: "Toufiq"}, validInput())
	if err != nil {
		t.Fatal(err)
	}
	if len(saver.saved) != 1 || len(saver.shared) != 1 || got.ShareToken == nil || *got.ShareToken != "token-r1" {
		t.Fatalf("saved=%d shared=%v result=%+v", len(saver.saved), saver.shared, got)
	}
	in := saver.saved[0]
	if in.ProjectID != "proj" || in.OrganizationID != "org" || in.CreatedByUserID != "u1" || in.CreatedBy != "Toufiq" || in.Skill == nil || *in.Skill != "Site report" {
		t.Fatalf("save input = %+v", in)
	}
	if !strings.Contains(in.HTML, "#7c3aed") || !strings.Contains(in.Summary, "Search clicks up") {
		t.Error("report must use the workspace accent and summarize highlights")
	}
	if fmt.Sprint(got.Included) != "[search rankings audit backlinks]" {
		t.Errorf("included = %v", got.Included)
	}
	if len(got.Skipped) != 1 || got.Skipped[0].Section != SectionTraffic || !strings.Contains(got.Skipped[0].Reason, "not connected") {
		t.Errorf("skipped = %+v", got.Skipped)
	}
}

func TestGenerateSkipsFailedSourceInsteadOfFailing(t *testing.T) {
	saver := &fakeSaver{}
	svc := newService(&fakeSearch{err: errors.New("google 503")}, saver)
	svc.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	got, err := svc.Generate(t.Context(), "org", "proj", Author{}, validInput())
	if err != nil {
		t.Fatal(err)
	}
	if got.Skipped[0].Section != SectionSearch || !strings.Contains(got.Skipped[0].Reason, "Try again") {
		t.Errorf("skipped = %+v", got.Skipped)
	}
	if strings.Contains(saver.saved[0].HTML, `id="search"`) {
		t.Error("failed section must not render")
	}
}

func TestGenerateRefusesWhenNoSectionHasData(t *testing.T) {
	saver := &fakeSaver{}
	svc := &Service{Store: fakeStore{project: Project{Domain: "example.com"}}, Search: &fakeSearch{data: &Search{}}, Reports: saver, Now: func() time.Time { return fixedNow }}
	_, err := svc.Generate(t.Context(), "org", "proj", Author{}, Input{Title: "x", PeriodDays: 7, Sections: AllSections})
	typed, ok := errors.AsType[*Error](err)
	if !ok || typed.Code != "no_report_data" || !strings.Contains(typed.Message, "no search data") || !strings.Contains(typed.Message, "Rank Tracking") {
		t.Fatalf("err = %v", err)
	}
	if len(saver.saved) != 0 {
		t.Fatal("nothing should be saved")
	}
}

func TestGenerateValidatesInput(t *testing.T) {
	tests := map[string]Input{
		"empty title":    {Title: "  ", PeriodDays: 28, Sections: AllSections},
		"long title":     {Title: strings.Repeat("é", 121), PeriodDays: 28, Sections: AllSections},
		"bad period":     {Title: "x", PeriodDays: 30, Sections: AllSections},
		"no sections":    {Title: "x", PeriodDays: 28},
		"bad section":    {Title: "x", PeriodDays: 28, Sections: []Section{"ads"}},
		"long note":      {Title: "x", PeriodDays: 28, Sections: AllSections, Note: strings.Repeat("a", maxNoteChars+1)},
		"long nextsteps": {Title: "x", PeriodDays: 28, Sections: AllSections, NextSteps: strings.Repeat("a", maxNoteChars+1)},
	}
	for name, in := range tests {
		t.Run(name, func(t *testing.T) {
			saver := &fakeSaver{}
			search := &fakeSearch{data: fullSearch()}
			_, err := newService(search, saver).Generate(t.Context(), "org", "proj", Author{}, in)
			if typed, ok := errors.AsType[*Error](err); !ok || typed.Code != "validation_error" {
				t.Fatalf("err = %v", err)
			}
			if search.calls.Load() != 0 || len(saver.saved) != 0 {
				t.Fatal("invalid input must not read sources or save")
			}
		})
	}
}

func TestGenerateDedupesDoubleClicks(t *testing.T) {
	saver := &fakeSaver{}
	search := &fakeSearch{data: fullSearch(), gate: make(chan struct{})}
	svc := newService(search, saver)
	var wg sync.WaitGroup
	results := make(chan Result, 8)
	for range 8 {
		wg.Go(func() {
			got, err := svc.Generate(t.Context(), "org", "proj", Author{}, validInput())
			if err != nil {
				t.Error(err)
			}
			results <- got
		})
	}
	time.Sleep(100 * time.Millisecond)
	close(search.gate)
	wg.Wait()
	close(results)
	if len(saver.saved) != 1 || search.calls.Load() != 1 {
		t.Fatalf("8 concurrent clicks saved %d reports and read search %d times", len(saver.saved), search.calls.Load())
	}
	for got := range results {
		if got.ReportID != "r1" {
			t.Fatalf("result = %+v", got)
		}
	}
}

func TestGenerateKeepsReportWhenShareFails(t *testing.T) {
	saver := &failingShare{}
	svc := newService(&fakeSearch{data: fullSearch()}, nil)
	svc.Reports = saver
	got, err := svc.Generate(t.Context(), "org", "proj", Author{}, validInput())
	if err != nil || got.ReportID != "r1" || got.ShareToken != nil {
		t.Fatalf("got %+v, %v", got, err)
	}
}

type failingShare struct{ fakeSaver }

func (f *failingShare) Share(context.Context, string, string) (reports.Metadata, error) {
	return reports.Metadata{}, errors.New("db down")
}

// --- handler ---

func TestHandler(t *testing.T) {
	saver := &fakeSaver{}
	svc := newService(&fakeSearch{data: fullSearch()}, saver)
	mux := http.NewServeMux()
	identity := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := auth.WithUser(r.Context(), auth.User{ID: "u1", Name: " "})
			next.ServeHTTP(w, r.WithContext(auth.WithProjectOrganization(ctx, "org")))
		})
	}
	Mount(mux, Deps{Service: svc, WithSession: identity, WithProjectAccess: func(h http.Handler) http.Handler { return h }})
	tests := []struct {
		body   string
		status int
		want   string
	}{
		{`{"title":"A","periodDays":28,"sections":["search"],"note":"","nextSteps":"","share":false}`, 200, `"reportId":"r1"`},
		{`{"title":"A","periodDays":28,"sections":["search"],"extra":1}`, 400, "invalid_request"},
		{`{"title":"A"} {}`, 400, "invalid_request"},
		{`{"title":"","periodDays":28,"sections":["search"]}`, 400, "validation_error"},
	}
	for _, tt := range tests {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/projects/p1/reports/generate", strings.NewReader(tt.body)))
		if rec.Code != tt.status || !strings.Contains(rec.Body.String(), tt.want) {
			t.Errorf("%s = %d %s", tt.body, rec.Code, rec.Body)
		}
	}
	if len(saver.saved) != 1 || saver.saved[0].CreatedBy != "Workspace member" || saver.saved[0].ProjectID != "p1" {
		t.Fatalf("saved = %+v", saver.saved)
	}
	saver.saveErr = &reports.Error{Code: "VALIDATION_ERROR", Message: "A report with this title already exists in the project."}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/projects/p1/reports/generate", strings.NewReader(`{"title":"B","periodDays":7,"sections":["audit"]}`)))
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "already exists") {
		t.Fatalf("duplicate title = %d %s", rec.Code, rec.Body)
	}
}
