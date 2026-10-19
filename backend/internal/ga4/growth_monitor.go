package ga4

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/toufiqqureshi/uplumo/backend/internal/gsc"
)

// GrowthMonitorInput explicitly opts a project into free weekly Google reads.
type GrowthMonitorInput struct {
	Enabled   bool   `json:"enabled"`
	GoalEvent string `json:"goalEvent"`
}

// GrowthDigest keeps only the top three actions, not a second unbounded report.
type GrowthDigest struct {
	GeneratedAt time.Time       `json:"generatedAt"`
	Range       DateRange       `json:"range"`
	GoalEvent   string          `json:"goalEvent"`
	Pages       []GrowthPage    `json:"pages"`
	Alerts      []GrowthFinding `json:"alerts"`
	Limited     bool            `json:"limited"`
	Truncated   bool            `json:"truncated"`
}

// GrowthMonitorState is project-scoped configuration and its last successful check.
type GrowthMonitorState struct {
	Enabled     bool          `json:"enabled"`
	GoalEvent   string        `json:"goalEvent"`
	NextCheckAt *time.Time    `json:"nextCheckAt"`
	CheckedAt   *time.Time    `json:"checkedAt"`
	Digest      *GrowthDigest `json:"digest"`
	LastError   *string       `json:"lastError"`
}

// Monitor returns only configuration and digest within the authorized project.
func (r *GrowthRepository) Monitor(ctx context.Context, org, project string) (GrowthMonitorState, error) {
	var state GrowthMonitorState
	err := r.DB.QueryRow(ctx, `SELECT enabled,goal_event,next_check_at,checked_at,digest,last_error FROM go_search_growth_monitors WHERE organization_id=$1 AND project_id=$2`, org, project).Scan(&state.Enabled, &state.GoalEvent, &state.NextCheckAt, &state.CheckedAt, &state.Digest, &state.LastError)
	if errors.Is(err, pgx.ErrNoRows) {
		return state, nil
	}
	if err != nil {
		return state, fmt.Errorf("read growth monitor: %w", err)
	}
	return state, nil
}

// SaveGrowthMonitor changes explicit opt-in while preserving an identical schedule.
func (s *Service) SaveGrowthMonitor(ctx context.Context, search *gsc.Service, org, project string, in GrowthMonitorInput) (GrowthMonitorState, error) {
	if err := validateGrowthInput(GrowthInput{GoalEvent: in.GoalEvent}); err != nil {
		return GrowthMonitorState{}, err
	}
	// Turning off does not require a working Google grant.
	if !in.Enabled {
		_, err := s.Growth.DB.Exec(ctx, `UPDATE go_search_growth_monitors SET enabled=false,revision=revision+1,updated_at=now() WHERE organization_id=$1 AND project_id=$2`, org, project)
		if err != nil {
			return GrowthMonitorState{}, fmt.Errorf("disable growth monitor: %w", err)
		}
		return s.Growth.Monitor(ctx, org, project)
	}
	if in.GoalEvent == "" {
		return GrowthMonitorState{}, invalid("Choose your successful outcome before enabling weekly checks.")
	}
	c, err := s.growthConnection(ctx, project)
	if err != nil {
		return GrowthMonitorState{}, err
	}
	if search == nil || search.Connections == nil {
		return GrowthMonitorState{}, reportErr("gsc_unavailable", "Connect Search Console first.")
	}
	scope, err := search.Connections.GetByProjectID(ctx, org, project)
	if err != nil {
		return GrowthMonitorState{}, mapOpportunityGSCErr(err)
	}
	if err := s.validateGrowthGoal(ctx, c, in.GoalEvent); err != nil {
		return GrowthMonitorState{}, err
	}
	_, err = s.Growth.DB.Exec(ctx, `INSERT INTO go_search_growth_monitors(project_id,organization_id,enabled,goal_event,property_id,site_url) SELECT id,organization_id,true,$3,$4,$5 FROM projects WHERE id=$2 AND organization_id=$1 ON CONFLICT(project_id) DO UPDATE SET enabled=true,goal_event=excluded.goal_event,property_id=excluded.property_id,site_url=excluded.site_url,revision=CASE WHEN go_search_growth_monitors.enabled AND go_search_growth_monitors.goal_event=excluded.goal_event AND go_search_growth_monitors.property_id=excluded.property_id AND go_search_growth_monitors.site_url=excluded.site_url THEN go_search_growth_monitors.revision ELSE go_search_growth_monitors.revision+1 END,next_check_at=CASE WHEN go_search_growth_monitors.enabled AND go_search_growth_monitors.goal_event=excluded.goal_event AND go_search_growth_monitors.property_id=excluded.property_id AND go_search_growth_monitors.site_url=excluded.site_url THEN go_search_growth_monitors.next_check_at ELSE now() END,digest=CASE WHEN go_search_growth_monitors.goal_event=excluded.goal_event AND go_search_growth_monitors.property_id=excluded.property_id AND go_search_growth_monitors.site_url=excluded.site_url THEN go_search_growth_monitors.digest ELSE NULL END,last_error=NULL,updated_at=now() WHERE go_search_growth_monitors.organization_id=excluded.organization_id`, org, project, in.GoalEvent, c.PropertyID, scope.SiteURL)
	if err != nil {
		return GrowthMonitorState{}, fmt.Errorf("save growth monitor: %w", err)
	}
	return s.Growth.Monitor(ctx, org, project)
}

