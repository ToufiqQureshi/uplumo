package analytics

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/pgdb"
)

var errEnquiryInput = errors.New("invalid enquiry settings")
var errChangeLimit = errors.New("this project has 100 measurement baselines; remove an old baseline first")

// EnquiryMetric counts visitor-days, not identifiable people or confirmed leads.
type EnquiryMetric struct {
	Visitors     int64   `json:"visitors"`
	GoalVisitors int64   `json:"goalVisitors"`
	Rate         float64 `json:"rate"`
}

// EnquiryPage compares two equal, complete UTC windows on one page.
type EnquiryPage struct {
	Path   string        `json:"path"`
	Before EnquiryMetric `json:"before"`
	After  EnquiryMetric `json:"after"`
	Signal string        `json:"signal"`
}

// EnquiryChange freezes pre-change evidence; it is never rewritten by later traffic.
type EnquiryChange struct {
	ID            string        `json:"id"`
	Path          string        `json:"path"`
	EventName     string        `json:"eventName"`
	Note          string        `json:"note"`
	Baseline      EnquiryMetric `json:"baseline"`
	BaselineStart time.Time     `json:"baselineStart"`
	BaselineEnd   time.Time     `json:"baselineEnd"`
	CreatedAt     time.Time     `json:"createdAt"`
}

// EnquiryReport separates missing instrumentation from observed zero events.
type EnquiryReport struct {
	EventName      string          `json:"eventName"`
	SiteConfigured bool            `json:"siteConfigured"`
	GoalObserved   bool            `json:"goalObserved"`
	BeforeStart    time.Time       `json:"beforeStart"`
	CurrentStart   time.Time       `json:"currentStart"`
	Until          time.Time       `json:"until"`
	Pages          []EnquiryPage   `json:"pages"`
	Changes        []EnquiryChange `json:"changes"`
}

// EnquiryComparison reports a fixed baseline against a complete post-change week.
type EnquiryComparison struct {
	Change     EnquiryChange `json:"change"`
	After      EnquiryMetric `json:"after"`
	AfterStart time.Time     `json:"afterStart"`
	AfterEnd   time.Time     `json:"afterEnd"`
	Ready      bool          `json:"ready"`
}

func validGoal(name string) bool {
	return name != "" && utf8.ValidString(name) && utf8.RuneCountInString(name) <= maxEventName && !strings.ContainsFunc(name, unicode.IsControl)
}
func enquiryPath(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || raw == "" || len(raw) > 2048 || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || u.Host != "" || u.Scheme != "" || strings.ContainsAny(raw, "?#") || u.RawQuery != "" || u.Fragment != "" || strings.ContainsFunc(raw, unicode.IsControl) {
		return "", errEnquiryInput
	}
	return u.EscapedPath(), nil
}
func enquirySignal(before, after EnquiryMetric, observed bool) string {
	if !observed {
		return "verify_tracking"
	}
	if before.Visitors < 30 || after.Visitors < 30 {
		return "low_data"
	}
	traffic := float64(after.Visitors) / float64(before.Visitors)
	if traffic < 0.8 || traffic > 1.2 {
		return "traffic_changed"
	}
	if before.GoalVisitors >= 5 && after.Rate <= before.Rate*0.5 {
		return "enquiry_drop"
	}
	if after.GoalVisitors == 0 {
		return "no_goal_events"
	}
	return "stable"
}

// SaveEnquiryGoal selects an existing or deliberately instrumented custom event.
// No event is collected automatically and ingestion/privacy rules do not change.
func (s *Service) SaveEnquiryGoal(ctx context.Context, project, name string) error {
	name = strings.TrimSpace(name)
	if !validGoal(name) {
		return errEnquiryInput
	}
	_, err := s.repo.db.Exec(ctx, `INSERT INTO go_analytics_enquiry_goals(project_id,event_name) SELECT id,$2 FROM projects WHERE id=$1 AND archived_at IS NULL ON CONFLICT(project_id) DO UPDATE SET event_name=EXCLUDED.event_name,updated_at=now()`, project, name)
	return err
}

