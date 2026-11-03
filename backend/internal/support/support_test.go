package support

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"image"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/database"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/ids"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/pgdb"
)

func input(t *testing.T) Input {
	t.Helper()
	id, err := ids.New()
	if err != nil {
		t.Fatal(err)
	}
	return Input{Kind: "bug", Description: "The credits display is wrong.", PagePath: "/billing", AppVersion: "test", SubmissionID: id}
}
func screenshot(t *testing.T) string {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes())
}
func TestValidationBoundsUntrustedDiagnostics(t *testing.T) {
	for _, mutate := range []func(*Input){
		func(i *Input) { i.Kind = "other" }, func(i *Input) { i.Description = "short" }, func(i *Input) { i.Description = strings.Repeat("x", 5001) }, func(i *Input) { i.SubmissionID = "not-uuid" },
		func(i *Input) { i.PagePath = "https://evil.test" }, func(i *Input) { i.PagePath = "//evil.test" }, func(i *Input) { i.PagePath = "/billing?token=secret" }, func(i *Input) { i.PagePath = "/a#secret" }, func(i *Input) { i.PagePath = "/a%0afoo" }, func(i *Input) { i.PagePath = "/a%5cfoo" },
		func(i *Input) { i.RequestID = strings.Repeat("a", 129) }, func(i *Input) { i.AppVersion = "v1\nsecret" }, func(i *Input) { i.Screenshot = "data:image/svg+xml;base64,AAAA" }, func(i *Input) {
			i.Screenshot = "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("not an image"))
		}, func(i *Input) {
			i.Screenshot = "data:image/png;base64," + strings.Repeat("A", base64.StdEncoding.EncodedLen(MaxScreenshotBytes)+1)
		},
	} {
		in := input(t)
		mutate(&in)
		if _, _, err := validate(in); !errors.Is(err, ErrInvalid) {
			t.Fatalf("invalid input accepted: %+v", in)
		}
	}
	in := input(t)
	in.Screenshot = screenshot(t)
	if data, mime, err := validate(in); err != nil || len(data) == 0 || mime != "image/png" {
		t.Fatalf("valid screenshot: %s %v", mime, err)
	}
}
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
func seed(t *testing.T, db *pgxpool.Pool) auth.User {
	t.Helper()
	id := rand.Text()
	user := auth.User{ID: "support-user-" + id, OrganizationID: "support-org-" + id, Email: id + "@example.test", Role: "owner"}
	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO organization(id,name,slug,created_at) VALUES($1,'Support test',$1,now())`, []any{user.OrganizationID}},
		{`INSERT INTO "user"(id,name,email,email_verified) VALUES($1,'Reporter',$2,true)`, []any{user.ID, user.Email}},
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
func TestCreateIdempotencyQuotaAndPrivateScreenshots(t *testing.T) {
	db := pool(t)
	ctx := t.Context()
	user := seed(t, db)
	other := seed(t, db)
	s := Service{DB: db}
	in := input(t)
	in.Screenshot = screenshot(t)
	var wg sync.WaitGroup
	results := make(chan string, 2)
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); id, err := s.Create(ctx, user, in); results <- id; errs <- err }()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var ticket string
	for id := range results {
		if ticket != "" && id != ticket {
			t.Fatal("retry created a second report")
		}
		ticket = id
	}
	in.Description = "Changed payload for same submission."
	if _, err := s.Create(ctx, user, in); !errors.Is(err, ErrConflict) {
		t.Fatalf("changed retry: %v", err)
	}
	if _, _, err := s.Screenshot(ctx, other, ticket); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-tenant read: %v", err)
	}
	if data, mime, err := s.Screenshot(ctx, user, ticket); err != nil || len(data) == 0 || mime != "image/png" {
		t.Fatalf("own screenshot: %s %v", mime, err)
	}
	// Ordinary customers invited to a staff-owned workspace are not staff themselves.
	s.StaffEmails = []string{strings.ToLower(user.Email)}
	if _, err := db.Exec(ctx, `INSERT INTO member(id,organization_id,user_id,role,created_at) VALUES($1,$2,$3,'member',now())`, rand.Text(), user.OrganizationID, other.ID); err != nil {
		t.Fatal(err)
	}
	third := seed(t, db)
	thirdInput := input(t)
	thirdInput.Screenshot = screenshot(t)
	thirdTicket, err := s.Create(ctx, third, thirdInput)
	if err != nil {
		t.Fatal(err)
	}
	invited := other
	invited.OrganizationID = user.OrganizationID
	if _, _, err = s.Screenshot(ctx, invited, thirdTicket); !errors.Is(err, ErrNotFound) {
		t.Fatalf("member inherited staff: %v", err)
	}
	if _, _, err = s.Screenshot(ctx, user, thirdTicket); err != nil {
		t.Fatalf("verified staff denied: %v", err)
	}
	if _, err = db.Exec(ctx, `UPDATE "user" SET email_verified=false WHERE id=$1`, user.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.Screenshot(ctx, user, thirdTicket); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unverified staff accepted: %v", err)
	}
	for range 4 {
		if _, err := s.Create(ctx, user, input(t)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.Create(ctx, user, input(t)); !errors.Is(err, ErrRateLimit) {
		t.Fatalf("sixth report accepted: %v", err)
	}
	// Previously accepted retries remain valid after the quota is reached.
	in.Description = "The credits display is wrong."
	if id, err := s.Create(ctx, user, in); err != nil || id != ticket {
		t.Fatalf("retry after quota: %s %v", id, err)
	}
	if _, err = db.Exec(ctx, `DELETE FROM "user" WHERE id=$1`, user.ID); err != nil {
		t.Fatal(err)
	}
	var n int
	if err = db.QueryRow(ctx, `SELECT count(*) FROM go_support_reports WHERE user_id=$1`, user.ID).Scan(&n); err != nil || n != 0 {
		t.Fatalf("user deletion retained reports: %d %v", n, err)
	}
}

type fakeMailer struct {
	err   error
	calls int
	vars  map[string]string
}

func (m *fakeMailer) SendTransactional(_ context.Context, to, template string, vars map[string]string) error {
	m.calls++
	m.vars = vars
	if to != "support@uplumo.com" || template != "template" {
		return errors.New("wrong mail destination")
	}
	return m.err
}
func TestNotificationFailureRetainsSavedReportAndRetries(t *testing.T) {
	db := pool(t)
	ctx := t.Context()
	user := seed(t, db)
	s := Service{DB: db, PublicURL: "https://uplumo.com"}
	in := input(t)
	in.Screenshot = screenshot(t)
	id, err := s.Create(ctx, user, in)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.NotifyPending(ctx); err != nil {
		t.Fatal(err)
	}
	var attempts int
	if err = db.QueryRow(ctx, `SELECT notification_attempts FROM go_support_reports WHERE id=$1`, id).Scan(&attempts); err != nil || attempts != 0 {
		t.Fatalf("missing config consumed attempt: %d %v", attempts, err)
	}
	m := &fakeMailer{err: errors.New("email down")}
	s.Mailer = m
	s.TemplateID = "template"
	if _, err = s.NotifyPending(ctx); err == nil {
		t.Fatal("email failure not returned")
	}
	var notified bool
	if err = db.QueryRow(ctx, `SELECT notification_attempts,notified_at IS NOT NULL FROM go_support_reports WHERE id=$1`, id).Scan(&attempts, &notified); err != nil || attempts != 1 || notified {
		t.Fatalf("failure state: %d %v %v", attempts, notified, err)
	}
	if _, err = s.NotifyPending(ctx); err != nil {
		t.Fatal(err)
	}
	if m.calls != 1 {
		t.Fatal("backoff not respected")
	}
	_, err = db.Exec(ctx, `UPDATE go_support_reports SET next_notification_at=now() WHERE id=$1`, id)
	if err != nil {
		t.Fatal(err)
	}
	m.err = nil
	if n, err := s.NotifyPending(ctx); err != nil || n != 1 {
		t.Fatalf("retry: %d %v", n, err)
	}
	if m.vars["screenshotUrl"] != "https://uplumo.com/api/v1/support/reports/"+id+"/screenshot" {
		t.Fatal("missing private screenshot link")
	}
	if _, err = s.NotifyPending(ctx); err != nil || m.calls != 2 {
		t.Fatalf("resent completed notification: %d %v", m.calls, err)
	}
}
func TestHTTPRequiresSessionAndBoundsJSON(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	for _, test := range []struct {
		name, body string
		user       auth.User
		want       int
	}{
		{"unauthenticated", `{}`, auth.User{}, 401},
		{"no workspace", `{}`, auth.User{ID: "u"}, 403},
		{"unknown fields", `{"unexpected":true}`, auth.User{ID: "u", OrganizationID: "org"}, 400},
		{"trailing JSON", `{} {}`, auth.User{ID: "u", OrganizationID: "org"}, 400},
		{"oversized", strings.Repeat("x", (2<<20)+1), auth.User{ID: "u", OrganizationID: "org"}, 400},
	} {
		t.Run(test.name, func(t *testing.T) {
			mux := http.NewServeMux()
			Mount(mux, Deps{Service: &Service{}, Logger: logger, WithSession: func(h http.Handler) http.Handler { return h }})
			r := httptest.NewRequest("POST", "/api/v1/support/reports", strings.NewReader(test.body))
			if test.user.ID != "" {
				r = r.WithContext(auth.WithUser(r.Context(), test.user))
			}
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Code != test.want {
				t.Fatalf("status %d want %d: %s", w.Code, test.want, w.Body.String())
			}
		})
	}
}
