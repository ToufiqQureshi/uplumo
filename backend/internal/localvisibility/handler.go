package localvisibility

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/httpx"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

const maxBody = 4 << 10

var (
	uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	keyPattern  = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)
)

// Deps are what the routes need.
type Deps struct {
	Logger            *slog.Logger
	Service           *Service
	Markets           Markets
	Plans             PaidPlans
	WithSession       func(http.Handler) http.Handler
	WithProjectAccess func(http.Handler) http.Handler
}

type inputError string

func (e inputError) Error() string { return string(e) }

// Mount registers the Local Visibility routes. Only find, runs/start,
// watch (turning it on) and reply can spend credits; each needs a click.
func Mount(mux *http.ServeMux, d Deps) {
	protect := func(h http.Handler) http.Handler { return d.WithSession(d.WithProjectAccess(h)) }
	routes := map[string]func(*http.Request, string, string) (any, error){
		"overview":   d.overview,
		"find":       d.find,
		"business":   d.saveBusiness,
		"runs/start": d.start,
		"runs/get":   d.getRun,
		"watch":      d.watch,
		"reply":      d.reply,
	}
	for path, fn := range routes {
		mux.Handle("POST /api/v1/projects/{projectId}/local/"+path, protect(d.handle(fn)))
	}
}

// decode reads the body, which handle limits to maxBody.
func decode(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return inputError("The request is not valid JSON.")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return inputError("The request must contain exactly one JSON value.")
	}
	return nil
}

func (d Deps) handle(fn func(*http.Request, string, string) (any, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if d.Service == nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, "local_unavailable", "Local Visibility is not set up on this server.")
			return
		}
		org, ok := auth.ProjectOrganizationFromContext(r.Context())
		if !ok {
			httpx.WriteError(w, http.StatusInternalServerError, "internal", "Something went wrong. Please try again.")
			return
		}
		if d.Plans != nil {
			paid, err := d.Plans.HasPaidPlan(r.Context(), org)
			if err != nil {
				d.Logger.ErrorContext(r.Context(), "check local visibility plan", "err", err)
				httpx.WriteError(w, http.StatusInternalServerError, "internal", "Something went wrong. Please try again.")
				return
			}
			if !paid {
				httpx.WriteError(w, http.StatusPaymentRequired, "payment_required", "Local Visibility is on the Pro and Business plans.")
				return
			}
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxBody)
		result, err := fn(r, org, r.PathValue("projectId"))
		d.write(w, r, result, err)
	})
}

func (d Deps) write(w http.ResponseWriter, r *http.Request, result any, err error) {
	input, isInput := errors.AsType[inputError](err)
	switch {
	case err == nil:
		httpx.WriteJSON(w, http.StatusOK, result)
	case isInput:
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", input.Error())
	case errors.Is(err, ErrInvalidInput):
		httpx.WriteError(w, http.StatusBadRequest, "invalid_request", trimPrefix(err))
	case errors.Is(err, ErrNoBusiness):
		httpx.WriteError(w, http.StatusConflict, "no_business", "Find and choose your business first.")
	case errors.Is(err, ErrNeedsCheck):
		httpx.WriteError(w, http.StatusConflict, "needs_check", "Run the profile check first: its top 3 are the competitors.")
	case errors.Is(err, ErrBusy):
		httpx.WriteError(w, http.StatusConflict, "busy", "A search is already running. Wait a moment.")
	case errors.Is(err, ErrNotFound):
		httpx.WriteError(w, http.StatusNotFound, "not_found", "Not found. Reload the page.")
	case r.Context().Err() != nil:
		d.Logger.InfoContext(r.Context(), "local visibility request canceled", "err", err)
	case spend.WriteHTTPError(w, err):
	case errors.Is(err, dataforseo.ErrBillingIssue):
		d.Logger.ErrorContext(r.Context(), "DataForSEO billing issue", "err", err)
		httpx.WriteError(w, http.StatusServiceUnavailable, "provider_billing_issue", "Google data is temporarily unavailable. Please try again later.")
	default:
		d.Logger.ErrorContext(r.Context(), "local visibility request failed", "err", err)
		httpx.WriteError(w, http.StatusBadGateway, "provider_error", "Google data is temporarily unavailable. Please try again.")
	}
}

// trimPrefix drops the "invalid input: " sentinel from a message.
func trimPrefix(err error) string {
	msg := err.Error()
	prefix := ErrInvalidInput.Error() + ": "
	if len(msg) > len(prefix) && msg[:len(prefix)] == prefix {
		return msg[len(prefix):]
	}
	return "The request is not valid."
}

func (d Deps) overview(r *http.Request, _, projectID string) (any, error) {
	var req struct{}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	return d.Service.Overview(r.Context(), projectID)
}

func (d Deps) find(r *http.Request, org, projectID string) (any, error) {
	var req struct {
		Query string `json:"query"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	if d.Markets == nil {
		return nil, errors.New("project markets are not configured")
	}
	return d.Service.Find(r.Context(), d.Markets, org, projectID, req.Query)
}

func (d Deps) saveBusiness(r *http.Request, org, projectID string) (any, error) {
	var req SaveRequest
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	if d.Markets == nil {
		return nil, errors.New("project markets are not configured")
	}
	return d.Service.SaveBusiness(r.Context(), d.Markets, org, projectID, req)
}

func (d Deps) start(r *http.Request, org, projectID string) (any, error) {
	var req StartRequest
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	return d.Service.Start(r.Context(), org, projectID, req)
}

func (d Deps) getRun(r *http.Request, _, projectID string) (any, error) {
	var req struct {
		RunID string `json:"runId"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	if !uuidPattern.MatchString(req.RunID) {
		return nil, ErrNotFound
	}
	return d.Service.RunStatus(r.Context(), projectID, req.RunID)
}

func (d Deps) watch(r *http.Request, org, projectID string) (any, error) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	return d.Service.SetWatch(r.Context(), org, projectID, req.Enabled)
}

func (d Deps) reply(r *http.Request, org, projectID string) (any, error) {
	var req struct {
		ReviewID       string `json:"reviewId"`
		Rewrite        bool   `json:"rewrite"`
		IdempotencyKey string `json:"idempotencyKey"`
	}
	if err := decode(r, &req); err != nil {
		return nil, err
	}
	if req.ReviewID == "" || len(req.ReviewID) > 300 || !keyPattern.MatchString(req.IdempotencyKey) {
		return nil, inputError("reviewId and idempotencyKey are required.")
	}
	if d.Service.LLM == nil || d.Service.Meter == nil {
		return nil, inputError("AI drafts are not available on this server.")
	}
	reply, err := d.Service.DraftReply(r.Context(), org, projectID, req.ReviewID, req.Rewrite, req.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	return map[string]string{"reply": reply}, nil
}
