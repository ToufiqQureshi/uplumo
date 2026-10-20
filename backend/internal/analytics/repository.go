package analytics

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// topLimit bounds the ranked lists in a summary.
const topLimit = 50

type repository struct {
	db *pgxpool.Pool
}

// ensureSite returns the site key of projectID's site, creating the site
// with newKey when there is none. The no-op update makes RETURNING yield the
// existing row on conflict, so concurrent callers all get the same key.
func (r repository) ensureSite(ctx context.Context, projectID, newKey string) (string, error) {
	var key string
	err := r.db.QueryRow(ctx, `
		INSERT INTO go_analytics_sites (project_id, site_key) VALUES ($1, $2)
		ON CONFLICT (project_id) DO UPDATE SET project_id = EXCLUDED.project_id
		RETURNING site_key`,
		projectID, newKey,
	).Scan(&key)
	if err != nil {
		return "", fmt.Errorf("ensure analytics site: %w", err)
	}
	return key, nil
}

func (r repository) siteIDByProject(ctx context.Context, projectID string) (int64, error) {
	var id int64
	if err := r.db.QueryRow(ctx, `SELECT id FROM go_analytics_sites WHERE project_id = $1`, projectID).Scan(&id); err != nil {
		return 0, fmt.Errorf("load analytics site id: %w", err)
	}
	return id, nil
}

// site is a tracked site and its project's domain ("" when unset).
type site struct {
	ID     int64
	Domain string
}

// siteByKey returns the site with siteKey whose project is active, or
// ErrUnknownSite.
func (r repository) siteByKey(ctx context.Context, siteKey string) (site, error) {
	var s site
	err := r.db.QueryRow(ctx, `
		SELECT s.id, coalesce(p.domain, '')
		FROM go_analytics_sites s
		JOIN projects p ON p.id = s.project_id
		WHERE s.site_key = $1 AND p.archived_at IS NULL`,
		siteKey,
	).Scan(&s.ID, &s.Domain)
	if errors.Is(err, pgx.ErrNoRows) {
		return site{}, ErrUnknownSite
	}
	if err != nil {
		return site{}, fmt.Errorf("load analytics site: %w", err)
	}
	return s, nil
}

type event struct {
	SiteID       int64
	OccurredAt   time.Time
	Path         string
	Source       source
	Campaign     campaign
	Device       string
	VisitorHash  []byte
	SessionHash  []byte
	PageviewHash []byte
	EngagementMS int64
	CountryCode  string
}

func (r repository) insertEvent(ctx context.Context, e event) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO go_analytics_events
			(site_id, occurred_at, path, referrer_host, channel, ai_source, device, visitor_hash,
			 country_code, utm_source, utm_medium, utm_campaign, utm_term, utm_content,
			 pageview_hash, session_hash, engaged_ms)
		VALUES ($1, $2, $3, nullif($4, ''), $5, nullif($6, ''), $7, $8, nullif($9, ''),
			nullif($10, ''), nullif($11, ''), nullif($12, ''), nullif($13, ''), nullif($14, ''),
			$15, $16, $17)
		ON CONFLICT (site_id, pageview_hash) WHERE pageview_hash IS NOT NULL
		DO UPDATE SET engaged_ms = GREATEST(go_analytics_events.engaged_ms, EXCLUDED.engaged_ms)`,
		e.SiteID, e.OccurredAt, e.Path, e.Source.ReferrerHost, e.Source.Channel,
		e.Source.AISource, e.Device, e.VisitorHash, e.CountryCode,
		e.Campaign.Source, e.Campaign.Medium, e.Campaign.Name,
		e.Campaign.Term, e.Campaign.Content, e.PageviewHash, e.SessionHash, e.EngagementMS,
	)
	if err != nil {
		return fmt.Errorf("insert analytics event: %w", err)
	}
	return nil
}

func (r repository) insertCustomEvent(ctx context.Context, siteID int64, at time.Time, name, path string, visitor, sessionHash, eventHash []byte) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO go_analytics_custom_events (site_id, occurred_at, name, path, visitor_hash, session_hash, event_hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (site_id, event_hash) WHERE event_hash IS NOT NULL DO NOTHING`,
		siteID, at, name, path, visitor, sessionHash, eventHash)
	if err != nil {
		return fmt.Errorf("insert custom event: %w", err)
	}
	return nil
}

