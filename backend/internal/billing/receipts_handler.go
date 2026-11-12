package billing

import (
	"errors"
	"net/http"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/httpx"
)

// mountReceipts registers the payments list, receipts and billing details.
// They are read from our own ledger: no Razorpay or other paid call.
func mountReceipts(mux *http.ServeMux, d Deps) {
	mux.Handle("GET /api/v1/billing/payments", d.WithSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := receiptsUser(w, r, d.Service)
		if !ok {
			return
		}
		payments, err := d.Service.Payments(r.Context(), user.OrganizationID)
		if err != nil {
			d.Logger.ErrorContext(r.Context(), "list payments", "err", err)
			httpx.WriteError(w, http.StatusInternalServerError, "internal", "Could not load your payments. Please try again.")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"payments": payments})
	})))
	mux.Handle("GET /api/v1/billing/payments/{id}/receipt", d.WithSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := receiptsUser(w, r, d.Service)
		if !ok {
			return
		}
		receipt, err := d.Service.Receipt(r.Context(), user.OrganizationID, r.PathValue("id"))
		switch {
		case errors.Is(err, errReceiptNotFound):
			httpx.WriteError(w, http.StatusNotFound, "not_found", "This receipt does not exist in this workspace.")
		case err != nil:
			d.Logger.ErrorContext(r.Context(), "read receipt", "err", err)
			httpx.WriteError(w, http.StatusInternalServerError, "internal", "Could not load the receipt. Please try again.")
		default:
			httpx.WriteJSON(w, http.StatusOK, receipt)
		}
	})))
	mux.Handle("GET /api/v1/billing/details", d.WithSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := receiptsUser(w, r, d.Service)
		if !ok {
			return
		}
		details, err := d.Service.BillingDetails(r.Context(), user.OrganizationID)
		if err != nil {
			d.Logger.ErrorContext(r.Context(), "read billing details", "err", err)
			httpx.WriteError(w, http.StatusInternalServerError, "internal", "Could not load your billing details. Please try again.")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, details)
	})))
	mux.Handle("POST /api/v1/billing/details", d.WithSession(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := receiptsUser(w, r, d.Service)
		if !ok {
			return
		}
		var in Details
		if err := httpx.DecodeJSONLimit(r.Body, 4<<10, &in); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Check your billing details and try again.")
			return
		}
		saved, err := d.Service.SaveBillingDetails(r.Context(), user.OrganizationID, in)
		if invalid, ok := errors.AsType[*detailsError](err); ok {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_details", invalid.msg)
			return
		}
		if err != nil {
			d.Logger.ErrorContext(r.Context(), "save billing details", "err", err)
			httpx.WriteError(w, http.StatusInternalServerError, "internal", "Could not save your billing details. Please try again.")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, saved)
	})))
}

// receiptsUser is the signed-in owner or admin of the active workspace;
// payments and billing details are hidden from other members.
func receiptsUser(w http.ResponseWriter, r *http.Request, svc *Service) (auth.User, bool) {
	user, ok := billingUser(w, r, svc)
	if !ok {
		return user, false
	}
	if user.Role != "owner" && user.Role != "admin" {
		httpx.WriteError(w, http.StatusForbidden, "forbidden", "Only workspace owners and admins can see payments and receipts.")
		return user, false
	}
	return user, true
}