func growthDigest(report GrowthReport) GrowthDigest {
	digest := GrowthDigest{GeneratedAt: report.GeneratedAt, Range: report.Range, GoalEvent: report.GoalEvent, Pages: []GrowthPage{}, Alerts: []GrowthFinding{}, Limited: report.Limited, Truncated: report.Truncated}
	for _, page := range report.Pages {
		if len(page.Findings) > 0 && len(digest.Pages) < 3 {
			digest.Pages = append(digest.Pages, page)
		}
		if report.Limited {
			continue
		}
		// Require both search loss and a meaningful selected-goal rate drop. A
		// declining click count alone does not establish a business-impact alert.
		if page.Previous == nil || page.Previous.Sessions == nil || page.Current.Sessions == nil || *page.Previous.Sessions < 100 || *page.Current.Sessions < 100 || page.Previous.GoalRate == nil || page.Current.GoalRate == nil || *page.Previous.GoalRate**page.Previous.Sessions < 5 {
			continue
		}
		if page.Previous.Clicks >= 30 && page.Current.Clicks <= page.Previous.Clicks*.8 && *page.Current.GoalRate < *page.Previous.GoalRate*.7 && len(digest.Alerts) < 5 {
			digest.Alerts = append(digest.Alerts, GrowthFinding{Kind: "business_drop", Title: "Search clicks and your selected outcome rate both fell", Evidence: fmt.Sprintf("%s: clicks %.0f → %.0f; %s session rate %.1f%% → %.1f%%.", page.Page, page.Previous.Clicks, page.Current.Clicks, report.GoalEvent, *page.Previous.GoalRate*100, *page.Current.GoalRate*100), Action: "Verify goal tracking, audience mix and the conversion path before editing content. Compare seasonal demand; this is an investigation signal.", Confidence: "Sufficient sample for review; cause unverified"})
		}
	}
	return digest
}

// CheckGrowthMonitors processes one due project per tick. Atomic claiming and
// a revision token prevent replicas/configuration changes overwriting each other.
// Failure waits a day; success waits a week. No paid provider or delivery call.
func (s *Service) CheckGrowthMonitors(ctx context.Context, search *gsc.Service) error {
	if s == nil || s.Growth == nil || search == nil {
		return nil
	}
	var org, project, goal, property, site string
	var revision int64
	err := s.Growth.DB.QueryRow(ctx, `UPDATE go_search_growth_monitors SET next_check_at=now()+interval '1 day',revision=revision+1 WHERE project_id=(SELECT project_id FROM go_search_growth_monitors WHERE enabled AND next_check_at<=now() ORDER BY next_check_at LIMIT 1 FOR UPDATE SKIP LOCKED) RETURNING organization_id,project_id,goal_event,property_id,site_url,revision`).Scan(&org, &project, &goal, &property, &site, &revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("claim growth monitor: %w", err)
	}
	work, cancel := context.WithTimeout(ctx, 100*time.Second)
	defer cancel()
	if s.GrowthMonitorAllowed != nil {
		allowed, checkErr := s.GrowthMonitorAllowed(work, org)
		if checkErr != nil {
			err = checkErr
		} else if !allowed {
			err = invalid("The current plan does not include scheduled Google checks.")
		}
	}
	var c Connection
	if err == nil {
		c, err = s.Connections.GetByProjectID(work, project)
	}
	if err == nil {
		scope, scopeErr := search.Connections.GetByProjectID(work, org, project)
		err = scopeErr
		if err == nil && (c.PropertyID != property || scope.SiteURL != site) {
			err = invalid("The connected properties changed. Review and re-enable weekly monitoring.")
		}
	}
	var report GrowthReport
	if err == nil {
		report, err = s.GetGrowthReport(work, search, org, project, GrowthInput{GoalEvent: goal})
	}
	if err != nil {
		// Store a safe status only, never raw provider errors or tokens.
		_, saveErr := s.Growth.DB.Exec(ctx, `UPDATE go_search_growth_monitors SET last_error='Weekly check could not complete. Review your plan, Google connections, selected goal and property, then try again.',updated_at=now() WHERE organization_id=$1 AND project_id=$2 AND revision=$3 AND enabled`, org, project, revision)
		if saveErr != nil {
			return fmt.Errorf("record monitor failure: %w", saveErr)
		}
		return fmt.Errorf("run growth monitor: %w", err)
	}
	digest := growthDigest(report)
	_, err = s.Growth.DB.Exec(ctx, `UPDATE go_search_growth_monitors SET digest=$4,checked_at=now(),last_error=NULL,next_check_at=now()+interval '7 days',updated_at=now() WHERE organization_id=$1 AND project_id=$2 AND revision=$3 AND enabled`, org, project, revision, digest)
	if err != nil {
		return fmt.Errorf("store growth digest: %w", err)
	}
	return nil
}
