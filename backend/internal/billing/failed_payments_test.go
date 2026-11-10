package billing

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/razorpay"
)

// fakePayments serves Razorpay's GET /payments from items, 100 per page.
func fakePayments(t *testing.T, items []razorpay.PaymentAttempt, status int) (*Service, *atomic.Int32) {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != http.MethodGet || r.URL.Path != "/payments" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if status != 0 {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":{"code":"SERVER_ERROR","description":"down"}}`))
			return
		}
		skip, _ := strconv.Atoi(r.URL.Query().Get("skip"))
		page := items[min(skip, len(items)):min(skip+100, len(items))]
		if err := json.NewEncoder(w).Encode(map[string]any{"items": page}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(srv.Close)
	return NewService(nil, razorpay.NewClient(srv.URL, "rzp_test_fake", "fake-secret"), "plan_fake", "fake-webhook-secret"), &calls
}

func TestListFailedPayments(t *testing.T) {
	now := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	at := func(h int) int64 { return now.Add(time.Duration(-h) * time.Hour).Unix() }
	items := []razorpay.PaymentAttempt{
		// Newest first, as Razorpay lists them.
		{ID: "pay_retry_ok", OrderID: "order_a", Status: "captured", Email: "Asha@Example.test", CreatedAt: at(1)},
		{ID: "pay_timeout", OrderID: "order_a", Status: "failed", Method: "upi", Email: "asha@example.test", Amount: 499900, Currency: "INR",
			ErrorReason: "payment_timed_out", ErrorDescription: "Payment was unsuccessful as you could not complete it in time.", CreatedAt: at(2)},
		{ID: "pay_lost", OrderID: "order_b", Status: "failed", Method: "card", Email: "ravi@example.test", Amount: 6900, Currency: "USD",
			ErrorReason: "card_declined", ErrorDescription: "Carte refusée — 卡被拒绝", CreatedAt: at(3)},
		// A capture before the failure does not recover it.
		{ID: "pay_older", OrderID: "order_c", Status: "captured", Email: "ravi@example.test", CreatedAt: at(4)},
		{ID: "pay_no_email", Status: "failed", Method: "upi", CreatedAt: at(5)},
		{ID: "pay_created", Status: "created", Email: "ravi@example.test", CreatedAt: at(6)},
	}
	svc, _ := fakePayments(t, items, 0)

	got, err := svc.ListFailedPayments(t.Context(), now)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Since.Equal(now.Add(-30*24*time.Hour)) || got.Truncated {
		t.Fatalf("since %v truncated %v", got.Since, got.Truncated)
	}
	type row struct {
		id        string
		paidLater bool
	}
	var rows []row
	for _, p := range got.Payments {
		rows = append(rows, row{p.ID, p.PaidLater})
	}
	want := []row{{"pay_timeout", true}, {"pay_lost", false}, {"pay_no_email", false}}
	if !slices.Equal(rows, want) {
		t.Fatalf("rows = %v, want %v", rows, want)
	}
	first := got.Payments[0]
	if first.Email != "asha@example.test" || first.Method != "upi" || first.Amount != 499900 || first.Currency != "INR" ||
		first.Code != "payment_timed_out" || first.Reason == "" || !first.At.Equal(now.Add(-2*time.Hour)) {
		t.Fatalf("first = %+v", first)
	}
	if got.Payments[1].Reason != "Carte refusée — 卡被拒绝" {
		t.Fatalf("unicode reason = %q", got.Payments[1].Reason)
	}
}

func TestListFailedPaymentsEmptyAndPaging(t *testing.T) {
	now := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name          string
		n             int
		wantCalls     int32
		wantTruncated bool
	}{
		{"none", 0, 1, false},
		{"exactly one page", 100, 2, false},
		{"over the 1,000 cap", 1050, 10, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			items := make([]razorpay.PaymentAttempt, tc.n)
			for i := range items {
				items[i] = razorpay.PaymentAttempt{ID: fmt.Sprintf("pay_%d", i), Status: "failed", CreatedAt: now.Unix()}
			}
			svc, calls := fakePayments(t, items, 0)
			got, err := svc.ListFailedPayments(t.Context(), now)
			if err != nil {
				t.Fatal(err)
			}
			if got.Payments == nil || len(got.Payments) != min(tc.n, 1000) || got.Truncated != tc.wantTruncated || calls.Load() != tc.wantCalls {
				t.Fatalf("payments %d truncated %v calls %d", len(got.Payments), got.Truncated, calls.Load())
			}
		})
	}
}

func TestListFailedPaymentsErrors(t *testing.T) {
	svc, _ := fakePayments(t, nil, http.StatusInternalServerError)
	if _, err := svc.ListFailedPayments(t.Context(), time.Now()); !errors.Is(err, ErrPaymentProvider) {
		t.Fatalf("provider down: err = %v, want ErrPaymentProvider", err)
	}
	if _, err := NewService(nil, nil, "", "").ListFailedPayments(t.Context(), time.Now()); !errors.Is(err, ErrPaymentUnavailable) {
		t.Fatalf("no Razorpay: err = %v, want ErrPaymentUnavailable", err)
	}
}
