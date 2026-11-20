package freeaudit

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/toufiqqureshi/uplumo/backend/internal/audit"
)

// ReportPath answers the free audit page's request for a report. It takes
// url, type=seo|aeo (default seo) and cached=1; cached=1 only reads the cache,
// so switching between the two reports never runs a check.
const ReportPath = "/free-seo-audit/report"

// DocxPath serves a cached report as a Word file. It never runs a check.
const DocxPath = "/free-seo-audit/report.docx"

// maxInputBytes bounds the address a visitor may send.
const maxInputBytes = 2048

// docxContentType is the media type of a .docx file.
const docxContentType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"

// Pages renders the report and errors as HTML for the free audit page
// (site.Site).
type Pages interface {
	ServeAuditReport(w http.ResponseWriter, r *http.Request, report audit.PageReport)
	ServeAuditError(w http.ResponseWriter, status int, message string)
}

// Deps are what the routes need.
type Deps struct {
	Service  *Service
	Pages    Pages
	ClientIP func(*http.Request) string
	// Logo is the PNG drawn at the top of the Word file; nil draws none.
	Logo   []byte
	Logger *slog.Logger
}

// Mount registers the report routes. They need no session.
func Mount(mux *http.ServeMux, d Deps) {
	mux.HandleFunc("GET "+ReportPath, func(w http.ResponseWriter, r *http.Request) {
		input := r.URL.Query().Get("url")
		if input == "" || len(input) > maxInputBytes {
			d.Pages.ServeAuditError(w, http.StatusBadRequest, "Enter a website address, like example.com.")
			return
		}
		kind, ok := ParseKind(r.URL.Query().Get("type"))
		if !ok {
			d.Pages.ServeAuditError(w, http.StatusBadRequest, "Choose the SEO audit or the AEO audit.")
			return
		}
		if r.URL.Query().Get("cached") == "1" {
			report, found, err := d.Service.Cached(r.Context(), input, d.ClientIP(r), kind)
			if err != nil {
				d.writeError(w, r, err)
				return
			}
			if !found {
				d.Pages.ServeAuditError(w, http.StatusNotFound, "This report is no longer kept. Run the audit again to see it.")
				return
			}
			d.Pages.ServeAuditReport(w, r, report)
			return
		}
		report, err := d.Service.Run(r.Context(), input, d.ClientIP(r), kind)
		if err != nil {
			d.writeError(w, r, err)
			return
		}
		d.Pages.ServeAuditReport(w, r, report)
	})
	mux.HandleFunc("GET "+DocxPath, func(w http.ResponseWriter, r *http.Request) {
		input := r.URL.Query().Get("url")
		if input == "" || len(input) > maxInputBytes {
			d.Pages.ServeAuditError(w, http.StatusBadRequest, "Enter a website address, like example.com.")
			return
		}
		kind, ok := ParseKind(r.URL.Query().Get("type"))
		if !ok {
			d.Pages.ServeAuditError(w, http.StatusBadRequest, "Choose the SEO audit or the AEO audit.")
			return
		}
		report, ok, err := d.Service.Cached(r.Context(), input, d.ClientIP(r), kind)
		if err != nil {
			d.writeError(w, r, err)
			return
		}
		if !ok {
			d.Pages.ServeAuditError(w, http.StatusNotFound, "Run the audit for this page first, then download the Word file.")
			return
		}
		body, err := BuildDocx(report, d.Logo)
		if err != nil {
			d.Logger.ErrorContext(r.Context(), "build free audit docx", "err", err)
			d.Pages.ServeAuditError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
			return
		}
		h := w.Header()
		h.Set("Content-Type", docxContentType)
		h.Set("Content-Disposition", `attachment; filename="uplumo-`+kind+`-audit-`+fileHost(report.URL)+`.docx"`)
		h.Set("Cache-Control", "private, no-store")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Robots-Tag", "noindex")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	})
}

// writeError answers a failed lookup or check with the message for it.
func (d Deps) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalidURL):
		d.Pages.ServeAuditError(w, http.StatusBadRequest, "We could not find that website. Check the address, like example.com, and try again.")
	case errors.Is(err, ErrBlocked):
		d.Pages.ServeAuditError(w, http.StatusBadRequest, "That address is private or not allowed. Enter a public website.")
	case errors.Is(err, ErrTooMany):
		d.Pages.ServeAuditError(w, http.StatusTooManyRequests, "You have run many fresh checks today. Pages checked in the last 24 hours still open, or create a free account to audit your whole site.")
	case errors.Is(err, ErrBusy):
		d.Pages.ServeAuditError(w, http.StatusServiceUnavailable, "Many audits are running right now. Please try again in a minute.")
	default:
		d.Logger.ErrorContext(r.Context(), "free audit failed", "err", err)
		d.Pages.ServeAuditError(w, http.StatusInternalServerError, "Something went wrong. Please try again.")
	}
}

var unsafeFileChars = regexp.MustCompile(`[^a-z0-9.-]`)

// fileHost is the checked host made safe for a file name ("site" if none).
func fileHost(pageURL string) string {
	parsed, err := url.Parse(pageURL)
	if err != nil {
		return "site"
	}
	host := unsafeFileChars.ReplaceAllString(strings.ToLower(parsed.Hostname()), "")
	if host == "" {
		return "site"
	}
	return host
}
