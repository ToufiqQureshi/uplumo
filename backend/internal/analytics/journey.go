package analytics

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/httpx"
)

// AudienceDay counts recognized browsers once per UTC day. Unknown pageviews
// are a coverage measure, not a count of identifiable people.
type AudienceDay struct {
	Date              string `json:"date"`
	NewBrowsers       int64  `json:"newBrowsers"`
	ReturningBrowsers int64  `json:"returningBrowsers"`
	UnknownPageviews  int64  `json:"unknownPageviews"`
}

type destination struct{ Host, Path, Type string }

func clickDestination(raw string, page *url.URL, domain string) destination {
	d := destination{Type: "unknown"}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return d
	}
	d.Host, d.Path, d.Type = normalizeHost(u.Hostname()), u.EscapedPath(), "external"
	if d.Path == "" {
		d.Path = "/"
	}
	if d.Host == normalizeHost(page.Hostname()) || domain != "" && withinDomain(d.Host, domain) {
		d.Type = "internal"
	}
	return d
}

// ClickRow describes click intent and observed same-session internal arrivals.
type ClickRow struct {
	Page        string `json:"page"`
	Name        string `json:"name"`
	Host        string `json:"host"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	Clicks      int64  `json:"clicks"`
	VisitorDays int64  `json:"visitorDays"`
	Arrivals    int64  `json:"arrivals"`
}

// ClickReport is a bounded, project-authorized page of grouped click activity.
type ClickReport struct {
	Rows    []ClickRow `json:"rows"`
	HasMore bool       `json:"hasMore"`
}

// Clicks reads a bounded report; no external destination is requested.
func (s *Service) Clicks(ctx context.Context, project string, from, to time.Time, limit, offset int) (ClickReport, error) {
	start, end := from.UTC().Truncate(24*time.Hour), to.UTC().Truncate(24*time.Hour).AddDate(0, 0, 1)
	if !end.After(start) || end.Sub(start) > maxRangeDays*24*time.Hour || limit < 1 || limit > 100 || offset < 0 || offset > 99900 {
		return ClickReport{}, ErrInvalidRange
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	return s.repo.clicks(ctx, project, start, end, limit, offset)
}

func (r repository) clicks(ctx context.Context, project string, start, end time.Time, limit, offset int) (ClickReport, error) {
	rows, err := r.db.Query(ctx, `
		SELECT c.path, c.name, coalesce(c.destination_host,''), coalesce(c.destination_path,''), c.destination_type,
			count(*), count(DISTINCT c.visitor_hash), count(*) FILTER (WHERE c.destination_type='internal' AND EXISTS (
				SELECT 1 FROM go_analytics_events e
				WHERE e.site_id=c.site_id AND e.session_hash=c.session_hash
					AND e.page_host=c.destination_host AND e.path=c.destination_path
					AND e.pageview_hash <> c.pageview_hash
					AND e.occurred_at >= c.occurred_at AND e.occurred_at <= c.occurred_at + interval '30 minutes'
			))
		FROM go_analytics_custom_events c
		WHERE c.site_id=(SELECT id FROM go_analytics_sites WHERE project_id=$1)
			AND c.occurred_at >= $2 AND c.occurred_at < $3 AND c.is_click
		GROUP BY c.path,c.name,c.destination_host,c.destination_path,c.destination_type
		ORDER BY count(*) DESC,c.path,c.name,c.destination_host NULLS FIRST,c.destination_path NULLS FIRST,c.destination_type
		LIMIT $4 OFFSET $5`, project, start, end, limit+1, offset)
	if err != nil {
		return ClickReport{}, fmt.Errorf("query click destinations: %w", err)
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[ClickRow])
	if err != nil {
		return ClickReport{}, fmt.Errorf("read click destinations: %w", err)
	}
	if items == nil {
		items = []ClickRow{}
	}
	more := len(items) > limit
	if more {
		items = items[:limit]
	}
	return ClickReport{Rows: items, HasMore: more}, nil
}

func clicksHandler(logger *slog.Logger, svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		from, fe := time.Parse(time.DateOnly, q.Get("from"))
		to, te := time.Parse(time.DateOnly, q.Get("to"))
		page, limit, pe := httpx.ParsePageLimit(q)
		if fe != nil || te != nil || pe != nil {
			httpx.WriteError(w, 400, "invalid_request", "Choose valid from/to dates, page 1–1000 and limit 1–100.")
			return
		}
		report, err := svc.Clicks(r.Context(), r.PathValue("projectId"), from, to, limit, (page-1)*limit)
		if errors.Is(err, ErrInvalidRange) {
			httpx.WriteError(w, 400, "invalid_range", "Choose a forward date range of at most 366 days.")
			return
		}
		if err != nil {
			logger.ErrorContext(r.Context(), "analytics clicks", "err", err)
			httpx.WriteError(w, 500, "internal", "Could not load click destinations.")
			return
		}
		httpx.WriteJSON(w, 200, report)
	}
}
