package citefix

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/httpx"
)

// Reporter builds a project's report. Store is the production one.
type Reporter interface {
	Report(ctx context.Context, organizationID, projectID string) (Report, error)
}

// Deps wires the route. Reporter is nil when the database is not configured;
// the route then answers 503.
type Deps struct {
	Logger            *slog.Logger
	Reporter          Reporter
	WithSession       func(http.Handler) http.Handler
	WithProjectAccess func(http.Handler) http.Handler
}

// Mount registers the report route. Any member of the workspace may call it:
// it reads the project's own audit and spends nothing.
func Mount(mux *http.ServeMux, deps Deps) {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	h := handler{deps}
	mux.Handle("POST /api/v1/projects/{projectId}/cite-fix/report", deps.WithSession(deps.WithProjectAccess(http.HandlerFunc(h.report))))
}

type handler struct{ Deps }

// maxRequestBytes is generous for the empty object the route takes.
const maxRequestBytes = 1 << 10

func (h handler) report(w http.ResponseWriter, r *http.Request) {
	if h.Reporter == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "cite_fix_unavailable", "Cite Fix is not available on this server.")
		return
	}
	if !decodeEmpty(w, r) {
		return
	}
	organizationID, ok := auth.ProjectOrganizationFromContext(r.Context())
	if !ok || organizationID == "" {
		// Project access already refused a foreign project; this guards the
		// plain handler path.
		httpx.WriteError(w, http.StatusNotFound, "project_not_found", "Project not found.")
		return
	}
	report, err := h.Reporter.Report(r.Context(), organizationID, r.PathValue("projectId"))
	if err != nil {
		if r.Context().Err() == nil {
			h.Logger.ErrorContext(r.Context(), "build cite fix report", "project_id", r.PathValue("projectId"), "err", err)
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "Something went wrong. Please try again.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, report)
}

// decodeEmpty accepts exactly one empty JSON object, like the other project
// routes reject unknown fields.
func decodeEmpty(w http.ResponseWriter, r *http.Request) bool {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBytes))
	trimmed := bytes.TrimSpace(body)
	if err != nil || len(trimmed) == 0 || trimmed[0] != '{' {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Send an empty JSON object.")
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.DisallowUnknownFields()
	var input struct{}
	if err := decoder.Decode(&input); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Send an empty JSON object.")
		return false
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The request must contain exactly one JSON value.")
		return false
	}
	return true
}
