package sitereport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/toufiqqureshi/uplumo/backend/internal/branding"
	"github.com/toufiqqureshi/uplumo/backend/internal/reports"
	"golang.org/x/sync/singleflight"
)

// generateTimeout bounds the Google reads and the save of one report.
const generateTimeout = 60 * time.Second

// Store reads the stored project, audit and backlink rows.
type Store interface {
	Project(ctx context.Context, organizationID, projectID string) (Project, error)
	LatestAudit(ctx context.Context, projectID, domain string) (*Audit, error)
	LatestBacklinks(ctx context.Context, projectID, domain string) (*Backlinks, error)
}

// SearchSource reads Search Console; ErrNotConnected when the project has none.
type SearchSource interface {
	Search(ctx context.Context, organizationID, projectID string, p Period) (*Search, error)
}

// TrafficSource reads Google Analytics; ErrNotConnected when the project has none.
type TrafficSource interface {
	Traffic(ctx context.Context, projectID string, p Period) (*Traffic, error)
}

// RankSource reads stored rank snapshots; nil when nothing is tracked.
type RankSource interface {
	Rankings(ctx context.Context, projectID, domain string, p Period) (*Rankings, error)
}

// ReportSaver stores the rendered document as a project report.
type ReportSaver interface {
	Save(ctx context.Context, in reports.SaveInput) (reports.SaveResult, error)
	Share(ctx context.Context, projectID, reportID string) (reports.Metadata, error)
}

// Service builds and saves site reports.
type Service struct {
	Store     Store
	Search    SearchSource
	Traffic   TrafficSource
	Rankings  RankSource
	Analytics func(context.Context, string, Period) (*AnalyticsSnapshot, error)
	AI        func(context.Context, string, string) (*AIVisibility, error)
	Local     func(context.Context, string, string, time.Time) (*LocalVisibility, error)
	Reports   ReportSaver
	Branding  func(ctx context.Context, organizationID string) (branding.Branding, bool, error)
	Logger    *slog.Logger
	Now       func() time.Time
	flight    singleflight.Group
}

// Author is the signed-in member who clicked "Create report".
type Author struct{ UserID, Name string }

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Generate collects every chosen section, renders one document and saves it.
// A double click with identical inputs shares one run instead of saving twice.
func (s *Service) Generate(ctx context.Context, organizationID, projectID string, author Author, in Input) (Result, error) {
	if s == nil || s.Store == nil || s.Reports == nil {
		return Result{}, &Error{Code: "reports_unavailable", Message: "Site reports are not available on this server."}
	}
	in, err := normalize(in)
	if err != nil {
		return Result{}, err
	}
	key, err := json.Marshal(struct {
		OrganizationID, ProjectID string
		Author                    Author
		Input                     Input
	}{organizationID, projectID, author, in})
	if err != nil {
		return Result{}, fmt.Errorf("encode report generation key: %w", err)
	}
	out, err, _ := s.flight.Do(string(key), func() (any, error) {
		run, cancel := context.WithTimeout(context.WithoutCancel(ctx), generateTimeout)
		defer cancel()
		return s.generate(run, organizationID, projectID, author, in)
	})
	if err != nil {
		return Result{}, err
	}
	return out.(Result), nil
}

func (s *Service) generate(ctx context.Context, organizationID, projectID string, author Author, in Input) (Result, error) {
	draft, snap, err := s.prepare(ctx, organizationID, projectID, in)
	if err != nil {
		return Result{}, err
	}
	saved, err := s.Reports.Save(ctx, reports.SaveInput{
		ProjectID: projectID, OrganizationID: organizationID, Title: in.Title, Summary: summary(snap),
		HTML: draft.HTML, Skill: new("Site report"), CreatedBy: author.Name, CreatedByUserID: author.UserID,
	})
	if err != nil {
		return Result{}, err
	}
	result := Result{ReportID: saved.ReportID, Title: saved.Title, Included: includedSections(snap), Skipped: draft.Skipped}
	if in.Share {
		meta, err := s.Reports.Share(ctx, projectID, saved.ReportID)
		if err != nil {
			s.log(ctx, "share generated site report", err)
		} else {
			result.ShareToken = meta.ShareToken
		}
	}
	return result, nil
}

