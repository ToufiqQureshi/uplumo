package billing

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/plans"
	"github.com/toufiqqureshi/uplumo/backend/internal/razorpay"
)

var usdTierPlanIDs = map[string]string{PlanBasic: "plan_usd_basic", PlanPro: "plan_usd_pro", plans.Business: "plan_usd_business"}

// sellInUSD makes the fixture sell all three plans in rupees and dollars.
func (f *fixture) sellInUSD(t *testing.T) {
	t.Helper()
	f.sellTiers()
	f.svc.SetUSDPlanIDs(usdTierPlanIDs)
	if err := f.svc.EnableUSD(t.Context()); err != nil {
		t.Fatalf("EnableUSD() = %v", err)
	}
}

// usdEvent is a webhook for a subscription to plan's dollar Razorpay plan.
func (f *fixture) usdEvent(plan, name, subID, status string) []byte {
	return bytes.Replace(f.event(name, subID, status, t1), []byte(testPlanID), []byte(usdTierPlanIDs[plan]), 1)
}

func (f *fixture) storedCurrency(t *testing.T) string {
	t.Helper()
	var currency string
	if err := f.pool.QueryRow(t.Context(), `SELECT currency FROM go_billing_subscriptions WHERE organization_id = $1`, f.org).Scan(&currency); err != nil {
		t.Fatal(err)
	}
	return currency
}

func TestUSDCheckoutBillsTheDollarPlan(t *testing.T) {
	f := newFixture(t.Context(), t)
	f.sellInUSD(t)
	if !f.svc.SellsInUSD() || f.svc.CurrencyFor("US") != plans.USD || f.svc.CurrencyFor("IN") != plans.INR || f.svc.CurrencyFor("") != plans.INR {
		t.Fatal("with dollar plans verified, the US must pay in dollars and India (or unknown) in rupees")
	}
	for i, plan := range plans.Paid() {
		if i > 0 {
			if _, err := f.pool.Exec(t.Context(), `DELETE FROM go_billing_subscriptions WHERE organization_id = $1`, f.org); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := f.svc.CheckoutPlan(t.Context(), f.org, plan, plans.USD); err != nil {
			t.Fatalf("CheckoutPlan(%s, USD) = %v", plan, err)
		}
		if got := f.lastPlanID.Load(); got != usdTierPlanIDs[plan] {
			t.Fatalf("Razorpay plan_id = %v, want the dollar plan %s", got, usdTierPlanIDs[plan])
		}
		if got := f.storedCurrency(t); got != plans.USD {
			t.Fatalf("stored currency = %s, want USD", got)
		}
	}
}

// A dollar price must never show unless checkout can charge exactly it.
// Every failure leaves everyone on rupees and creates nothing payable.
func TestEnableUSDFailsClosed(t *testing.T) {
	for name, setup := range map[string]func(*fixture){
		"a plan has no dollar id": func(f *fixture) {
			f.svc.SetUSDPlanIDs(map[string]string{PlanBasic: "plan_usd_basic", PlanPro: "plan_usd_pro"})
		},
		"a dollar plan has the wrong amount": func(f *fixture) {
			f.svc.SetUSDPlanIDs(usdTierPlanIDs)
			f.planOverride = func(p *razorpay.Plan) {
				if p.ID == "plan_usd_pro" {
					p.Item.Amount = 3800
				}
			}
		},
		"a dollar plan is in rupees": func(f *fixture) {
			f.svc.SetUSDPlanIDs(usdTierPlanIDs)
			f.planOverride = func(p *razorpay.Plan) {
				if p.ID == "plan_usd_business" {
					p.Item.Currency = plans.INR
				}
			}
		},
		"Razorpay is down": func(f *fixture) {
			f.svc.SetUSDPlanIDs(usdTierPlanIDs)
			f.failWith = http.StatusInternalServerError
		},
		"a dollar id is a rupee id": func(f *fixture) {
			f.svc.SetUSDPlanIDs(map[string]string{PlanBasic: tierPlanIDs[PlanBasic], PlanPro: "plan_usd_pro", plans.Business: "plan_usd_business"})
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t.Context(), t)
			f.sellTiers()
			setup(f)
			if err := f.svc.EnableUSD(t.Context()); err == nil {
				t.Fatal("EnableUSD() succeeded, want an error")
			}
			f.failWith = 0
			if f.svc.SellsInUSD() || f.svc.CurrencyFor("US") != plans.INR {
				t.Fatal("dollar billing switched on after a failed check")
			}
			if _, err := f.svc.CheckoutPlan(t.Context(), f.org, PlanPro, plans.USD); !errors.Is(err, ErrPlanUnavailable) {
				t.Fatalf("dollar checkout = %v, want ErrPlanUnavailable", err)
			}
			if n := f.calls.Load(); n != 0 {
				t.Fatalf("Razorpay created %d subscriptions while dollars are off", n)
			}
		})
	}
}

