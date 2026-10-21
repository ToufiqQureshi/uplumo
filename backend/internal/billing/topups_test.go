package billing

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/toufiqqureshi/uplumo/backend/internal/razorpay"
)

func TestTopUpPackConfiguration(t *testing.T) {
	for _, raw := range []string{`[{"id":"pack","name":"Pack","priceInr":0,"credits":100,"expiresAfterDays":30}]`, `[{"id":"pack","name":"Pack","priceInr":1,"credits":-1,"expiresAfterDays":30}]`, `[{"id":"pack","name":"Pack","priceInr":1,"credits":100,"expiresAfterDays":0}]`, `[{"unknown":true}]`, `not-json`} {
		if (&Service{}).ConfigureTopUps(raw) == nil {
			t.Fatalf("invalid pack accepted: %s", raw)
		}
	}
	svc := &Service{}
	if err := svc.ConfigureTopUps(""); err != nil {
		t.Fatal(err)
	}
	if len(svc.TopUpPacks()) != 0 {
		t.Fatal("unconfigured packs advertised")
	}
}
func TestTopUpVerifiedPaymentExactlyOnce(t *testing.T) {
	ctx := t.Context()
	f := newFixture(ctx, t)
	f.deliver(ctx, t, f.event("subscription.activated", "sub_1", "active", t1))
	var created atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/orders" {
			t.Error("unexpected payment endpoint", r.URL.Path)
		}
		var req struct {
			Amount   int64  `json:"amount"`
			Currency string `json:"currency"`
			Receipt  string `json:"receipt"`
			Partial  bool   `json:"partial_payment"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if req.Amount != 99900 || req.Currency != "INR" || req.Partial {
			t.Error("browser influenced order amount", req)
		}
		created.Add(1)
		if err := json.NewEncoder(w).Encode(razorpay.Order{ID: "order_" + rand.Text(), Amount: req.Amount, Currency: req.Currency, Receipt: req.Receipt}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	f.svc.razorpay = razorpay.NewClient(server.URL, testKeyID, "test-secret")
	if err := f.svc.ConfigureTopUps(`[{"id":"extra","name":"Extra usage","priceInr":999,"credits":1000,"expiresAfterDays":90}]`); err != nil {
		t.Fatal(err)
	}
	order, err := f.svc.CheckoutTopUp(ctx, f.org, "extra")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.topUpStatus(ctx, "foreign-org", order.ID); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("foreign workspace read payment", err)
	}
	again, err := f.svc.CheckoutTopUp(ctx, f.org, "extra")
	if err != nil || again.OrderID != order.OrderID || created.Load() != 1 {
		t.Fatalf("duplicate checkout=%+v %v", again, err)
	}
	ledger := NewCreditLedgerService(f.pool)
	balance, err := ledger.GetBalance(ctx, f.org)
	if err != nil || balance != 0 {
		t.Fatal("order creation credited the account", balance, err)
	}
	event := func(amount int64, currency, status string, captured bool) []byte {
		return fmt.Appendf(nil, `{"event":"payment.captured","payload":{"payment":{"entity":{"id":"pay_topup","order_id":%q,"amount":%d,"currency":%q,"status":%q,"captured":%t}}}}`, order.OrderID, amount, currency, status, captured)
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
			_, err := f.svc.HandleWebhook(context.Background(), good, sign(good), "different-delivery-"+rand.Text())
			if err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	balance, err = ledger.GetBalance(ctx, f.org)
	if err != nil || balance != 1000 {
		t.Fatal("duplicate or missing grant", balance, err)
	}
	var count int
	var expires time.Time
	if err = f.pool.QueryRow(ctx, `SELECT count(*),min(expires_at) FROM go_billing_credit_grants WHERE organization_id=$1`, f.org).Scan(&count, &expires); err != nil {
		t.Fatal(err)
	}
	if count != 1 || expires.Before(time.Now().Add(89*24*time.Hour)) {
		t.Fatal("wrong top-up expiry", count, expires)
	}
	var grantRows int
	if err = f.pool.QueryRow(ctx, `SELECT count(*) FROM go_billing_credit_ledger WHERE organization_id=$1 AND type='grant'`, f.org).Scan(&grantRows); err != nil || grantRows != 1 {
		t.Fatal("grant ledger replayed", grantRows, err)
	}
}

// A bought top-up must not shrink the plan credits a renewal grants: the
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

	orderID := "order_" + rand.Text()
	if _, err := f.pool.Exec(ctx, `INSERT INTO go_billing_topups(id,organization_id,pack_id,amount_paise,credits,expires_after_days,order_id,status) VALUES($1,$2,'extra',99900,10000,90,$3,'ready')`, "tup_"+rand.Text(), f.org, orderID); err != nil {
		t.Fatal(err)
	}
	paid := fmt.Appendf(nil, `{"event":"payment.captured","payload":{"payment":{"entity":{"id":"pay_%s","order_id":%q,"amount":99900,"currency":"INR","status":"captured","captured":true}}}}`, rand.Text(), orderID)
	if _, err := f.svc.HandleWebhook(ctx, paid, sign(paid), "evt-"+rand.Text()); err != nil {
		t.Fatal(err)
	}

	charge()
	ledger := NewCreditLedgerService(f.pool)
	if got, err := ledger.GetBalance(ctx, f.org); err != nil || got != 30000 {
		t.Fatalf("balance = %d, %v; want 10000 rolled-over plan + 10000 top-up + 10000 new plan credits", got, err)
	}
}

// Closing the payment window must not lock the workspace to that pack: a new
// pack starts a new order, and a late payment on the old order still grants.
func TestTopUpCheckoutCanSwitchToAnotherPack(t *testing.T) {
	ctx := t.Context()
	f := newFixture(ctx, t)
	f.sellTiers()
	f.deliver(ctx, t, f.eventOnPlan(PlanPro, "subscription.activated", "sub_1", "active"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req razorpay.Order
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		req.ID = "order_" + rand.Text()
		if err := json.NewEncoder(w).Encode(req); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(server.Close)
	f.svc.razorpay = razorpay.NewClient(server.URL, testKeyID, "test-secret")
	if err := f.svc.ConfigureTopUps(`[{"id":"small","name":"Small","priceInr":499,"credits":400,"expiresAfterDays":90},{"id":"large","name":"Large","priceInr":999,"credits":1000,"expiresAfterDays":90}]`); err != nil {
		t.Fatal(err)
	}
	abandoned, err := f.svc.CheckoutTopUp(ctx, f.org, "small")
	if err != nil {
		t.Fatal(err)
	}
	switched, err := f.svc.CheckoutTopUp(ctx, f.org, "large")
	if err != nil {
		t.Fatalf("switching pack after closing checkout: %v", err)
	}
	if switched.OrderID == abandoned.OrderID || switched.Amount != 99900 {
		t.Fatalf("switched checkout = %+v, want a new order for the large pack", switched)
	}

	late := fmt.Appendf(nil, `{"event":"payment.captured","payload":{"payment":{"entity":{"id":"pay_%s","order_id":%q,"amount":49900,"currency":"INR","status":"captured","captured":true}}}}`, rand.Text(), abandoned.OrderID)
	if _, err := f.svc.HandleWebhook(ctx, late, sign(late), "evt-"+rand.Text()); err != nil {
		t.Fatal(err)
	}
	if got, err := NewCreditLedgerService(f.pool).GetBalance(ctx, f.org); err != nil || got < 400 {
		t.Fatalf("balance = %d, %v; a captured payment on the abandoned order must still grant its 400 credits", got, err)
	}
}
