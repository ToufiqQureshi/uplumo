package customers

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/csv"
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
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

func seed(t *testing.T, db *pgxpool.Pool, verified bool) auth.User {
	t.Helper()
	id := strings.ToLower(rand.Text())
	user := auth.User{ID: "customers-user-" + id, OrganizationID: "customers-org-" + id, Email: id + "@example.test", Role: "owner"}
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO organization(id,name,slug,created_at) VALUES($1,'Acme',$1,now())`, []any{user.OrganizationID}},
		{`INSERT INTO "user"(id,name,email,email_verified) VALUES($1,'=HYPERLINK("x")',$2,$3)`, []any{user.ID, user.Email, verified}},
		{`INSERT INTO member(id,organization_id,user_id,role,created_at) VALUES($1,$2,$3,'owner',now())`, []any{"member-" + id, user.OrganizationID, user.ID}},
	} {
		if _, err := db.Exec(t.Context(), q.sql, q.args...); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = db.Exec(ctx, `DELETE FROM organization WHERE id=$1`, user.OrganizationID)
		_, _ = db.Exec(ctx, `DELETE FROM "user" WHERE id=$1`, user.ID)
	})
	return user
}

func TestOnlyVerifiedStaffSeeCustomersWithOnboardingAndSites(t *testing.T) {
	db := pool(t)
	ctx := t.Context()
	staff := seed(t, db, true)
	unverifiedStaff := seed(t, db, false)
	customer := seed(t, db, true)
	for _, q := range []string{
		`INSERT INTO projects(id,organization_id,name) VALUES('default-'||$1,$1,'Default')`,
		`INSERT INTO projects(id,organization_id,name,domain) VALUES('site-'||$1,$1,'Shop','shop.com')`,
		`INSERT INTO go_billing_subscriptions(organization_id,razorpay_subscription_id,plan,status,event_at) VALUES($1,'sub-'||$1,'pro','active',now())`,
	} {
		if _, err := db.Exec(ctx, q, customer.OrganizationID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(ctx, `INSERT INTO user_onboarding_answers(user_id,organization_id,interested_features,work_for,found_via) VALUES($1,$2,'["Rank tracking"]','My clients','YouTube')`, customer.ID, customer.OrganizationID); err != nil {
		t.Fatal(err)
	}
	s := &Service{DB: db, StaffEmails: []string{staff.Email, unverifiedStaff.Email}}

	for _, u := range []auth.User{customer, unverifiedStaff} {
		if _, err := s.List(ctx, u); err != ErrForbidden {
			t.Fatalf("List(%s) error = %v, want ErrForbidden", u.Email, err)
		}
	}
	list, err := s.List(ctx, staff)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(list, func(c Customer) bool { return c.UserID == customer.ID })
	if i < 0 {
		t.Fatal("customer missing from list")
	}
	got := list[i]
	if got.Plan != "pro" || got.ActiveSites != 1 || !slices.Equal(got.Domains, []string{"shop.com"}) ||
		got.InterestedFeatures != `["Rank tracking"]` || got.WorkFor != "My clients" || got.FoundVia != "YouTube" || got.Workspace != "Acme" {
		t.Fatalf("customer row = %+v", got)
	}

	var buf bytes.Buffer
	if err := WriteCSV(&buf, []Customer{got}); err != nil {
		t.Fatal(err)
	}
	records, err := csv.NewReader(&buf).ReadAll()
	if err != nil || len(records) != 2 {
		t.Fatalf("csv = %v, %v", records, err)
	}
	if name := records[1][1]; name != `'=HYPERLINK("x")` {
		t.Fatalf("name cell = %q, want formula neutralised", name)
	}
}

func TestHandlersRequireSessionAndStaff(t *testing.T) {
	db := pool(t)
	staff := seed(t, db, true)
	customer := seed(t, db, true)
	mux := http.NewServeMux()
	var as *auth.User
	Mount(mux, Deps{Service: &Service{DB: db, StaffEmails: []string{staff.Email}}, WithSession: func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if as != nil {
				r = r.WithContext(auth.WithUser(r.Context(), *as))
			}
			next.ServeHTTP(w, r)
		})
	}})
	for _, c := range []struct {
		user *auth.User
		path string
		want int
	}{
		{nil, "/api/v1/admin/customers", 401},
		{&customer, "/api/v1/admin/customers", 403},
		{&customer, "/api/v1/admin/customers.csv", 403},
		{&staff, "/api/v1/admin/customers", 200},
		{&staff, "/api/v1/admin/customers.csv", 200},
	} {
		as = c.user
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest("GET", c.path, nil))
		if rec.Code != c.want {
			t.Fatalf("%s as %v = %d, want %d", c.path, c.user, rec.Code, c.want)
		}
	}
}
