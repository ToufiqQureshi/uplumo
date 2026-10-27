package citefix

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
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

// serve mounts the route with project access that grants organizationID
// ("" leaves the request without one).
func serve(reporter Reporter, organizationID, body string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	pass := func(next http.Handler) http.Handler { return next }
	Mount(mux, Deps{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Reporter: reporter, WithSession: pass,
		WithProjectAccess: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if organizationID != "" {
					r = r.WithContext(auth.WithProjectOrganization(r.Context(), organizationID))
				}
				next.ServeHTTP(w, r)
			})
		},
	})
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/projects/proj-1/cite-fix/report", strings.NewReader(body)))
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
		"too large":         {&fakeReporter{}, "org-1", `{` + strings.Repeat(" ", 2<<10) + `}`, http.StatusBadRequest, "invalid_request"},
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