// eventsInRange selects the project's events from $2 (inclusive) to $3
// (exclusive). A project without a site matches no events.
const eventsInRange = `
	FROM go_analytics_events
	WHERE site_id = (SELECT id FROM go_analytics_sites WHERE project_id = $1)
		AND occurred_at >= $2 AND occurred_at < $3`

// summary aggregates the project's events in [start, end), whole UTC days,
// with every query sent in one round trip.
func (r repository) summary(ctx context.Context, projectID string, start, end time.Time) (Summary, error) {
	s := Summary{}
	args := []any{projectID, start, end}
	b := &pgx.Batch{}

	b.Queue(`SELECT count(DISTINCT visitor_hash), count(*)`+eventsInRange, args...).
		QueryRow(func(row pgx.Row) error { return row.Scan(&s.Visitors, &s.Pageviews) })

	b.Queue(`
		WITH daily AS (
			SELECT (occurred_at AT TIME ZONE 'UTC')::date AS day,
				count(DISTINCT visitor_hash) AS visitors, count(*) AS pageviews`+eventsInRange+`
			GROUP BY 1
		)
		SELECT to_char(d, 'YYYY-MM-DD'), coalesce(daily.visitors, 0), coalesce(daily.pageviews, 0)
		FROM generate_series($2 AT TIME ZONE 'UTC', $3 AT TIME ZONE 'UTC' - interval '1 day', interval '1 day') AS d
		LEFT JOIN daily ON daily.day = d::date
		ORDER BY d`, args...).
		Query(collectInto(&s.Series, pgx.RowToStructByPos[DayCount]))

	b.Queue(`SELECT path, count(*), coalesce((avg(engaged_ms) FILTER (WHERE session_hash IS NOT NULL))::bigint, 0)`+eventsInRange+`
		GROUP BY path ORDER BY 2 DESC, path LIMIT $4`, append(args, topLimit)...).
		Query(collectInto(&s.TopPages, pgx.RowToStructByPos[PageCount]))

	b.Queue(`SELECT channel, count(DISTINCT visitor_hash)`+eventsInRange+`
		AND channel <> $4
		GROUP BY channel ORDER BY 2 DESC, channel`, append(args, channelInternal)...).
		Query(collectInto(&s.Channels, pgx.RowToStructByPos[ChannelCount]))

	b.Queue(`SELECT ai_source, count(DISTINCT visitor_hash)`+eventsInRange+`
		AND ai_source IS NOT NULL
		GROUP BY ai_source ORDER BY 2 DESC, ai_source`, args...).
		Query(collectInto(&s.AISources, pgx.RowToStructByPos[AISourceCount]))

	b.Queue(`SELECT referrer_host, count(DISTINCT visitor_hash)`+eventsInRange+`
		AND referrer_host IS NOT NULL
		GROUP BY referrer_host ORDER BY 2 DESC, referrer_host LIMIT $4`, append(args, topLimit)...).
		Query(collectInto(&s.Referrers, pgx.RowToStructByPos[ReferrerCount]))

	b.Queue(`SELECT device, count(DISTINCT visitor_hash)`+eventsInRange+`
		GROUP BY device ORDER BY 2 DESC, device`, args...).
		Query(collectInto(&s.Devices, pgx.RowToStructByPos[DeviceCount]))

	b.Queue(`SELECT name, count(*), count(DISTINCT visitor_hash)
		FROM go_analytics_custom_events
		WHERE site_id = (SELECT id FROM go_analytics_sites WHERE project_id = $1)
			AND occurred_at >= $2 AND occurred_at < $3
		GROUP BY name ORDER BY 2 DESC, name LIMIT $4`, append(args, topLimit)...).
		Query(collectInto(&s.Events, pgx.RowToStructByPos[EventCount]))

	b.Queue(`WITH goal AS (
			SELECT event_name FROM go_analytics_enquiry_goals WHERE project_id = $1
		), conversions AS (
			SELECT DISTINCT session_hash FROM go_analytics_custom_events
			WHERE site_id = (SELECT id FROM go_analytics_sites WHERE project_id = $1)
				AND occurred_at >= $2 AND occurred_at < $3 AND session_hash IS NOT NULL
				AND name = (SELECT event_name FROM goal)
		), first_page AS (
			SELECT DISTINCT ON (e.session_hash) e.session_hash, e.visitor_hash,
				e.utm_source, e.utm_medium, e.utm_campaign, e.utm_term, e.utm_content
			FROM go_analytics_events e
			JOIN (SELECT DISTINCT session_hash`+eventsInRange+`
				AND session_hash IS NOT NULL) in_range USING (session_hash)
			WHERE e.site_id = (SELECT id FROM go_analytics_sites WHERE project_id = $1)
				AND e.session_hash IS NOT NULL
			ORDER BY e.session_hash, e.occurred_at, e.pageview_hash
		)
		SELECT coalesce(f.utm_source, ''), coalesce(f.utm_medium, ''), coalesce(f.utm_campaign, ''),
			coalesce(f.utm_term, ''), coalesce(f.utm_content, ''),
			count(DISTINCT f.visitor_hash), count(DISTINCT f.session_hash),
			count(DISTINCT f.session_hash) FILTER (WHERE c.session_hash IS NOT NULL)
		FROM first_page f
		LEFT JOIN conversions c ON c.session_hash = f.session_hash
		WHERE f.utm_source IS NOT NULL OR f.utm_medium IS NOT NULL OR f.utm_campaign IS NOT NULL OR f.utm_term IS NOT NULL OR f.utm_content IS NOT NULL
		GROUP BY 1, 2, 3, 4, 5 ORDER BY 8 DESC, 1, 2, 3, 4, 5 LIMIT $4`, append(args, topLimit)...).
		Query(collectInto(&s.Campaigns, pgx.RowToStructByPos[CampaignCount]))

	b.Queue(`WITH goal AS (
			SELECT event_name FROM go_analytics_enquiry_goals WHERE project_id = $1
		), conversions AS (
			SELECT DISTINCT session_hash FROM go_analytics_custom_events
			WHERE site_id = (SELECT id FROM go_analytics_sites WHERE project_id = $1)
				AND occurred_at >= $2 AND occurred_at < $3 AND session_hash IS NOT NULL
				AND name = (SELECT event_name FROM goal)
		), first_page AS (
			SELECT DISTINCT ON (e.session_hash) e.session_hash, e.channel
			FROM go_analytics_events e
			JOIN (SELECT DISTINCT session_hash`+eventsInRange+`
				AND session_hash IS NOT NULL) in_range USING (session_hash)
			WHERE e.site_id = (SELECT id FROM go_analytics_sites WHERE project_id = $1)
			ORDER BY e.session_hash, e.occurred_at, e.pageview_hash
		)
		SELECT f.channel, count(*), count(*) FILTER (WHERE c.session_hash IS NOT NULL)
		FROM first_page f LEFT JOIN conversions c USING (session_hash)
		WHERE f.channel <> $4
		GROUP BY f.channel ORDER BY 2 DESC, 1`, append(args, channelInternal)...).
		Query(collectInto(&s.AcquisitionChannels, pgx.RowToStructByPos[AcquisitionCount]))

	b.Queue(`WITH goal AS (
			SELECT event_name FROM go_analytics_enquiry_goals WHERE project_id = $1
		), conversions AS (
			SELECT DISTINCT session_hash FROM go_analytics_custom_events
			WHERE site_id = (SELECT id FROM go_analytics_sites WHERE project_id = $1)
				AND occurred_at >= $2 AND occurred_at < $3 AND session_hash IS NOT NULL
				AND name = (SELECT event_name FROM goal)
		), sessions AS (
			SELECT e.session_hash, count(*) AS pageviews, sum(e.engaged_ms) AS engaged_ms,
				bool_or(c.session_hash IS NOT NULL) AS converted
			FROM go_analytics_events e
			LEFT JOIN conversions c ON c.session_hash = e.session_hash
			WHERE e.site_id = (SELECT id FROM go_analytics_sites WHERE project_id = $1)
				AND e.occurred_at >= $2 AND e.occurred_at < $3 AND e.session_hash IS NOT NULL
			GROUP BY e.session_hash
		)
		SELECT count(*), count(*) FILTER (WHERE engaged_ms >= 10000 OR pageviews >= 2 OR converted),
			coalesce(sum(engaged_ms), 0), coalesce(avg(engaged_ms)::bigint, 0),
			coalesce((SELECT event_name FROM goal), ''),
			count(*) FILTER (WHERE converted)
		FROM sessions`, args...).
		QueryRow(func(row pgx.Row) error {
			return row.Scan(&s.Sessions, &s.EngagedSessions, &s.TotalEngagementMS,
				&s.AvgEngagementMS, &s.GoalEventName, &s.ConversionSessions)
		})

	// Landing and exit pages come from the whole session, like the campaign
	// reports above: a session that crosses the range's edge keeps its real
	// first and last page.
	b.Queue(`WITH in_range AS (
			SELECT DISTINCT session_hash`+eventsInRange+`
				AND session_hash IS NOT NULL
		), first_page AS (
			SELECT DISTINCT ON (e.session_hash) e.session_hash, e.path, e.engaged_ms
			FROM go_analytics_events e JOIN in_range USING (session_hash)
			WHERE e.site_id = (SELECT id FROM go_analytics_sites WHERE project_id = $1)
			ORDER BY e.session_hash, e.occurred_at, e.pageview_hash
		)
		SELECT path, count(*), coalesce(avg(engaged_ms)::bigint, 0)
		FROM first_page GROUP BY path ORDER BY 2 DESC, path LIMIT $4`, append(args, topLimit)...).
		Query(collectInto(&s.Entries, pgx.RowToStructByPos[EntryExitCount]))

	b.Queue(`WITH in_range AS (
			SELECT DISTINCT session_hash`+eventsInRange+`
				AND session_hash IS NOT NULL
		), last_page AS (
			SELECT DISTINCT ON (e.session_hash) e.session_hash, e.path, e.engaged_ms
			FROM go_analytics_events e JOIN in_range USING (session_hash)
			WHERE e.site_id = (SELECT id FROM go_analytics_sites WHERE project_id = $1)
			ORDER BY e.session_hash, e.occurred_at DESC, e.pageview_hash DESC
		)
		SELECT path, count(*), coalesce(avg(engaged_ms)::bigint, 0)
		FROM last_page GROUP BY path ORDER BY 2 DESC, path LIMIT $4`, append(args, topLimit)...).
		Query(collectInto(&s.Exits, pgx.RowToStructByPos[EntryExitCount]))

	if err := r.db.SendBatch(ctx, b).Close(); err != nil {
		return Summary{}, fmt.Errorf("query analytics summary: %w", err)
	}
	if s.Sessions > 0 {
		s.BounceRate = float64(s.Sessions-s.EngagedSessions) / float64(s.Sessions)
		s.ConversionRate = float64(s.ConversionSessions) / float64(s.Sessions)
	}
	return s, nil
}

