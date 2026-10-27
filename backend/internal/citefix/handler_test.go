package citefix

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

type fakeReporter struct {
	report Report
	err    error
	calls  int
	org    string
	proj   string
}

func (f *fakeReporter) Report(_ context.Context, organizationID, projectID string) (Report, error) {
	f.calls++
	f.org, f.proj = organizationID, projectID
	return f.report, f.err
}

// serve posts body to the report route; serveDeps posts to any route with
// project access that grants organizationID ("" leaves the request without
// one).
func serve(reporter Reporter, organizationID, body string) *httptest.ResponseRecorder {
	return serveDeps(Deps{Reporter: reporter}, organizationID, "report", body)
}

func serveDeps(deps Deps, organizationID, route, body string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	deps.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	deps.WithSession = func(next http.Handler) http.Handler { return next }
	deps.WithProjectAccess = func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if organizationID != "" {
				r = r.WithContext(auth.WithProjectOrganization(r.Context(), organizationID))
			}
			next.ServeHTTP(w, r)
		})
	}
	Mount(mux, deps)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/projects/proj-1/cite-fix/"+route, strings.NewReader(body)))
	return rec
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error body %q: %v", rec.Body.String(), err)
	}
	return body.Error.Code
}

func TestHandlerServesTheReport(t *testing.T) {
	score := 72
	reporter := &fakeReporter{report: Report{State: StateReady, Score: &score, Categories: []CategoryScore{}, Fixes: []Fix{}, Pages: []PageScore{}}}
	rec := serve(reporter, "org-1", `{}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	if reporter.org != "org-1" || reporter.proj != "proj-1" {
		t.Fatalf("reporter asked for %s/%s", reporter.org, reporter.proj)
	}
	var got Report
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.State != StateReady || *got.Score != 72 {
		t.Fatalf("body = %s (%v)", rec.Body, err)
	}
	if !strings.Contains(rec.Body.String(), `"fixes":[]`) || !strings.Contains(rec.Body.String(), `"running":null`) {
		t.Fatalf("empty lists must be arrays and running explicit: %s", rec.Body)
	}
}

func TestHandlerRejectsBadRequests(t *testing.T) {
	for name, tt := range map[string]struct {
		reporter Reporter
		org      string
		body     string
		status   int
		code     string
	}{
		"no server support": {nil, "org-1", `{}`, http.StatusServiceUnavailable, "cite_fix_unavailable"},
		"empty body":        {&fakeReporter{}, "org-1", ``, http.StatusBadRequest, "invalid_request"},
		"not an object":     {&fakeReporter{}, "org-1", `[]`, http.StatusBadRequest, "invalid_request"},
		"unknown field":     {&fakeReporter{}, "org-1", `{"auditId":"a"}`, http.StatusBadRequest, "invalid_request"},
		"two values":        {&fakeReporter{}, "org-1", `{} {}`, http.StatusBadRequest, "invalid_request"},
		"too large":         {&fakeReporter{}, "org-1", `{` + strings.Repeat(" ", 5<<10) + `}`, http.StatusBadRequest, "invalid_request"},
		"no workspace":      {&fakeReporter{}, "", `{}`, http.StatusNotFound, "project_not_found"},
		"store fails":       {&fakeReporter{err: errors.New("db down")}, "org-1", `{}`, http.StatusInternalServerError, "internal"},
	} {
		t.Run(name, func(t *testing.T) {
			rec := serve(tt.reporter, tt.org, tt.body)
			if rec.Code != tt.status || errorCode(t, rec) != tt.code {
				t.Fatalf("got %d %s, want %d %s", rec.Code, rec.Body, tt.status, tt.code)
			}
			// Only the failing store may have been asked for a report.
			if f, ok := tt.reporter.(*fakeReporter); ok && f.err == nil && f.calls != 0 {
				t.Fatal("a rejected request reached the reporter")
			}
		})
	}
}

type fakeRunner struct {
	offer   AgentOffer
	result  AgentResult
	err     error
	input   AgentInput
	runs    int
	results []AgentResult
}

func (f *fakeRunner) Offer(context.Context, string) (AgentOffer, error) { return f.offer, nil }

func (f *fakeRunner) Run(_ context.Context, _, _ string, input AgentInput) (AgentResult, error) {
	f.runs++
	f.input = input
	return f.result, f.err
}

func (f *fakeRunner) Results(context.Context, string, string) ([]AgentResult, error) {
	return f.results, nil
}

func TestReportCarriesTheAgentOffer(t *testing.T) {
	reporter := &fakeReporter{report: emptyReport(StateNoScan)}
	// Without OpenRouter the agent is unavailable, at its prices.
	rec := serve(reporter, "org-1", `{}`)
	var report Report
	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil || report.Agent.Available || report.Agent.DeepMaxCredits != DeepMaxCredits || report.Agent.MaxPages != MaxAgentPages {
		t.Fatalf("offer without agent = %+v (%v)", report.Agent, err)
	}
	runner := &fakeRunner{offer: AgentOffer{Available: true, PlanAllowed: true, QuickMaxCredits: QuickMaxCredits}}
	rec = serveDeps(Deps{Reporter: reporter, Agent: runner}, "org-1", "report", `{}`)
	if err := json.Unmarshal(rec.Body.Bytes(), &report); err != nil || !report.Agent.Available || !report.Agent.PlanAllowed {
		t.Fatalf("offer with agent = %+v (%v)", report.Agent, err)
	}
}

func TestAgentRunRoute(t *testing.T) {
	runner := &fakeRunner{result: AgentResult{URL: "https://x.example/", Credits: 7}}
	rec := serveDeps(Deps{Reporter: &fakeReporter{}, Agent: runner}, "org-1", "agent/run", `{"url":"https://x.example/","mode":"quick","idempotencyKey":"k1"}`)
	if rec.Code != http.StatusOK || runner.input != (AgentInput{URL: "https://x.example/", Mode: "quick", IdempotencyKey: "k1"}) || !strings.Contains(rec.Body.String(), `"credits":7`) {
		t.Fatalf("run = %d %s, input %+v", rec.Code, rec.Body, runner.input)
	}
	for name, tt := range map[string]struct {
		err    error
		status int
		code   string
	}{
		"validation":   {&AgentValidationError{Message: "Choose Quick or Deep."}, http.StatusBadRequest, "invalid_request"},
		"no credits":   {fmt.Errorf("hold: %w", spend.ErrNoCredits), http.StatusPaymentRequired, "insufficient_credits"},
		"daily limit":  {fmt.Errorf("hold: %w", spend.ErrWorkspaceDailyLimit), http.StatusTooManyRequests, "daily_credit_limit"},
		"plan":         {ErrAgentPlanRequired, http.StatusPaymentRequired, "plan_required"},
		"not in scan":  {ErrAgentPageNotFound, http.StatusNotFound, "page_not_found"},
		"busy":         {ErrAgentBusy, http.StatusConflict, "agent_busy"},
		"rate":         {ErrAgentRateLimited, http.StatusTooManyRequests, "rate_limited"},
		"page fetch":   {fmt.Errorf("%w: status 404", ErrAgentPageFetch), http.StatusUnprocessableEntity, "page_unreadable"},
		"model failed": {fmt.Errorf("%w: analyst", ErrAgentFailed), http.StatusBadGateway, "agent_failed"},
		"other":        {errors.New("db down"), http.StatusInternalServerError, "internal"},
	} {
		t.Run(name, func(t *testing.T) {
			rec := serveDeps(Deps{Reporter: &fakeReporter{}, Agent: &fakeRunner{err: tt.err}}, "org-1", "agent/run", `{"url":"https://x.example/","mode":"quick","idempotencyKey":"k"}`)
			if rec.Code != tt.status || errorCode(t, rec) != tt.code {
				t.Fatalf("got %d %s, want %d %s", rec.Code, rec.Body, tt.status, tt.code)
			}
		})
	}
	// Unknown fields and a missing agent are refused before any run.
	refused := &fakeRunner{}
	if rec := serveDeps(Deps{Reporter: &fakeReporter{}, Agent: refused}, "org-1", "agent/run", `{"url":"u","mode":"deep","idempotencyKey":"k","maxCredits":999}`); rec.Code != http.StatusBadRequest || refused.runs != 0 {
		t.Fatalf("unknown field: %d, runs %d", rec.Code, refused.runs)
	}
	if rec := serveDeps(Deps{Reporter: &fakeReporter{}}, "org-1", "agent/run", `{}`); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("no agent: %d", rec.Code)
	}
}

func TestAgentResultsRoute(t *testing.T) {
	rec := serveDeps(Deps{Reporter: &fakeReporter{}}, "org-1", "agent/results", `{}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"results":[]`) {
		t.Fatalf("without agent = %d %s", rec.Code, rec.Body)
	}
	runner := &fakeRunner{results: []AgentResult{{URL: "https://x.example/"}}}
	rec = serveDeps(Deps{Reporter: &fakeReporter{}, Agent: runner}, "org-1", "agent/results", `{}`)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"url":"https://x.example/"`) {
		t.Fatalf("results = %d %s", rec.Code, rec.Body)
	}
}
