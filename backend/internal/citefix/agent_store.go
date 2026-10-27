package citefix

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/toufiqqureshi/uplumo/backend/internal/audit"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/ids"
)

// agentTarget is the page as the latest scan saw it.
type agentTarget struct {
	URL    string
	Failed []string
}

// maxRecentResults bounds the saved results the page loads.
const maxRecentResults = 50

// agentPage finds url in the project's latest completed scan, scoped to the
// workspace. The agent only fetches pages the customer's own scan found, so
// it can never be pointed at another site.
func (s Store) agentPage(ctx context.Context, organizationID, projectID, pageURL string) (agentTarget, error) {
	var page Page
	var signalsJSON *string
	var scannedAt time.Time
	err := s.DB.QueryRow(ctx, `SELECT pg.url, COALESCE(pg.title, ''), COALESCE(pg.status_code, 0), pg.is_indexable,
			pg.word_count, pg.external_link_count, pg.citation_signals_json, latest.completed_at
		FROM go_audit_pages pg JOIN (
			SELECT a.id, a.completed_at FROM go_audits a JOIN projects p ON p.id = a.project_id AND p.organization_id = $1
			WHERE a.project_id = $2 AND a.status = 'completed' AND a.completed_at IS NOT NULL
			ORDER BY a.started_at DESC LIMIT 1) latest ON latest.id = pg.audit_id
		WHERE pg.url = $3`, organizationID, projectID, pageURL).
		Scan(&page.URL, &page.Title, &page.StatusCode, &page.Indexable, &page.WordCount, &page.ExternalLinks, &signalsJSON, &scannedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return agentTarget{}, ErrAgentPageNotFound
	}
	if err != nil {
		return agentTarget{}, fmt.Errorf("find agent page: %w", err)
	}
	if signalsJSON == nil || !isContentPage(page) {
		return agentTarget{}, ErrAgentPageNotFound
	}
	var signals audit.AIPageSignals
	if err := json.Unmarshal([]byte(*signalsJSON), &signals); err != nil {
		return agentTarget{}, fmt.Errorf("decode citation signals for %s: %w", page.URL, err)
	}
	page.Signals = &signals
	// The page alone, scored for its own failed checks.
	report := Build(Scan{CompletedAt: scannedAt}, []Page{page}, nil)
	target := agentTarget{URL: page.URL, Failed: []string{}}
	if len(report.Pages) == 1 {
		target.Failed = report.Pages[0].Failed
	}
	return target, nil
}

// runByKey replays a request already made with this key: its saved result,
// never a second charge.
func (s Store) runByKey(ctx context.Context, organizationID, key string) (AgentResult, bool, error) {
	var status string
	var resultJSON *string
	err := s.DB.QueryRow(ctx, `SELECT status, result_json FROM go_citefix_agent_runs
		WHERE organization_id = $1 AND idempotency_key = $2`, organizationID, key).Scan(&status, &resultJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return AgentResult{}, false, nil
	}
	if err != nil {
		return AgentResult{}, false, fmt.Errorf("find agent run by key: %w", err)
	}
	switch {
	case status == "running":
		return AgentResult{}, false, ErrAgentBusy
	case status == "failed" || resultJSON == nil:
		return AgentResult{}, false, ErrAgentFailed
	}
	result, err := unmarshalResult(*resultJSON)
	return result, err == nil, err
}

// cachedRun is the newest completed run on the same content and mode.
func (s Store) cachedRun(ctx context.Context, projectID, pageURL, mode, hash string) (AgentResult, bool, error) {
	var resultJSON string
	err := s.DB.QueryRow(ctx, `SELECT result_json FROM go_citefix_agent_runs
		WHERE project_id = $1 AND url = $2 AND mode = $3 AND content_hash = $4 AND status = 'completed' AND result_json IS NOT NULL
		ORDER BY created_at DESC LIMIT 1`, projectID, pageURL, mode, hash).Scan(&resultJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return AgentResult{}, false, nil
	}
	if err != nil {
		return AgentResult{}, false, fmt.Errorf("find cached agent run: %w", err)
	}
	result, err := unmarshalResult(resultJSON)
	return result, err == nil, err
}