// collectInto returns a batch callback that collects every row into dst.
func collectInto[T any](dst *[]T, fn pgx.RowToFunc[T]) func(pgx.Rows) error {
	return func(rows pgx.Rows) error {
		var err error
		*dst, err = pgx.CollectRows(rows, fn)
		return err
	}
}

// countries returns the first recorded country per daily visitor hash. The
// membership query keeps a foreign site indistinguishable from a missing one.
func (r repository) countries(ctx context.Context, siteID int64, userID string, start, end time.Time, limit, offset int) ([]CountryCount, error) {
	var allowed bool
	err := r.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM go_analytics_sites s
			JOIN projects p ON p.id = s.project_id
			JOIN member m ON m.organization_id = p.organization_id
			WHERE s.id = $1 AND p.archived_at IS NULL AND m.user_id = $2
		)`, siteID, userID).Scan(&allowed)
	if err != nil {
		return nil, fmt.Errorf("authorize analytics site: %w", err)
	}
	if !allowed {
		return nil, ErrUnknownSite
	}
	rows, err := r.db.Query(ctx, `
		WITH visits AS (
			SELECT DISTINCT ON (visitor_hash) visitor_hash, country_code
			FROM go_analytics_events
			WHERE site_id = $1 AND occurred_at >= $2 AND occurred_at < $3
			ORDER BY visitor_hash, occurred_at
		), counted AS (
			SELECT country_code, count(*) AS visitors FROM visits
			WHERE country_code IS NOT NULL GROUP BY country_code
		)
		SELECT country_code, visitors, (SELECT count(*) FROM visits) AS total
		FROM counted ORDER BY visitors DESC, country_code LIMIT $4 OFFSET $5`,
		siteID, start, end, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query analytics countries: %w", err)
	}
	defer rows.Close()
	result := make([]CountryCount, 0)
	for rows.Next() {
		var row CountryCount
		var total int64
		if err := rows.Scan(&row.Code, &row.Visitors, &total); err != nil {
			return nil, fmt.Errorf("scan analytics country: %w", err)
		}
		row.Pct = float64(row.Visitors) * 100 / float64(total)
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read analytics countries: %w", err)
	}
	return result, nil
}
