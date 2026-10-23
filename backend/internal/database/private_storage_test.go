package database

import (
	"context"
	"crypto/rand"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestServerTablesDenyDirectClientAccess(t *testing.T) {
	ctx := t.Context()
	pool := openFresh(ctx, t)
	// Reproduce the permissive grants on an existing Supabase table.
	if _, err := pool.Exec(ctx, `CREATE TABLE private_fixture(id int); INSERT INTO private_fixture VALUES(1); GRANT ALL ON private_fixture TO PUBLIC`); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	var unprotected int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM pg_class WHERE relnamespace='public'::regnamespace AND relkind='r' AND NOT relrowsecurity`).Scan(&unprotected); err != nil || unprotected != 0 {
		t.Fatalf("tables without RLS = %d, err=%v", unprotected, err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	role := pgx.Identifier{"launch_client_" + strings.ToLower(rand.Text())}.Sanitize()
	for _, sql := range []string{
		"CREATE ROLE " + role + " NOLOGIN",
		"GRANT USAGE ON SCHEMA public TO " + role,
		"SET LOCAL ROLE " + role,
	} {
		if _, err := tx.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	var allowed bool
	if err := tx.QueryRow(ctx, `SELECT has_table_privilege(current_user,'private_fixture','TRUNCATE')`).Scan(&allowed); err != nil || allowed {
		t.Fatalf("client retains destructive public grant: %v, err=%v", allowed, err)
	}
	if _, err := tx.Exec(ctx, "RESET ROLE"); err != nil {
		t.Fatal(err)
	}
	// Even accidentally restored SELECT grants must not expose auth/billing.
	for _, table := range []string{"private_fixture", `"user"`, "session", "account", "go_billing_credits", "go_billing_credit_ledger"} {
		if _, err := tx.Exec(ctx, "GRANT SELECT ON "+table+" TO "+role+"; SET LOCAL ROLE "+role); err != nil {
			t.Fatal(err)
		}
		var count int
		if err := tx.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("client can read %s: count=%d, err=%v", table, count, err)
		}
		if _, err := tx.Exec(ctx, "RESET ROLE"); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM private_fixture").Scan(&count); err != nil || count != 1 {
		t.Fatalf("trusted server lost access: count=%d, err=%v", count, err)
	}
}
