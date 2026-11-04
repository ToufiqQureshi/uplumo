package citefix

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	// fixProofWindow is the longest period compared on each side of a fix.
	fixProofWindow = 28 * 24 * time.Hour
	// fixProofLookback bounds which fixes are followed up.
	fixProofLookback = 180 * 24 * time.Hour
	maxFixProofs     = 10
)

// FixProof follows one page the agent fixed: AI-assistant visitors to that
// page and the leads in their visits, for the same number of days before and
// after the fix, and whether the latest AI check cites the page. The link
// is likely, not proven: AI assistants do not say who clicked.
type FixProof struct {
	URL             string    `json:"url"`
	FixedAt         time.Time `json:"fixedAt"`
	Days            int       `json:"days"`
	AIVisitorsAfter int64     `json:"aiVisitorsAfter"`
	AIVisitorsPrior int64     `json:"aiVisitorsBefore"`
	LeadsAfter      int64     `json:"leadsAfter"`
	LeadsPrior      int64     `json:"leadsBefore"`
	// LeadsTracked is false until the owner picks lead actions in Analytics.
	LeadsTracked bool `json:"leadsTracked"`
	// CitedNow is whether the latest AI Visibility check cites this page.
	CitedNow bool `json:"citedNow"`
}

// fixProofs follows up the project's agent-fixed pages, newest fix first.
// It reads Uplumo's own analytics only: no paid call.
func (s Store) fixProofs(ctx context.Context, organizationID, projectID string, proof *Proof, now time.Time) ([]FixProof, error) {
	rows, err := s.DB.Query(ctx, `SELECT url, max(finished_at) FROM go_citefix_agent_runs
		WHERE organization_id = $1 AND project_id = $2 AND status = 'completed' AND NOT cached
			AND finished_at >= $3
		GROUP BY url ORDER BY 2 DESC LIMIT $4`, organizationID, projectID, now.Add(-fixProofLookback), maxFixProofs)
	if err != nil {
		return nil, fmt.Errorf("list fixed pages: %w", err)
	}
	fixes, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (FixProof, error) {
		var f FixProof
		return f, row.Scan(&f.URL, &f.FixedAt)
	})
	if err != nil {
		return nil, fmt.Errorf("read fixed pages: %w", err)
	}
	cited := map[string]bool{}
	if proof != nil {
		for _, page := range proof.YourPages {
			cited[canonicalCitedURL(page.URL)] = true
		}
	}
	for i := range fixes {
		f := &fixes[i]
		span := min(now.Sub(f.FixedAt), fixProofWindow)
		f.Days = int(span.Hours() / 24)
		f.CitedNow = cited[canonicalCitedURL(f.URL)]
		parsed, err := url.Parse(f.URL)
		if err != nil || span <= 0 {
			continue
		}
		path := parsed.EscapedPath()
		if path == "" {
			path = "/"
		}
		if err := s.aiVisits(ctx, projectID, path, f.FixedAt.Add(-span), f.FixedAt, &f.AIVisitorsPrior, &f.LeadsPrior, &f.LeadsTracked); err != nil {
			return nil, err
		}
		if err := s.aiVisits(ctx, projectID, path, f.FixedAt, f.FixedAt.Add(span), &f.AIVisitorsAfter, &f.LeadsAfter, &f.LeadsTracked); err != nil {
			return nil, err
		}
	}
	return fixes, nil
}

// aiVisits counts, in [start, end), the visitor-days that came to path from
// an AI assistant and those of them whose visit included a chosen lead
// action. A project without Analytics counts zero.
func (s Store) aiVisits(ctx context.Context, projectID, path string, start, end time.Time, visitors, leads *int64, tracked *bool) error {
	err := s.DB.QueryRow(ctx, `WITH site AS (
			SELECT id FROM go_analytics_sites WHERE project_id = $1
		), ai AS (
			SELECT DISTINCT visitor_hash, session_hash FROM go_analytics_events
			WHERE site_id = (SELECT id FROM site) AND path = $2 AND ai_source IS NOT NULL
				AND occurred_at >= $3 AND occurred_at < $4
		), lead_names AS (
			SELECT name FROM go_analytics_lead_events WHERE site_id = (SELECT id FROM site)
		)
		SELECT (SELECT count(DISTINCT visitor_hash) FROM ai),
			(SELECT count(DISTINCT c.visitor_hash) FROM go_analytics_custom_events c
				JOIN ai USING (session_hash)
				WHERE c.site_id = (SELECT id FROM site) AND c.name IN (SELECT name FROM lead_names)
					AND c.occurred_at >= $3 AND c.occurred_at < $4),
			EXISTS (SELECT 1 FROM lead_names)`, projectID, path, start, end).Scan(visitors, leads, tracked)
	if err != nil {
		return fmt.Errorf("count AI visits to %s: %w", path, err)
	}
	return nil
}