// recordCachedRun saves a free replay under the request's key, so a retry of
// the same request answers the same way.
func (s Store) recordCachedRun(ctx context.Context, organizationID, projectID string, input AgentInput, hash string, result AgentResult) error {
	encoded, err := marshalResult(result)
	if err != nil {
		return err
	}
	id, err := ids.New()
	if err != nil {
		return fmt.Errorf("new agent run id: %w", err)
	}
	_, err = s.DB.Exec(ctx, `INSERT INTO go_citefix_agent_runs
		(id, organization_id, project_id, url, mode, idempotency_key, content_hash, status, credits, cached, result_json, finished_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'completed', 0, true, $8, now())
		ON CONFLICT (organization_id, idempotency_key) DO NOTHING`,
		id, organizationID, projectID, input.URL, input.Mode, input.IdempotencyKey, hash, encoded)
	if err != nil {
		return fmt.Errorf("record cached agent run: %w", err)
	}
	return nil
}

// claimRun starts a run. It refuses a second run on the same page and mode
// while one is running, and more than MaxAgentPages runs per project in
// agentRateWindow. A run cut off by a restart is freed after staleRunAfter.
func (s Store) claimRun(ctx context.Context, organizationID, projectID string, input AgentInput, hash string, now time.Time) (string, error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin agent claim: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	// One claim per project at a time, so the rate count cannot race.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('citefix-agent:' || $1))`, projectID); err != nil {
		return "", fmt.Errorf("lock agent claims: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE go_citefix_agent_runs SET status = 'failed', error_code = 'stale', finished_at = now()
		WHERE project_id = $1 AND status = 'running' AND created_at < $2`, projectID, now.Add(-staleRunAfter)); err != nil {
		return "", fmt.Errorf("free stale agent runs: %w", err)
	}
	var recent int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM go_citefix_agent_runs
		WHERE project_id = $1 AND created_at > $2 AND NOT cached`,
		projectID, now.Add(-agentRateWindow)).Scan(&recent); err != nil {
		return "", fmt.Errorf("count recent agent runs: %w", err)
	}
	if recent >= MaxAgentPages {
		return "", ErrAgentRateLimited
	}
	id, err := ids.New()
	if err != nil {
		return "", fmt.Errorf("new agent run id: %w", err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO go_citefix_agent_runs (id, organization_id, project_id, url, mode, idempotency_key, content_hash, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`, id, organizationID, projectID, input.URL, input.Mode, input.IdempotencyKey, hash, now)
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
		// Same key from a double click, or the same page already running.
		return "", ErrAgentBusy
	}
	if err != nil {
		return "", fmt.Errorf("claim agent run: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit agent claim: %w", err)
	}
	return id, nil
}

func (s Store) completeRun(ctx context.Context, runID string, credits int64, result AgentResult) error {
	encoded, err := marshalResult(result)
	if err != nil {
		return err
	}
	if _, err := s.DB.Exec(ctx, `UPDATE go_citefix_agent_runs SET status = 'completed', credits = $2, result_json = $3, finished_at = now()
		WHERE id = $1`, runID, credits, encoded); err != nil {
		return fmt.Errorf("complete agent run: %w", err)
	}
	return nil
}

func (s Store) failRun(ctx context.Context, runID, code string, credits int64) error {
	if _, err := s.DB.Exec(ctx, `UPDATE go_citefix_agent_runs SET status = 'failed', error_code = $2, credits = $3, finished_at = now()
		WHERE id = $1`, runID, code, credits); err != nil {
		return fmt.Errorf("fail agent run: %w", err)
	}
	return nil
}

// Results lists the newest completed result per page and mode, for the page
// to show without paying again.
func (s Store) Results(ctx context.Context, organizationID, projectID string) ([]AgentResult, error) {
	ctx, cancel := context.WithTimeout(ctx, storeTimeout)
	defer cancel()
	rows, err := s.DB.Query(ctx, `SELECT DISTINCT ON (r.url, r.mode) r.result_json FROM go_citefix_agent_runs r
		JOIN projects p ON p.id = r.project_id AND p.organization_id = $1
		WHERE r.project_id = $2 AND r.organization_id = $1 AND r.status = 'completed' AND r.result_json IS NOT NULL
		ORDER BY r.url, r.mode, r.created_at DESC LIMIT $3`, organizationID, projectID, maxRecentResults)
	if err != nil {
		return nil, fmt.Errorf("list agent results: %w", err)
	}
	defer rows.Close()
	results := []AgentResult{}
	for rows.Next() {
		var resultJSON string
		if err := rows.Scan(&resultJSON); err != nil {
			return nil, fmt.Errorf("scan agent result: %w", err)
		}
		result, err := unmarshalResult(resultJSON)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read agent results: %w", err)
	}
	return results, nil
}

func unmarshalResult(encoded string) (AgentResult, error) {
	var result AgentResult
	if err := json.Unmarshal([]byte(encoded), &result); err != nil {
		return AgentResult{}, fmt.Errorf("decode cite fix agent result: %w", err)
	}
	return result, nil
}