// A checkout in one currency, abandoned, then retried in the other (the
// customer turned a VPN on) must not leave two payable subscriptions.
func TestSwitchingCurrencyCancelsTheUnpaidSubscription(t *testing.T) {
	f := newFixture(t.Context(), t)
	f.sellInUSD(t)
	first, err := f.svc.CheckoutPlan(t.Context(), f.org, PlanPro, plans.INR)
	if err != nil {
		t.Fatal(err)
	}
	again, err := f.svc.CheckoutPlan(t.Context(), f.org, PlanPro, plans.INR)
	if err != nil || again.SubscriptionID != first.SubscriptionID {
		t.Fatalf("same plan, same currency = %+v, %v; want %s reused", again, err, first.SubscriptionID)
	}
	usd, err := f.svc.CheckoutPlan(t.Context(), f.org, PlanPro, plans.USD)
	if err != nil || usd.SubscriptionID == first.SubscriptionID {
		t.Fatalf("same plan in dollars = %+v, %v; want a new subscription", usd, err)
	}
	if len(f.cancels) != 1 || !strings.Contains(f.cancels[0].Path, first.SubscriptionID) || f.cancels[0].AtCycleEnd {
		t.Fatalf("cancels = %+v, want the unpaid rupee subscription cancelled now", f.cancels)
	}
	if got := f.storedCurrency(t); got != plans.USD {
		t.Fatalf("stored currency = %s, want USD", got)
	}
	if n := f.calls.Load(); n != 2 {
		t.Fatalf("Razorpay created %d subscriptions, want 2", n)
	}
}

func TestDoubleClickedUSDCheckoutCreatesOneSubscription(t *testing.T) {
	f := newFixture(t.Context(), t)
	f.sellInUSD(t)
	var wg sync.WaitGroup
	ids := make([]string, 5)
	for i := range ids {
		wg.Go(func() {
			checkout, err := f.svc.CheckoutPlan(t.Context(), f.org, PlanBasic, plans.USD)
			if err != nil {
				t.Errorf("checkout %d: %v", i, err)
			}
			ids[i] = checkout.SubscriptionID
		})
	}
	wg.Wait()
	if n := f.calls.Load(); n != 1 {
		t.Fatalf("Razorpay created %d subscriptions for one double click, want 1", n)
	}
	if ids[0] == "" || slices.ContainsFunc(ids, func(id string) bool { return id != ids[0] }) {
		t.Fatalf("checkouts got %v, want one subscription", ids)
	}
}

// A dollar subscription keeps renewing and earning its credits even when
// dollar sales are off at this start (Razorpay was down during the check).
func TestDollarSubscriptionRenewsWhileDollarSalesAreOff(t *testing.T) {
	f := newFixture(t.Context(), t)
	f.sellTiers()
	f.svc.SetUSDPlanIDs(usdTierPlanIDs) // never enabled
	ledger := NewCreditLedgerService(f.pool)

	f.deliver(t.Context(), t, f.usdEvent(PlanPro, "subscription.charged", "sub_usd", "active"))
	status, err := f.svc.Status(t.Context(), f.org)
	if err != nil || status.Plan != PlanPro || status.Currency != plans.USD {
		t.Fatalf("status = %+v, %v; want Pro billed in USD", status, err)
	}
	if got, err := ledger.GetBalance(t.Context(), f.org); err != nil || got != 10000 {
		t.Fatalf("balance = %d, %v; want Pro's 10,000 monthly credits", got, err)
	}
	// The same event delivered again grants nothing more.
	if got := f.deliverAs(t.Context(), t, "evt-other-"+f.org, f.usdEvent(PlanPro, "subscription.charged", "sub_usd", "active")); got != WebhookDuplicate {
		t.Fatalf("redelivered charge = %s, want duplicate", got)
	}
	if got, err := ledger.GetBalance(t.Context(), f.org); err != nil || got != 10000 {
		t.Fatalf("balance after redelivery = %d, %v; want 10,000", got, err)
	}
}

