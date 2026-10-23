package billing

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

func TestStaleProviderHoldWithUnknownOutcomeIsNotRefunded(t *testing.T) {
	pool := setupTestDB(t)
	ledger := NewCreditLedgerService(pool)
	org := spendOrg(t.Context(), t, pool, 1000)
	guard := testGuard(t, pool, ledger, noDailyLimit, true)
	ctx, cancel := context.WithCancel(t.Context())
	hold, err := guard.Hold(ctx, spend.Request{OrganizationID: org, Feature: spend.Backlinks, EstimateUSD: 0.1})
	cancel() // simulate a process disappearing before recording the provider outcome
	if err != nil {
		t.Fatal(err)
	}
	id := hold.(*creditHold).reservationID
	if _, err := pool.Exec(t.Context(), `UPDATE go_billing_credit_reservations SET updated_at=now()-interval '2 hours' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.RefundStaleReservations(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := balanceOf(t.Context(), t, ledger, org); got != 872 {
		t.Fatalf("unknown provider outcome restored credits: balance = %d, want 872 still reserved", got)
	}
}

func TestZeroEstimateRecordsActualProviderSpendOnce(t *testing.T) {
	for _, balance := range []int64{1000, 0} {
		t.Run(fmt.Sprint(balance), func(t *testing.T) {
			pool := setupTestDB(t)
			ledger := NewCreditLedgerService(pool)
			org := spendOrg(t.Context(), t, pool, balance)
			guard := testGuard(t, pool, ledger, noDailyLimit, true)
			hold, err := guard.Hold(t.Context(), spend.Request{OrganizationID: org, Feature: spend.Backlinks})
			if err != nil {
				t.Fatal(err)
			}
			if err := hold.Settle(t.Context(), 0.02); err != nil {
				t.Fatal(err)
			}
			if cost, _ := providerSpend(t, pool, org); cost != 0.02 {
				t.Fatalf("provider spend = %v, want exactly 0.02", cost)
			}
		})
	}
}

func TestFailedSettlementIsRecoveredWithoutRefundingPaidWork(t *testing.T) {
	pool := setupTestDB(t)
	ledger := NewCreditLedgerService(pool)
	org := spendOrg(t.Context(), t, pool, 1000)
	guard := testGuard(t, pool, ledger, noDailyLimit, true)
	hold, err := guard.Hold(t.Context(), spend.Request{OrganizationID: org, Feature: spend.Backlinks, EstimateUSD: 0.1})
	if err != nil {
		t.Fatal(err)
	}
	name := pgx.Identifier{"audit_settle_" + rand.Text()}.Sanitize()
	if _, err := pool.Exec(t.Context(), fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.organization_id='%s' AND NEW.type='settle' THEN RAISE EXCEPTION 'audit settlement failure'; END IF; RETURN NEW; END $$;
CREATE TRIGGER %s BEFORE INSERT ON go_billing_credit_ledger FOR EACH ROW EXECUTE FUNCTION %s()`, name, org, name, name)); err != nil {
		t.Fatal(err)
	}
	cleanup := func() {
		if _, err := pool.Exec(context.Background(), fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON go_billing_credit_ledger; DROP FUNCTION IF EXISTS %s()`, name, name)); err != nil {
			t.Error(err)
		}
	}
	t.Cleanup(cleanup)
	if err := hold.Settle(t.Context(), 0.02); err == nil {
		t.Fatal("injected settlement failure was ignored")
	}
	cleanup()
	id := hold.(*creditHold).reservationID
	if _, err := pool.Exec(t.Context(), `UPDATE go_billing_credit_reservations SET updated_at=now()-interval '2 hours' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := ledger.RefundStaleReservations(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := balanceOf(t.Context(), t, ledger, org); got != 974 {
		t.Fatalf("balance = %d, want 974 after recovering the 26-credit charge", got)
	}
	if cost, _ := providerSpend(t, pool, org); cost != 0.02 {
		t.Fatalf("provider cost = %v, want exactly 0.02", cost)
	}
	if _, err := ledger.RefundStaleReservations(t.Context()); err != nil {
		t.Fatal(err)
	}
	if cost, _ := providerSpend(t, pool, org); cost != 0.02 {
		t.Fatalf("reconciliation repeated provider spend: %v", cost)
	}
}
