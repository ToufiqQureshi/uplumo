package staff

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/database"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/pgdb"
)

func pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("TEST_DATABASE_URL required")
		}
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := pgdb.Open(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	if err = database.Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	return db
}

func exec(t *testing.T, db *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := db.Exec(t.Context(), sql, args...); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
}

// seed creates a verified user owning a one-person workspace.
func seed(t *testing.T, db *pgxpool.Pool) auth.User {
	t.Helper()
	id := strings.ToLower(rand.Text())
	u := auth.User{ID: "staff-test-user-" + id, OrganizationID: "staff-test-org-" + id, Email: id + "@example.test", Role: "owner"}
	exec(t, db, `INSERT INTO organization(id,name,slug,created_at) VALUES($1,'Acme',$1,now())`, u.OrganizationID)
	exec(t, db, `INSERT INTO "user"(id,name,email,email_verified) VALUES($1,'Asha',$2,true)`, u.ID, u.Email)
	exec(t, db, `INSERT INTO member(id,organization_id,user_id,role,created_at) VALUES($1,$2,$3,'owner',now())`, "member-"+id, u.OrganizationID, u.ID)
	t.Cleanup(func() {
		ctx := context.WithoutCancel(t.Context())
		_, _ = db.Exec(ctx, `DELETE FROM organization WHERE id=$1`, u.OrganizationID)
		_, _ = db.Exec(ctx, `DELETE FROM "user" WHERE id=$1`, u.ID)
		_, _ = db.Exec(ctx, `DELETE FROM go_billing_payments WHERE organization_id=$1`, u.OrganizationID)
		_, _ = db.Exec(ctx, `DELETE FROM go_staff_actions WHERE target_user_id=$1 OR actor_user_id=$1`, u.ID)
	})
	return u
}

