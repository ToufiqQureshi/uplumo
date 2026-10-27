package aisearch

import (
	"errors"
	"net/http"
	"strings"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/httpx"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

func mountTracking(mux *http.ServeMux, d Deps) {
	for _, route := range []struct{ method, path, action string }{
		{"GET", "", "state"}, {"GET", "/runs/{runId}", "answers"}, {"POST", "/save", "save"}, {"POST", "/estimate", "estimate"}, {"POST", "/run", "run"},
	} {
		mux.Handle(route.method+" /api/v1/projects/{projectId}/ai-search/tracking"+route.path, d.WithSession(d.WithProjectAccess(trackingHandler(d, route.action))))
	}
	mux.Handle("POST /api/v1/projects/{projectId}/ai-search/tracking/runs/{runId}/sentiment", d.WithSession(d.WithProjectAccess(sentimentHandler(d))))
}

func sentimentHandler(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.Sentiment == nil {
			httpx.WriteError(w, 503, "sentiment_unavailable", "AI sentiment is not available on this server.")
			return
		}
		org, ok := auth.ProjectOrganizationFromContext(r.Context())
		if !ok {
			httpx.WriteError(w, 403, "no_organization", "Open a project first.")
			return
		}
		var in struct{}
		if err := httpx.DecodeJSONLimit(r.Body, 1024, &in); err != nil {
			httpx.WriteError(w, 400, "invalid_request", "Send an empty request body.")
			return
		}
		key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
		if len(key) < 8 || len(key) > 128 {
			httpx.WriteError(w, 400, "invalid_request", "A valid request key is required.")
			return
		}
		out, err := d.Sentiment.Classify(r.Context(), org, r.PathValue("projectId"), r.PathValue("runId"), key)
		switch {
		case err == nil:
			httpx.WriteJSON(w, 200, out)
		case errors.Is(err, errTrackingPlan):
			httpx.WriteError(w, 402, "payment_required", err.Error())
		case errors.Is(err, errSentimentNoCredits):
			httpx.WriteError(w, 402, "insufficient_credits", spend.Message(spend.ErrNoCredits))
		case errors.Is(err, errSentimentBusy):
			httpx.WriteError(w, 409, "sentiment_running", "These answers are already being classified. Wait a few seconds.")
		case errors.Is(err, errTrackingMissing):
			httpx.WriteError(w, 404, "not_found", "That check was not found.")
		case errors.Is(err, errSentimentUnusable):
			httpx.WriteError(w, 502, "sentiment_unusable", "The AI answer could not be read. Nothing was charged; try again.")
		case errors.Is(err, errSentimentUnavailable):
			httpx.WriteError(w, 503, "sentiment_unavailable", "AI sentiment is not available on this server.")
		default:
			if r.Context().Err() == nil {
				d.Logger.ErrorContext(r.Context(), "AI sentiment failed", "err", err)
			}
			httpx.WriteError(w, 500, "internal", "Could not classify the answers. Try again.")
		}
	}
}
func trackingHandler(d Deps, action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.Tracking == nil {
			httpx.WriteError(w, 503, "tracking_unavailable", "AI tracking is unavailable.")
			return
		}
		org, ok := auth.ProjectOrganizationFromContext(r.Context())
		if !ok {
			httpx.WriteError(w, 403, "no_organization", "Open a project first.")
			return
		}
		project := r.PathValue("projectId")
		var out any
		var err error
		switch action {
		case "state":
			out, err = d.Tracking.State(r.Context(), org, project)
		case "answers":
			out, err = d.Tracking.Answers(r.Context(), org, project, r.PathValue("runId"))
		case "save", "estimate":
			var cfg TrackingConfig
			err = httpx.DecodeJSONLimit(r.Body, 64<<10, &cfg)
			if err != nil {
				err = badRequest("Enter a valid tracking configuration.")
				break
			}
			if err = cfg.validate(); err != nil {
				break
			}
			if action == "estimate" {
				out = cfg.estimate()
			} else {
				err = d.Tracking.Save(r.Context(), org, project, cfg)
				if err == nil {
					out, err = d.Tracking.State(r.Context(), org, project)
				}
			}
		case "run":
			var in struct {
				MaxCredits int64 `json:"maxCredits"`
			}
			if err = httpx.DecodeJSONLimit(r.Body, 1024, &in); err != nil {
				err = badRequest("Approve the check's credit ceiling.")
				break
			}
			key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
			if len(key) < 8 || len(key) > 128 {
				err = badRequest("A valid request key is required.")
				break
			}
			var id string
			id, err = d.Tracking.Start(r.Context(), org, project, key, false, in.MaxCredits)
			out = map[string]string{"runId": id}
		}
		var invalid badRequest
		switch {
		case err == nil:
			httpx.WriteJSON(w, 200, out)
		case errors.As(err, &invalid):
			httpx.WriteError(w, 400, "invalid_request", invalid.Error())
		case errors.Is(err, errTrackingPlan):
			httpx.WriteError(w, 402, "payment_required", err.Error())
		case errors.Is(err, errTrackingBusy):
			httpx.WriteError(w, 409, "check_running", err.Error())
		case errors.Is(err, errTrackingMissing):
			httpx.WriteError(w, 404, "not_found", "Save tracking first.")
		default:
			d.Logger.ErrorContext(r.Context(), "AI tracking request failed", "err", err)
			httpx.WriteError(w, 500, "internal", "Could not load AI tracking. Try again.")
		}
	}
}
