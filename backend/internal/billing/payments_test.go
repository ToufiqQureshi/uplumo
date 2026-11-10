package billing

import (
	"crypto/rand"
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

	// The two refund events Razorpay can send for a payment, in its shape:
	// the refund and the payment with its running refunded total.
	refund := func(event string, refunded int64) []byte {
		body, err := json.Marshal(map[string]any{"entity": "event", "event": event, "created_at": t2.Unix(), "payload": map[string]any{
			"refund":  map[string]any{"entity": map[string]any{"id": "rfnd_" + f.org, "payment_id": pay, "amount": 80000, "status": "processed"}},
			"payment": map[string]any{"entity": map[string]any{"id": pay, "amount": 249900, "amount_refunded": refunded, "currency": "INR", "status": "refunded"}},
		}})
		if err != nil {
			t.Fatal(err)
		}
		return body
	}
	// Razorpay has no payment.refunded event; it must not be what updates refunds.
	if got := f.deliverAs(t.Context(), t, "evt_old_"+f.org, refund("payment.refunded", 249900)); got != WebhookIgnored {
		t.Fatalf("payment.refunded = %s, want ignored", got)
	}
	for _, step := range []struct {
		name     string
		refunded int64
		want     int64
	}{
		{"first refund", 80000, 80000},
		{"redelivery", 80000, 80000},
		{"second refund", 180000, 180000},
		{"older event delivered late never lowers it", 80000, 180000},
		{"more than paid is capped", 999999, 249900},
	} {
		if got := f.deliverAs(t.Context(), t, "evt_"+rand.Text(), refund("refund.processed", step.refunded)); got != WebhookProcessed {
			t.Fatalf("%s: refund = %s, want processed", step.name, got)
		}
		if got := f.ledger(t)[pay].Refunded; got != step.want {
			t.Fatalf("%s: refunded = %d, want %d", step.name, got, step.want)
		}
	}
	other, _ := json.Marshal(map[string]any{"event": "refund.processed", "created_at": t2.Unix(), "payload": map[string]any{
		"payment": map[string]any{"entity": map[string]any{"id": "pay_not_ours_" + f.org, "amount": 100, "amount_refunded": 100, "currency": "INR"}},
	}})
	if got := f.deliverAs(t.Context(), t, "evt_"+rand.Text(), other); got != WebhookIgnored {
		t.Fatalf("refund of another business's payment = %s, want ignored", got)
	}
}

func TestSyncRaisesRefundsTheWebhookMissed(t *testing.T) {
	f := newFixture(t.Context(), t)
	pay, other := "pay_"+f.org, "pay_other_"+f.org
	f.deliver(t.Context(), t, f.chargedWithPayment(t, pay, 99900))
	f.refunds = []razorpay.Refund{
		{ID: "rfnd_1", PaymentID: pay, Amount: 50000, Status: "processed"},
		{ID: "rfnd_2", PaymentID: pay, Amount: 30000, Status: "processed"},
		{ID: "rfnd_3", PaymentID: pay, Amount: 19900, Status: "pending"},
		{ID: "rfnd_4", PaymentID: pay, Amount: 19900, Status: "failed"},
		{ID: "rfnd_5", PaymentID: other, Amount: 100, Status: "processed"},
	}
	for run, want := range []int{1, 0} {
		got, err := f.svc.SyncSubscriptionPayments(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if got.RefundsUpdated != want {
			t.Fatalf("sync run %d updated %d refunds, want %d", run, got.RefundsUpdated, want)
		}
	}
	ledger := f.ledger(t)
	if ledger[pay].Refunded != 80000 {
		t.Fatalf("refunded = %d, want 80000 (processed only)", ledger[pay].Refunded)
	}
	if _, ok := ledger[other]; ok {
		t.Fatal("a refund of a payment outside the ledger was added to it")
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
