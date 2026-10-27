package aisearch

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/pgdb"
)

const trackingQueue = "ai_visibility"

// Up to 60 sequential model requests (60 seconds each), the standard
// scraper queue (45 minutes), and 15 minutes for submission/collection.
const trackingRunTimeout = 2 * time.Hour

// Save replaces approved configuration without changing historical answers.
func (s *Tracking) Save(ctx context.Context, org, project string, c TrackingConfig) error {
	if err := c.validate(); err != nil {
		return err
	}
	if err := s.allowed(ctx, org); err != nil {
		return err
	}
	if c.Enabled && c.MaxCreditsPerRun < c.estimate().Credits {
		return badRequest("Approve the complete per-check credit ceiling before scheduling.")
	}
	if c.Enabled && s.Search == nil {
		return badRequest("Configure the data provider before enabling checks.")
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return err
	}
	tag, err := s.DB.Exec(ctx, `INSERT INTO go_ai_trackers(project_id,organization_id,configuration,next_check_at)
 SELECT id,organization_id,$3,CASE WHEN $4 THEN now() ELSE NULL END FROM projects WHERE id=$1 AND organization_id=$2 AND archived_at IS NULL
 ON CONFLICT(project_id) DO UPDATE SET configuration=EXCLUDED.configuration,
 next_check_at=CASE WHEN NOT $4 THEN NULL WHEN go_ai_trackers.next_check_at IS NULL OR go_ai_trackers.configuration->>'interval' <> EXCLUDED.configuration->>'interval' THEN now() ELSE go_ai_trackers.next_check_at END,updated_at=now()
 WHERE go_ai_trackers.organization_id=$2`, project, org, raw, c.Enabled)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return errTrackingMissing
	}
	if !c.Enabled {
		_, err = s.DB.Exec(ctx, `WITH stopped AS (UPDATE go_ai_tracking_runs SET status='failed',error_code='schedule_paused',finished_at=now() WHERE project_id=$1 AND organization_id=$2 AND trigger='scheduled' AND status='queued' RETURNING id) UPDATE go_ai_tracking_answers SET status='failed',error_code='schedule_paused' WHERE run_id IN (SELECT id FROM stopped)`, project, org)
	}
	return err
}

// State reads the owning workspace's configuration and recent run summaries.
func (s *Tracking) State(ctx context.Context, org, project string) (TrackingState, error) {
	out := TrackingState{Runs: []TrackingRun{}, ProviderConfigured: s.Search != nil}
	var raw []byte
	err := s.DB.QueryRow(ctx, `SELECT configuration,next_check_at FROM go_ai_trackers WHERE organization_id=$1 AND project_id=$2`, org, project).Scan(&raw, &out.NextCheckAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	var c TrackingConfig
	if err = json.Unmarshal(raw, &c); err != nil {
		return out, err
	}
	out.Configuration = &c
	rows, err := s.DB.Query(ctx, `SELECT r.id,r.configuration,r.scope_key,r.trigger,r.status,r.error_code,r.created_at,r.finished_at,
 count(a.id) FILTER(WHERE a.status='completed'),count(a.id) FILTER(WHERE a.status='failed'),
 count(a.id) FILTER(WHERE a.status='completed' AND a.answer<>''),
 count(a.id) FILTER(WHERE a.status='completed' AND a.answer<>'' AND a.brands->0->>'mentioned'='true'),
 count(a.id) FILTER(WHERE a.status='completed' AND a.answer<>'' AND a.brands->0->>'cited'='true')
 FROM go_ai_tracking_runs r LEFT JOIN go_ai_tracking_answers a ON a.run_id=r.id
 WHERE r.organization_id=$1 AND r.project_id=$2 AND r.id IN (SELECT id FROM go_ai_tracking_runs WHERE project_id=$2 AND organization_id=$1 ORDER BY created_at DESC LIMIT 30) GROUP BY r.id ORDER BY r.created_at DESC LIMIT 30`, org, project)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var run TrackingRun
		var cfg []byte
		if err = rows.Scan(&run.ID, &cfg, &run.ScopeKey, &run.Trigger, &run.Status, &run.ErrorCode, &run.CreatedAt, &run.FinishedAt, &run.Completed, &run.Failed, &run.Answered, &run.Mentioned, &run.Cited); err != nil {
			return out, err
		}
		if err = json.Unmarshal(cfg, &run.Configuration); err != nil {
			return out, err
		}
		out.Runs = append(out.Runs, run)
	}
	return out, rows.Err()
}

