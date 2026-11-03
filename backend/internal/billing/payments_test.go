package billing

import (
	"encoding/json"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/razorpay"
)

type ledgerRow struct {
	Kind, Item, Currency, Source string
	Amount, Refunded             int64
}

func (f *fixture) ledger(t *testing.T) map[string]ledgerRow {
	t.Helper()
	rows, err := f.pool.Query(t.Context(), `SELECT payment_id,kind,item,currency,source,amount,amount_refunded FROM go_billing_payments WHERE organization_id=$1`, f.org)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]ledgerRow{}
	for rows.Next() {
		var id string
		var r ledgerRow
		if err := rows.Scan(&id, &r.Kind, &r.Item, &r.Currency, &r.Source, &r.Amount, &r.Refunded); err != nil {
			t.Fatal(err)
		}
		out[id] = r
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(t.Context(), `DELETE FROM go_billing_payments WHERE organization_id=$1`, f.org)
	})
	return out
}

// chargedWithPayment is a subscription.charged body carrying its payment.
func (f *fixture) chargedWithPayment(t *testing.T, paymentID string, amount int64) []byte {
	t.Helper()
	var event map[string]any
	if err := json.Unmarshal(f.event("subscription.charged", "sub_1", "active", t1), &event); err != nil {
		t.Fatal(err)
	}
	event["payload"].(map[string]any)["payment"] = map[string]any{"entity": map[string]any{
		"id": paymentID, "amount": amount, "currency": "INR", "status": "captured", "created_at": t1.Unix(),
	}}
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func TestChargedWebhookRecordsThePaymentOnceAndRefundsUpdateIt(t *testing.T) {
	f := newFixture(t.Context(), t)
	pay := "pay_" + f.org
	body := f.chargedWithPayment(t, pay, 249900)
	f.deliver(t.Context(), t, body)
	if got := f.deliverAs(t.Context(), t, "evt_again_"+f.org, body); got != WebhookDuplicate {
		t.Fatalf("redelivery = %s, want duplicate", got)
	}
	if got := f.ledger(t); len(got) != 1 || got[pay] != (ledgerRow{Kind: "subscription", Item: PlanPro, Currency: "INR", Source: sourceWebhook, Amount: 249900}) {
		t.Fatalf("ledger = %+v", got)
	}

	refund, err := json.Marshal(map[string]any{"event": "payment.refunded", "created_at": t2.Unix(), "payload": map[string]any{
		"payment": map[string]any{"entity": map[string]any{"id": pay, "amount": 249900, "amount_refunded": 100000, "currency": "INR"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if got := f.deliverAs(t.Context(), t, "evt_refund_"+f.org, refund); got != WebhookProcessed {
			t.Fatalf("refund = %s, want processed", got)
		}
	}
	if got := f.ledger(t)[pay].Refunded; got != 100000 {
		t.Fatalf("refunded = %d, want 100000", got)
	}
}

func TestSyncCopiesPaidInvoicesOnceAndSkipsUnpaid(t *testing.T) {
	f := newFixture(t.Context(), t)
	f.deliver(t.Context(), t, f.event("subscription.activated", "sub_1", "active", t1))
	paid := t1.Unix()
	f.invoices = map[string][]razorpay.Invoice{"sub_1": {
		{ID: "inv_1", PaymentID: "pay_a_" + f.org, Status: "paid", AmountPaid: 249900, Currency: "INR", PaidAt: &paid},
		{ID: "inv_2", PaymentID: "pay_b_" + f.org, Status: "paid", AmountPaid: 249900, Currency: "INR", PaidAt: &paid},
		{ID: "inv_3", Status: "issued", AmountPaid: 0, Currency: "INR"},
	}}
	for run, wantAdded := range []int{2, 0} {
		got, err := f.svc.SyncSubscriptionPayments(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if got.Added != wantAdded {
			t.Fatalf("sync run %d added %d, want %d", run, got.Added, wantAdded)
		}
	}
	if got := f.ledger(t); len(got) != 2 || got["pay_a_"+f.org].Source != sourceSync {
		t.Fatalf("ledger = %+v", got)
	}
}
