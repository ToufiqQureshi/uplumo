package freeaudit

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/site"
)

// docxRouteMux mounts the report routes on a fresh mux.
func docxRouteMux(t *testing.T, svc *Service) *http.ServeMux {
	t.Helper()
	pages, err := site.New(&url.URL{Scheme: "https", Host: "uplumo.com"})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	Mount(mux, Deps{Service: svc, Pages: pages, Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		ClientIP: func(*http.Request) string { return "203.0.113.7" }, Logo: site.ReportLogoPNG()})
	return mux
}

func getDocx(t *testing.T, mux *http.ServeMux, query string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, DocxPath+query, nil))
	return rec
}

func TestDocxRouteServesOnlyCachedReports(t *testing.T) {
	checker := &fakeChecker{reachable: true}
	svc := newService(t, checker)
	mux := docxRouteMux(t, svc)

	cases := map[string]struct {
		query  string
		status int
		run    bool
	}{
		"not cached yet":  {"?url=example.com", http.StatusNotFound, false},
		"missing address": {"", http.StatusBadRequest, false},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rec := getDocx(t, mux, tc.query)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tc.status, rec.Body.String())
			}
			if checker.checks.Load() != 0 {
				t.Fatalf("the docx route ran %d fresh checks, want 0", checker.checks.Load())
			}
		})
	}

	// A check fills the cache; the Word file then comes from it.
	if _, err := svc.Run(context.Background(), "example.com", "203.0.113.7"); err != nil {
		t.Fatal(err)
	}
	if checker.checks.Load() != 1 {
		t.Fatalf("checks = %d, want 1", checker.checks.Load())
	}
	rec := getDocx(t, mux, "?url="+url.QueryEscape("example.com"))
	if rec.Code != http.StatusOK {
		t.Fatalf("cached report status = %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != docxContentType {
		t.Fatalf("content type = %q", got)
	}
	if got := rec.Header().Get("Content-Disposition"); got != `attachment; filename="uplumo-audit-example.com.docx"` {
		t.Fatalf("content disposition = %q", got)
	}
	reader, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
	if err != nil {
		t.Fatalf("response is not a zip: %v", err)
	}
	if len(reader.File) == 0 {
		t.Fatalf("empty docx")
	}
	if checker.checks.Load() != 1 {
		t.Fatalf("checks after the download = %d, want still 1", checker.checks.Load())
	}

	// An address that was never checked is still not checked.
	rec = getDocx(t, mux, "?url="+url.QueryEscape("other.example"))
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "Run the audit") {
		t.Fatalf("uncached address: status %d body %q", rec.Code, rec.Body.String())
	}
	if checker.checks.Load() != 1 {
		t.Fatalf("checks = %d, want 1 after an uncached request", checker.checks.Load())
	}
}

func TestDocxRouteRefusesBadAddresses(t *testing.T) {
	checker := &fakeChecker{reachable: true}
	svc := newService(t, checker)
	mux := docxRouteMux(t, svc)
	rec := getDocx(t, mux, "?url="+url.QueryEscape("http://127.0.0.1/"))
	if rec.Code == http.StatusOK {
		t.Fatalf("a private address produced a report: %d", rec.Code)
	}
	if checker.checks.Load() != 0 {
		t.Fatalf("checks = %d, want 0", checker.checks.Load())
	}
}
