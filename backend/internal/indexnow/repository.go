package indexnow

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Repository stores keys in Postgres.
type Repository struct{ DB *pgxpool.Pool }

// Domain returns the active project's website.
func (r Repository) Domain(ctx context.Context, organizationID, projectID string) (*string, error) {
	var domain *string
	err := r.DB.QueryRow(ctx, `SELECT domain FROM projects WHERE organization_id=$1 AND id=$2 AND archived_at IS NULL`,
		organizationID, projectID).Scan(&domain)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrProjectNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load project %s domain: %w", projectID, err)
	}
	return domain, nil
}

// Key returns the project's key, or "".
func (r Repository) Key(ctx context.Context, projectID string) (string, error) {
	var key string
	err := r.DB.QueryRow(ctx, `SELECT key FROM go_indexnow_keys WHERE project_id=$1`, projectID).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("load indexnow key for %s: %w", projectID, err)
	}
	return key, nil
}

// CreateKey stores key unless the project has one, and returns the stored
// key either way.
func (r Repository) CreateKey(ctx context.Context, projectID, key string) (string, error) {
	if _, err := r.DB.Exec(ctx, `INSERT INTO go_indexnow_keys (project_id, key) VALUES ($1, $2) ON CONFLICT (project_id) DO NOTHING`,
		projectID, key); err != nil {
		return "", fmt.Errorf("create indexnow key for %s: %w", projectID, err)
	}
	// A separate statement: a key a concurrent request just committed is
	// invisible to the insert's own snapshot.
	stored, err := r.Key(ctx, projectID)
	if err != nil {
		return "", err
	}
	if stored == "" {
		return "", fmt.Errorf("create indexnow key for %s: no key after insert", projectID)
	}
	return stored, nil
}

// HourlyLimit is how many IndexNow requests (submissions, key-file checks
// and automatic submissions after a publish) one workspace may make an hour.
// Every workspace submits from Uplumo's servers, so a workspace that floods
// Bing could get those servers rate-limited for every customer.
const HourlyLimit = 60

// RedisLimiter allows HourlyLimit requests per workspace per clock hour,
// shared across server instances. It fails closed: a submission is free,
// so waiting for Redis costs the customer little.
type RedisLimiter struct{ Client *redis.Client }

// Allow records one request and reports whether it is within the limit.
func (l RedisLimiter) Allow(ctx context.Context, organizationID string) (bool, error) {
	if l.Client == nil {
		return false, errors.New("indexnow limiter is not configured")
	}
	key := "indexnow:rl:" + organizationID + ":" + time.Now().UTC().Format("2006010215")
	count, err := l.Client.Eval(ctx, `local count = redis.call('INCR', KEYS[1]); if count == 1 then redis.call('EXPIRE', KEYS[1], 7200) end; return count`, []string{key}).Int64()
	if err != nil {
		return false, fmt.Errorf("count indexnow request: %w", err)
	}
	return count <= HourlyLimit, nil
}