func TestUSDPackIsChargedAndGrantedInDollarsOnly(t *testing.T) {
	ctx := t.Context()
	f := newFixture(ctx, t)
	f.sellInUSD(t)
	f.deliver(ctx, t, f.eventOnPlan(PlanPro, "subscription.activated", "sub_1", "active"))

	order, err := f.svc.CheckoutTopUp(ctx, f.org, "credits_5000", plans.USD)
	if err != nil || order.Amount != 1900 || order.Currency != plans.USD || order.Credits != 5000 {
		t.Fatalf("USD pack order = %+v, %v; want $19.00 for 5,000 credits", order, err)
	}
	// A double click gets the same order back.
	if again, err := f.svc.CheckoutTopUp(ctx, f.org, "credits_5000", plans.USD); err != nil || again.OrderID != order.OrderID || f.orders.Load() != 1 {
		t.Fatalf("second click = %+v, %v with %d orders; want the same order", again, err, f.orders.Load())
	}
	// The same pack in rupees is another price: a new order, the dollar one
	// released.
	inr, err := f.svc.CheckoutTopUp(ctx, f.org, "credits_5000", plans.INR)
	if err != nil || inr.OrderID == order.OrderID || inr.Amount != 149900 || inr.Currency != plans.INR {
		t.Fatalf("rupee pack after a dollar one = %+v, %v", inr, err)
	}

	ledger := NewCreditLedgerService(f.pool)
	before, err := ledger.GetBalance(ctx, f.org)
	if err != nil {
		t.Fatal(err)
	}
	pay := func(currency string, amount int64) (WebhookResult, error) {
		body := paymentEvent("payment.captured", "pay_"+rand.Text(), order.OrderID, amount, currency, "captured", true)
		return f.svc.HandleWebhook(ctx, body, sign(body), "evt-"+rand.Text())
	}
	// 1900 paise is not $19: the currency must match, not only the number.
	if _, err := pay(plans.INR, 1900); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("rupee payment on a dollar order = %v, want ErrInvalidEvent", err)
	}
	if got, err := ledger.GetBalance(ctx, f.org); err != nil || got != before {
		t.Fatalf("a mismatched payment granted credits: %d -> %d, %v", before, got, err)
	}
	// The released dollar order still grants what was paid on it.
	if got, err := pay(plans.USD, 1900); err != nil || got != WebhookProcessed {
		t.Fatalf("dollar payment = %s, %v", got, err)
	}
	if got, err := ledger.GetBalance(ctx, f.org); err != nil || got != before+5000 {
		t.Fatalf("balance = %d, %v; want %d", got, err, before+5000)
	}

	// A late authorization is captured in the order's currency.
	usd2, err := f.svc.CheckoutTopUp(ctx, f.org, "credits_15000", plans.USD)
	if err != nil || usd2.Amount != 5900 {
		t.Fatalf("15,000-credit pack = %+v, %v; want $59.00", usd2, err)
	}
	body := paymentEvent("payment.authorized", "pay_late_usd", usd2.OrderID, usd2.Amount, plans.USD, "authorized", false)
	if got, err := f.svc.HandleWebhook(ctx, body, sign(body), "evt-"+rand.Text()); err != nil || got != WebhookProcessed {
		t.Fatalf("late dollar authorization = %s, %v", got, err)
	}
	if want := "capture pay_late_usd 5900 USD"; !slices.Contains(f.payments, want) {
		t.Fatalf("payment calls = %v, want %q", f.payments, want)
	}
}

func TestUSDPackRefusedWhileDollarSalesAreOff(t *testing.T) {
	ctx := t.Context()
	f := newFixture(ctx, t)
	f.sellTiers()
	f.svc.SetUSDPlanIDs(usdTierPlanIDs) // never enabled
	f.deliver(ctx, t, f.eventOnPlan(PlanPro, "subscription.activated", "sub_1", "active"))
	if _, err := f.svc.CheckoutTopUp(ctx, f.org, "credits_5000", plans.USD); !errors.Is(err, errTopUpUnavailable) {
		t.Fatalf("dollar pack while dollars are off = %v, want errTopUpUnavailable", err)
	}
	if n := f.orders.Load(); n != 0 {
		t.Fatalf("Razorpay got %d orders, want 0", n)
	}
}

// The customer pays the currency they were shown. When the server would
// bill another one, checkout stops before Razorpay is called.
func TestCheckoutHandlerRefusesACurrencyTheCustomerWasNotShown(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, tc := range []struct {
		name, country, body string
		wantCode            int
		wantCreated         int32
	}{
		{"US visitor shown dollars", "US", `{"plan":"pro","currency":"USD"}`, http.StatusOK, 1},
		{"US visitor shown rupees", "US", `{"plan":"pro","currency":"INR"}`, http.StatusConflict, 0},
		{"US visitor on an old app", "US", `{"plan":"pro"}`, http.StatusConflict, 0},
		{"Indian visitor shown rupees", "IN", `{"plan":"pro","currency":"INR"}`, http.StatusOK, 1},
		{"Indian visitor on an old app", "IN", `{"plan":"pro"}`, http.StatusOK, 1},
		{"Indian visitor shown dollars", "IN", `{"plan":"pro","currency":"USD"}`, http.StatusConflict, 0},
		{"unknown country shown dollars", "", `{"plan":"pro","currency":"USD"}`, http.StatusConflict, 0},
		{"a currency we do not sell", "US", `{"plan":"pro","currency":"EUR"}`, http.StatusConflict, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t.Context(), t)
			f.sellInUSD(t)
			ctx := auth.WithUser(context.Background(), auth.User{OrganizationID: f.org, Role: "owner"})
			req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/v1/billing/checkout", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			checkoutHandler(logger, f.svc, func(*http.Request) string { return tc.country }).ServeHTTP(rec, req)
			if rec.Code != tc.wantCode {
				t.Fatalf("status %d body %s, want %d", rec.Code, rec.Body.String(), tc.wantCode)
			}
			if tc.wantCode == http.StatusConflict && !strings.Contains(rec.Body.String(), "currency_changed") {
				t.Fatalf("body %s, want currency_changed", rec.Body.String())
			}
			if n := f.calls.Load(); n != tc.wantCreated {
				t.Fatalf("Razorpay created %d subscriptions, want %d", n, tc.wantCreated)
			}
		})
	}
}
