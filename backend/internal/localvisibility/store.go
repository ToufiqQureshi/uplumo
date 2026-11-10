package localvisibility

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Run kinds.
const (
	KindFind    = "find"
	KindCheck   = "check"
	KindGrid    = "grid"
	KindReviews = "reviews"
	KindWatch   = "watch"
)

// Run states.
const (
	StatusRunning = "running"
	StatusDone    = "done"
	StatusFailed  = "failed"
)

// Task states.
const (
	taskPosted = "posted"
	taskDone   = "done"
	taskFailed = "failed"
)

// Run is one paid check and, once done, its result.
type Run struct {
	ID             string          `json:"id"`
	ProjectID      string          `json:"-"`
	OrganizationID string          `json:"-"`
	Kind           string          `json:"kind"`
	InputHash      string          `json:"-"`
	Input          json.RawMessage `json:"-"`
	Status         string          `json:"status"`
	Result         json.RawMessage `json:"result"`
	Error          string          `json:"error"`
	CreatedAt      time.Time       `json:"createdAt"`
	FinishedAt     *time.Time      `json:"finishedAt"`
	// Cached is true when the answer is an earlier run of the same input.
	Cached bool `json:"cached"`
}

type storedTask struct {
	Key      string
	Endpoint string
	TaskID   string
	State    string
	Result   json.RawMessage
}

// Store is the feature's Postgres storage.
type Store struct{ DB *pgxpool.Pool }

const runColumns = `id::text, project_id, organization_id, kind, input_hash, input, status, result, error, created_at, finished_at`

func scanRun(row pgx.Row) (*Run, error) {
	var r Run
	err := row.Scan(&r.ID, &r.ProjectID, &r.OrganizationID, &r.Kind, &r.InputHash, &r.Input, &r.Status, &r.Result, &r.Error, &r.CreatedAt, &r.FinishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan local run: %w", err)
	}
	return &r, nil
}

// Business returns the project's business, or nil when none is set.
func (s Store) Business(ctx context.Context, projectID string) (*Business, error) {
	var b Business
	err := s.DB.QueryRow(ctx, `SELECT cid, place_id, title, address, category, latitude, longitude, search_term, language_code, updated_at
		FROM go_local_businesses WHERE project_id = $1`, projectID).Scan(
		&b.CID, &b.PlaceID, &b.Title, &b.Address, &b.Category, &b.Latitude, &b.Longitude, &b.SearchTerm, &b.LanguageCode, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load local business: %w", err)
	}
	return &b, nil
}

