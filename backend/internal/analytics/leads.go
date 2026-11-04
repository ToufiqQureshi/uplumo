package analytics

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

// maxLeadEvents bounds how many event names one project may count as leads.
const maxLeadEvents = 20

// ErrInvalidLeads means a lead selection has too many or malformed names.
var ErrInvalidLeads = fmt.Errorf("choose at most %d event names of 1 to %d characters", maxLeadEvents, maxEventName)

// ErrNoSite means the project has no analytics site yet.
var ErrNoSite = errors.New("project has no analytics site")

// LeadSummary counts the actions the owner chose as leads. Configured is
// false until the owner has picked at least one event, so the screen can ask
// instead of showing a misleading zero. Visitors is distinct visitor-days
// across every lead action, so one visitor who does two lead actions counts
// once.
type LeadSummary struct {
	Configured bool  `json:"configured"`
	Count      int64 `json:"count"`
	Visitors   int64 `json:"visitors"`
	// PreviousVisitors is Visitors for the same-length period just before
	// the range, for a "compared with last period" line.
	PreviousVisitors int64        `json:"previousVisitors"`
	Actions          []EventCount `json:"actions"`
	Pages            []LeadPage   `json:"pages"`
}

// SourceLeads is one acquisition channel's visitors and the visitors whose
// visit from it led to a lead action. A visit belongs to the channel of its
// first page, like AcquisitionChannels.
type SourceLeads struct {
	Channel      string `json:"channel"`
	Visitors     int64  `json:"visitors"`
	LeadVisitors int64  `json:"leadVisitors"`
}

// AILanding is where visits from one AI assistant landed, and how many of
// those visitors went on to a lead action in the same visit.
type AILanding struct {
	Source       string `json:"source"`
	Path         string `json:"path"`
	Visitors     int64  `json:"visitors"`
	LeadVisitors int64  `json:"leadVisitors"`
}

// LeadPage is a path's pageview visitors and the visitors who did a lead
// action on it, so the screen can point at the page that loses people.
type LeadPage struct {
	Path         string `json:"path"`
	Visitors     int64  `json:"visitors"`
	LeadVisitors int64  `json:"leadVisitors"`
}

// cleanLeadNames trims, validates and de-duplicates names, keeping order.
func cleanLeadNames(names []string) ([]string, error) {
	if len(names) > maxLeadEvents {
		return nil, ErrInvalidLeads
	}
	out := make([]string, 0, len(names))
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" || !utf8.ValidString(name) || utf8.RuneCountInString(name) > maxEventName ||
			strings.ContainsFunc(name, unicode.IsControl) {
			return nil, ErrInvalidLeads
		}
		if !slices.Contains(out, name) {
			out = append(out, name)
		}
	}
	return out, nil
}

// LeadEvents returns the event names projectID counts as leads, in the order
// they were saved. The caller must have authorized access to projectID.
func (s *Service) LeadEvents(ctx context.Context, projectID string) ([]string, error) {
	names, err := s.repo.leadNames(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("lead events for project %s: %w", projectID, err)
	}
	return names, nil
}

// SaveLeadEvents replaces projectID's lead events with names. An empty list
// clears them. The caller must have authorized access to projectID.
func (s *Service) SaveLeadEvents(ctx context.Context, projectID string, names []string) ([]string, error) {
	clean, err := cleanLeadNames(names)
	if err != nil {
		return nil, err
	}
	if err := s.repo.saveLeadNames(ctx, projectID, clean); err != nil {
		return nil, fmt.Errorf("save lead events for project %s: %w", projectID, err)
	}
	return clean, nil
}

func (r repository) leadNames(ctx context.Context, projectID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT l.name FROM go_analytics_lead_events l
		JOIN go_analytics_sites s ON s.id = l.site_id
		WHERE s.project_id = $1
		ORDER BY l.created_at, l.name`, projectID)
	if err != nil {
		return nil, fmt.Errorf("query lead events: %w", err)
	}
	names, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("read lead events: %w", err)
	}
	return names, nil
}

// saveLeadNames swaps the site's lead events in one transaction, so a reader
// sees either the old set or the new one. Row order in created_at keeps the
// owner's order; the per-row offset makes it strict within one save.
func (r repository) saveLeadNames(ctx context.Context, projectID string, names []string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin lead events: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var siteID int64
	err = tx.QueryRow(ctx, `SELECT id FROM go_analytics_sites WHERE project_id = $1 FOR UPDATE`, projectID).Scan(&siteID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNoSite
	}
	if err != nil {
		return fmt.Errorf("lock analytics site: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM go_analytics_lead_events WHERE site_id = $1`, siteID); err != nil {
		return fmt.Errorf("clear lead events: %w", err)
	}
	if len(names) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO go_analytics_lead_events (site_id, name, created_at)
			SELECT $1, name, now() + (ord * interval '1 microsecond')
			FROM unnest($2::text[]) WITH ORDINALITY AS t(name, ord)`, siteID, names); err != nil {
			return fmt.Errorf("insert lead events: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit lead events: %w", err)
	}
	return nil
}
