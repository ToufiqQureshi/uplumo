package backlinks

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toufiqqureshi/uplumo/backend/internal/database"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/ids"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/pgdb"
)

// openBacklinksPool opens the test Postgres and applies migrations + the legacy
// schema the organization FK needs.
func openBacklinksPool(t *testing.T) *pgxpool.Pool {
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

type callCounter struct {
	mu        sync.Mutex
	summaries int
	total     int
}

func (c *callCounter) snapshot() (summaries, total int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.summaries, c.total
}

func persistenceOverviewHandler(counter *callCounter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		counter.mu.Lock()
		counter.total++
		if strings.HasSuffix(r.URL.Path, "/summary/live") {
			counter.summaries++
		}
		counter.mu.Unlock()
		if strings.HasSuffix(r.URL.Path, "/summary/live") {
			providerOK(w, r.URL.Path, map[string]any{"rank": 52, "backlinks": 90, "referring_pages": 20, "referring_domains": 10})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/history/live") {
			providerOK(w, r.URL.Path, map[string]any{"items": []any{map[string]any{"date": "2026-10-08T00:00:00Z", "backlinks": 90}}})
			return
		}
		http.NotFound(w, r)
	}
}

func newBacklinksPersistenceService(t *testing.T, pool *pgxpool.Pool, counter *callCounter) *Service {
	t.Helper()
	recorder := &usageRecorder{}
	svc := newService(testClient(t, persistenceOverviewHandler(counter), recorder), &memoryCache{values: map[string][]byte{}},
		slog.New(slog.NewTextHandler(io.Discard, nil)), func() time.Time { return time.Date(2026, 10, 9, 11, 12, 13, 0, time.UTC) })
	svc.DB = pool
	return svc
}

func newOrg(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	org, err := ids.New()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), `INSERT INTO organization (id, name, slug, created_at) VALUES ($1, 'Org', $1, now())`, org); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.WithoutCancel(context.Background()), `DELETE FROM organization WHERE id = $1`, org)
	})
	return org
}

func TestBacklinksOverviewSnapshotServedWithoutProviderCall(t *testing.T) {
	pool := openBacklinksPool(t)
	org := newOrg(t, pool)
	counter := &callCounter{}
	in := lookupInput{Target: "example.com", Scope: string(ScopeDomain)}
	if _, err := newBacklinksPersistenceService(t, pool, counter).Overview(t.Context(), org, in); err != nil {
		t.Fatal(err)
	}
	before, _ := counter.snapshot()
	// A fresh service has a new Redis cache, so this call must read Postgres.
	got, err := newBacklinksPersistenceService(t, pool, counter).Overview(t.Context(), org, in)
	if err != nil {
		t.Fatal(err)
	}
	if got.Summary.Backlinks == nil || *got.Summary.Backlinks != 90 {
		t.Fatalf("overview = %#v, want the persisted summary", got)
	}
	if after, _ := counter.snapshot(); after != before {
		t.Fatalf("provider summaries moved from %d; a fresh snapshot must not re-buy", before)
	}
}

func TestBacklinksOverviewExpiredSnapshotTriggersOnePurchase(t *testing.T) {
	pool := openBacklinksPool(t)
	org := newOrg(t, pool)
	counter := &callCounter{}
	in := lookupInput{Target: "example.com", Scope: string(ScopeDomain)}
	if _, err := newBacklinksPersistenceService(t, pool, counter).Overview(t.Context(), org, in); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(t.Context(), `UPDATE go_backlinks_overviews SET fetched_at = now() - interval '31 days' WHERE organization_id = $1`, org); err != nil {
		t.Fatal(err)
	}
	before, _ := counter.snapshot()
	if _, err := newBacklinksPersistenceService(t, pool, counter).Overview(t.Context(), org, in); err != nil {
		t.Fatal(err)
	}
	if after, _ := counter.snapshot(); after-before != 1 {
		t.Fatalf("expired snapshot bought %d summary tasks, want exactly 1", after-before)
	}
}

func TestBacklinksOverviewSnapshotIsTenantScoped(t *testing.T) {
	pool := openBacklinksPool(t)
	firstOrg, secondOrg := newOrg(t, pool), newOrg(t, pool)
	counter := &callCounter{}
	in := lookupInput{Target: "example.com", Scope: string(ScopeDomain)}
	if _, err := newBacklinksPersistenceService(t, pool, counter).Overview(t.Context(), firstOrg, in); err != nil {
		t.Fatal(err)
	}
	before, _ := counter.snapshot()
	if _, err := newBacklinksPersistenceService(t, pool, counter).Overview(t.Context(), secondOrg, in); err != nil {
		t.Fatal(err)
	}
	if after, _ := counter.snapshot(); after-before != 1 {
		t.Fatalf("another organization read a stored snapshot (summaries=%d); it must buy its own", after-before)
	}
}

func TestBacklinksOverviewReturnsThePaidResultWhenTheDBFails(t *testing.T) {
	pool := openBacklinksPool(t)
	org := newOrg(t, pool)
	counter := &callCounter{}
	svc := newBacklinksPersistenceService(t, pool, counter)
	pool.Close() // every snapshot read/write now errors
	got, err := svc.Overview(t.Context(), org, lookupInput{Target: "example.com", Scope: string(ScopeDomain)})
	if err != nil {
		t.Fatalf("Overview failed on a DB error: %v", err)
	}
	if got.Summary.Backlinks == nil {
		t.Fatalf("paid result not returned when the DB is down: %#v", got)
	}
}
