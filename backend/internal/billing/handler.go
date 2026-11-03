package billing

import (
	"cmp"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/plans"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/httpx"
)

const maxWebhookBody = 256 << 10

type statusResponse struct {
	Status
	// CheckoutCurrency is what a new checkout from this request is billed in.
	CheckoutCurrency   string    `json:"checkoutCurrency"`
	PaymentsConfigured bool      `json:"paymentsConfigured"`
	AvailablePlans     []string  `json:"availablePlans"`
	Usage              PlanUsage `json:"usage"`
}

// Deps contains billing services and the root router's authentication middleware.
type Deps struct {
	Logger      *slog.Logger
	Service     *Service
	WithSession func(http.Handler) http.Handler
	Credits     *CreditLedgerService
	// Country is the ISO country code of the request's trusted client
	// address, or "" when unknown. It decides the checkout currency. Nil
	// means unknown for every request: everyone is billed in rupees.
	Country func(*http.Request) string
}

// checkoutCurrency is the currency a checkout from r is billed in.
func checkoutCurrency(svc *Service, country func(*http.Request) string, r *http.Request) string {
	if svc == nil || country == nil {
		return plans.INR
	}
	return svc.CurrencyFor(country(r))
}

// Mount registers billing routes on mux.
func Mount(mux *http.ServeMux, d Deps) {
	mountTopUps(mux, d)
	mux.Handle("POST /webhooks/razorpay", webhookHandler(d.Logger, d.Service))
	mux.Handle("GET /api/v1/billing/plans", d.WithSession(plansHandler(d.Service, d.Country)))
	mux.Handle("GET /api/v1/billing/status", d.WithSession(statusHandler(d.Logger, d.Service, d.Country)))
	mux.Handle("POST /api/v1/billing/checkout", d.WithSession(checkoutHandler(d.Logger, d.Service, d.Country)))
	mux.Handle("POST /api/v1/billing/cancel", d.WithSession(cancelHandler(d.Logger, d.Service, d.Country)))
	mux.Handle("GET /api/v1/billing/credits", d.WithSession(creditsHandler(d.Logger, d.Service, d.Credits)))
	mux.Handle("POST /api/v1/billing/usage-events", d.WithSession(usageEventsHandler(d.Logger, d.Credits)))
	mux.Handle("GET /api/v1/billing/usage", d.WithSession(usageHandler(d.Logger, d.Service, d.Credits, time.Now)))
}

// planInfo is a plan as the billing screens show it.
type planInfo struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	PriceINR   int      `json:"priceInr"`
	PriceUSD   int      `json:"priceUsd"`
	Summary    string   `json:"summary"`
	Highlights []string `json:"highlights"`
}

// plansHandler returns the plan catalog, so the app shows the same prices
// and promises as the public pricing page, and the currency a checkout from
// this request is billed in. Both prices are sent so a subscription already
// running in the other currency still shows what it costs.
func plansHandler(svc *Service, country func(*http.Request) string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		currency := checkoutCurrency(svc, country, r)
		catalog := plans.Catalog()
		out := make([]planInfo, len(catalog))
		for i, plan := range catalog {
			out[i] = planInfo{ID: plan.ID, Name: plan.Name, PriceINR: plan.PriceINR, PriceUSD: plan.PriceUSD, Summary: plan.Summary, Highlights: plan.HighlightsIn(currency)}
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"plans": out, "currency": currency})
	}
}

// statusHandler returns the plan of the signed-in user's active organization.
func statusHandler(logger *slog.Logger, svc *Service, country func(*http.Request) string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := billingUser(w, r, svc)
		if !ok {
			return
		}
		status, err := svc.Status(r.Context(), user.OrganizationID)
		if err != nil {
			logger.ErrorContext(r.Context(), "billing status", "err", err)
			httpx.WriteError(w, http.StatusInternalServerError, "internal", "Something went wrong. Please try again.")
			return
		}
		usage, err := svc.UsageAgainstPlan(r.Context(), user.OrganizationID)
		if err != nil {
			logger.ErrorContext(r.Context(), "billing plan usage", "err", err)
			httpx.WriteError(w, http.StatusInternalServerError, "internal", "Something went wrong. Please try again.")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, statusResponse{Status: status, CheckoutCurrency: checkoutCurrency(svc, country, r), PaymentsConfigured: svc.PaymentsConfigured(), AvailablePlans: svc.AvailablePlans(), Usage: usage})
	}
}

