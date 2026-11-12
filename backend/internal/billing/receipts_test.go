package billing

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
)

// receiptsServer serves the receipt routes as user.
func receiptsServer(svc *Service, user auth.User) http.Handler {
	mux := http.NewServeMux()
	mountReceipts(mux, Deps{
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		Service: svc,
		WithSession: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r.WithContext(auth.WithUser(r.Context(), user)))
			})
		},
	})
	return mux
}

func call(t *testing.T, h http.Handler, method, path, body string) (int, string) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func TestReceiptRoutesRefuseMembersBeforeTouchingTheDatabase(t *testing.T) {
	// No database: a request that is not refused first panics.
	h := receiptsServer(&Service{}, auth.User{OrganizationID: "org-1", Role: "member"})
	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/billing/payments"},
		{http.MethodGet, "/api/v1/billing/payments/pay_1/receipt"},
		{http.MethodGet, "/api/v1/billing/details"},
		{http.MethodPost, "/api/v1/billing/details"},
	} {
		code, body := call(t, h, route.method, route.path, `{"name":"x"}`)
		if code != http.StatusForbidden || !strings.Contains(body, `"forbidden"`) {
			t.Fatalf("%s %s = %d %s, want 403", route.method, route.path, code, body)
		}
	}
}

func TestPaymentsAndReceiptsShowOnlyTheWorkspacesOwnPayments(t *testing.T) {
	ctx := t.Context()
	f := newFixture(ctx, t)
	other := newFixture(ctx, t)
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.WithoutCancel(ctx), `DELETE FROM go_billing_payments WHERE organization_id = ANY($1)`, []string{f.org, other.org})
	})
	march := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	record := func(org, id, kind, item string, amount int64, currency string, at time.Time) {
		t.Helper()
		if err := f.svc.repo.recordPayment(ctx, f.pool, id, org, kind, item, amount, currency, at, sourceWebhook); err != nil {
			t.Fatal(err)
		}
	}
	record(f.org, "pay_a_"+f.org, "subscription", PlanPro, 249900, "INR", march)
	record(f.org, "pay_b_"+f.org, "topup", "credits_5000", 1900, "USD", march.AddDate(0, 1, 0))
	record(other.org, "pay_x_"+other.org, "subscription", PlanPro, 249900, "INR", march)
	// A webhook redelivery must not give the payment a second number.
	record(f.org, "pay_a_"+f.org, "subscription", PlanPro, 249900, "INR", march)
	if _, _, err := f.svc.repo.raiseRefunded(ctx, "pay_b_"+f.org, 500); err != nil {
		t.Fatal(err)
	}

	h := receiptsServer(f.svc, auth.User{OrganizationID: f.org, Role: "admin"})
	code, body := call(t, h, http.MethodGet, "/api/v1/billing/payments", "")
	if code != http.StatusOK {
		t.Fatalf("payments = %d %s", code, body)
	}
	var list struct{ Payments []PaymentReceipt }
	if err := json.Unmarshal([]byte(body), &list); err != nil {
		t.Fatal(err)
	}
	if len(list.Payments) != 2 {
		t.Fatalf("payments = %+v, want this workspace's two", list.Payments)
	}
	newest, oldest := list.Payments[0], list.Payments[1]
	if newest.PaymentID != "pay_b_"+f.org || newest.Item != "5,000 credits" || newest.Currency != "USD" || newest.Amount != 1900 || newest.Refunded != 500 {
		t.Fatalf("newest = %+v", newest)
	}
	if oldest.Item != "Uplumo Pro plan" || !strings.HasPrefix(oldest.ReceiptNo, "UPL-2026-") || oldest.ReceiptNo >= newest.ReceiptNo {
		t.Fatalf("oldest = %+v, newest number %s", oldest, newest.ReceiptNo)
	}

	// Another workspace's payment id answers as if it did not exist.
	if code, body := call(t, h, http.MethodGet, "/api/v1/billing/payments/pay_x_"+other.org+"/receipt", ""); code != http.StatusNotFound {
		t.Fatalf("other workspace's receipt = %d %s, want 404", code, body)
	}

	if code, body := call(t, h, http.MethodPost, "/api/v1/billing/details", `{"name":"  Müller GmbH  ","address":"Straße 1\r\n10115 Berlin","country":"Germany","taxId":"DE123456789"}`); code != http.StatusOK {
		t.Fatalf("save details = %d %s", code, body)
	}
	code, body = call(t, h, http.MethodGet, "/api/v1/billing/payments/pay_a_"+f.org+"/receipt", "")
	if code != http.StatusOK {
		t.Fatalf("receipt = %d %s", code, body)
	}
	var receipt Receipt
	if err := json.Unmarshal([]byte(body), &receipt); err != nil {
		t.Fatal(err)
	}
	want := Details{Name: "Müller GmbH", Address: "Straße 1\n10115 Berlin", Country: "Germany", TaxID: "DE123456789"}
	if receipt.ReceiptNo != oldest.ReceiptNo || receipt.Amount != 249900 || receipt.BilledTo != want {
		t.Fatalf("receipt = %+v", receipt)
	}

	// The other workspace sees none of it.
	otherH := receiptsServer(other.svc, auth.User{OrganizationID: other.org, Role: "owner"})
	if _, body := call(t, otherH, http.MethodGet, "/api/v1/billing/details", ""); !strings.Contains(body, `"name":""`) {
		t.Fatalf("other workspace's details = %s, want empty", body)
	}
}

func TestSaveBillingDetailsValidates(t *testing.T) {
	f := newFixture(t.Context(), t)
	h := receiptsServer(f.svc, auth.User{OrganizationID: f.org, Role: "owner"})
	tests := []struct {
		name, body string
		wantCode   int
		wantBody   string
	}{
		{"name only", `{"name":"Acme"}`, http.StatusOK, `"name":"Acme"`},
		{"longest name", `{"name":"` + strings.Repeat("é", 200) + `"}`, http.StatusOK, `"name":"é`},
		{"unicode", `{"name":"株式会社テスト","country":"日本"}`, http.StatusOK, `"country":"日本"`},
		{"blank name", `{"name":"   "}`, http.StatusBadRequest, "Name is required."},
		{"name too long", `{"name":"` + strings.Repeat("é", 201) + `"}`, http.StatusBadRequest, "Name must be 200 characters or fewer."},
		{"newline in name", `{"name":"A\nB"}`, http.StatusBadRequest, "Name has invalid characters."},
		{"control in tax id", `{"name":"A","taxId":"X\u0000"}`, http.StatusBadRequest, "Tax ID has invalid characters."},
		{"tax id too long", `{"name":"A","taxId":"` + strings.Repeat("9", 41) + `"}`, http.StatusBadRequest, "Tax ID must be 40 characters or fewer."},
		{"body too large", `{"name":"A","address":"` + strings.Repeat("a", 5000) + `"}`, http.StatusBadRequest, "invalid_request"},
		{"not json", `name=A`, http.StatusBadRequest, "invalid_request"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, body := call(t, h, http.MethodPost, "/api/v1/billing/details", tt.body)
			if code != tt.wantCode || !strings.Contains(body, tt.wantBody) {
				t.Fatalf("= %d %s, want %d with %s", code, body, tt.wantCode, tt.wantBody)
			}
		})
	}
}