// Enquiries reads two weeks of saved first-party evidence without a paid provider.
func (s *Service) Enquiries(ctx context.Context, project string) (EnquiryReport, error) {
	end := s.now().UTC().Truncate(24 * time.Hour)
	out := EnquiryReport{BeforeStart: end.AddDate(0, 0, -14), CurrentStart: end.AddDate(0, 0, -7), Until: end, Pages: []EnquiryPage{}, Changes: []EnquiryChange{}}
	err := s.repo.db.QueryRow(ctx, `SELECT COALESCE((SELECT event_name FROM go_analytics_enquiry_goals WHERE project_id=$1),''),EXISTS(SELECT 1 FROM go_analytics_sites WHERE project_id=$1)`, project).Scan(&out.EventName, &out.SiteConfigured)
	if err != nil {
		return out, err
	}
	if out.EventName != "" {
		if err = s.repo.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM go_analytics_custom_events WHERE site_id=(SELECT id FROM go_analytics_sites WHERE project_id=$1) AND name=$2 AND occurred_at >= $3 AND occurred_at < $4)`, project, out.EventName, end.AddDate(0, 0, -28), end).Scan(&out.GoalObserved); err != nil {
			return out, err
		}
		out.Pages, err = enquiryPages(ctx, s.repo.db, project, out.EventName, "", out.BeforeStart, out.CurrentStart, end)
		if err != nil {
			return out, err
		}
		for i := range out.Pages {
			p := &out.Pages[i]
			p.Signal = enquirySignal(p.Before, p.After, out.GoalObserved)
		}
	}
	out.Changes, err = s.repo.enquiryChanges(ctx, project)
	return out, err
}

type enquiryQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func enquiryPages(ctx context.Context, db enquiryQuerier, project, event, path string, start, split, end time.Time) ([]EnquiryPage, error) {
	rows, err := db.Query(ctx, `WITH visits AS (
 SELECT path,visitor_hash,occurred_at >= $3 AS current FROM go_analytics_events
 WHERE site_id=(SELECT id FROM go_analytics_sites WHERE project_id=$1) AND occurred_at >= $2 AND occurred_at < $4 AND ($6='' OR path=$6)
 ), goals AS (
 SELECT DISTINCT path,visitor_hash,occurred_at >= $3 AS current FROM go_analytics_custom_events
 WHERE site_id=(SELECT id FROM go_analytics_sites WHERE project_id=$1) AND occurred_at >= $2 AND occurred_at < $4 AND name=$5 AND ($6='' OR path=$6)
 ) SELECT v.path,count(DISTINCT v.visitor_hash) FILTER(WHERE NOT v.current),count(DISTINCT g.visitor_hash) FILTER(WHERE NOT v.current),
 count(DISTINCT v.visitor_hash) FILTER(WHERE v.current),count(DISTINCT g.visitor_hash) FILTER(WHERE v.current)
 FROM visits v LEFT JOIN goals g ON g.path=v.path AND g.visitor_hash=v.visitor_hash AND g.current=v.current
 GROUP BY v.path ORDER BY count(DISTINCT v.visitor_hash) DESC,v.path LIMIT 50`, project, start, split, end, event, path)
	if err != nil {
		return nil, fmt.Errorf("read enquiry page evidence: %w", err)
	}
	defer rows.Close()
	out := []EnquiryPage{}
	for rows.Next() {
		var p EnquiryPage
		if err = rows.Scan(&p.Path, &p.Before.Visitors, &p.Before.GoalVisitors, &p.After.Visitors, &p.After.GoalVisitors); err != nil {
			return nil, err
		}
		p.Before.setRate()
		p.After.setRate()
		out = append(out, p)
	}
	return out, rows.Err()
}
func (m *EnquiryMetric) setRate() {
	if m.Visitors > 0 {
		m.Rate = float64(m.GoalVisitors) * 100 / float64(m.Visitors)
	}
}
func (r repository) enquiryChanges(ctx context.Context, project string) ([]EnquiryChange, error) {
	rows, err := r.db.Query(ctx, `SELECT id,path,event_name,note,baseline,baseline_start,baseline_end,created_at FROM go_analytics_enquiry_changes WHERE project_id=$1 ORDER BY created_at DESC LIMIT 20`, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []EnquiryChange{}
	for rows.Next() {
		var c EnquiryChange
		var raw []byte
		if err = rows.Scan(&c.ID, &c.Path, &c.EventName, &c.Note, &raw, &c.BaselineStart, &c.BaselineEnd, &c.CreatedAt); err != nil {
			return nil, err
		}
		if err = json.Unmarshal(raw, &c.Baseline); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// SaveEnquiryChange records the last complete week before a deliberate page edit.
// The project row serializes the cap and request key; replay returns the same id.
func (s *Service) SaveEnquiryChange(ctx context.Context, project, path, note, key string) (string, error) {
	path, err := enquiryPath(strings.TrimSpace(path))
	note = strings.TrimSpace(note)
	if err != nil || note == "" || utf8.RuneCountInString(note) > 500 || len(key) < 8 || len(key) > 128 {
		return "", errEnquiryInput
	}
	id := "eqc_" + rand.Text()
	err = pgdb.InTx(ctx, s.repo.db, func(tx pgx.Tx) error {
		var locked string
		if err := tx.QueryRow(ctx, `SELECT id FROM projects WHERE id=$1 AND archived_at IS NULL FOR UPDATE`, project).Scan(&locked); err != nil {
			return err
		}
		err := tx.QueryRow(ctx, `SELECT id FROM go_analytics_enquiry_changes WHERE project_id=$1 AND request_key=$2`, project, key).Scan(&locked)
		if err == nil {
			id = locked
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var n int
		if err = tx.QueryRow(ctx, `SELECT count(*) FROM go_analytics_enquiry_changes WHERE project_id=$1`, project).Scan(&n); err != nil {
			return err
		}
		if n >= 100 {
			return errChangeLimit
		}
		var event string
		if err = tx.QueryRow(ctx, `SELECT event_name FROM go_analytics_enquiry_goals WHERE project_id=$1`, project).Scan(&event); errors.Is(err, pgx.ErrNoRows) {
			return errEnquiryInput
		} else if err != nil {
			return err
		}
		end := s.now().UTC().Truncate(24 * time.Hour)
		start := end.AddDate(0, 0, -7)
		// Reuse the transaction connection while holding the project lock.
		pages, err := enquiryPages(ctx, tx, project, event, path, start, end, end)
		if err != nil {
			return err
		}
		baseline := EnquiryMetric{}
		if len(pages) > 0 {
			baseline = pages[0].Before
		}
		raw, err := json.Marshal(baseline)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO go_analytics_enquiry_changes(id,project_id,path,event_name,note,baseline,baseline_start,baseline_end,request_key,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, id, project, path, event, note, raw, start, end, key, s.now().UTC())
		return err
	})
	return id, err
}

// CompareEnquiryChange waits for seven complete days after the saved change.
// It reports association only; changing traffic mix can explain differences.
func (s *Service) CompareEnquiryChange(ctx context.Context, project, id string) (EnquiryComparison, error) {
	var out EnquiryComparison
	var raw []byte
	err := s.repo.db.QueryRow(ctx, `SELECT id,path,event_name,note,baseline,baseline_start,baseline_end,created_at FROM go_analytics_enquiry_changes WHERE project_id=$1 AND id=$2`, project, id).Scan(&out.Change.ID, &out.Change.Path, &out.Change.EventName, &out.Change.Note, &raw, &out.Change.BaselineStart, &out.Change.BaselineEnd, &out.Change.CreatedAt)
	if err != nil {
		return out, err
	}
	if err = json.Unmarshal(raw, &out.Change.Baseline); err != nil {
		return out, err
	}
	out.AfterStart = out.Change.CreatedAt.UTC().Truncate(24*time.Hour).AddDate(0, 0, 1)
	out.AfterEnd = out.AfterStart.AddDate(0, 0, 7)
	out.Ready = !s.now().UTC().Before(out.AfterEnd)
	if !out.Ready {
		return out, nil
	}
	pages, err := enquiryPages(ctx, s.repo.db, project, out.Change.EventName, out.Change.Path, out.AfterStart, out.AfterEnd, out.AfterEnd)
	if err != nil {
		return out, err
	}
	if len(pages) > 0 {
		out.After = pages[0].Before
	}
	return out, nil
}

// DeleteEnquiryChange removes only a baseline from the authorized project.
func (s *Service) DeleteEnquiryChange(ctx context.Context, project, id string) error {
	_, err := s.repo.db.Exec(ctx, `DELETE FROM go_analytics_enquiry_changes WHERE project_id=$1 AND id=$2`, project, id)
	return err
}