// Preview renders a report without saving it or creating a share token.
func (s *Service) Preview(ctx context.Context, organizationID, projectID string, in Input) (Preview, error) {
	if s == nil || s.Store == nil {
		return Preview{}, &Error{Code: "reports_unavailable", Message: "Site reports are not available on this server."}
	}
	in, err := normalize(in)
	if err != nil {
		return Preview{}, err
	}
	run, cancel := context.WithTimeout(ctx, generateTimeout)
	defer cancel()
	draft, _, err := s.prepare(run, organizationID, projectID, in)
	return draft, err
}

func (s *Service) prepare(ctx context.Context, organizationID, projectID string, in Input) (Preview, Snapshot, error) {
	now := s.now()
	period, err := resolvePeriod(in, now)
	if err != nil {
		return Preview{}, Snapshot{}, err
	}
	project, err := s.Store.Project(ctx, organizationID, projectID)
	if err != nil {
		return Preview{}, Snapshot{}, err
	}
	snap := Snapshot{Project: project, Period: period, GeneratedAt: now.UTC(), Note: in.Note,
		WorkCompleted: in.WorkCompleted, NextSteps: in.NextSteps, Accent: s.accent(ctx, organizationID)}
	skipped := s.collect(ctx, organizationID, projectID, in.Sections, &snap)
	if len(includedSections(snap)) == 0 {
		return Preview{}, Snapshot{}, &Error{Code: "no_report_data", Message: noDataMessage(skipped)}
	}
	html, err := Render(snap)
	if err != nil {
		return Preview{}, Snapshot{}, err
	}
	return Preview{HTML: html, Skipped: skipped}, snap, nil
}

