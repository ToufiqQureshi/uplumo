package billing

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/toufiqqureshi/uplumo/backend/internal/plans"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/pgdb"
)

// TopUpPack is operator-configured commercial policy; no new price is activated
// until this configuration is supplied. Existing subscription prices stay canonical.
type TopUpPack struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	PriceINR         int64  `json:"priceInr"`
	Credits          int64  `json:"credits"`
	ExpiresAfterDays int    `json:"expiresAfterDays"`
}

// TopUpCheckout describes a server-priced Razorpay order.
type TopUpCheckout struct {
	ID               string `json:"id"`
	OrderID          string `json:"orderId"`
	KeyID            string `json:"keyId"`
	Amount           int64  `json:"amount"`
	Currency         string `json:"currency"`
	Credits          int64  `json:"credits"`
	ExpiresAfterDays int    `json:"expiresAfterDays"`
	Status           string `json:"status"`
}

var errTopUpPending = errors.New("a top-up checkout is already being prepared; try again shortly")
var errTopUpUnavailable = errors.New("this top-up pack is unavailable")
var errTopUpPlan = errors.New("an active paid plan is required to buy credits")

// ConfigureTopUps validates the operator's explicitly approved credit packs.
func (s *Service) ConfigureTopUps(raw string) error {
	packs := []TopUpPack{}
	if strings.TrimSpace(raw) != "" {
		decoder := json.NewDecoder(strings.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&packs); err != nil {
			return fmt.Errorf("BILLING_TOPUP_PACKS: %w", err)
		}
		if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
			return errors.New("BILLING_TOPUP_PACKS must contain one JSON array")
		}
	}
	if len(packs) > 10 {
		return errors.New("BILLING_TOPUP_PACKS accepts at most ten packs")
	}
	seen := map[string]bool{}
	for _, p := range packs {
		if p.ID == "" || len(p.ID) > 40 || p.Name == "" || len(p.Name) > 80 || seen[p.ID] || p.PriceINR < 1 || p.PriceINR > 100000 || p.Credits < 1 || p.Credits > 1000000 || p.ExpiresAfterDays < 1 || p.ExpiresAfterDays > 730 {
			return errors.New("BILLING_TOPUP_PACKS has an invalid or duplicate pack")
		}
		seen[p.ID] = true
	}
	s.topUpPacks = slices.Clone(packs)
	return nil
}

// TopUpPacks lists configured offers only while payment processing is enabled.
func (s *Service) TopUpPacks() []TopUpPack {
	if s == nil || !s.PaymentsConfigured() || len(s.topUpPacks) == 0 {
		return []TopUpPack{}
	}
	return slices.Clone(s.topUpPacks)
}

// CheckoutTopUp reuses a pending order and never grants credits before capture.
func (s *Service) CheckoutTopUp(ctx context.Context, org, packID string) (TopUpCheckout, error) {
	if !s.PaymentsConfigured() {
		return TopUpCheckout{}, ErrPaymentUnavailable
	}
	i := slices.IndexFunc(s.topUpPacks, func(p TopUpPack) bool { return p.ID == packID })
	if i < 0 {
		return TopUpCheckout{}, errTopUpUnavailable
	}
	paid, err := s.Gate(plans.Keywords).HasPaidPlan(ctx, org)
	if err != nil {
		return TopUpCheckout{}, err
	}
	if !paid {
		return TopUpCheckout{}, errTopUpPlan
	}
	p := s.topUpPacks[i]
	out := TopUpCheckout{ID: "tup_" + rand.Text(), KeyID: s.razorpay.KeyID(), Amount: p.PriceINR * 100, Currency: "INR", Credits: p.Credits, ExpiresAfterDays: p.ExpiresAfterDays, Status: "creating"}
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
		var existingPack string
		err := tx.QueryRow(ctx, `SELECT id,pack_id,COALESCE(order_id,''),amount_paise,credits,expires_after_days,status FROM go_billing_topups WHERE organization_id=$1 AND status IN ('creating','ready')`, org).Scan(&out.ID, &existingPack, &out.OrderID, &out.Amount, &out.Credits, &out.ExpiresAfterDays, &out.Status)
		if err == nil {
			if out.Status == "creating" {
				return errTopUpPending
			}
			if existingPack != packID {
				return fmt.Errorf("%w: finish the existing credit-pack checkout first", errTopUpPending)
			}
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO go_billing_topups(id,organization_id,pack_id,amount_paise,credits,expires_after_days) VALUES($1,$2,$3,$4,$5,$6)`, out.ID, org, p.ID, out.Amount, out.Credits, out.ExpiresAfterDays)
		fresh = err == nil
		return err
	})
	if err != nil || !fresh {
		return out, err
	}
	order, err := s.razorpay.CreateOrder(ctx, out.ID, out.Amount)
	// Persist even if the browser closed. A ready order is reused on the next
	// checkout, and only the signed captured-payment webhook grants credits.
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

// applyTopUpPayment runs only after the original webhook signature is verified.
// It binds a captured payment to our stored order/amount; browser callbacks,
// notes and duplicate event ids cannot create credit grants.
func (s *Service) applyTopUpPayment(ctx context.Context, body []byte) (WebhookResult, error) {
	var e struct {
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
		var days int
		err := tx.QueryRow(ctx, `SELECT id,organization_id,status,amount_paise,credits,expires_after_days FROM go_billing_topups WHERE order_id=$1 FOR UPDATE`, p.OrderID).Scan(&id, &org, &status, &amount, &credits, &days)
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
		expiry := time.Now().UTC().AddDate(0, 0, days)
		if err = grantCreditsExpiringTx(ctx, tx, org, credits, "topup:"+p.ID, &expiry); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE go_billing_topups SET status='paid',payment_id=$2 WHERE id=$1`, id, p.ID)
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
