package indexnow

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/toufiqqureshi/uplumo/backend/internal/database"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/pgdb"
)

func TestRepositoryKeepsOneKeyPerProject(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("TEST_DATABASE_URL must be set in CI")
		}
		t.Skip("TEST_DATABASE_URL not set; skipping PostgreSQL integration test")
	}
	pool, err := pgdb.Open(ctx, url)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	suffix := strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	organizationID, projectID, archivedID := "indexnow-org-"+suffix, "indexnow-project-"+suffix, "indexnow-archived-"+suffix
	if _, err := pool.Exec(ctx, `INSERT INTO organization (id, name, slug, created_at) VALUES ($1, 'IndexNow test', $1, now())`, organizationID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		if _, err := pool.Exec(cleanupCtx, `DELETE FROM organization WHERE id=$1`, organizationID); err != nil {
			t.Errorf("clean up indexnow fixture: %v", err)
		}
	})
	if _, err := pool.Exec(ctx, `INSERT INTO projects (id, organization_id, name, domain) VALUES ($1, $2, $1, 'example.com'), ($3, $2, $3, 'old.example')`, projectID, organizationID, archivedID); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE projects SET archived_at=now()::text WHERE id=$1`, archivedID); err != nil {
		t.Fatal(err)
	}
	repo := Repository{DB: pool}

	if domain, err := repo.Domain(ctx, organizationID, projectID); err != nil || domain == nil || *domain != "example.com" {
		t.Fatalf("domain = %v, %v", domain, err)
	}
	if _, err := repo.Domain(ctx, "other-org", projectID); !errors.Is(err, ErrProjectNotFound) {
		t.Errorf("another organization: err = %v", err)
	}
	if _, err := repo.Domain(ctx, organizationID, archivedID); !errors.Is(err, ErrProjectNotFound) {
		t.Errorf("archived project: err = %v", err)
	}
	if key, err := repo.Key(ctx, projectID); err != nil || key != "" {
		t.Fatalf("key before set up = %q, %v", key, err)
	}

	// Concurrent first uses must all end up with the one stored key.
	candidates := []string{strings.Repeat("a", 32), strings.Repeat("b", 32), strings.Repeat("c", 32), strings.Repeat("d", 32)}
	got := make([]string, len(candidates))
	var wg sync.WaitGroup
	for i, candidate := range candidates {
		wg.Go(func() {
			key, err := repo.CreateKey(ctx, projectID, candidate)
			if err != nil {
				t.Error(err)
			}
			got[i] = key
		})
	}
	wg.Wait()
	stored, err := repo.Key(ctx, projectID)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range got {
		if key != stored {
			t.Fatalf("CreateKey answered %v, stored %q", got, stored)
		}
	}
	if _, err := repo.CreateKey(ctx, "indexnow-other-"+suffix, "not-hex"); err == nil {
		t.Error("stored a key that is not 32 hex characters")
	}
}

func TestRedisLimiterIsPerWorkspace(t *testing.T) {
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("TEST_REDIS_URL must be set in CI")
		}
		t.Skip("TEST_REDIS_URL not set; skipping Redis integration test")
	}
	options, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatal(err)
	}
	client := redis.NewClient(options)
	t.Cleanup(func() { _ = client.Close() })
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	limiter := RedisLimiter{Client: client}
	suffix := time.Now().UTC().Format("150405.000000000")
	heavy, quiet := "indexnow-heavy-"+suffix, "indexnow-quiet-"+suffix

	// The count resets each clock hour; wait out an hour that is about to
	// roll over so the test cannot straddle two buckets.
	if time.Now().Minute() == 59 && time.Now().Second() > 50 {
		time.Sleep(15 * time.Second)
	}
	for i := range HourlyLimit {
		if ok, err := limiter.Allow(ctx, heavy); err != nil || !ok {
			t.Fatalf("request %d = %v, %v", i+1, ok, err)
		}
	}
	if ok, err := limiter.Allow(ctx, heavy); err != nil || ok {
		t.Fatalf("request %d = %v, %v, want refused", HourlyLimit+1, ok, err)
	}
	// One heavy workspace must not use up anyone else's submissions.
	if ok, err := limiter.Allow(ctx, quiet); err != nil || !ok {
		t.Fatalf("another workspace = %v, %v", ok, err)
	}
	if ok, err := (RedisLimiter{}).Allow(ctx, quiet); err == nil || ok {
		t.Fatal("a limiter without Redis allowed a request")
	}
}
