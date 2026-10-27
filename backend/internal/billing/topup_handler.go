package billing

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/httpx"
)

func mountTopUps(mux *http.ServeMux, d Deps) {
	mux.Handle("GET /api/v1/billing/topups", d.WithSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := billingUser(w, r, d.Service)
		if !ok {
			return
		}
		offers, err := d.Service.TopUpOffers(r.Context(), user.OrganizationID)
		if err != nil {
			d.Logger.ErrorContext(r.Context(), "list top-up packs", "err", err)
			httpx.WriteError(w, 500, "internal", "Could not load packs.")
			return
		}
		httpx.WriteJSON(w, 200, offers)
	})))
	mux.Handle("POST /api/v1/billing/topups/checkout", d.WithSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := billingUser(w, r, d.Service)
		if !ok {
			return
		}
		if user.Role != "owner" && user.Role != "admin" {
			httpx.WriteError(w, 403, "forbidden", "Only workspace owners and admins can buy packs.")
			return
		}
		var in struct {
			PackID string `json:"packId"`
		}
		if err := httpx.DecodeJSONLimit(r.Body, 1024, &in); err != nil {
			httpx.WriteError(w, 400, "invalid_request", "Choose a pack.")
			return
		}
		out, err := d.Service.CheckoutTopUp(r.Context(), user.OrganizationID, in.PackID)
		switch {
		case err == nil:
			httpx.WriteJSON(w, 200, out)
		case errors.Is(err, errTopUpPending):
			httpx.WriteError(w, 409, "checkout_pending", err.Error())
		case errors.Is(err, errTopUpUnavailable):
			httpx.WriteError(w, 400, "pack_unavailable", err.Error())
		case errors.Is(err, errTopUpPlan):
			httpx.WriteError(w, 402, "payment_required", err.Error())
		default:
			d.Logger.ErrorContext(r.Context(), "create credit top-up", "err", err)
			httpx.WriteError(w, 503, "payment_unavailable", "Could not start checkout. Try again shortly.")
		}
	})))
	mux.Handle("GET /api/v1/billing/topups/{topupId}", d.WithSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := billingUser(w, r, d.Service)
		if !ok {
			return
		}
		status, err := d.Service.topUpStatus(r.Context(), user.OrganizationID, r.PathValue("topupId"))
		if errors.Is(err, pgx.ErrNoRows) {
			httpx.WriteError(w, 404, "not_found", "Top-up not found.")
			return
		}
		if err != nil {
			d.Logger.ErrorContext(r.Context(), "read credit top-up", "err", err)
			httpx.WriteError(w, 500, "internal", "Could not load payment status.")
			return
		}
		httpx.WriteJSON(w, 200, map[string]string{"status": status})
	})))
}
