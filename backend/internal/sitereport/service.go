package sitereport

import (
	"context"
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
	LatestAudit(ctx context.Context, projectID string) (*Audit, error)
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
	Store    Store
	Search   SearchSource
	Traffic  TrafficSource
	Rankings RankSource
	Reports  ReportSaver
	Branding func(ctx context.Context, organizationID string) (branding.Branding, bool, error)
	Logger   *slog.Logger
	Now      func() time.Time
	flight   singleflight.Group
}

// Author is the signed-in member who clicked "Create report".
type Author struct{ UserID, Name string }

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Generate collects every chosen section, renders one document and saves it.
// A double click with the same title shares one run instead of saving twice.
func (s *Service) Generate(ctx context.Context, organizationID, projectID string, author Author, in Input) (Result, error) {
	if s == nil || s.Store == nil || s.Reports == nil {
		return Result{}, &Error{Code: "reports_unavailable", Message: "Site reports are not available on this server."}
	}
	in, err := normalize(in)
	if err != nil {
		return Result{}, err
	}
	key := organizationID + "\x00" + projectID + "\x00" + strings.ToLower(in.Title)
	out, err, _ := s.flight.Do(key, func() (any, error) {
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
	project, err := s.Store.Project(ctx, organizationID, projectID)
	if err != nil {
		return Result{}, err
	}
	snap := Snapshot{Project: project, Period: periodFor(s.now(), in.PeriodDays), GeneratedAt: s.now().UTC(), Note: in.Note, NextSteps: in.NextSteps, Accent: s.accent(ctx, organizationID)}
	skipped := s.collect(ctx, organizationID, projectID, in.Sections, &snap)
	included := includedSections(snap)
	if len(included) == 0 {
		return Result{}, &Error{Code: "no_report_data", Message: noDataMessage(skipped)}
	}
	html, err := Render(snap)
	if err != nil {
		return Result{}, err
	}
	saved, err := s.Reports.Save(ctx, reports.SaveInput{
		ProjectID: projectID, OrganizationID: organizationID, Title: in.Title, Summary: summary(snap),
		HTML: html, Skill: new("Site report"), CreatedBy: author.Name, CreatedByUserID: author.UserID,
	})
	if err != nil {
		return Result{}, err
	}
	result := Result{ReportID: saved.ReportID, Title: saved.Title, Included: included, Skipped: skipped}
	if in.Share {
		meta, err := s.Reports.Share(ctx, projectID, saved.ReportID)
		if err != nil {
			// The report is saved; the client link can still be created from
			// the report page, so this is not a failed click.
			s.log(ctx, "share generated site report", err)
		} else {
			result.ShareToken = meta.ShareToken
		}
	}
	return result, nil
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
				case data.Sessions.Value == nil || *data.Sessions.Value == 0:
					skip(section, "Google Analytics recorded no organic search visits in this period.")
				default:
					snap.Traffic = data
				}
			})
		case SectionRankings:
			wg.Go(func() {
				var data *Rankings
				var err error
				if s.Rankings != nil {
					data, err = s.Rankings.Rankings(ctx, projectID, snap.Project.Domain, snap.Period)
				}
				switch {
				case err != nil:
					fail(section, "rank tracking", err)
				case data == nil || data.Total == 0:
					skip(section, "No keyword has a finished rank check yet. Add keywords in Rank Tracking first.")
				default:
					snap.Rankings = data
				}
			})
		case SectionAudit:
			wg.Go(func() {
				data, err := s.Store.LatestAudit(ctx, projectID)
				switch {
				case err != nil:
					fail(section, "site audit", err)
				case data == nil:
					skip(section, "No site audit has finished yet. Run one from Site Audit first.")
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
	in.NextSteps = strings.TrimSpace(in.NextSteps)
	switch {
	case in.Title == "":
		return in, invalid("Add a report title.")
	case utf8.RuneCountInString(in.Title) > maxTitleChars:
		return in, invalid(fmt.Sprintf("Keep the title to %d characters or fewer.", maxTitleChars))
	case utf8.RuneCountInString(in.Note) > maxNoteChars || utf8.RuneCountInString(in.NextSteps) > maxNoteChars:
		return in, invalid(fmt.Sprintf("Keep the message and next steps to %d characters each.", maxNoteChars))
	case !slices.Contains(PeriodDays, in.PeriodDays):
		return in, invalid("Choose a 7, 28 or 90 day period.")
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
	start := end.AddDate(0, 0, -(days - 1))
	prevEnd := start.AddDate(0, 0, -1)
	return Period{Days: days, Start: start, End: end, PrevStart: prevEnd.AddDate(0, 0, -(days - 1)), PrevEnd: prevEnd}
}

func includedSections(s Snapshot) []Section {
	present := map[Section]bool{
		SectionSearch: s.Search != nil, SectionTraffic: s.Traffic != nil, SectionRankings: s.Rankings != nil,
		SectionAudit: s.Audit != nil, SectionBacklinks: s.Backlinks != nil,
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
