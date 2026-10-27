package billing

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/toufiqqureshi/uplumo/backend/internal/plans"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/pgdb"
)

// TopUpCheckout describes a server-priced Razorpay order for one pack.
type TopUpCheckout struct {
	ID       string `json:"id"`
	PackID   string `json:"packId"`
	OrderID  string `json:"orderId"`
	KeyID    string `json:"keyId"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	Credits  int64  `json:"credits"`
	Sites    int    `json:"sites"`
	Status   string `json:"status"`
}

// planCreditLifetime stands for "until the paid plan ends": a pack credit
// grant carries it as its expiry, and the hourly credit worker ends it once
// the plan has ended (expirePackCreditsOfEndedPlans).
const planCreditLifetime = 50 * 365 * 24 * time.Hour

// planEndGrace is how long a workspace without a paying subscription keeps
// its pack credits, so a plan change done by hand (cancel, then subscribe
// again) does not wipe them.
const planEndGrace = 24 * time.Hour

var errTopUpPending = errors.New("a checkout is already being prepared; try again shortly")
var errTopUpUnavailable = errors.New("this pack is unavailable")
var errTopUpPlan = errors.New("your plan does not include this pack")

// TopUpOffers lists the packs orgID's plan may buy, and none while payments
// are not configured.
func (s *Service) TopUpOffers(ctx context.Context, orgID string) ([]plans.Pack, error) {
	offers := []plans.Pack{}
	if !s.PaymentsConfigured() {
		return offers, nil
	}
	definition, err := s.entitlement(ctx, orgID)
	if err != nil {
		return nil, err
	}
	for _, pack := range plans.Packs() {
		if definition.Allows(pack.Feature) {
			offers = append(offers, pack)
		}
	}
	return offers, nil
}

// CheckoutTopUp reuses a pending order and never grants anything before a
// captured payment.
func (s *Service) CheckoutTopUp(ctx context.Context, org, packID string) (TopUpCheckout, error) {
	if !s.PaymentsConfigured() {
		return TopUpCheckout{}, ErrPaymentUnavailable
	}
	p, ok := plans.GetPack(packID)
	if !ok {
		return TopUpCheckout{}, errTopUpUnavailable
	}
	allowed, err := s.Allows(ctx, org, p.Feature)
	if err != nil {
		return TopUpCheckout{}, err
	}
	if !allowed {
		return TopUpCheckout{}, errTopUpPlan
	}
	days := 0
	if p.Sites > 0 {
		days = plans.SiteAddOnDays
	}
	out := TopUpCheckout{ID: "tup_" + rand.Text(), PackID: p.ID, KeyID: s.razorpay.KeyID(), Amount: p.PriceINR * 100, Currency: "INR", Credits: p.Credits, Sites: p.Sites, Status: "creating"}
	fresh := false
	err = pgdb.InTx(ctx, s.repo.db, func(tx pgx.Tx) error {
		var orgID string
		if err := tx.QueryRow(ctx, `SELECT id FROM organization WHERE id=$1 FOR UPDATE`, org).Scan(&orgID); err != nil {
			return err
		}
		// An interrupted preparation never reached the browser. Keep any order
		// binding for later signed captures, but release the checkout slot.
		if _, err := tx.Exec(ctx, `UPDATE go_billing_topups SET status='failed' WHERE organization_id=$1 AND status='creating' AND created_at<now()-interval '5 minutes'`, org); err != nil {
			return err
		}
		var existing TopUpCheckout
		err := tx.QueryRow(ctx, `SELECT id,pack_id,COALESCE(order_id,''),amount_paise,credits,sites,status FROM go_billing_topups WHERE organization_id=$1 AND status IN ('creating','ready')`, org).Scan(&existing.ID, &existing.PackID, &existing.OrderID, &existing.Amount, &existing.Credits, &existing.Sites, &existing.Status)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
		case err != nil:
			return err
		case existing.Status == "creating":
			return errTopUpPending
		case existing.PackID == packID && existing.Amount == out.Amount:
			existing.KeyID = out.KeyID
			existing.Currency = out.Currency
			out = existing
			return nil
		default:
			// The customer closed the payment window and chose another pack.
			// Release the slot; the old order stays bound to this row, so a
			// late signed capture on it still grants what it paid for.
			if _, err := tx.Exec(ctx, `UPDATE go_billing_topups SET status='failed' WHERE id=$1`, existing.ID); err != nil {
				return err
			}
		}
		_, err = tx.Exec(ctx, `INSERT INTO go_billing_topups(id,organization_id,pack_id,amount_paise,credits,sites,expires_after_days) VALUES($1,$2,$3,$4,$5,$6,$7)`, out.ID, org, p.ID, out.Amount, out.Credits, out.Sites, days)
		fresh = err == nil
		return err
	})
	if err != nil || !fresh {
		return out, err
	}
	order, err := s.razorpay.CreateOrder(ctx, out.ID, out.Amount)
	// Persist even if the browser closed. A ready order is reused on the next
	// checkout, and only the signed captured-payment webhook grants anything.
	persist, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err != nil {
		_, writeErr := s.repo.db.Exec(persist, `UPDATE go_billing_topups SET status='failed' WHERE id=$1 AND status='creating'`, out.ID)
		return TopUpCheckout{}, errors.Join(ErrPaymentProvider, err, writeErr)
	}
	tag, err := s.repo.db.Exec(persist, `UPDATE go_billing_topups SET order_id=$2,status=CASE WHEN status='creating' THEN 'ready' ELSE status END WHERE id=$1`, out.ID, order.ID)
	if err != nil {
		return TopUpCheckout{}, fmt.Errorf("save top-up order: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return TopUpCheckout{}, ErrPaymentProvider
	}
	out.OrderID = order.ID
	out.Status = "ready"
	return out, nil
}

// webhookPayment is the payment entity of a payment.* webhook.
type webhookPayment struct {
	Payload struct {
		Payment *struct {
			Entity struct {
				ID       string `json:"id"`
				OrderID  string `json:"order_id"`
				Amount   int64  `json:"amount"`
				Currency string `json:"currency"`
				Status   string `json:"status"`
				Captured bool   `json:"captured"`
			} `json:"entity"`
		} `json:"payment"`
	} `json:"payload"`
}

// captureTopUpPayment captures an authorized payment on one of our pack
// orders. Razorpay captures by itself only within its auto-capture window;
// a slower authorization (a late UPI approval) would otherwise take the
// customer's money, grant nothing and be refunded days later, while the
// customer pays again. The capture's payment.captured webhook then grants
// the pack. Payments on orders that are not ours are left alone.
func (s *Service) captureTopUpPayment(ctx context.Context, body []byte) (WebhookResult, error) {
	var e webhookPayment
	if err := json.Unmarshal(body, &e); err != nil {
		return "", ErrInvalidEvent
	}
	if e.Payload.Payment == nil || e.Payload.Payment.Entity.OrderID == "" {
		return WebhookIgnored, nil
	}
	p := e.Payload.Payment.Entity
	if p.ID == "" || p.Status != "authorized" {
		return WebhookIgnored, nil
	}
	var status string
	var amount int64
	err := s.repo.db.QueryRow(ctx, `SELECT status,amount_paise FROM go_billing_topups WHERE order_id=$1`, p.OrderID).Scan(&status, &amount)
	if errors.Is(err, pgx.ErrNoRows) || status == "paid" {
		return WebhookIgnored, nil
	}
	if err != nil {
		return "", fmt.Errorf("load top-up for order %s: %w", p.OrderID, err)
	}
	if p.Currency != "INR" || p.Amount != amount {
		return "", ErrInvalidEvent
	}
	if _, err := s.razorpay.CapturePayment(ctx, p.ID, amount); err != nil {
		return "", fmt.Errorf("%w: %w", ErrPaymentProvider, err)
	}
	return WebhookProcessed, nil
}

// applyTopUpPayment runs only after the original webhook signature is verified.
// It binds a captured payment to our stored order/amount; browser callbacks,
// notes and duplicate event ids cannot create grants.
func (s *Service) applyTopUpPayment(ctx context.Context, body []byte) (WebhookResult, error) {
	var e webhookPayment
	if err := json.Unmarshal(body, &e); err != nil {
		return "", ErrInvalidEvent
	}
	if e.Payload.Payment == nil {
		return WebhookIgnored, nil
	}
	p := e.Payload.Payment.Entity
	if p.OrderID == "" {
		return WebhookIgnored, nil
	}
	if p.ID == "" || p.Status != "captured" || !p.Captured {
		return "", ErrInvalidEvent
	}
	result := WebhookIgnored
	err := pgdb.InTx(ctx, s.repo.db, func(tx pgx.Tx) error {
		var id, org, status string
		var amount, credits int64
		var sites, days int
		err := tx.QueryRow(ctx, `SELECT id,organization_id,status,amount_paise,credits,sites,expires_after_days FROM go_billing_topups WHERE order_id=$1 FOR UPDATE`, p.OrderID).Scan(&id, &org, &status, &amount, &credits, &sites, &days)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		if p.Currency != "INR" || p.Amount != amount {
			return ErrInvalidEvent
		}
		if status == "paid" {
			result = WebhookDuplicate
			return nil
		}
		now := time.Now().UTC()
		if credits > 0 {
			expiry := now.Add(planCreditLifetime)
			if days > 0 {
				// A pack bought before packs lasted for the plan keeps the
				// validity it was sold with.
				expiry = now.AddDate(0, 0, days)
			}
			if err = grantCreditsExpiringTx(ctx, tx, org, credits, "topup:"+p.ID, &expiry); err != nil {
				return err
			}
		}
		var sitesUntil *time.Time
		if sites > 0 {
			until := now.AddDate(0, 0, days)
			sitesUntil = &until
		}
		_, err = tx.Exec(ctx, `UPDATE go_billing_topups SET status='paid',payment_id=$2,sites_until=$3 WHERE id=$1`, id, p.ID, sitesUntil)
		result = WebhookProcessed
		return err
	})
	return result, err
}

func (s *Service) topUpStatus(ctx context.Context, org, id string) (string, error) {
	var status string
	err := s.repo.db.QueryRow(ctx, `SELECT status FROM go_billing_topups WHERE id=$1 AND organization_id=$2`, id, org).Scan(&status)
	return status, err
}

// ExpirePackCreditsOfEndedPlans ends the pack credits of every workspace
// whose paid plan has ended: no subscription in a paying state for longer
// than planEndGrace. The ended grants then expire through ExpireCredits.
func (s *CreditLedgerService) ExpirePackCreditsOfEndedPlans(ctx context.Context) (int64, error) {
	tag, err := s.db.Exec(ctx, `
		UPDATE go_billing_credit_grants g SET expires_at = now()
		WHERE g.top_up AND g.remaining_credits > 0 AND g.expires_at > now()
			AND NOT EXISTS (
				SELECT 1 FROM go_billing_subscriptions s
				WHERE s.organization_id = g.organization_id
					AND (s.status = ANY($1) OR s.updated_at > now() - $2::interval))`,
		proStatuses, fmt.Sprintf("%d seconds", int(planEndGrace.Seconds())),
	)
	if err != nil {
		return 0, fmt.Errorf("end pack credits of ended plans: %w", err)
	}
	if n := tag.RowsAffected(); n > 0 {
		slog.InfoContext(ctx, "pack credits ended with their plan", "grants", n)
	}
	return tag.RowsAffected(), nil
}
