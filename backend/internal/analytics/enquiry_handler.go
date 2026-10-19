package analytics

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/httpx"
)

func mountEnquiries(mux *http.ServeMux, d Deps) {
	for _, route := range []struct{ method, path, action string }{
		{"GET", "", "report"}, {"POST", "/goal", "goal"}, {"POST", "/changes", "save"}, {"GET", "/changes/{changeId}", "compare"}, {"POST", "/changes/{changeId}/delete", "delete"},
	} {
		mux.Handle(route.method+" /api/v1/projects/{projectId}/analytics/enquiries"+route.path, d.WithSession(d.WithProjectAccess(enquiryHandler(d, route.action))))
	}
}
func enquiryHandler(d Deps, action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.Service == nil {
			httpx.WriteError(w, 503, "unavailable", "Enquiry tracking is unavailable.")
			return
		}
		var out any = map[string]bool{"saved": true}
		var err error
		project := r.PathValue("projectId")
		switch action {
		case "report":
			out, err = d.Service.Enquiries(r.Context(), project)
		case "compare":
			out, err = d.Service.CompareEnquiryChange(r.Context(), project, r.PathValue("changeId"))
		case "delete":
			err = d.Service.DeleteEnquiryChange(r.Context(), project, r.PathValue("changeId"))
		case "goal":
			var in struct {
				EventName string `json:"eventName"`
			}
			if err = httpx.DecodeJSONLimit(r.Body, 4096, &in); err != nil {
				err = errEnquiryInput
				break
			}
			err = d.Service.SaveEnquiryGoal(r.Context(), project, in.EventName)
		case "save":
			var in struct {
				Path string `json:"path"`
				Note string `json:"note"`
			}
			if err = httpx.DecodeJSONLimit(r.Body, 8192, &in); err != nil {
				err = errEnquiryInput
				break
			}
			var id string
			id, err = d.Service.SaveEnquiryChange(r.Context(), project, in.Path, in.Note, strings.TrimSpace(r.Header.Get("Idempotency-Key")))
			out = map[string]string{"id": id}
		}
		switch {
		case err == nil:
			httpx.WriteJSON(w, 200, out)
		case errors.Is(err, errEnquiryInput):
			httpx.WriteError(w, 400, "invalid_request", "Choose a goal event (1–64 characters), a page path such as /services, and a change note up to 500 characters.")
		case errors.Is(err, errChangeLimit):
			httpx.WriteError(w, 409, "baseline_limit", err.Error())
		case errors.Is(err, pgx.ErrNoRows):
			httpx.WriteError(w, 404, "not_found", "Measurement baseline not found.")
		default:
			d.Logger.ErrorContext(r.Context(), "enquiry evidence request failed", "err", err)
			httpx.WriteError(w, 500, "internal", "Could not load enquiry evidence. Try again.")
		}
	}
}
