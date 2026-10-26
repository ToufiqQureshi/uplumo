package dashboard

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/httpx"
)

// LayoutStore is the storage the handlers need.
type LayoutStore interface {
	Get(ctx context.Context, userID, projectID string) ([]Card, error)
	Save(ctx context.Context, userID, projectID string, cards []Card) error
}

// Deps provides the dependencies of the layout handlers.
type Deps struct {
	Logger            *slog.Logger
	Store             LayoutStore
	WithSession       func(http.Handler) http.Handler
	WithProjectAccess func(http.Handler) http.Handler
}

// Mount registers the layout endpoints. Both need a session and access to
// the project; a layout belongs to the signed-in member, never to the request.
func Mount(mux *http.ServeMux, d Deps) {
	protect := func(h http.HandlerFunc) http.Handler { return d.WithSession(d.WithProjectAccess(h)) }
	mux.Handle("POST /api/v1/projects/{projectId}/dashboard/layout/get", protect(d.get))
	mux.Handle("POST /api/v1/projects/{projectId}/dashboard/layout/save", protect(d.save))
}

type layoutBody struct {
	Cards []Card `json:"cards"`
}

func (d Deps) get(w http.ResponseWriter, r *http.Request) {
	var in struct{}
	if !decode(w, r, &in) {
		return
	}
	userID, ok := d.ready(w, r)
	if !ok {
		return
	}
	cards, err := d.Store.Get(r.Context(), userID, r.PathValue("projectId"))
	if err != nil {
		d.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, layoutBody{Cards: cards})
}

func (d Deps) save(w http.ResponseWriter, r *http.Request) {
	var in layoutBody
	if !decode(w, r, &in) {
		return
	}
	if in.Cards == nil {
		invalid(w)
		return
	}
	userID, ok := d.ready(w, r)
	if !ok {
		return
	}
	if err := d.Store.Save(r.Context(), userID, r.PathValue("projectId"), in.Cards); err != nil {
		if errors.Is(err, ErrInvalidLayout) {
			invalid(w)
			return
		}
		d.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, layoutBody{Cards: in.Cards})
}

func (d Deps) ready(w http.ResponseWriter, r *http.Request) (string, bool) {
	u, ok := auth.UserFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Sign in to continue.")
		return "", false
	}
	if d.Store == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "dashboard_unavailable", "Dashboard settings are not available.")
		return "", false
	}
	return u.ID, true
}

func (d Deps) fail(w http.ResponseWriter, r *http.Request, err error) {
	if r.Context().Err() != nil {
		return
	}
	if d.Logger != nil {
		d.Logger.ErrorContext(r.Context(), "dashboard layout failed", "err", err)
	}
	httpx.WriteError(w, http.StatusInternalServerError, "internal", "Something went wrong.")
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		invalid(w)
		return false
	}
	if err := dec.Decode(new(any)); !errors.Is(err, io.EOF) {
		invalid(w)
		return false
	}
	return true
}

func invalid(w http.ResponseWriter) {
	httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Choose a valid dashboard layout.")
}
