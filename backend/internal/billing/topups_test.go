package billing

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/toufiqqureshi/uplumo/backend/internal/plans"
)

// paymentEvent is a payment webhook body on orderID.
func paymentEvent(name, paymentID, orderID string, amount int64, currency, status string, captured bool) []byte {
	return fmt.Appendf(nil, `{"event":%q,"payload":{"payment":{"entity":{"id":%q,"order_id":%q,"amount":%d,"currency":%q,"status":%q,"captured":%t}}}}`, name, paymentID, orderID, amount, currency, status, captured)
}

func (f *fixture) onPlan(ctx context.Context, t *testing.T, plan string) {
	t.Helper()
	f.sellTiers()
	f.deliver(ctx, t, f.eventOnPlan(plan, "subscription.activated", "sub_"+plan, "active"))
}

func (f *fixture) capture(ctx context.Context, t *testing.T, order TopUpCheckout) {
	t.Helper()
	paid := paymentEvent("payment.captured", "pay_"+rand.Text(), order.OrderID, order.Amount, "INR", "captured", true)
	if got, err := f.svc.HandleWebhook(ctx, paid, sign(paid), "evt-"+rand.Text()); err != nil || got != WebhookProcessed {
		t.Fatalf("captured payment = %q, %v", got, err)
	}
}

func TestPacksFollowThePlan(t *testing.T) {
	ctx := t.Context()
	for _, tc := range []struct {
		plan string
		want []string
	}{
		{PlanFree, nil},
		{PlanBasic, nil},
		{PlanPro, []string{"credits_5000", "credits_15000"}},
		{plans.Business, []string{"credits_5000", "credits_15000", "sites_5"}},
	} {
		t.Run(tc.plan, func(t *testing.T) {
			f := newFixture(ctx, t)
			f.sellTiers()
			if tc.plan != PlanFree {
				f.onPlan(ctx, t, tc.plan)
			}
			offers, err := f.svc.TopUpOffers(ctx, f.org)
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, p := range offers {
				got = append(got, p.ID)
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("offers = %v, want %v", got, tc.want)
			}
			for _, pack := range plans.Packs() {
				_, err := f.svc.CheckoutTopUp(ctx, f.org, pack.ID)
				if slices.Contains(tc.want, pack.ID) {
					if err != nil {
						t.Errorf("checkout %s on %s: %v", pack.ID, tc.plan, err)
					}
				} else if !errors.Is(err, errTopUpPlan) {
					t.Errorf("checkout %s on %s = %v, want errTopUpPlan", pack.ID, tc.plan, err)
				}
			}
		})
	}
	if offers, err := NewReadOnlyService(nil).TopUpOffers(ctx, "any"); err != nil || len(offers) != 0 {
		t.Fatalf("packs offered without payments: %v, %v", offers, err)
	}
}

