package contenteditor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/singleflight"
)

// AnalysisCache coalesces identical reviews before any paid model call.
type AnalysisCache interface {
	GetOrBuild(context.Context, string, string, func() (WriterOutput, error)) (WriterOutput, error)
}

func analysisKey(input WriterInput) string {
	input.IdempotencyKey = "" // Transport retries are not new content.
	data, _ := json.Marshal([]any{"review-v1", writerModel, writerSystemPrompt(WriterAnalyze, input.Mode), input})
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// PostgresAnalysisCache stores private reports under their authorized
// workspace for seven days. Identical requests in this process share one
// build without holding a database connection while they wait: the build
// itself needs pooled connections for its credit hold and settlement.
type PostgresAnalysisCache struct {
	DB      *pgxpool.Pool
	Logger  *slog.Logger
	flights singleflight.Group
}

// GetOrBuild returns the cached report for this workspace and input, or runs
// build once and caches its result. A report that was built (and billed) is
// returned even when caching it fails.
func (c *PostgresAnalysisCache) GetOrBuild(ctx context.Context, org, key string, build func() (WriterOutput, error)) (WriterOutput, error) {
	if c.DB == nil || org == "" || len(key) != 64 {
		return WriterOutput{}, ErrWriterUnavailable
	}
	if out, ok, err := c.cached(ctx, org, key); err != nil || ok {
		return out, err
	}
	result := c.flights.DoChan(org+":"+key, func() (any, error) {
		// A waiter may have finished this review while we queued.
		if out, ok, err := c.cached(ctx, org, key); err != nil || ok {
			return out, err
		}
		out, err := build()
		if err != nil {
			return out, err
		}
		c.save(ctx, org, key, out)
		return out, nil
	})
	select {
	case <-ctx.Done():
		return WriterOutput{}, ctx.Err()
	case r := <-result:
		if r.Err != nil {
			return WriterOutput{}, r.Err
		}
		return r.Val.(WriterOutput), nil
	}
}

func (c *PostgresAnalysisCache) cached(ctx context.Context, org, key string) (WriterOutput, bool, error) {
	var out WriterOutput
	err := c.DB.QueryRow(ctx, `SELECT report FROM go_content_reviews WHERE organization_id=$1 AND input_hash=$2 AND expires_at>now()`, org, key).Scan(&out.Content)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, false, nil
	}
	if err != nil {
		return out, false, fmt.Errorf("read content review cache: %w", err)
	}
	return out, true, nil
}

// save caches a delivered, billed report even if the browser disconnected.
func (c *PostgresAnalysisCache) save(ctx context.Context, org, key string, out WriterOutput) {
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	_, err := c.DB.Exec(saveCtx, `INSERT INTO go_content_reviews (organization_id, input_hash, report, expires_at)
		VALUES ($1,$2,$3,now()+interval '7 days') ON CONFLICT (organization_id,input_hash)
		DO UPDATE SET report=EXCLUDED.report, expires_at=EXCLUDED.expires_at`, org, key, out.Content)
	if err != nil {
		if c.Logger != nil {
			c.Logger.ErrorContext(saveCtx, "cache content review", "err", err)
		}
		return
	}
	_, _ = c.DB.Exec(saveCtx, `DELETE FROM go_content_reviews WHERE organization_id=$1 AND expires_at<=now()`, org)
}
