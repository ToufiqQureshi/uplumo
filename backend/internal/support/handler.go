package support

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/httpx"
)

type Deps struct {
	Service     *Service
	Logger      *slog.Logger
	WithSession func(http.Handler) http.Handler
}

func Mount(mux *http.ServeMux, d Deps) {
	mux.Handle("POST /api/v1/support/reports", d.WithSession(http.HandlerFunc(d.create)))
	mux.Handle("GET /api/v1/support/reports/{ticketID}/screenshot", d.WithSession(http.HandlerFunc(d.screenshot)))
}
func (d Deps) user(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u.ID == "" {
		httpx.WriteError(w, 401, "unauthenticated", "Sign in to report a problem.")
		return u, false
	}
	if u.OrganizationID == "" {
		httpx.WriteError(w, 403, "no_organization", "Open a workspace first.")
		return u, false
	}
	if d.Service == nil {
		httpx.WriteError(w, 503, "unavailable", "Support reports are temporarily unavailable. Email support@uplumo.com.")
		return u, false
	}
	return u, true
}
func (d Deps) create(w http.ResponseWriter, r *http.Request) {
	u, ok := d.user(w, r)
	if !ok {
		return
	}
	var in Input
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&in); err != nil {
		d.fail(w, r, ErrInvalid)
		return
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		d.fail(w, r, ErrInvalid)
		return
	}
	id, err := d.Service.Create(r.Context(), u, in)
	if err != nil {
		d.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, map[string]string{"ticketId": id})
}
func (d Deps) screenshot(w http.ResponseWriter, r *http.Request) {
	u, ok := d.user(w, r)
	if !ok {
		return
	}
	data, mime, err := d.Service.Screenshot(r.Context(), u, r.PathValue("ticketID"))
	if err != nil {
		d.fail(w, r, err)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", `attachment; filename="screenshot"`)
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(data)
}
func (d Deps) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalid):
		httpx.WriteError(w, 400, "invalid_request", "Choose a report type, describe the problem in 10 to 5,000 characters, and attach only a PNG or JPEG up to 1 MB.")
	case errors.Is(err, ErrConflict):
		httpx.WriteError(w, 409, "conflict", "This report was already submitted with different details. Start a new report.")
	case errors.Is(err, ErrRateLimit):
		w.Header().Set("Retry-After", "3600")
		httpx.WriteError(w, 429, "rate_limited", "You can send up to five reports an hour. Try again later or email support@uplumo.com.")
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, 404, "not_found", "Screenshot not found.")
	default:
		if d.Logger != nil {
			d.Logger.ErrorContext(r.Context(), "support report failed", "err", err)
		}
		httpx.WriteError(w, 500, "internal", "Could not save your report. Try again with the same details or email support@uplumo.com.")
	}
}
