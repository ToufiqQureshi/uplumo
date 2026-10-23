package dataforseo

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend/spendtest"
)

func TestExplicitProviderRejectionReleasesTheHold(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }))
			defer server.Close()
			guard := &spendtest.Guard{}
			client := guardedClient(t, server, &testRecorder{}, guard, 0)
			_, err := client.Do(t.Context(), "org", spend.Backlinks, http.MethodPost, "/v3/backlinks/summary/live", []byte(`[{}]`), false)
			settled := guard.Settled()
			if err == nil || errors.Is(err, ErrOutcomeUnknown) || len(settled) != 1 || settled[0] != 0 {
				t.Fatalf("rejected task left a hold: err=%v, settled=%v", err, settled)
			}
		})
	}
}

func TestUnconfirmedProviderResponseNeverRefundsTheHold(t *testing.T) {
	for _, body := range []string{"", `{"tasks":`, `{}`, `null`, `{"tasks":[]}`, `{"tasks":[{"path":["v3","backlinks"],"status_code":20000}]}`} {
		t.Run(body, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls++
				_, _ = io.WriteString(w, body)
			}))
			defer server.Close()
			for _, budgeted := range []bool{false, true} {
				guard := &spendtest.Guard{}
				client := guardedClient(t, server, &testRecorder{}, guard, 0)
				var err error
				if budgeted {
					err = client.DoBudgeted(t.Context(), "org", spend.Backlinks, 0.1, func(request DoBudgetedRequest) error {
						_, _ = request(http.MethodPost, "/v3/backlinks/summary/live", []byte(`[{}]`), false)
						return nil // a caller cannot swallow an uncertain purchase into a refund
					})
				} else {
					_, err = client.Do(t.Context(), "org", spend.Backlinks, http.MethodPost, "/v3/backlinks/summary/live", []byte(`[{}]`), false)
				}
				if err == nil || len(guard.Settled()) != 0 || len(guard.Held()) != 1 {
					t.Fatalf("budgeted=%v: err=%v, settled=%v; want an error and the hold kept for reconciliation", budgeted, err, guard.Settled())
				}
			}
			if calls != 2 {
				t.Fatalf("paid calls=%d, want one per explicit operation", calls)
			}
		})
	}
}

func TestAccountCapRejectionReleasesCreditsAndStopsBudgetLegs(t *testing.T) {
	for _, body := range []string{`{"status_code":40203,"tasks":null}`, `{"status_code":20000,"tasks":[{"status_code":40203}]}`} {
		t.Run(body, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls++; _, _ = io.WriteString(w, body) }))
			defer server.Close()
			guard := &spendtest.Guard{}
			client := guardedClient(t, server, &testRecorder{}, guard, 0)
			err := client.DoBudgeted(t.Context(), "org", spend.Backlinks, 0.1, func(request DoBudgetedRequest) error {
				payload, err := request(http.MethodPost, "/v3/backlinks/summary/live", []byte(`[{}]`), false)
				if err != nil {
					return err
				}
				_, err = Results(payload)
				return err
			})
			settled := guard.Settled()
			if !errors.Is(err, ErrBillingIssue) || errors.Is(err, ErrOutcomeUnknown) || len(settled) != 1 || settled[0] != 0 || calls != 1 {
				t.Fatalf("account cap: err=%v, settled=%v, calls=%d; want known refusal with zero charge and one request", err, settled, calls)
			}
		})
	}
}
