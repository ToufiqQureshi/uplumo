package sitereport

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// examplesPerIssue caps the page URLs listed under one audit issue.
const examplesPerIssue = 5

// Repository reads the stored project, audit and backlink rows.
type Repository struct{ DB *pgxpool.Pool }

// Project returns an active project inside the organization.
func (r Repository) Project(ctx context.Context, organizationID, projectID string) (Project, error) {
	var name string
	var domain *string
	err := r.DB.QueryRow(ctx, `SELECT name, domain FROM projects WHERE id=$1 AND organization_id=$2 AND archived_at IS NULL`, projectID, organizationID).Scan(&name, &domain)
	if errors.Is(err, pgx.ErrNoRows) {
		return Project{}, ErrProjectNotFound
	}
	if err != nil {
		return Project{}, fmt.Errorf("load report project: %w", err)
	}
	out := Project{Name: name}
	if domain != nil {
		out.Domain = *domain
	}
	return out, nil
}

// LatestAudit returns the newest completed audit, or nil when none finished.
func (r Repository) LatestAudit(ctx context.Context, projectID string) (*Audit, error) {
	var auditID string
	var completedAt *time.Time
	var startedAt time.Time
	out := Audit{}
	err := r.DB.QueryRow(ctx, `SELECT id, pages_crawled, started_at, completed_at FROM go_audits
		WHERE project_id=$1 AND status='completed' ORDER BY started_at DESC, id DESC LIMIT 1`, projectID).Scan(&auditID, &out.PagesCrawled, &startedAt, &completedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load latest audit: %w", err)
	}
	out.CompletedAt = startedAt
	if completedAt != nil {
		out.CompletedAt = *completedAt
	}
	rows, err := r.DB.Query(ctx, `SELECT issue_type, severity, count(DISTINCT page_url)::int,
		(array_agg(DISTINCT page_url ORDER BY page_url))[1:$2]
		FROM go_audit_issues WHERE audit_id=$1 GROUP BY issue_type, severity`, auditID, examplesPerIssue)
	if err != nil {
		return nil, fmt.Errorf("query audit issues: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item Issue
		if err := rows.Scan(&item.Type, &item.Severity, &item.Pages, &item.Examples); err != nil {
			return nil, fmt.Errorf("read audit issue: %w", err)
		}
		switch item.Severity {
		case "critical":
			out.Critical += item.Pages
		case "warning":
			out.Warning += item.Pages
		default:
			out.Info += item.Pages
		}
		out.Issues = append(out.Issues, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit issues: %w", err)
	}
	sortIssues(out.Issues)
	lighthouse, err := r.lighthouse(ctx, auditID)
	if err != nil {
		return nil, err
	}
	out.Lighthouse = lighthouse
	return &out, nil
}

func (r Repository) lighthouse(ctx context.Context, auditID string) (*Lighthouse, error) {
	var out Lighthouse
	var perf, a11y, best, seo *float64
	err := r.DB.QueryRow(ctx, `SELECT count(*)::int, avg(performance_score)::float8, avg(accessibility_score)::float8, avg(best_practices_score)::float8, avg(seo_score)::float8,
		avg(lcp_ms)::float8, avg(cls)::float8, avg(inp_ms)::float8
		FROM go_audit_lighthouse_results WHERE audit_id=$1 AND strategy='mobile' AND error_message IS NULL`, auditID).Scan(
		&out.Pages, &perf, &a11y, &best, &seo, &out.LCPMillis, &out.CLS, &out.INPMillis)
	if err != nil {
		return nil, fmt.Errorf("load lighthouse averages: %w", err)
	}
	if out.Pages == 0 {
		return nil, nil
	}
	out.Performance, out.Accessibility, out.BestPractices, out.SEO = roundScore(perf), roundScore(a11y), roundScore(best), roundScore(seo)
	return &out, nil
}

// LatestBacklinks returns the newest stored snapshot for the domain, or nil.
func (r Repository) LatestBacklinks(ctx context.Context, projectID, domain string) (*Backlinks, error) {
	var out Backlinks
	var captured string
	err := r.DB.QueryRow(ctx, `SELECT rank, backlinks, referring_domains, new_backlinks, lost_backlinks, new_referring_domains, lost_referring_domains, captured_at
		FROM backlink_snapshots WHERE project_id=$1 AND domain=$2 ORDER BY id DESC LIMIT 1`, projectID, domain).Scan(
		&out.Rank, &out.Backlinks, &out.ReferringDomains, &out.NewBacklinks, &out.LostBacklinks, &out.NewReferringDomains, &out.LostReferringDomains, &captured)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load backlink snapshot: %w", err)
	}
	if t, err := time.Parse(time.RFC3339Nano, captured); err == nil {
		out.CapturedAt = t
	}
	return &out, nil
}

func roundScore(v *float64) *int {
	if v == nil {
		return nil
	}
	n := int(*v + 0.5)
	return &n
}

func sortIssues(rows []Issue) {
	rank := map[string]int{"critical": 0, "warning": 1}
	weight := func(s string) int {
		if n, ok := rank[s]; ok {
			return n
		}
		return 2
	}
	slices.SortFunc(rows, func(a, b Issue) int {
		if d := weight(a.Severity) - weight(b.Severity); d != 0 {
			return d
		}
		if d := b.Pages - a.Pages; d != 0 {
			return d
		}
		switch {
		case a.Type < b.Type:
			return -1
		case a.Type > b.Type:
			return 1
		}
		return 0
	})
}