func TestTopUpVerifiedPaymentExactlyOnce(t *testing.T) {
	ctx := t.Context()
	f := newFixture(ctx, t)
	f.deliver(ctx, t, f.event("subscription.activated", "sub_1", "active", t1))
	order, err := f.svc.CheckoutTopUp(ctx, f.org, "credits_5000")
	if err != nil {
		t.Fatal(err)
	}
	if order.Amount != 149900 || order.Credits != 5000 || order.Currency != "INR" {
		t.Fatalf("order = %+v, want ₹1,499 for 5,000 credits", order)
	}
	if _, err := f.svc.topUpStatus(ctx, "foreign-org", order.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("foreign workspace read payment", err)
	}
	again, err := f.svc.CheckoutTopUp(ctx, f.org, "credits_5000")
	if err != nil || again.OrderID != order.OrderID || f.orders.Load() != 1 {
		t.Fatalf("duplicate checkout=%+v %v, %d orders", again, err, f.orders.Load())
	}
	ledger := NewCreditLedgerService(f.pool)
	if balance, err := ledger.GetBalance(ctx, f.org); err != nil || balance != 0 {
		t.Fatal("order creation credited the account", balance, err)
	}
	event := func(amount int64, currency, status string, captured bool) []byte {
		return paymentEvent("payment.captured", "pay_topup", order.OrderID, amount, currency, status, captured)
	}
	if _, err = f.svc.HandleWebhook(ctx, event(order.Amount, "INR", "captured", true), "forged", "evt"); !errors.Is(err, ErrInvalidSignature) {
		t.Fatal("unsigned payment accepted")
	}
	for _, bad := range [][]byte{event(order.Amount-1, "INR", "captured", true), event(order.Amount, "USD", "captured", true), event(order.Amount, "INR", "authorized", false)} {
		if _, err = f.svc.HandleWebhook(ctx, bad, sign(bad), "evt"); !errors.Is(err, ErrInvalidEvent) {
			t.Fatal("incorrect payment accepted", err)
		}
	}
	good := event(order.Amount, "INR", "captured", true)
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			if _, err := f.svc.HandleWebhook(context.Background(), good, sign(good), "different-delivery-"+rand.Text()); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if balance, err := ledger.GetBalance(ctx, f.org); err != nil || balance != 5000 {
		t.Fatal("duplicate or missing grant", balance, err)
	}
	var count int
	var expires time.Time
	if err = f.pool.QueryRow(ctx, `SELECT count(*),min(expires_at) FROM go_billing_credit_grants WHERE organization_id=$1 AND top_up`, f.org).Scan(&count, &expires); err != nil {
		t.Fatal(err)
	}
	if count != 1 || expires.Before(time.Now().AddDate(10, 0, 0)) {
		t.Fatal("pack credits must last while the plan does", count, expires)
	}
}

// Pack credits last while the plan is paid for and end once it has ended,
// after a grace day for a plan change done by hand. Plan credits stay.
func TestPackCreditsEndWithThePlan(t *testing.T) {
	ctx := t.Context()
	f := newFixture(ctx, t)
	f.deliver(ctx, t, f.event("subscription.charged", "sub_1", "active", t1))
	order, err := f.svc.CheckoutTopUp(ctx, f.org, "credits_5000")
	if err != nil {
		t.Fatal(err)
	}
	f.capture(ctx, t, order)
	ledger := NewCreditLedgerService(f.pool)
	end := func() int64 {
		t.Helper()
		if _, err := ledger.ExpirePackCreditsOfEndedPlans(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := ledger.ExpireCredits(ctx); err != nil {
			t.Fatal(err)
		}
		balance, err := ledger.GetBalance(ctx, f.org)
		if err != nil {
			t.Fatal(err)
		}
		return balance
	}
	if got := end(); got != 15000 {
		t.Fatalf("balance on a live plan = %d, want 10,000 plan + 5,000 pack", got)
	}
	f.deliver(ctx, t, f.event("subscription.cancelled", "sub_1", "cancelled", t1.Add(time.Hour)))
	if got := end(); got != 15000 {
		t.Fatalf("balance within the grace day = %d, want the pack kept", got)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE go_billing_subscriptions SET updated_at = now() - interval '25 hours' WHERE organization_id = $1`, f.org); err != nil {
		t.Fatal(err)
	}
	if got := end(); got != 10000 {
		t.Fatalf("balance after the plan ended = %d, want only the plan credits", got)
	}
}

// A payment authorized after Razorpay's auto-capture window is captured, so
// the customer gets the pack instead of a refund days later and a second
// payment now.
func TestLateAuthorizedPackPaymentIsCaptured(t *testing.T) {
	ctx := t.Context()
	f := newFixture(ctx, t)
	f.deliver(ctx, t, f.event("subscription.activated", "sub_1", "active", t1))
	order, err := f.svc.CheckoutTopUp(ctx, f.org, "credits_5000")
	if err != nil {
		t.Fatal(err)
	}
	authorized := func(paymentID, orderID string, amount int64) (WebhookResult, error) {
		body := paymentEvent("payment.authorized", paymentID, orderID, amount, "INR", "authorized", false)
		return f.svc.HandleWebhook(ctx, body, sign(body), "evt-"+rand.Text())
	}
	if _, err := authorized("pay_wrong", order.OrderID, order.Amount-100); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("authorized payment of the wrong amount = %v, want ErrInvalidEvent", err)
	}
	if got, err := authorized("pay_other", "order_not_ours", 100); err != nil || got != WebhookIgnored {
		t.Fatalf("payment on a foreign order = %q, %v", got, err)
	}
	if got, err := authorized("pay_late", order.OrderID, order.Amount); err != nil || got != WebhookProcessed {
		t.Fatalf("late authorization = %q, %v", got, err)
	}
	if want := []string{"capture pay_late 149900"}; !slices.Equal(f.payments, want) {
		t.Fatalf("payment calls = %v, want %v", f.payments, want)
	}

	// Razorpay captured it itself meanwhile: not an error.
	f.captureAnswer = func(w http.ResponseWriter) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"BAD_REQUEST_ERROR","description":"This payment has already been captured"}}`))
	}
	if got, err := authorized("pay_late", order.OrderID, order.Amount); err != nil || got != WebhookProcessed {
		t.Fatalf("authorization of a captured payment = %q, %v", got, err)
	}
	// Any other refusal is retried by Razorpay's redelivery.
	f.captureAnswer = func(w http.ResponseWriter) { w.WriteHeader(http.StatusBadGateway) }
	if _, err := authorized("pay_late", order.OrderID, order.Amount); !errors.Is(err, ErrPaymentProvider) {
		t.Fatalf("failed capture = %v, want ErrPaymentProvider", err)
	}

	f.captureAnswer = nil
	f.capture(ctx, t, order)
	calls := len(f.payments)
	if got, err := authorized("pay_late", order.OrderID, order.Amount); err != nil || got != WebhookIgnored || len(f.payments) != calls {
		t.Fatalf("authorization on a paid pack = %q, %v, calls %v", got, err, f.payments)
	}
}

