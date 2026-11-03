package customers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/httpx"
)

// Deps wires the customer routes to the service and session middleware.
type Deps struct {
	Service     *Service
	Logger      *slog.Logger
	WithSession func(http.Handler) http.Handler
}

// Mount registers the staff access, customer list and CSV routes.
func Mount(mux *http.ServeMux, d Deps) {
	mux.Handle("GET /api/v1/admin/access", d.WithSession(http.HandlerFunc(d.access)))
	mux.Handle("GET /api/v1/admin/customers", d.WithSession(http.HandlerFunc(d.list)))
	mux.Handle("GET /api/v1/admin/customers.csv", d.WithSession(http.HandlerFunc(d.csv)))
}

func (d Deps) access(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFromContext(r.Context())
	staff := false
	if d.Service != nil {
		var err error
		if staff, err = d.Service.IsStaff(r.Context(), u); err != nil {
			d.fail(w, r, err)
			return
		}
	}
	w.Header().Set("Cache-Control", "private, no-store")
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"staff": staff})
}

func (d Deps) load(w http.ResponseWriter, r *http.Request) ([]Customer, bool) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok || u.ID == "" {
		httpx.WriteError(w, 401, "unauthenticated", "Sign in first.")
		return nil, false
	}
	if d.Service == nil {
		d.fail(w, r, ErrForbidden)
		return nil, false
	}
	list, err := d.Service.List(r.Context(), u)
	if err != nil {
		d.fail(w, r, err)
		return nil, false
	}
	w.Header().Set("Cache-Control", "private, no-store")
	return list, true
}

func (d Deps) list(w http.ResponseWriter, r *http.Request) {
	if list, ok := d.load(w, r); ok {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"customers": list})
	}
}

func (d Deps) csv(w http.ResponseWriter, r *http.Request) {
	list, ok := d.load(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="uplumo-customers-`+time.Now().UTC().Format("2006-01-02")+`.csv"`)
	if err := WriteCSV(w, list); err != nil && d.Logger != nil {
		d.Logger.ErrorContext(r.Context(), "customers csv failed", "err", err)
	}
}

func (d Deps) fail(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, ErrForbidden) {
		httpx.WriteError(w, 403, "forbidden", "Only Uplumo staff can see customers.")
		return
	}
	if d.Logger != nil {
		d.Logger.ErrorContext(r.Context(), "customers failed", "err", err)
	}
	httpx.WriteError(w, 500, "internal", "Could not load customers. Try again.")
}
