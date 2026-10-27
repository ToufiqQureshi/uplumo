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
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

// Reporter builds a project's report. Store is the production one.
type Reporter interface {
	Report(ctx context.Context, organizationID, projectID string) (Report, error)
}

// AgentRunner runs the paid agent and lists its saved results. *Agent is
// the production one.
type AgentRunner interface {
	Offer(ctx context.Context, organizationID string) (AgentOffer, error)
	Run(ctx context.Context, organizationID, projectID string, input AgentInput) (AgentResult, error)
	Results(ctx context.Context, organizationID, projectID string) ([]AgentResult, error)
}

// Deps wires the routes. Reporter is nil when the database is not
// configured; the routes then answer 503. Agent is nil without OpenRouter:
// the report says the agent is unavailable and its routes answer 503.
type Deps struct {
	Logger            *slog.Logger
	Reporter          Reporter
	Agent             AgentRunner
	WithSession       func(http.Handler) http.Handler
	WithProjectAccess func(http.Handler) http.Handler
}

// Mount registers the routes. Any member of the workspace may call them;
// only agent/run spends credits.
func Mount(mux *http.ServeMux, deps Deps) {
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	h := handler{deps}
	protect := func(next http.HandlerFunc) http.Handler { return deps.WithSession(deps.WithProjectAccess(next)) }
	mux.Handle("POST /api/v1/projects/{projectId}/cite-fix/report", protect(h.report))
	mux.Handle("POST /api/v1/projects/{projectId}/cite-fix/agent/run", protect(h.run))
	mux.Handle("POST /api/v1/projects/{projectId}/cite-fix/agent/results", protect(h.results))
}

type handler struct{ Deps }

// maxRequestBytes is generous for the small objects the routes take.
const maxRequestBytes = 4 << 10

func (h handler) report(w http.ResponseWriter, r *http.Request) {
	if h.Reporter == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "cite_fix_unavailable", "Cite Fix is not available on this server.")
		return
	}
	var input struct{}
	organizationID, ok := h.begin(w, r, &input)
	if !ok {
		return
	}
	report, err := h.Reporter.Report(r.Context(), organizationID, r.PathValue("projectId"))
	if err == nil {
		report.Agent, err = h.offer(r.Context(), organizationID)
	}
	if err != nil {
		h.internal(w, r, "build cite fix report", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, report)
}

// offer describes the agent; without one it is unavailable at its prices.
func (h handler) offer(ctx context.Context, organizationID string) (AgentOffer, error) {
	if h.Agent == nil {
		return (*Agent)(nil).Offer(ctx, organizationID)
	}
	return h.Agent.Offer(ctx, organizationID)
}

func (h handler) run(w http.ResponseWriter, r *http.Request) {
	if h.Agent == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "cite_fix_agent_unavailable", "The Cite Fix agent is not available on this server.")
		return
	}
	var input AgentInput
	organizationID, ok := h.begin(w, r, &input)
	if !ok {
		return
	}
	result, err := h.Agent.Run(r.Context(), organizationID, r.PathValue("projectId"), input)
	if err == nil {
		httpx.WriteJSON(w, http.StatusOK, result)
		return
	}
	if validation, ok := errors.AsType[*AgentValidationError](err); ok {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", validation.Message)
		return
	}
	if spend.WriteHTTPError(w, err) {
		return
	}
	switch {
	case errors.Is(err, ErrAgentUnavailable):
		httpx.WriteError(w, http.StatusServiceUnavailable, "cite_fix_agent_unavailable", "The Cite Fix agent is not available on this server.")
	case errors.Is(err, ErrAgentPlanRequired):
		httpx.WriteError(w, http.StatusPaymentRequired, "plan_required", "The Cite Fix agent is available on every paid plan.")
	case errors.Is(err, ErrAgentPageNotFound):
		httpx.WriteError(w, http.StatusNotFound, "page_not_found", "This page is not in your latest scan. Scan your site again, then try.")
	case errors.Is(err, ErrAgentBusy):
		httpx.WriteError(w, http.StatusConflict, "agent_busy", "This page is already being worked on. Wait for it to finish.")
	case errors.Is(err, ErrAgentRateLimited):
		httpx.WriteError(w, http.StatusTooManyRequests, "rate_limited", "You have run the agent on 20 pages in the last 10 minutes. Wait a few minutes, then continue.")
	case errors.Is(err, ErrAgentPageFetch):
		h.Logger.WarnContext(r.Context(), "cite fix agent could not read the page", "err", err)
		httpx.WriteError(w, http.StatusUnprocessableEntity, "page_unreadable", "We could not read this page right now (it may have moved or be down). No credits were used.")
	case errors.Is(err, ErrAgentFailed):
		h.Logger.ErrorContext(r.Context(), "cite fix agent failed", "err", err)
		httpx.WriteError(w, http.StatusBadGateway, "agent_failed", "The AI could not finish this page. You were not charged; try again.")
	default:
		h.internal(w, r, "run cite fix agent", err)
	}
}

func (h handler) results(w http.ResponseWriter, r *http.Request) {
	if h.Agent == nil {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"results": []AgentResult{}})
		return
	}
	var input struct{}
	organizationID, ok := h.begin(w, r, &input)
	if !ok {
		return
	}
	results, err := h.Agent.Results(r.Context(), organizationID, r.PathValue("projectId"))
	if err != nil {
		h.internal(w, r, "list cite fix agent results", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"results": results})
}

// begin decodes the body and returns the project's workspace.
func (h handler) begin(w http.ResponseWriter, r *http.Request, dst any) (string, bool) {
	if !decode(w, r, dst) {
		return "", false
	}
	organizationID, ok := auth.ProjectOrganizationFromContext(r.Context())
	if !ok || organizationID == "" {
		// Project access already refused a foreign project; this guards the
		// plain handler path.
		httpx.WriteError(w, http.StatusNotFound, "project_not_found", "Project not found.")
		return "", false
	}
	return organizationID, true
}

func (h handler) internal(w http.ResponseWriter, r *http.Request, what string, err error) {
	if r.Context().Err() == nil {
		h.Logger.ErrorContext(r.Context(), what, "project_id", r.PathValue("projectId"), "err", err)
	}
	httpx.WriteError(w, http.StatusInternalServerError, "internal", "Something went wrong. Please try again.")
}

// decode accepts exactly one JSON object with no unknown fields.
func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRequestBytes))
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
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "The request must contain exactly one JSON value.")
		return false
	}
	return true
}
