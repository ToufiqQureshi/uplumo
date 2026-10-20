package indexnow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/httpx"
)

const (
	// maxRequestBody fits MaxURLs typical URLs, not MaxURLs of maxURLLength.
	maxRequestBody = 2 << 20
	maxURLLength   = 2048
	// outboundTimeout covers checking key files and posting to engines for
	// up to maxHosts hosts. The handler lifts the server's write deadline
	// to match.
	outboundTimeout = 60 * time.Second
)

// Deps wires the IndexNow routes.
type Deps struct {
	Logger *slog.Logger
	// Service is nil when IndexNow is not configured; routes answer 503.
	Service           *Service
	WithSession       func(http.Handler) http.Handler
	WithProjectAccess func(http.Handler) http.Handler
}

type handler struct{ Deps }

// Mount registers the IndexNow routes.
func Mount(mux *http.ServeMux, deps Deps) {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	h := handler{deps}
	protect := func(next http.HandlerFunc) http.Handler { return deps.WithSession(deps.WithProjectAccess(next)) }
	mux.Handle("POST /api/v1/projects/{projectId}/indexnow/get", protect(h.get))
	mux.Handle("POST /api/v1/projects/{projectId}/indexnow/setup", protect(h.setUp))
	mux.Handle("POST /api/v1/projects/{projectId}/indexnow/check", protect(h.check))
	mux.Handle("POST /api/v1/projects/{projectId}/indexnow/submit", protect(h.submit))
}

type setupView struct {
	Included bool   `json:"included"`
	Domain   string `json:"domain"`
	Key      string `json:"key"`
}

func (h handler) get(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := h.start(w, r)
	if !ok || !decodeEmpty(w, r) {
		return
	}
	included, err := h.Service.Included(r.Context(), organizationID)
	if err != nil {
		h.writeError(w, r, "check indexnow plan", err)
		return
	}
	setup, err := h.Service.Get(r.Context(), organizationID, r.PathValue("projectId"))
	if err != nil {
		h.writeError(w, r, "load indexnow setup", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, setupView{Included: included, Domain: setup.Domain, Key: setup.Key})
}

func (h handler) setUp(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := h.start(w, r)
	if !ok || !decodeEmpty(w, r) {
		return
	}
	setup, err := h.Service.SetUp(r.Context(), organizationID, r.PathValue("projectId"))
	if err != nil {
		h.writeError(w, r, "set up indexnow", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, setupView{Included: true, Domain: setup.Domain, Key: setup.Key})
}

func (h handler) check(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := h.start(w, r)
	if !ok || !decodeEmpty(w, r) {
		return
	}
	h.extendWriteDeadline(w, r)
	ctx, cancel := context.WithTimeout(r.Context(), outboundTimeout)
	defer cancel()
	files, err := h.Service.CheckKeyFiles(ctx, organizationID, r.PathValue("projectId"))
	if err != nil {
		h.writeError(w, r, "check indexnow key files", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"files": files})
}

func (h handler) submit(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := h.start(w, r)
	if !ok {
		return
	}
	var input struct {
		URLs []string `json:"urls"`
	}
	if !decode(w, r, &input) {
		return
	}
	if len(input.URLs) == 0 || len(input.URLs) > MaxURLs {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Send between 1 and 10,000 URLs.")
		return
	}
	for _, raw := range input.URLs {
		if len(raw) > maxURLLength {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Each URL must be at most 2,048 characters.")
			return
		}
	}
	h.extendWriteDeadline(w, r)
	ctx, cancel := context.WithTimeout(r.Context(), outboundTimeout)
	defer cancel()
	result, err := h.Service.Submit(ctx, organizationID, r.PathValue("projectId"), input.URLs)
	if err != nil {
		h.writeError(w, r, "submit indexnow urls", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

// start answers 503 when IndexNow is not configured and returns the
// project's organization.
func (h handler) start(w http.ResponseWriter, r *http.Request) (string, bool) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "indexnow_unavailable", "IndexNow is not available on this server.")
		return "", false
	}
	organizationID, _ := auth.ProjectOrganizationFromContext(r.Context())
	return organizationID, true
}

// extendWriteDeadline gives a request that reaches customers' sites and the
// engines longer than the server's 30 second write timeout.
func (h handler) extendWriteDeadline(w http.ResponseWriter, r *http.Request) {
	if err := http.NewResponseController(w).SetWriteDeadline(time.Now().Add(outboundTimeout + 10*time.Second)); err != nil {
		h.Logger.WarnContext(r.Context(), "extend indexnow write deadline", "err", err)
	}
}

func (h handler) writeError(w http.ResponseWriter, r *http.Request, operation string, err error) {
	switch {
	case errors.Is(err, ErrNotIncluded):
		httpx.WriteError(w, http.StatusPaymentRequired, "plan_required", "IndexNow is available on Basic, Pro and Agency.")
	case errors.Is(err, ErrProjectNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "This project is no longer available.")
	case errors.Is(err, ErrNoDomain):
		httpx.WriteError(w, http.StatusBadRequest, "no_website", "Add the project's website in project settings first.")
	case errors.Is(err, ErrNotSetUp):
		httpx.WriteError(w, http.StatusConflict, "indexnow_not_set_up", "Set up IndexNow for this project first.")
	case errors.Is(err, ErrRateLimited):
		httpx.WriteError(w, http.StatusTooManyRequests, "rate_limited", "This workspace used its 60 IndexNow requests for this hour. Try again next hour.")
	default:
		if r.Context().Err() != nil {
			return
		}
		h.Logger.ErrorContext(r.Context(), operation, "err", err)
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "Something went wrong. Please try again.")
	}
}

func decodeEmpty(w http.ResponseWriter, r *http.Request) bool {
	var empty map[string]json.RawMessage
	if !decode(w, r, &empty) {
		return false
	}
	if len(empty) != 0 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "This request takes no fields.")
		return false
	}
	return true
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBody))
	trimmed := bytes.TrimSpace(body)
	if err != nil || len(trimmed) == 0 || trimmed[0] != '{' {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Send a JSON object.")
		return false
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Send a JSON object with the expected fields.")
		return false
	}
	return true
}