// SaveBusiness sets the project's business. A different business clears
// the watch, which follows the old business's competitors.
func (s Store) SaveBusiness(ctx context.Context, organizationID, projectID string, b Business) error {
	return pgx.BeginFunc(ctx, s.DB, func(tx pgx.Tx) error {
		var previous string
		err := tx.QueryRow(ctx, `SELECT cid FROM go_local_businesses WHERE project_id = $1 FOR UPDATE`, projectID).Scan(&previous)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("lock local business: %w", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO go_local_businesses
			(project_id, organization_id, cid, place_id, title, address, category, latitude, longitude, search_term, language_code, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,now())
			ON CONFLICT (project_id) DO UPDATE SET organization_id=EXCLUDED.organization_id, cid=EXCLUDED.cid, place_id=EXCLUDED.place_id,
			title=EXCLUDED.title, address=EXCLUDED.address, category=EXCLUDED.category, latitude=EXCLUDED.latitude,
			longitude=EXCLUDED.longitude, search_term=EXCLUDED.search_term, language_code=EXCLUDED.language_code, updated_at=now()`,
			projectID, organizationID, b.CID, b.PlaceID, b.Title, b.Address, b.Category, b.Latitude, b.Longitude, b.SearchTerm, b.LanguageCode); err != nil {
			return fmt.Errorf("save local business: %w", err)
		}
		if previous != "" && previous != b.CID {
			if _, err := tx.Exec(ctx, `UPDATE go_local_watch SET enabled=false, updated_at=now() WHERE project_id=$1`, projectID); err != nil {
				return fmt.Errorf("stop local watch: %w", err)
			}
		}
		return nil
	})
}

// LatestRuns returns the newest run of each kind for the project.
func (s Store) LatestRuns(ctx context.Context, projectID string) (map[string]*Run, error) {
	rows, err := s.DB.Query(ctx, `SELECT DISTINCT ON (kind) `+runColumns+`
		FROM go_local_runs WHERE project_id = $1 ORDER BY kind, created_at DESC`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list local runs: %w", err)
	}
	defer rows.Close()
	out := map[string]*Run{}
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out[run.Kind] = run
	}
	return out, rows.Err()
}

// LatestDone returns the newest finished run of kind, or nil.
func (s Store) LatestDone(ctx context.Context, projectID, kind string, before time.Time) (*Run, error) {
	return scanRun(s.DB.QueryRow(ctx, `SELECT `+runColumns+` FROM go_local_runs
		WHERE project_id=$1 AND kind=$2 AND status='done' AND created_at < $3 ORDER BY created_at DESC LIMIT 1`, projectID, kind, before))
}

// Cached returns a finished run of the same input newer than since. A run
// that came back with parts missing is not reused, so asking again buys
// them instead of showing the gap for a day.
func (s Store) Cached(ctx context.Context, projectID, kind, inputHash string, since time.Time) (*Run, error) {
	return scanRun(s.DB.QueryRow(ctx, `SELECT `+runColumns+` FROM go_local_runs
		WHERE project_id=$1 AND kind=$2 AND input_hash=$3 AND status='done' AND created_at > $4
		AND COALESCE(jsonb_array_length(result->'missing'), 0) = 0
		ORDER BY created_at DESC LIMIT 1`, projectID, kind, inputHash, since))
}

// CreateRun starts a run. When one of the kind is already running for the
// project, that run is returned and created is false: a double click or a
// second tab never pays twice.
func (s Store) CreateRun(ctx context.Context, r Run) (*Run, bool, error) {
	run, err := scanRun(s.DB.QueryRow(ctx, `INSERT INTO go_local_runs (project_id, organization_id, kind, input_hash, input)
		VALUES ($1,$2,$3,$4,$5) ON CONFLICT (project_id, kind) WHERE status = 'running' DO NOTHING
		RETURNING `+runColumns, r.ProjectID, r.OrganizationID, r.Kind, r.InputHash, r.Input))
	if err != nil || run != nil {
		return run, run != nil, err
	}
	run, err = scanRun(s.DB.QueryRow(ctx, `SELECT `+runColumns+` FROM go_local_runs
		WHERE project_id=$1 AND kind=$2 AND status='running'`, r.ProjectID, r.Kind))
	if err == nil && run == nil {
		err = errors.New("local run conflict without a running run")
	}
	return run, false, err
}

// Run loads one run of the project.
func (s Store) Run(ctx context.Context, projectID, runID string) (*Run, error) {
	return scanRun(s.DB.QueryRow(ctx, `SELECT `+runColumns+` FROM go_local_runs WHERE id::text=$1 AND project_id=$2`, runID, projectID))
}

func (s Store) runByID(ctx context.Context, runID string) (*Run, error) {
	return scanRun(s.DB.QueryRow(ctx, `SELECT `+runColumns+` FROM go_local_runs WHERE id::text=$1`, runID))
}

// FinishRun stores a run's outcome once; a finished run is never changed.
func (s Store) FinishRun(ctx context.Context, runID, status string, result any, message string) error {
	var raw []byte
	if result != nil {
		var err error
		if raw, err = json.Marshal(result); err != nil {
			return fmt.Errorf("encode local result: %w", err)
		}
	}
	_, err := s.DB.Exec(ctx, `UPDATE go_local_runs SET status=$2, result=$3, error=$4, finished_at=now()
		WHERE id::text=$1 AND status='running'`, runID, status, raw, message)
	if err != nil {
		return fmt.Errorf("finish local run: %w", err)
	}
	return nil
}

// RecordTask stores a posted task the moment the provider accepted it.
func (s Store) RecordTask(ctx context.Context, runID string, t task, taskID string) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO go_local_tasks (run_id, task_key, endpoint, task_id) VALUES ($1::uuid,$2,$3,$4)
		ON CONFLICT (run_id, task_key) DO NOTHING`, runID, t.Key, t.Endpoint, taskID)
	if err != nil {
		return fmt.Errorf("record local task: %w", err)
	}
	return nil
}

// FailTask marks a task that was never posted, or whose outcome is unknown,
// so the run stops waiting for it.
func (s Store) FailTask(ctx context.Context, runID string, t task) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO go_local_tasks (run_id, task_key, endpoint, task_id, state) VALUES ($1::uuid,$2,$3,'','failed')
		ON CONFLICT (run_id, task_key) DO NOTHING`, runID, t.Key, t.Endpoint)
	if err != nil {
		return fmt.Errorf("fail local task: %w", err)
	}
	return nil
}

// SetTask stores a collected task's state and result.
func (s Store) SetTask(ctx context.Context, runID, key, state string, result json.RawMessage) error {
	_, err := s.DB.Exec(ctx, `UPDATE go_local_tasks SET state=$3, result=$4 WHERE run_id=$1::uuid AND task_key=$2`, runID, key, state, []byte(result))
	if err != nil {
		return fmt.Errorf("update local task: %w", err)
	}
	return nil
}

// tasks lists a run's tasks by key.
func (s Store) tasks(ctx context.Context, runID string) (map[string]storedTask, error) {
	rows, err := s.DB.Query(ctx, `SELECT task_key, endpoint, task_id, state, result FROM go_local_tasks WHERE run_id=$1::uuid`, runID)
	if err != nil {
		return nil, fmt.Errorf("list local tasks: %w", err)
	}
	defer rows.Close()
	out := map[string]storedTask{}
	for rows.Next() {
		var t storedTask
		if err := rows.Scan(&t.Key, &t.Endpoint, &t.TaskID, &t.State, &t.Result); err != nil {
			return nil, fmt.Errorf("scan local task: %w", err)
		}
		out[t.Key] = t
	}
	return out, rows.Err()
}

// Watch reports whether the weekly watch is on and when it runs next.
func (s Store) Watch(ctx context.Context, projectID string) (bool, *time.Time, error) {
	var enabled bool
	var next time.Time
	err := s.DB.QueryRow(ctx, `SELECT enabled, next_run_at FROM go_local_watch WHERE project_id=$1`, projectID).Scan(&enabled, &next)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil, nil
	}
	if err != nil {
		return false, nil, fmt.Errorf("load local watch: %w", err)
	}
	if !enabled {
		return false, nil, nil
	}
	return true, &next, nil
}

// SetWatch turns the weekly watch on or off. Turning it on schedules the
// next run a week out: the run the customer starts by hand is this week's.
func (s Store) SetWatch(ctx context.Context, organizationID, projectID string, enabled bool, next time.Time) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO go_local_watch (project_id, organization_id, enabled, next_run_at) VALUES ($1,$2,$3,$4)
		ON CONFLICT (project_id) DO UPDATE SET organization_id=EXCLUDED.organization_id, enabled=EXCLUDED.enabled,
		next_run_at=CASE WHEN EXCLUDED.enabled AND NOT go_local_watch.enabled THEN EXCLUDED.next_run_at ELSE go_local_watch.next_run_at END,
		updated_at=now()`, projectID, organizationID, enabled, next)
	if err != nil {
		return fmt.Errorf("save local watch: %w", err)
	}
	return nil
}