func TestBlockEndsSessionsAndUnblockRestores(t *testing.T) {
	db := pool(t)
	staff, user := seed(t, db), seed(t, db)
	a := Actions{DB: db, StaffEmails: []string{staff.Email}}
	exec(t, db, `INSERT INTO session(id,token,user_id,expires_at,updated_at) VALUES($1,$1,$2,now()+interval '1 day',now())`, "tok-"+user.ID, user.ID)

	if err := a.Block(t.Context(), staff.ID, user.ID, "spam"); err != nil {
		t.Fatal(err)
	}
	if err := a.Block(t.Context(), staff.ID, user.ID, "again"); err != nil {
		t.Fatalf("second block must be a no-op: %v", err)
	}
	var sessions, blocks int
	if err := db.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM session WHERE user_id=$1),(SELECT count(*) FROM go_user_blocks WHERE user_id=$1)`, user.ID).Scan(&sessions, &blocks); err != nil {
		t.Fatal(err)
	}
	if sessions != 0 || blocks != 1 {
		t.Fatalf("after block sessions=%d blocks=%d, want 0 and 1", sessions, blocks)
	}
	if err := a.Unblock(t.Context(), staff.ID, user.ID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(t.Context(), `SELECT count(*) FROM go_user_blocks WHERE user_id=$1`, user.ID).Scan(&blocks); err != nil || blocks != 0 {
		t.Fatalf("after unblock blocks=%d, %v", blocks, err)
	}
	actions, err := a.Recent(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var mine []string
	for _, v := range actions {
		if v.TargetID == user.ID {
			mine = append(mine, v.Action)
		}
	}
	if strings.Join(mine, ",") != "unblock,block,block" {
		t.Fatalf("logged actions = %v", mine)
	}
}

func TestStaffAndSelfAreProtected(t *testing.T) {
	db := pool(t)
	staff, other := seed(t, db), seed(t, db)
	a := Actions{DB: db, StaffEmails: []string{staff.Email, other.Email}}
	for _, target := range []string{staff.ID, other.ID} {
		if err := a.Block(t.Context(), staff.ID, target, ""); !errors.Is(err, ErrProtectedUser) {
			t.Fatalf("Block(%s) = %v, want ErrProtectedUser", target, err)
		}
		if err := a.Delete(t.Context(), staff.ID, target, ""); !errors.Is(err, ErrProtectedUser) {
			t.Fatalf("Delete(%s) = %v, want ErrProtectedUser", target, err)
		}
	}
}

func TestDeleteRefusesPaidPlanAndWrongConfirmThenErasesKeepingPayments(t *testing.T) {
	db := pool(t)
	staff, user := seed(t, db), seed(t, db)
	a := Actions{DB: db, StaffEmails: []string{staff.Email}}
	exec(t, db, `INSERT INTO go_billing_subscriptions(organization_id,razorpay_subscription_id,plan,status,event_at) VALUES($1,'sub-'||$1,'pro','active',now())`, user.OrganizationID)
	exec(t, db, `INSERT INTO go_billing_payments(payment_id,organization_id,kind,item,amount,currency,paid_at,source) VALUES('pay-'||$1,$1,'subscription','pro',249900,'INR',now(),'webhook')`, user.OrganizationID)

	if err := a.Delete(t.Context(), staff.ID, user.ID, user.Email); !errors.Is(err, ErrLiveSubscription) {
		t.Fatalf("delete with live plan = %v, want ErrLiveSubscription", err)
	}
	exec(t, db, `UPDATE go_billing_subscriptions SET status='cancelled' WHERE organization_id=$1`, user.OrganizationID)
	if err := a.Delete(t.Context(), staff.ID, user.ID, "someone@else.test"); !errors.Is(err, ErrConfirmMismatch) {
		t.Fatalf("delete with wrong confirm = %v, want ErrConfirmMismatch", err)
	}
	if err := a.Delete(t.Context(), staff.ID, user.ID, strings.ToUpper(user.Email)); err != nil {
		t.Fatal(err)
	}
	var users, orgs, payments, logged int
	if err := db.QueryRow(t.Context(), `SELECT (SELECT count(*) FROM "user" WHERE id=$1),(SELECT count(*) FROM organization WHERE id=$2),
		(SELECT count(*) FROM go_billing_payments WHERE organization_id=$2),(SELECT count(*) FROM go_staff_actions WHERE target_user_id=$1 AND action='delete')`,
		user.ID, user.OrganizationID).Scan(&users, &orgs, &payments, &logged); err != nil {
		t.Fatal(err)
	}
	if users != 0 || orgs != 0 || payments != 1 || logged != 1 {
		t.Fatalf("after delete users=%d orgs=%d payments=%d logged=%d; want 0 0 1 1", users, orgs, payments, logged)
	}
}

func TestOverviewCountsRealRevenueAndMRR(t *testing.T) {
	db := pool(t)
	user := seed(t, db)
	s := Stats{DB: db}
	before, err := s.Overview(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	exec(t, db, `INSERT INTO go_billing_subscriptions(organization_id,razorpay_subscription_id,plan,status,event_at) VALUES($1,'sub-'||$1,'pro','active',now())`, user.OrganizationID)
	exec(t, db, `INSERT INTO go_billing_payments(payment_id,organization_id,kind,item,amount,amount_refunded,currency,paid_at,source) VALUES('pay-'||$1,$1,'subscription','pro',249900,49900,'INR',now(),'webhook')`, user.OrganizationID)
	after, err := s.Overview(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if after.PayingWorkspaces-before.PayingWorkspaces != 1 {
		t.Fatalf("paying workspaces +%d, want +1", after.PayingWorkspaces-before.PayingWorkspaces)
	}
	if got := after.MRR[0].Amount - before.MRR[0].Amount; after.MRR[0].Currency != "INR" || got != 249900 {
		t.Fatalf("INR MRR +%d, want +249900 (Pro ₹2,499)", got)
	}
	if got := after.CollectedThisMonth[0].Amount - before.CollectedThisMonth[0].Amount; got != 200000 {
		t.Fatalf("collected this month +%d, want +200000 net of the refund", got)
	}
	if len(after.SignupsByDay) != 30 {
		t.Fatalf("signups by day has %d days, want 30", len(after.SignupsByDay))
	}
	if _, err := s.Marketing(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Finance(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Subscriptions(t.Context()); err != nil {
		t.Fatal(err)
	}
}

type fixedStaff map[string]bool

func (f fixedStaff) IsStaff(_ context.Context, u auth.User) (bool, error) { return f[u.ID], nil }

func TestRoutesAreHiddenFromEveryoneButStaff(t *testing.T) {
	db := pool(t)
	staff, user := seed(t, db), seed(t, db)
	var as *auth.User
	mux := http.NewServeMux()
	Mount(mux, Deps{
		Staff: fixedStaff{staff.ID: true}, Stats: Stats{DB: db}, Actions: Actions{DB: db, StaffEmails: []string{staff.Email}},
		WithSession: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if as != nil {
					r = r.WithContext(auth.WithUser(r.Context(), *as))
				}
				next.ServeHTTP(w, r)
			})
		},
	})
	call := func(u *auth.User, method, path, body string) int {
		as = u
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, path, bytes.NewBufferString(body)))
		return rec.Code
	}
	block := `{"userId":"` + user.ID + `"}`
	for _, path := range []string{"/api/v1/staff/overview", "/api/v1/staff/subscriptions", "/api/v1/staff/marketing", "/api/v1/staff/finance", "/api/v1/staff/actions"} {
		if got := call(&user, "GET", path, ""); got != 404 {
			t.Fatalf("customer GET %s = %d, want 404", path, got)
		}
		if got := call(nil, "GET", path, ""); got != 404 {
			t.Fatalf("anonymous GET %s = %d, want 404", path, got)
		}
		if got := call(&staff, "GET", path, ""); got != 200 {
			t.Fatalf("staff GET %s = %d, want 200", path, got)
		}
	}
	if got := call(&user, "POST", "/api/v1/staff/users/block", block); got != 404 {
		t.Fatalf("customer block = %d, want 404", got)
	}
	if got := call(&staff, "POST", "/api/v1/staff/payments/sync", ""); got != 503 {
		t.Fatalf("sync without Razorpay = %d, want 503", got)
	}
	if got := call(&user, "GET", "/api/v1/staff/payments/failed", ""); got != 404 {
		t.Fatalf("customer failed payments = %d, want 404", got)
	}
	if got := call(&staff, "GET", "/api/v1/staff/payments/failed", ""); got != 503 {
		t.Fatalf("failed payments without Razorpay = %d, want 503", got)
	}
	if got := call(&staff, "POST", "/api/v1/staff/users/block", block); got != 200 {
		t.Fatalf("staff block = %d, want 200", got)
	}
}
