package sitereport

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/httpx"
	"github.com/toufiqqureshi/uplumo/backend/internal/reports"
)

const maxRequestBody = 64 << 10

// Deps provides dependencies for the site report handler.
type Deps struct {
	Logger            *slog.Logger
	Service           *Service
	WithSession       func(http.Handler) http.Handler
	WithProjectAccess func(http.Handler) http.Handler
}

// Mount registers the create and private preview endpoints.
func Mount(mux *http.ServeMux, d Deps) {
	for _, action := range []string{"generate", "preview"} {
		mux.Handle("POST /api/v1/projects/{projectId}/reports/"+action, d.WithSession(d.WithProjectAccess(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			d.generate(w, r, action == "preview")
		}))))
	}
}

func (d Deps) generate(w http.ResponseWriter, r *http.Request, preview bool) {
	if d.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "reports_unavailable", "Site reports are not available on this server.")
		return
	}
	var in Input
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBody))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&in); err != nil || decoder.Decode(new(any)) != io.EOF {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The request must be a single JSON object with valid fields.")
		return
	}
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "unauthenticated", "Sign in to continue.")
		return
	}
	orgID, ok := auth.ProjectOrganizationFromContext(r.Context())
	if !ok {
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "Something went wrong.")
		return
	}
	if preview {
		result, err := d.Service.Preview(r.Context(), orgID, r.PathValue("projectId"), in)
		if err != nil {
			d.writeError(w, r, err)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		httpx.WriteJSON(w, http.StatusOK, result)
		return
	}
	name := strings.TrimSpace(user.Name)
	if name == "" {
		name = "Workspace member"
	}
	result, err := d.Service.Generate(r.Context(), orgID, r.PathValue("projectId"), Author{UserID: user.ID, Name: name}, in)
	if err != nil {
		d.writeError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (d Deps) writeError(w http.ResponseWriter, r *http.Request, err error) {
	if typed, ok := errors.AsType[*Error](err); ok {
		status := http.StatusBadRequest
		if typed.Code == "reports_unavailable" {
			status = http.StatusServiceUnavailable
		}
		httpx.WriteError(w, status, typed.Code, typed.Message)
		return
	}
	if typed, ok := errors.AsType[*reports.Error](err); ok {
		httpx.WriteError(w, http.StatusBadRequest, strings.ToLower(typed.Code), typed.Message)
		return
	}
	if errors.Is(err, ErrProjectNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "project_not_found", "Project not found.")
		return
	}
	if r.Context().Err() != nil {
		return
	}
	if d.Logger != nil {
		d.Logger.ErrorContext(r.Context(), "generate site report failed", "err", err)
	}
	httpx.WriteError(w, http.StatusInternalServerError, "internal", "Could not create the report. Please try again.")
}