// collect fills the chosen sections. Google reads run in parallel; a source
// that fails skips its section instead of failing the whole report.
func (s *Service) collect(ctx context.Context, organizationID, projectID string, sections []Section, snap *Snapshot) []SkippedPart {
	var mu sync.Mutex
	skipped := []SkippedPart{}
	skip := func(section Section, reason string) {
		mu.Lock()
		skipped = append(skipped, SkippedPart{Section: section, Reason: reason})
		mu.Unlock()
	}
	fail := func(section Section, source string, err error) {
		if errors.Is(err, ErrNotConnected) {
			skip(section, source+" is not connected for this project.")
			return
		}
		s.log(ctx, "load site report section", err, "section", string(section))
		skip(section, "Could not load "+source+" data right now. Try again in a few minutes.")
	}
	var wg sync.WaitGroup
	wantRankings := slices.Contains(sections, SectionRankings)
	wantCompetitors := slices.Contains(sections, SectionCompetitors)
	for _, section := range sections {
		switch section {
		case SectionSearch:
			wg.Go(func() {
				if s.Search == nil {
					fail(section, "Google Search Console", ErrNotConnected)
					return
				}
				data, err := s.Search.Search(ctx, organizationID, projectID, snap.Period)
				switch {
				case errors.Is(err, errSearchRangeUnavailable):
					skip(section, "Search Console does not cover the full selected period.")
				case err != nil:
					fail(section, "Google Search Console", err)
				case data.Current.Impressions == 0 && data.Previous.Impressions == 0:
					skip(section, "Google Search Console has no search data for this period yet.")
				default:
					snap.Search = data
				}
			})
		case SectionTraffic:
			wg.Go(func() {
				if s.Traffic == nil {
					fail(section, "Google Analytics", ErrNotConnected)
					return
				}
				data, err := s.Traffic.Traffic(ctx, projectID, snap.Period)
				switch {
				case err != nil:
					fail(section, "Google Analytics", err)
				case data == nil || !data.hasData():
					if data != nil && data.breakdownError != nil {
						fail(section, "Google Analytics", data.breakdownError)
					} else {
						skip(section, "Google Analytics recorded no traffic or key events in this period.")
					}
				default:
					if data.breakdownError != nil {
						s.log(ctx, "load Google Analytics report breakdown", data.breakdownError)
					}
					snap.Traffic = data
				}
			})
		case SectionAnalytics:
			wg.Go(func() {
				var data *AnalyticsSnapshot
				var err error
				if s.Analytics != nil {
					data, err = s.Analytics(ctx, projectID, snap.Period)
				}
				switch {
				case err != nil:
					fail(section, "Uplumo Analytics", err)
				case data == nil:
					skip(section, "Uplumo Analytics has no traffic or tracked activity in this period.")
				default:
					snap.Analytics = data
				}
			})
		case SectionAI:
			wg.Go(func() {
				var data *AIVisibility
				var err error
				if s.AI != nil {
					data, err = s.AI(ctx, organizationID, projectID)
				}
				switch {
				case err != nil:
					fail(section, "AI Visibility", err)
				case data == nil:
					skip(section, "No AI Visibility answers have been saved yet.")
				default:
					snap.AI = data
				}
			})
		case SectionLocal:
			wg.Go(func() {
				var data *LocalVisibility
				var err error
				if s.Local != nil {
					data, err = s.Local(ctx, organizationID, projectID, snap.GeneratedAt)
				}
				switch {
				case err != nil:
					fail(section, "Local Visibility", err)
				case data == nil:
					skip(section, "No Google Maps, profile or reviews checks have been saved yet.")
				default:
					snap.Local = data
				}
			})
		case SectionRankings, SectionCompetitors:
			if wantRankings && section == SectionCompetitors {
				continue
			}
			wg.Go(func() {
				var data *Rankings
				var err error
				if s.Rankings != nil {
					data, err = s.Rankings.Rankings(ctx, projectID, snap.Project.Domain, snap.Period)
				}
				if wantRankings {
					switch {
					case err != nil:
						fail(SectionRankings, "rank tracking", err)
					case data == nil || data.Total == 0:
						skip(SectionRankings, "No keyword has a finished rank check yet. Add keywords in Rank Tracking first.")
					default:
						snap.Rankings = data
					}
				}
				if !wantCompetitors {
					return
				}
				switch {
				case err != nil:
					fail(SectionCompetitors, "rank tracking", err)
				case data == nil || !slices.ContainsFunc(data.Rows, func(row RankRow) bool { return len(row.Competitors) > 0 }):
					skip(SectionCompetitors, "No competitor positions are saved for the tracked keywords yet.")
				default:
					snap.Competitors = data
				}
			})
		case SectionAudit:
			wg.Go(func() {
				data, err := s.Store.LatestAudit(ctx, projectID, snap.Project.Domain)
				switch {
				case err != nil:
					fail(section, "site audit", err)
				case data == nil:
					skip(section, "No site audit has finished for this domain. Run one from Site Audit first.")
				default:
					snap.Audit = data
				}
			})
		case SectionBacklinks:
			wg.Go(func() {
				var data *Backlinks
				var err error
				if snap.Project.Domain != "" {
					data, err = s.Store.LatestBacklinks(ctx, projectID, snap.Project.Domain)
				}
				switch {
				case err != nil:
					fail(section, "backlink", err)
				case data == nil:
					skip(section, "No backlink snapshot is saved yet. Refresh backlinks on the dashboard first.")
				default:
					snap.Backlinks = data
				}
			})
		}
	}
	wg.Wait()
	slices.SortFunc(skipped, func(a, b SkippedPart) int { return order(a.Section) - order(b.Section) })
	return skipped
}

func normalize(in Input) (Input, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Note = strings.TrimSpace(in.Note)
	in.WorkCompleted = strings.TrimSpace(in.WorkCompleted)
	in.NextSteps = strings.TrimSpace(in.NextSteps)
	switch {
	case in.Title == "":
		return in, invalid("Add a report title.")
	case utf8.RuneCountInString(in.Title) > maxTitleChars:
		return in, invalid(fmt.Sprintf("Keep the title to %d characters or fewer.", maxTitleChars))
	case utf8.RuneCountInString(in.Note) > maxNoteChars || utf8.RuneCountInString(in.NextSteps) > maxNoteChars || utf8.RuneCountInString(in.WorkCompleted) > maxNoteChars:
		return in, invalid(fmt.Sprintf("Keep the message, work completed and next steps to %d characters each.", maxNoteChars))
	case in.StartDate == "" && in.EndDate == "" && !slices.Contains(PeriodDays, in.PeriodDays):
		return in, invalid("Choose a preset or enter a custom date range.")
	case len(in.Sections) == 0:
		return in, invalid("Choose at least one section.")
	}
	seen := map[Section]bool{}
	for _, section := range in.Sections {
		if !slices.Contains(AllSections, section) {
			return in, invalid("Choose sections from the list.")
		}
		seen[section] = true
	}
	// A new slice: the caller's (and AllSections) must not be reordered.
	in.Sections = make([]Section, 0, len(seen))
	for _, section := range AllSections {
		if seen[section] {
			in.Sections = append(in.Sections, section)
		}
	}
	return in, nil
}

