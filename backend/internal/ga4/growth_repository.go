package ga4

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/ids"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/pgdb"
)

// GrowthRepository keeps a single short-lived report per project and durable,
// server-derived page baselines. All reads and writes include both scopes.
type GrowthRepository struct{ DB *pgxpool.Pool }

// GrowthAction preserves a page baseline and the first date its change was applied.
type GrowthAction struct {
	ID            string       `json:"id"`
	Page          string       `json:"page"`
	Note          string       `json:"note"`
	Status        string       `json:"status"`
	Baseline      GrowthPage   `json:"baseline"`
	Source        GrowthReport `json:"source"`
	BaselineRange DateRange    `json:"baselineRange"`
	CreatedAt     time.Time    `json:"createdAt"`
	AppliedAt     *time.Time   `json:"appliedAt"`
}

// Cached returns a matching unexpired report from the authorized project.
func (r *GrowthRepository) Cached(ctx context.Context, org, project, key string) (GrowthReport, bool, error) {
	var report GrowthReport
	err := r.DB.QueryRow(ctx, `SELECT report FROM go_search_growth_reports WHERE organization_id=$1 AND project_id=$2 AND cache_key=$3 AND created_at>now()-interval '10 minutes'`, org, project, key).Scan(&report)
	if errors.Is(err, pgx.ErrNoRows) {
		return report, false, nil
	}
	if err != nil {
		return report, false, fmt.Errorf("read growth report: %w", err)
	}
	return report, true, nil
}

// StoreReport replaces the single cached report for this authorized project.
func (r *GrowthRepository) StoreReport(ctx context.Context, org, project string, report GrowthReport) error {
	_, err := r.DB.Exec(ctx, `INSERT INTO go_search_growth_reports(project_id,organization_id,cache_key,report) SELECT id,organization_id,$3,$4 FROM projects WHERE id=$2 AND organization_id=$1 ON CONFLICT(project_id) DO UPDATE SET cache_key=excluded.cache_key,report=excluded.report,created_at=now() WHERE go_search_growth_reports.organization_id=excluded.organization_id`, org, project, report.ID, report)
	if err != nil {
		return fmt.Errorf("store growth report: %w", err)
	}
	return nil
}

// SaveAction freezes a server-derived page baseline once, including concurrent saves.
func (r *GrowthRepository) SaveAction(ctx context.Context, org, project, reportID, page, note string) (string, error) {
	note = strings.TrimSpace(note)
	if utf8.RuneCountInString(note) < 1 || utf8.RuneCountInString(note) > 1000 || strings.ContainsRune(note, 0) {
		return "", invalid("Add a change note of 1–1,000 characters.")
	}
	report, ok, err := r.Cached(ctx, org, project, reportID)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", invalid("Refresh the analysis before saving an action.")
	}
	var selected *GrowthPage
	for i := range report.Pages {
		if report.Pages[i].Page == page {
			selected = &report.Pages[i]
			break
		}
	}
	if selected == nil || len(page) > 2048 {
		return "", invalid("Choose a page from this analysis.")
	}
	source := report
	source.Pages = nil
	id, err := ids.New()
	if err != nil {
		return "", err
	}
	err = pgdb.InTx(ctx, r.DB, func(tx pgx.Tx) error {
		// Serialize the project cap and duplicate-page creation across replicas.
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, "growth:"+org+":"+project); err != nil {
			return err
		}
		var existing string
		err := tx.QueryRow(ctx, `SELECT id FROM go_search_growth_actions WHERE organization_id=$1 AND project_id=$2 AND page=$3`, org, project, page).Scan(&existing)
		if err == nil {
			id = existing
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var count int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM go_search_growth_actions WHERE organization_id=$1 AND project_id=$2`, org, project).Scan(&count); err != nil {
			return err
		}
		if count >= 100 {
			return invalid("Track up to 100 page changes per project. Remove an old action to add another.")
		}
		_, err = tx.Exec(ctx, `INSERT INTO go_search_growth_actions(id,organization_id,project_id,page,note,baseline,source,baseline_range) SELECT $1,$2,id,$4,$5,$6,$7,$8 FROM projects WHERE id=$3 AND organization_id=$2`, id, org, project, page, note, selected, source, report.Range)
		return err
	})
	if err != nil {
		return "", fmt.Errorf("save growth action: %w", err)
	}
	return id, nil
}

// Actions lists bounded saved changes inside the organization and project.
func (r *GrowthRepository) Actions(ctx context.Context, org, project string) ([]GrowthAction, error) {
	rows, err := r.DB.Query(ctx, `SELECT id,page,note,status,baseline,source,baseline_range,created_at,applied_at FROM go_search_growth_actions WHERE organization_id=$1 AND project_id=$2 ORDER BY created_at DESC LIMIT 100`, org, project)
	if err != nil {
		return nil, fmt.Errorf("list growth actions: %w", err)
	}
	defer rows.Close()
	actions := []GrowthAction{}
	for rows.Next() {
		var a GrowthAction
		if err := rows.Scan(&a.ID, &a.Page, &a.Note, &a.Status, &a.Baseline, &a.Source, &a.BaselineRange, &a.CreatedAt, &a.AppliedAt); err != nil {
			return nil, fmt.Errorf("decode growth action: %w", err)
		}
		actions = append(actions, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read growth actions: %w", err)
	}
	return actions, nil
}

// UpdateAction changes status without moving the original applied timestamp.
func (r *GrowthRepository) UpdateAction(ctx context.Context, org, project, id, status string) error {
	if status != "applied" && status != "dismissed" && status != "planned" {
		return invalid("Choose planned, applied or dismissed.")
	}
	tag, err := r.DB.Exec(ctx, `UPDATE go_search_growth_actions SET status=$4,applied_at=CASE WHEN $4='applied' THEN COALESCE(applied_at,now()) ELSE applied_at END WHERE organization_id=$1 AND project_id=$2 AND id=$3`, org, project, id, status)
	if err != nil {
		return fmt.Errorf("update growth action: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return reportErr("growth_not_found", "This saved action was not found.")
	}
	return nil
}

// DeleteAction removes only a change in the authorized project.
func (r *GrowthRepository) DeleteAction(ctx context.Context, org, project, id string) error {
	tag, err := r.DB.Exec(ctx, `DELETE FROM go_search_growth_actions WHERE organization_id=$1 AND project_id=$2 AND id=$3`, org, project, id)
	if err != nil {
		return fmt.Errorf("delete growth action: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return reportErr("growth_not_found", "This saved action was not found.")
	}
	return nil
}