// A bought pack must not shrink the plan credits a renewal grants: the
// two-month rollover cap applies to plan credits only.
func TestTopUpCreditsDoNotReduceTheMonthlyPlanGrant(t *testing.T) {
	ctx := t.Context()
	f := newFixture(ctx, t)
	f.sellTiers()
	chargeAt := t1
	charge := func() {
		f.deliver(ctx, t, f.chargeOnPlan(PlanPro, chargeAt))
		chargeAt = chargeAt.Add(time.Second)
	}
	charge() // 10000 Pro credits

	order, err := f.svc.CheckoutTopUp(ctx, f.org, "credits_15000")
	if err != nil {
		t.Fatal(err)
	}
	f.capture(ctx, t, order)

	charge()
	if got, err := NewCreditLedgerService(f.pool).GetBalance(ctx, f.org); err != nil || got != 35000 {
		t.Fatalf("balance = %d, %v; want 10000 rolled-over plan + 15000 pack + 10000 new plan credits", got, err)
	}
}

// Closing the payment window must not lock the workspace to that pack: a new
// pack starts a new order, and a late payment on the old order still grants.
func TestTopUpCheckoutCanSwitchToAnotherPack(t *testing.T) {
	ctx := t.Context()
	f := newFixture(ctx, t)
	f.deliver(ctx, t, f.event("subscription.activated", "sub_1", "active", t1))
	abandoned, err := f.svc.CheckoutTopUp(ctx, f.org, "credits_5000")
	if err != nil {
		t.Fatal(err)
	}
	switched, err := f.svc.CheckoutTopUp(ctx, f.org, "credits_15000")
	if err != nil {
		t.Fatalf("switching pack after closing checkout: %v", err)
	}
	if switched.OrderID == abandoned.OrderID || switched.Amount != 399900 {
		t.Fatalf("switched checkout = %+v, want a new order for the 15,000 pack", switched)
	}
	f.capture(ctx, t, abandoned)
	if got, err := NewCreditLedgerService(f.pool).GetBalance(ctx, f.org); err != nil || got != 5000 {
		t.Fatalf("balance = %d, %v; a captured payment on the abandoned order must still grant its 5,000 credits", got, err)
	}
}