// checkoutHandler starts a subscription for the signed-in user's active
// organization, billed in the currency of where the request comes from. The
// browser sends the currency it showed the price in; when that is no longer
// the currency the server would bill (a VPN switched on, a trip abroad), the
// checkout is refused with 409 so the customer sees the new price before
// paying it.
func checkoutHandler(logger *slog.Logger, svc *Service, country func(*http.Request) string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := billingUser(w, r, svc)
		if !ok {
			return
		}
		if user.Role != "owner" && user.Role != "admin" {
			httpx.WriteError(w, http.StatusForbidden, "forbidden", "Only workspace owners and admins can change the plan.")
			return
		}
		var input struct {
			Plan     string `json:"plan"`
			Currency string `json:"currency"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil && !errors.Is(err, io.EOF) {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Choose a valid plan.")
			return
		}
		if input.Plan == "" {
			input.Plan = PlanPro
		}
		currency := checkoutCurrency(svc, country, r)
		// A browser built before dollar prices sends no currency; it showed
		// rupees.
		if shown := cmp.Or(input.Currency, plans.INR); shown != currency {
			httpx.WriteError(w, http.StatusConflict, "currency_changed", "Prices changed for your location. Check the new price and try again.")
			return
		}
		checkout, err := svc.CheckoutPlan(r.Context(), user.OrganizationID, input.Plan, currency)
		if errors.Is(err, ErrAlreadySubscribed) {
			httpx.WriteError(w, http.StatusConflict, "already_subscribed", "This workspace already has a subscription.")
			return
		}
		if err != nil {
			logger.ErrorContext(r.Context(), "billing checkout", "err", err)
			if errors.Is(err, ErrPlanUnavailable) {
				httpx.WriteError(w, http.StatusServiceUnavailable, "plan_unavailable", "Checkout for this plan is not configured yet. Please contact support.")
			} else if errors.Is(err, ErrPaymentUnavailable) {
				httpx.WriteError(w, http.StatusServiceUnavailable, "billing_unavailable", "Payments are not available yet. Please contact support.")
			} else if errors.Is(err, ErrPaymentProvider) {
				httpx.WriteError(w, http.StatusBadGateway, "payment_provider_error", "Could not start the payment. Please try again.")
			} else {
				httpx.WriteError(w, http.StatusInternalServerError, "internal", "Something went wrong. Please try again.")
			}
			return
		}
		httpx.WriteJSON(w, http.StatusOK, checkout)
	}
}

// cancelHandler cancels the subscription of the signed-in user's active
// organization and answers with the plan that results.
func cancelHandler(logger *slog.Logger, svc *Service, country func(*http.Request) string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := billingUser(w, r, svc)
		if !ok {
			return
		}
		if user.Role != "owner" && user.Role != "admin" {
			httpx.WriteError(w, http.StatusForbidden, "forbidden", "Only workspace owners and admins can change the plan.")
			return
		}
		status, err := svc.Cancel(r.Context(), user.OrganizationID)
		if errors.Is(err, ErrNoSubscription) {
			httpx.WriteError(w, http.StatusConflict, "no_subscription", "This workspace has no subscription to cancel.")
			return
		}
		if err != nil {
			logger.ErrorContext(r.Context(), "billing cancel", "err", err)
			if errors.Is(err, ErrPaymentUnavailable) {
				httpx.WriteError(w, http.StatusServiceUnavailable, "billing_unavailable", "Payments are not available. Please contact support to cancel.")
			} else if errors.Is(err, ErrPaymentProvider) {
				httpx.WriteError(w, http.StatusBadGateway, "payment_provider_error", "Could not cancel the plan. Please try again, or write to us.")
			} else {
				httpx.WriteError(w, http.StatusInternalServerError, "internal", "Something went wrong. Please try again.")
			}
			return
		}
		usage, err := svc.UsageAgainstPlan(r.Context(), user.OrganizationID)
		if err != nil {
			logger.ErrorContext(r.Context(), "billing plan usage after cancel", "err", err)
			httpx.WriteError(w, http.StatusInternalServerError, "internal", "Something went wrong. Please try again.")
			return
		}
		httpx.WriteJSON(w, http.StatusOK, statusResponse{Status: status, CheckoutCurrency: checkoutCurrency(svc, country, r), PaymentsConfigured: svc.PaymentsConfigured(), AvailablePlans: svc.AvailablePlans(), Usage: usage})
	}
}

// billingUser returns the signed-in user when billing is configured and the user has an active organization.
func billingUser(w http.ResponseWriter, r *http.Request, svc *Service) (auth.User, bool) {
	if svc == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "billing_unavailable", "Billing is not set up on this server.")
		return auth.User{}, false
	}
	user, ok := auth.UserFromContext(r.Context())
	if !ok || user.OrganizationID == "" {
		httpx.WriteError(w, http.StatusForbidden, "no_organization", "Open a workspace first, then try again.")
		return auth.User{}, false
	}
	return user, true
}

// webhookHandler applies Razorpay's subscription webhooks.
func webhookHandler(logger *slog.Logger, svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !svc.PaymentsConfigured() {
			httpx.WriteError(w, http.StatusServiceUnavailable, "billing_unavailable", "Payments are not configured on this server.")
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxWebhookBody))
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			httpx.WriteError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "The webhook is too large.")
			return
		}
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_event", "The webhook body could not be read.")
			return
		}

		result, err := svc.HandleWebhook(r.Context(), body, r.Header.Get("X-Razorpay-Signature"), r.Header.Get("X-Razorpay-Event-Id"))
		switch {
		case err == nil:
			httpx.WriteJSON(w, http.StatusOK, map[string]WebhookResult{"status": result})
		case errors.Is(err, ErrInvalidSignature):
			httpx.WriteError(w, http.StatusUnauthorized, "invalid_signature", "The webhook signature is not valid.")
		case errors.Is(err, ErrInvalidEvent):
			logger.WarnContext(r.Context(), "invalid razorpay webhook", "err", err)
			httpx.WriteError(w, http.StatusBadRequest, "invalid_event", "The webhook event is not valid.")
		default:
			logger.ErrorContext(r.Context(), "apply razorpay webhook", "err", err)
			httpx.WriteError(w, http.StatusInternalServerError, "internal", "Something went wrong. Please try again.")
		}
	}
}