// Answers reads evidence only when the run, project and organization match.
func (s *Tracking) Answers(ctx context.Context, org, project, runID string) ([]TrackingAnswer, error) {
	rows, err := s.DB.Query(ctx, `SELECT a.id,a.prompt,a.topic,a.engine,a.status,a.task_id,a.answer,a.citations,a.brands,a.error_code,a.collected_at,a.sentiment,a.sentiment_quote,a.sentiment_reason
 FROM go_ai_tracking_answers a JOIN go_ai_tracking_runs r ON r.id=a.run_id WHERE r.id=$1 AND r.organization_id=$2 AND r.project_id=$3 ORDER BY a.topic,a.prompt,a.engine`, runID, org, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TrackingAnswer{}
	for rows.Next() {
		var a TrackingAnswer
		var citations, brands []byte
		if err = rows.Scan(&a.ID, &a.Prompt, &a.Topic, &a.Engine, &a.Status, &a.TaskID, &a.Answer, &citations, &brands, &a.ErrorCode, &a.CollectedAt, &a.Sentiment, &a.SentimentQuote, &a.SentimentReason); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(citations, &a.Citations); err != nil {
			return nil, err
		}
		a.Brands, a.Recommended, err = splitStoredBrands(brands)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// Start atomically snapshots configuration, creates answers and writes the job outbox.
// The tracker row serializes edits, manual starts and scheduler claims.
func (s *Tracking) Start(ctx context.Context, org, project, key string, scheduled bool, maxCredits int64) (string, error) {
	if s.Search == nil {
		return "", badRequest("The data provider is unavailable.")
	}
	if err := s.allowed(ctx, org); err != nil {
		return "", err
	}
	id := "air_" + rand.Text()
	err := pgdb.InTx(ctx, s.DB, func(tx pgx.Tx) error {
		var raw []byte
		var next *time.Time
		err := tx.QueryRow(ctx, `SELECT t.configuration,t.next_check_at FROM go_ai_trackers t JOIN projects p ON p.id=t.project_id WHERE t.organization_id=$1 AND t.project_id=$2 AND p.archived_at IS NULL FOR UPDATE OF t`, org, project).Scan(&raw, &next)
		if errors.Is(err, pgx.ErrNoRows) {
			return errTrackingMissing
		}
		if err != nil {
			return err
		}
		if scheduled {
			if next == nil || next.After(time.Now()) {
				return nil
			}
			key = "scheduled:" + next.UTC().Format(time.RFC3339Nano)
		}
		var existing string
		err = tx.QueryRow(ctx, `SELECT id FROM go_ai_tracking_runs WHERE organization_id=$1 AND project_id=$2 AND request_key=$3`, org, project, key).Scan(&existing)
		if err == nil {
			id = existing
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var c TrackingConfig
		if err = json.Unmarshal(raw, &c); err != nil {
			return err
		}
		if err = c.validate(); err != nil {
			return err
		}
		if scheduled && !c.Enabled {
			return nil
		}
		var busy bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM go_ai_tracking_runs WHERE project_id=$1 AND status IN ('queued','running'))`, project).Scan(&busy); err != nil {
			return err
		}
		if busy {
			return errTrackingBusy
		}
		if c.estimate().Answers == 0 {
			return badRequest("Enable at least one question.")
		}
		if scheduled {
			maxCredits = c.MaxCreditsPerRun
		}
		if maxCredits < c.estimate().Credits {
			return badRequest("The check changed. Review and approve its current credit ceiling.")
		}
		trigger := "manual"
		if scheduled {
			trigger = "scheduled"
		}
		_, err = tx.Exec(ctx, `INSERT INTO go_ai_tracking_runs(id,project_id,organization_id,configuration,scope_key,trigger,request_key) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, project, org, raw, trackingScope(c), trigger, key)
		if err != nil {
			return err
		}
		for _, p := range c.Prompts {
			if p.Paused {
				continue
			}
			for _, engine := range c.Engines {
				_, err = tx.Exec(ctx, `INSERT INTO go_ai_tracking_answers(id,run_id,prompt,topic,engine) VALUES($1,$2,$3,$4,$5)`, "aia_"+rand.Text(), id, p.Text, p.Topic, engine)
				if err != nil {
					return err
				}
			}
		}
		payload, err := json.Marshal(map[string]string{"runId": id})
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO go_jobs(queue,idempotency_key,payload,max_attempts,timeout_seconds) VALUES($1,$2,$3,3,$4)`, trackingQueue, id, payload, int(trackingRunTimeout/time.Second))
		if err != nil {
			return err
		}
		if scheduled {
			n := nextTrackingCheck(*next, c.Interval)
			for !n.After(time.Now()) {
				n = nextTrackingCheck(n, c.Interval)
			}
			_, err = tx.Exec(ctx, `UPDATE go_ai_trackers SET next_check_at=$2 WHERE project_id=$1`, project, n)
		}
		return err
	})
	return id, err
}

// Tick claims due trackers and removes expired historical evidence.
func (s *Tracking) Tick(ctx context.Context) error {
	// A failed final job must release the project's single active-run slot.
	_, err := s.DB.Exec(ctx, `WITH stopped AS (
 UPDATE go_ai_tracking_runs r SET status='failed',error_code='job_exhausted',finished_at=now()
 FROM go_jobs j WHERE j.queue=$1 AND j.idempotency_key=r.id AND j.state='failed' AND r.status IN ('queued','running') RETURNING r.id
 ) UPDATE go_ai_tracking_answers SET status='failed',error_code='job_exhausted' WHERE run_id IN (SELECT id FROM stopped) AND status NOT IN ('completed','failed')`, trackingQueue)
	if err != nil {
		return err
	}
	rows, err := s.DB.Query(ctx, `SELECT t.organization_id,t.project_id FROM go_ai_trackers t JOIN projects p ON p.id=t.project_id WHERE t.next_check_at<=now() AND p.archived_at IS NULL ORDER BY t.next_check_at LIMIT 20`)
	if err != nil {
		return err
	}
	type due struct{ org, project string }
	pending := []due{}
	for rows.Next() {
		var d due
		if err = rows.Scan(&d.org, &d.project); err != nil {
			rows.Close()
			return err
		}
		pending = append(pending, d)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, d := range pending {
		if _, err = s.Start(ctx, d.org, d.project, "", true, 0); err != nil && !errors.Is(err, errTrackingBusy) && !errors.Is(err, errTrackingPlan) {
			return fmt.Errorf("start scheduled AI check: %w", err)
		}
	}
	// Retention bounds stored customer research; history reads never collect answers.
	_, err = s.DB.Exec(ctx, `DELETE FROM go_ai_tracking_runs WHERE id IN (SELECT id FROM go_ai_tracking_runs WHERE created_at<now()-interval '13 months' AND status NOT IN ('queued','running') LIMIT 100)`)
	return err
}

// splitStoredBrands separates the tracked brands' observations from the brands
// the AI recommended, which share the brands column.
func splitStoredBrands(raw []byte) ([]BrandObservation, []RecommendedBrand, error) {
	var stored []storedBrand
	if err := json.Unmarshal(raw, &stored); err != nil {
		return nil, nil, err
	}
	tracked, recommended := []BrandObservation{}, []RecommendedBrand{}
	for _, b := range stored {
		if b.Recommended {
			recommended = append(recommended, RecommendedBrand{Name: b.Name, Category: b.Category, Position: b.Position})
			continue
		}
		tracked = append(tracked, b.BrandObservation)
	}
	return tracked, recommended, nil
}