// periodFor ends the window on Search Console's last final day, so search and
// analytics numbers cover the same days.
func periodFor(now time.Time, days int) Period {
	today := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	end := today.AddDate(0, 0, -gscDelayDays)
	return makePeriod(end.AddDate(0, 0, -(days-1)), end, false)
}

func resolvePeriod(in Input, now time.Time) (Period, error) {
	if in.StartDate == "" && in.EndDate == "" {
		return periodFor(now, in.PeriodDays), nil
	}
	if in.PeriodDays != 0 {
		return Period{}, invalid("Choose either a preset or a custom date range.")
	}
	start, startErr := time.Parse(day, in.StartDate)
	end, endErr := time.Parse(day, in.EndDate)
	if startErr != nil || endErr != nil || start.Format(day) != in.StartDate || end.Format(day) != in.EndDate || start.After(end) {
		return Period{}, invalid("Enter valid From and To dates in order.")
	}
	maxDays := PeriodDays[len(PeriodDays)-1]
	if start.AddDate(0, 0, maxDays-1).Before(end) {
		return Period{}, invalid(fmt.Sprintf("Choose a date range of up to %d days.", maxDays))
	}
	latest := periodFor(now, 1).End
	if end.After(latest) {
		return Period{}, invalid("Choose a To date on or before " + latest.Format("2 Jan 2006") + ", when the search data is available.")
	}
	return makePeriod(start, end, true), nil
}

func makePeriod(start, end time.Time, custom bool) Period {
	days := int(end.Sub(start)/(24*time.Hour)) + 1
	prevEnd := start.AddDate(0, 0, -1)
	return Period{Days: days, Custom: custom, Start: start, End: end, PrevStart: prevEnd.AddDate(0, 0, -(days - 1)), PrevEnd: prevEnd}
}

func includedSections(s Snapshot) []Section {
	present := map[Section]bool{
		SectionSearch: s.Search != nil, SectionTraffic: s.Traffic != nil, SectionRankings: s.Rankings != nil,
		SectionAudit: s.Audit != nil, SectionBacklinks: s.Backlinks != nil,
		SectionAnalytics: s.Analytics != nil, SectionCompetitors: s.Competitors != nil,
		SectionAI: s.AI != nil, SectionLocal: s.Local != nil,
	}
	out := []Section{}
	for _, section := range AllSections {
		if present[section] {
			out = append(out, section)
		}
	}
	return out
}

func noDataMessage(skipped []SkippedPart) string {
	reasons := make([]string, 0, len(skipped))
	for _, part := range skipped {
		reasons = append(reasons, part.Reason)
	}
	return "None of the chosen sections has data to report yet. " + strings.Join(reasons, " ")
}

func order(section Section) int { return slices.Index(AllSections, section) }

func (s *Service) accent(ctx context.Context, organizationID string) string {
	if s.Branding == nil {
		return ""
	}
	b, ok, err := s.Branding(ctx, organizationID)
	if err != nil {
		s.log(ctx, "load report branding", err)
		return ""
	}
	if ok && hexColor.MatchString(b.AccentColor) {
		return b.AccentColor
	}
	return ""
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) log(ctx context.Context, msg string, err error, args ...any) {
	if s.Logger != nil && ctx.Err() == nil {
		s.Logger.ErrorContext(ctx, msg, append([]any{"err", err}, args...)...)
	}
}