// Business may add 5 websites for 30 days. Sites added under the add-on keep
// working after it runs out, but no new site fits, and add-ons bought one
// after another do not stack into a permanent allowance.
func TestSiteAddOnRaisesTheBusinessSiteLimit(t *testing.T) {
	ctx := t.Context()
	f := newFixture(ctx, t)
	f.onPlan(ctx, t, plans.Business)
	limit := func() int {
		t.Helper()
		got, err := f.svc.SiteLimit(ctx, f.org)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	if got := limit(); got != 20 {
		t.Fatalf("Business site limit = %d, want 20", got)
	}
	order, err := f.svc.CheckoutTopUp(ctx, f.org, "sites_5")
	if err != nil {
		t.Fatal(err)
	}
	if order.Amount != 99900 || order.Sites != 5 || order.Credits != 0 {
		t.Fatalf("site add-on order = %+v", order)
	}
	f.capture(ctx, t, order)
	if got := limit(); got != 25 {
		t.Fatalf("site limit with the add-on = %d, want 25", got)
	}
	if balance, err := NewCreditLedgerService(f.pool).GetBalance(ctx, f.org); err != nil || balance != 0 {
		t.Fatalf("a site add-on granted credits: %d, %v", balance, err)
	}
	var until time.Time
	if err := f.pool.QueryRow(ctx, `SELECT sites_until FROM go_billing_topups WHERE id = $1`, order.ID).Scan(&until); err != nil {
		t.Fatal(err)
	}
	if d := time.Until(until); d < 29*24*time.Hour || d > 31*24*time.Hour {
		t.Fatalf("add-on lasts until %v, want 30 days from payment", until)
	}

	addSites := func(n int) {
		t.Helper()
		for range n {
			if _, err := f.pool.Exec(ctx, `INSERT INTO projects (id, organization_id, name) VALUES ($1, $2, 'Site')`, "project-"+rand.Text(), f.org); err != nil {
				t.Fatal(err)
			}
		}
	}
	addSites(24)
	// Three add-ons that ran out one after another: at most 5 at a time.
	if _, err := f.pool.Exec(ctx, `UPDATE go_billing_topups SET sites_until = now() - interval '1 day' WHERE id = $1`, order.ID); err != nil {
		t.Fatal(err)
	}
	for _, ago := range []string{"40 days", "80 days"} {
		if _, err := f.pool.Exec(ctx, `INSERT INTO go_billing_topups (id, organization_id, pack_id, amount_paise, credits, sites, expires_after_days, order_id, status, sites_until) VALUES ($1, $2, 'sites_5', 99900, 0, 5, 30, $3, 'paid', now() - $4::interval)`, "tup_"+rand.Text(), f.org, "order_"+rand.Text(), ago); err != nil {
			t.Fatal(err)
		}
	}
	if got := limit(); got != 20 {
		t.Fatalf("site limit after the add-on ran out = %d, want 20", got)
	}
	usage, err := f.svc.UsageAgainstPlan(ctx, f.org)
	if err != nil || usage.Sites.Exceeded {
		t.Fatalf("24 sites after a 5-site add-on ran out = %+v, %v; want them kept working", usage.Sites, err)
	}
	addSites(2)
	usage, err = f.svc.UsageAgainstPlan(ctx, f.org)
	if err != nil || !usage.Sites.Exceeded {
		t.Fatalf("26 sites = %+v, %v; want over the limit: expired add-ons must not stack", usage.Sites, err)
	}
}
