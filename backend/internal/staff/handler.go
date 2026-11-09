package staff

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/billing"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/httpx"
)

// Checker reports whether a session user is verified staff (customers.Service).
type Checker interface {
	IsStaff(context.Context, auth.User) (bool, error)
}

// Deps wires the staff routes to their services and session middleware.
type Deps struct {
	Staff       Checker
	Stats       Stats
	Actions     Actions
	Billing     *billing.Service
	Logger      *slog.Logger
	WithSession func(http.Handler) http.Handler
}

// Mount registers every staff console route behind the staff gate.
func Mount(mux *http.ServeMux, d Deps) {
	get := func(path string, fn func(context.Context) (any, error)) {
		mux.Handle("GET "+path, d.WithSession(d.staffOnly(func(w http.ResponseWriter, r *http.Request, _ auth.User) {
			v, err := fn(r.Context())
			if err != nil {
				d.fail(w, r, err)
				return
			}
			httpx.WriteJSON(w, http.StatusOK, v)
		})))
	}
	get("/api/v1/staff/overview", func(ctx context.Context) (any, error) { return d.Stats.Overview(ctx) })
	get("/api/v1/staff/subscriptions", func(ctx context.Context) (any, error) {
		v, err := d.Stats.Subscriptions(ctx)
		return map[string]any{"subscriptions": v}, err
	})
	get("/api/v1/staff/marketing", func(ctx context.Context) (any, error) { return d.Stats.Marketing(ctx) })
	get("/api/v1/staff/finance", func(ctx context.Context) (any, error) { return d.Stats.Finance(ctx) })
	get("/api/v1/staff/actions", func(ctx context.Context) (any, error) {
		v, err := d.Actions.Recent(ctx)
		return map[string]any{"actions": v}, err
	})
	mux.Handle("POST /api/v1/staff/users/block", d.WithSession(d.staffOnly(d.userAction(func(ctx context.Context, actor auth.User, in userInput) error {
		return d.Actions.Block(ctx, actor.ID, in.UserID, in.Reason)
	}))))
	mux.Handle("POST /api/v1/staff/users/unblock", d.WithSession(d.staffOnly(d.userAction(func(ctx context.Context, actor auth.User, in userInput) error {
		return d.Actions.Unblock(ctx, actor.ID, in.UserID)
	}))))
	mux.Handle("POST /api/v1/staff/users/delete", d.WithSession(d.staffOnly(d.userAction(func(ctx context.Context, actor auth.User, in userInput) error {
		return d.Actions.Delete(ctx, actor.ID, in.UserID, in.ConfirmEmail)
	}))))
	mux.Handle("POST /api/v1/staff/payments/sync", d.WithSession(d.staffOnly(d.sync)))
	mux.Handle("GET /api/v1/staff/payments/failed", d.WithSession(d.staffOnly(d.failedPayments)))
}

// staffOnly answers 404 to everyone but verified staff, so the console's
// existence is not revealed.
func (d Deps) staffOnly(next func(http.ResponseWriter, *http.Request, auth.User)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := auth.UserFromContext(r.Context())
		if !ok || u.ID == "" || d.Staff == nil {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Not found.")
			return
		}
		staff, err := d.Staff.IsStaff(r.Context(), u)
		if err != nil {
			d.fail(w, r, err)
			return
		}
		if !staff {
			httpx.WriteError(w, http.StatusNotFound, "not_found", "Not found.")
			return
		}
		w.Header().Set("Cache-Control", "private, no-store")
		next(w, r, u)
	})
}

type userInput struct {
	UserID       string `json:"userId"`
	Reason       string `json:"reason"`
	ConfirmEmail string `json:"confirmEmail"`
}

func (d Deps) userAction(fn func(context.Context, auth.User, userInput) error) func(http.ResponseWriter, *http.Request, auth.User) {
	return func(w http.ResponseWriter, r *http.Request, actor auth.User) {
		var in userInput
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&in); err != nil || in.UserID == "" {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_request", "Choose a user.")
			return
		}
		if err := fn(r.Context(), actor, in); err != nil {
			d.fail(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}

func (d Deps) sync(w http.ResponseWriter, r *http.Request, actor auth.User) {
	if d.Billing == nil || !d.Billing.PaymentsConfigured() {
		httpx.WriteError(w, http.StatusServiceUnavailable, "unavailable", "Razorpay is not configured on this server.")
		return
	}
	result, err := d.Billing.SyncSubscriptionPayments(r.Context())
	if err != nil {
		d.fail(w, r, err)
		return
	}
	if err := d.Actions.LogSync(r.Context(), actor.ID, fmt.Sprintf("%d subscriptions, %d added, %d failed, %d refunds updated", result.Subscriptions, result.Added, result.Failed, result.RefundsUpdated)); err != nil {
		d.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

// failedPayments lists the last 30 days of failed payments from Razorpay,
// read when the console asks; nothing is stored.
func (d Deps) failedPayments(w http.ResponseWriter, r *http.Request, _ auth.User) {
	if d.Billing == nil || !d.Billing.PaymentsConfigured() {
		httpx.WriteError(w, http.StatusServiceUnavailable, "unavailable", "Razorpay is not configured on this server.")
		return
	}
	result, err := d.Billing.ListFailedPayments(r.Context(), time.Now())
	if errors.Is(err, billing.ErrPaymentProvider) {
		if d.Logger != nil {
			d.Logger.WarnContext(r.Context(), "list failed payments", "err", err)
		}
		httpx.WriteError(w, http.StatusBadGateway, "payment_provider_error", "Could not reach Razorpay. Try again.")
		return
	}
	if err != nil {
		d.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (d Deps) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrUserNotFound):
		httpx.WriteError(w, http.StatusNotFound, "user_not_found", "User not found.")
	case errors.Is(err, ErrProtectedUser), errors.Is(err, ErrLiveSubscription), errors.Is(err, ErrSharedWorkspace):
		httpx.WriteError(w, http.StatusConflict, "conflict", capitalize(err.Error()))
	case errors.Is(err, ErrConfirmMismatch):
		httpx.WriteError(w, http.StatusBadRequest, "confirm_mismatch", capitalize(err.Error()))
	default:
		if d.Logger != nil {
			d.Logger.ErrorContext(r.Context(), "staff console failed", "err", err)
		}
		httpx.WriteError(w, http.StatusInternalServerError, "internal", "Something went wrong. Try again.")
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return string(s[0]-32) + s[1:] + "."
}
