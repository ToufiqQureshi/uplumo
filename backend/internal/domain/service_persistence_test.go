package domain

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toufiqqureshi/uplumo/backend/internal/database"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/ids"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/pgdb"
)

// openDomainPool opens the test Postgres and applies migrations + the legacy
// schema the projects/organization FKs need.
func openDomainPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("TEST_DATABASE_URL must be set in CI")
		}
		t.Skip("TEST_DATABASE_URL not set; skipping Postgres integration test")
	}
	ctx := context.Background()
	pool, err := pgdb.Open(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	schema, err := os.ReadFile("../database/testdata/legacy_schema.sql")
	if err != nil {
		t.Fatalf("read legacy schema: %v", err)
	}
	if _, err := pool.Exec(ctx, string(schema)); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}
	return pool
}

// newPersistenceFixture gives a DB-backed Service plus its organization and
// project IDs, cleaned up afterwards.
func newPersistenceFixture(t *testing.T) (*pgxpool.Pool, *fakeProvider, *Service, string, string) {
	t.Helper()
	pool := openDomainPool(t)
	org, err := ids.New()
	if err != nil {
		t.Fatal(err)
	}
	project, err := ids.New()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO organization (id, name, slug, created_at) VALUES ($1, 'Org', $1, now())`, org); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO projects (id, organization_id, name) VALUES ($1, $2, 'Proj')`, project, org); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.WithoutCancel(context.Background()), `DELETE FROM organization WHERE id = $1`, org)
	})
	provider := &fakeProvider{result: overviewResult(1234.6, 99.4)}
	svc := newTestService(t, provider, &memoryCache{})
	svc.DB = pool
	return pool, provider, svc, org, project
}

// aFreshService has a new Redis cache but the same DB/provider, so the next
// Overview must read Postgres.
func aFreshService(t *testing.T, provider *fakeProvider, db *pgxpool.Pool) *Service {
	t.Helper()
	svc := newTestService(t, provider, &memoryCache{})
	svc.DB = db
	return svc
}

func TestDomainOverviewSnapshotServedWithoutProviderCall(t *testing.T) {
	pool, provider, svc, org, project := newPersistenceFixture(t)
	ctx := t.Context()
	if _, err := svc.Overview(ctx, org, project, "example.com", ScopeDomain, 2840, "en"); err != nil {
		t.Fatal(err)
	}
	before := provider.calls.Load()
	fresh := aFreshService(t, provider, pool)
	got, err := fresh.Overview(ctx, org, project, "example.com", ScopeDomain, 2840, "en")
	if err != nil {
		t.Fatal(err)
	}
	if got.OrganicTraffic == nil || *got.OrganicTraffic != 1235 {
		t.Fatalf("overview = %+v, want the persisted result", got)
	}
	if provider.calls.Load() != before {
		t.Fatalf("provider calls moved from %d; a fresh snapshot must not re-buy", before)
	}
}

func TestDomainOverviewExpiredSnapshotTriggersOnePurchase(t *testing.T) {
	pool, provider, svc, org, project := newPersistenceFixture(t)
	ctx := t.Context()
	if _, err := svc.Overview(ctx, org, project, "example.com", ScopeDomain, 2840, "en"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE go_domain_overviews SET fetched_at = now() - interval '31 days' WHERE organization_id = $1`, org); err != nil {
		t.Fatal(err)
	}
	before := provider.calls.Load()
	fresh := aFreshService(t, provider, pool)
	if _, err := fresh.Overview(ctx, org, project, "example.com", ScopeDomain, 2840, "en"); err != nil {
		t.Fatal(err)
	}
	if got := provider.calls.Load() - before; got != 1 {
		t.Fatalf("expired snapshot bought %d tasks, want exactly 1", got)
	}
}

func TestDomainOverviewSnapshotIsTenantScoped(t *testing.T) {
	pool, provider, svc, org, project := newPersistenceFixture(t)
	ctx := t.Context()
	if _, err := svc.Overview(ctx, org, project, "example.com", ScopeDomain, 2840, "en"); err != nil {
		t.Fatal(err)
	}
	otherOrg, err := ids.New()
	if err != nil {
		t.Fatal(err)
	}
	otherProject, err := ids.New()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO organization (id, name, slug, created_at) VALUES ($1, 'Other', $1, now())`, otherOrg); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO projects (id, organization_id, name) VALUES ($1, $2, 'Other')`, otherProject, otherOrg); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.WithoutCancel(context.Background()), `DELETE FROM organization WHERE id = $1`, otherOrg)
	})
	before := provider.calls.Load()
	fresh := aFreshService(t, provider, pool)
	if _, err := fresh.Overview(ctx, otherOrg, otherProject, "example.com", ScopeDomain, 2840, "en"); err != nil {
		t.Fatal(err)
	}
	if got := provider.calls.Load() - before; got != 1 {
		t.Fatalf("another organization read a stored snapshot (calls=%d); it must always buy its own", got)
	}
}

func TestDomainOverviewReturnsThePaidResultWhenTheDBFails(t *testing.T) {
	pool, provider, svc, org, project := newPersistenceFixture(t)
	pool.Close() // every snapshot read/write now errors
	got, err := svc.Overview(t.Context(), org, project, "example.com", ScopeDomain, 2840, "en")
	if err != nil {
		t.Fatalf("Overview failed on a DB error: %v", err)
	}
	if got.OrganicTraffic == nil {
		t.Fatalf("paid result not returned when the DB is down: %+v", got)
	}
	if provider.calls.Load() == 0 {
		t.Fatal("the provider was not called when the DB was down")
	}
}