type dueWatch struct{ ProjectID, OrganizationID string }

// claimDueWatches moves due watches a week ahead and returns them, so every
// server instance can tick and each watch still starts once.
func (s Store) claimDueWatches(ctx context.Context, now time.Time, limit int) ([]dueWatch, error) {
	rows, err := s.DB.Query(ctx, `UPDATE go_local_watch SET next_run_at = $1::timestamptz + interval '7 days'
		WHERE project_id IN (SELECT project_id FROM go_local_watch WHERE enabled AND next_run_at <= $1
			ORDER BY next_run_at LIMIT $2 FOR UPDATE SKIP LOCKED)
		RETURNING project_id, organization_id`, now, limit)
	if err != nil {
		return nil, fmt.Errorf("claim local watches: %w", err)
	}
	defer rows.Close()
	var out []dueWatch
	for rows.Next() {
		var w dueWatch
		if err := rows.Scan(&w.ProjectID, &w.OrganizationID); err != nil {
			return nil, fmt.Errorf("scan local watch: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// ReplyDrafts returns the saved drafts of the given reviews.
func (s Store) ReplyDrafts(ctx context.Context, projectID string, reviewIDs []string) (map[string]string, error) {
	rows, err := s.DB.Query(ctx, `SELECT review_id, reply FROM go_local_reply_drafts WHERE project_id=$1 AND review_id = ANY($2)`, projectID, reviewIDs)
	if err != nil {
		return nil, fmt.Errorf("list reply drafts: %w", err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var id, reply string
		if err := rows.Scan(&id, &reply); err != nil {
			return nil, fmt.Errorf("scan reply draft: %w", err)
		}
		out[id] = reply
	}
	return out, rows.Err()
}

// SaveReplyDraft stores the latest draft for a review.
func (s Store) SaveReplyDraft(ctx context.Context, projectID, reviewID, reply string) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO go_local_reply_drafts (project_id, review_id, reply) VALUES ($1,$2,$3)
		ON CONFLICT (project_id, review_id) DO UPDATE SET reply=EXCLUDED.reply, created_at=now()`, projectID, reviewID, reply)
	if err != nil {
		return fmt.Errorf("save reply draft: %w", err)
	}
	return nil
}
