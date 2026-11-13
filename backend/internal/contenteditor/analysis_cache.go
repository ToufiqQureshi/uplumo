package contenteditor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

// PostgresAnalysisCache stores private reports under their authorized workspace.
type PostgresAnalysisCache struct{ DB *pgxpool.Pool }

func (c PostgresAnalysisCache) GetOrBuild(ctx context.Context, org, key string, build func() (WriterOutput, error)) (out WriterOutput, retErr error) {
	if c.DB == nil || org == "" || len(key) != 64 {
		return out, ErrWriterUnavailable
	}
	conn, err := c.DB.Acquire(ctx)
	if err != nil {
		return out, fmt.Errorf("acquire content review cache: %w", err)
	}
	lockKey := "content-review:" + org + ":" + key
	locked := false
	defer func() {
		if !locked {
			conn.Release()
			return
		}
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		var released bool
		if err := conn.QueryRow(cleanup, `SELECT pg_advisory_unlock(hashtextextended($1, 0))`, lockKey).Scan(&released); err != nil || !released {
			raw := conn.Hijack()
			_ = raw.Close(cleanup)
			return
		}
		conn.Release()
	}()
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock(hashtextextended($1, 0))`, lockKey); err != nil {
		return out, err
	}
	locked = true
	err = conn.QueryRow(ctx, `SELECT report FROM go_content_reviews WHERE organization_id=$1 AND input_hash=$2 AND expires_at>now()`, org, key).Scan(&out.Content)
	if err == nil {
		return out, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	out, err = build()
	if err != nil {
		return out, err
	}
	// Persist a delivered, billed result even if the browser disconnected.
	saveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	_, err = conn.Exec(saveCtx, `INSERT INTO go_content_reviews (organization_id, input_hash, report, expires_at)
		VALUES ($1,$2,$3,now()+interval '7 days') ON CONFLICT (organization_id,input_hash)
		DO UPDATE SET report=EXCLUDED.report, expires_at=EXCLUDED.expires_at`, org, key, out.Content)
	if err != nil {
		return WriterOutput{}, fmt.Errorf("save content review: %w", err)
	}
	_, _ = conn.Exec(saveCtx, `DELETE FROM go_content_reviews WHERE organization_id=$1 AND expires_at<=now()`, org)
	return out, nil
}
