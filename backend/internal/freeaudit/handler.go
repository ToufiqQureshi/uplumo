package freeaudit

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/toufiqqureshi/uplumo/backend/internal/audit"
)

// ReportPath answers the free audit page's request for a report.
const ReportPath = "/free-seo-audit/report"

// maxInputBytes bounds the address a visitor may send.
const maxInputBytes = 2048

// Pages renders the report and errors as HTML for the free audit page
// (site.Site).
type Pages interface {
	ServeAuditReport(w http.ResponseWriter, r *http.Request, report audit.PageReport)
	ServeAuditError(w http.ResponseWriter, status int, message string)
}

// Deps are what the route needs.
type Deps struct {
	Service  *Service
	Pages    Pages
	ClientIP func(*http.Request) string
	Logger   *slog.Logger
}

// Mount registers GET /free-seo-audit/report?url=. It needs no session.
func Mount(mux *http.ServeMux, d Deps) {
	mux.HandleFunc("GET "+ReportPath, func(w http.ResponseWriter, r *http.Request) {
		input := r.URL.Query().Get("url")
		if input == "" || len(input) > maxInputBytes {
			d.Pages.ServeAuditError(w, http.StatusBadRequest, "Enter a website address, like example.com.")
			return
		}
		report, err := d.Service.Run(r.Context(), input, d.ClientIP(r))
		switch {
		case err == nil:
			d.Pages.ServeAuditReport(w, r, report)
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
	})
}
