package billing

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/toufiqqureshi/uplumo/backend/internal/plans"
	"github.com/toufiqqureshi/uplumo/backend/internal/razorpay"
)

// Payment sources stored in go_billing_payments.source.
const (
	sourceWebhook = "webhook"
	sourceSync    = "razorpay_sync"
)

// execer is a pool or a transaction.
type execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// recordPayment adds one captured payment to the revenue ledger. The payment
// id is the key, so webhook redeliveries and repeated syncs never count a
// payment twice. Amounts are in the currency's subunits.
func (r repository) recordPayment(ctx context.Context, q execer, paymentID, orgID, kind, item string, amount int64, currency string, paidAt time.Time, source string) error {
	if paymentID == "" || amount <= 0 {
		return nil
	}
	if currency != plans.INR && currency != plans.USD {
		return fmt.Errorf("record payment %s: unsupported currency %q", paymentID, currency)
	}
	_, err := q.Exec(ctx, `INSERT INTO go_billing_payments(payment_id,organization_id,kind,item,amount,currency,paid_at,source)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT (payment_id) DO NOTHING`,
		paymentID, orgID, kind, item, amount, currency, paidAt.UTC(), source)
	if err != nil {
		return fmt.Errorf("record payment %s: %w", paymentID, err)
	}
	return nil
}

// recordSubscriptionCharge stores the payment a subscription.charged event carries.
func (s *Service) recordSubscriptionCharge(ctx context.Context, orgID, plan string, event razorpay.Event) error {
	if event.Event != "subscription.charged" || event.Payload.Payment == nil {
		return nil
	}
	p := event.Payload.Payment.Entity
	paidAt := time.Unix(p.CreatedAt, 0)
	if p.CreatedAt == 0 {
		paidAt = time.Unix(event.CreatedAt, 0)
	}
	return s.repo.recordPayment(ctx, s.repo.db, p.ID, orgID, "subscription", plan, p.Amount, p.Currency, paidAt, sourceWebhook)
}

// applyRefund keeps the ledger's refunded amount equal to Razorpay's. Razorpay
// sends the payment's running refunded total, so redeliveries are harmless.
func (s *Service) applyRefund(ctx context.Context, event razorpay.Event) (WebhookResult, error) {
	if event.Payload.Payment == nil || event.Payload.Payment.Entity.ID == "" {
		return WebhookIgnored, nil
	}
	p := event.Payload.Payment.Entity
	tag, err := s.repo.db.Exec(ctx, `UPDATE go_billing_payments SET amount_refunded=LEAST($2,amount) WHERE payment_id=$1`, p.ID, p.AmountRefunded)
	if err != nil {
		return "", fmt.Errorf("record refund of payment %s: %w", p.ID, err)
	}
	if tag.RowsAffected() == 0 {
		return WebhookIgnored, nil
	}
	return WebhookProcessed, nil
}

// SyncResult reports one Razorpay sync.
type SyncResult struct {
	Subscriptions int `json:"subscriptions"`
	Added         int `json:"added"`
	Failed        int `json:"failed"`
}

// SyncSubscriptionPayments copies the paid invoices of every stored
// subscription into the ledger, for charges made before webhooks recorded
// them. It only reads from Razorpay and is safe to run again.
func (s *Service) SyncSubscriptionPayments(ctx context.Context) (SyncResult, error) {
	var result SyncResult
	if !s.PaymentsConfigured() {
		return result, ErrPaymentUnavailable
	}
	rows, err := s.repo.db.Query(ctx, `SELECT organization_id, razorpay_subscription_id, plan FROM go_billing_subscriptions`)
	if err != nil {
		return result, fmt.Errorf("list subscriptions to sync: %w", err)
	}
	type sub struct{ org, id, plan string }
	var subs []sub
	for rows.Next() {
		var v sub
		if err := rows.Scan(&v.org, &v.id, &v.plan); err != nil {
			rows.Close()
			return result, fmt.Errorf("scan subscription to sync: %w", err)
		}
		subs = append(subs, v)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("list subscriptions to sync: %w", err)
	}
	for _, v := range subs {
		result.Subscriptions++
		invoices, err := s.razorpay.ListSubscriptionInvoices(ctx, v.id)
		if err != nil {
			result.Failed++
			slog.WarnContext(ctx, "razorpay payment sync failed for subscription", "subscription", v.id, "err", err)
			continue
		}
		for _, inv := range invoices {
			if inv.Status != "paid" || inv.PaymentID == "" || inv.AmountPaid <= 0 || inv.PaidAt == nil {
				continue
			}
			tag, err := s.repo.db.Exec(ctx, `INSERT INTO go_billing_payments(payment_id,organization_id,kind,item,amount,currency,paid_at,source)
				VALUES($1,$2,'subscription',$3,$4,$5,$6,$7) ON CONFLICT (payment_id) DO NOTHING`,
				inv.PaymentID, v.org, v.plan, inv.AmountPaid, inv.Currency, time.Unix(*inv.PaidAt, 0).UTC(), sourceSync)
			if err != nil {
				return result, fmt.Errorf("record invoice %s: %w", inv.ID, err)
			}
			result.Added += int(tag.RowsAffected())
		}
	}
	return result, nil
}
