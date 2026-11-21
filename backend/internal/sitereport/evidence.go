package sitereport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/aisearch"
	"github.com/toufiqqureshi/uplumo/backend/internal/analytics"
	"github.com/toufiqqureshi/uplumo/backend/internal/localvisibility"
)

const evidenceRows = 12

// AnalyticsSnapshot contains only aggregates already shown in Uplumo Analytics.
type AnalyticsSnapshot struct {
	Visitors, PreviousVisitors, Sessions, Pageviews int64
	Series                                          []analytics.DayCount
	Channels                                        []analytics.AcquisitionCount
	Entries                                         []analytics.EntryExitCount
	Referrers                                       []analytics.ReferrerCount
	Events                                          []analytics.EventCount
	LeadActions                                     []analytics.EventCount
	AILandings                                      []analytics.AILanding
	LeadsConfigured                                 bool
}

// UplumoAnalytics reads the same project aggregates as the Analytics screen.
type UplumoAnalytics struct{ Service *analytics.Service }

func (a UplumoAnalytics) Read(ctx context.Context, projectID string, p Period) (*AnalyticsSnapshot, error) {
	data, err := a.Service.Summarize(ctx, projectID, p.Start, p.End)
	if err != nil {
		return nil, err
	}
	if data.Pageviews == 0 && data.Sessions == 0 && data.PreviousVisitors == 0 && len(data.Events) == 0 {
		return nil, nil
	}
	out := &AnalyticsSnapshot{
		Visitors: data.Visitors, PreviousVisitors: data.PreviousVisitors, Sessions: data.Sessions, Pageviews: data.Pageviews,
		Series: data.Series, Channels: firstN(data.AcquisitionChannels, evidenceRows),
		Entries: firstN(data.Entries, evidenceRows), Referrers: firstN(data.Referrers, evidenceRows),
		Events: firstN(data.Events, evidenceRows), LeadActions: firstN(data.Leads.Actions, evidenceRows),
		AILandings: firstN(data.AILandings, evidenceRows), LeadsConfigured: data.Leads.Configured,
	}
	for i := range out.Entries {
		out.Entries[i].Path = reportPath(out.Entries[i].Path)
	}
	for i := range out.AILandings {
		out.AILandings[i].Path = reportPath(out.AILandings[i].Path)
	}
	return out, nil
}

// AIVisibility is one saved dashboard run, with answer-level source evidence.
type AIVisibility struct {
	CheckedAt       time.Time
	Brand           string
	Answered, Cited int
	Evidence        []AIAnswer
}

// AIAnswer shows a saved response excerpt and its cited pages.
type AIAnswer struct {
	Prompt, Engine, Excerpt string
	CheckedAt               *time.Time
	Sources                 []AISource
	SourceCount             int
}

// AISource identifies a cited page at its position in the saved source list.
type AISource struct {
	Position           int
	Brand, Domain, URL string
	Owned              bool
}

// AITracking reads evidence from completed AI visibility checks.
type AITracking struct{ Service *aisearch.Tracking }

func (a AITracking) Read(ctx context.Context, org, projectID string) (*AIVisibility, error) {
	state, err := a.Service.State(ctx, org, projectID)
	if err != nil {
		return nil, err
	}
	for _, run := range state.Runs {
		if (run.Status != "completed" && run.Status != "partial") || run.Answered == 0 {
			continue
		}
		answers, err := a.Service.Answers(ctx, org, projectID, run.ID)
		if err != nil {
			return nil, err
		}
		return citationEvidence(run, answers), nil
	}
	return nil, nil
}

func citationEvidence(run aisearch.TrackingRun, answers []aisearch.TrackingAnswer) *AIVisibility {
	out := &AIVisibility{CheckedAt: run.CreatedAt, Brand: run.Configuration.Brand, Answered: run.Answered, Cited: run.Cited}
	if run.FinishedAt != nil {
		out.CheckedAt = *run.FinishedAt
	}
	for _, answer := range answers {
		if answer.Status != "completed" || strings.TrimSpace(answer.Answer) == "" {
			continue
		}
		row := AIAnswer{Prompt: answer.Prompt, Engine: engineName(answer.Engine), Excerpt: excerpt(answer.Answer, 600), CheckedAt: answer.CollectedAt}
		for i, citation := range answer.Citations {
			if safeHref(citation.URL) == "" {
				continue
			}
			domain := evidenceDomain(citation.URL)
			owned := citation.MatchedBrand || matchesDomain(domain, run.Configuration.Domain)
			source := AISource{Position: i + 1, Domain: domain, URL: citation.URL, Owned: owned}
			if owned {
				source.Brand = run.Configuration.Brand
			} else {
				for _, brand := range run.Configuration.Competitors {
					if matchesDomain(domain, brand.Domain) {
						source.Brand = brand.Name
						break
					}
				}
			}
			row.Sources = append(row.Sources, source)
		}
		row.SourceCount = len(row.Sources)
		// Keep tracked domains visible before filling the remaining slots with other sources.
		slices.SortStableFunc(row.Sources, func(a, b AISource) int { return sourcePriority(a) - sourcePriority(b) })
		row.Sources = firstN(row.Sources, 6)
		slices.SortFunc(row.Sources, func(a, b AISource) int { return a.Position - b.Position })
		out.Evidence = append(out.Evidence, row)
	}
	slices.SortStableFunc(out.Evidence, func(a, b AIAnswer) int {
		return answerPriority(a) - answerPriority(b)
	})
	out.Evidence = firstN(out.Evidence, evidenceRows)
	return out
}

func sourcePriority(source AISource) int {
	switch {
	case source.Owned:
		return 0
	case source.Brand != "":
		return 1
	default:
		return 2
	}
}

func answerPriority(answer AIAnswer) int {
	for _, source := range answer.Sources {
		if source.Owned {
			return 0
		}
	}
	if len(answer.Sources) > 0 {
		return 1
	}
	return 2
}

func engineName(engine string) string {
	switch engine {
	case "chat_gpt":
		return "ChatGPT"
	case "google_ai_overview":
		return "Google AI Overviews"
	case "gemini":
		return "Gemini"
	case "claude":
		return "Claude"
	case "perplexity":
		return "Perplexity"
	default:
		return strings.ReplaceAll(engine, "_", " ")
	}
}

func evidenceDomain(raw string) string {
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(parsed.Hostname()), "www.")
}

func matchesDomain(host, domain string) bool {
	domain = evidenceDomain(domain)
	return host != "" && domain != "" && (host == domain || strings.HasSuffix(host, "."+domain))
}

func reportPath(path string) string {
	if end := strings.IndexAny(path, "?#"); end >= 0 {
		return path[:end]
	}
	return path
}

func excerpt(value string, limit int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return strings.TrimSpace(string(runes[:limit])) + "…"
}

// LocalVisibility contains finished Google Maps, profile and review checks.
type LocalVisibility struct {
	Business                   *localvisibility.Business
	Check                      *localvisibility.CheckResult
	Grid                       *localvisibility.GridResult
	Reviews                    *localvisibility.ReviewsResult
	CheckAt, GridAt, ReviewsAt time.Time
	PreviousReviews            *localvisibility.ReviewsResult
	PreviousReviewsAt          time.Time
}

// LocalSnapshots reads saved checks for the project's current business listing.
type LocalSnapshots struct {
	Store localvisibility.Store
	Plans localvisibility.PaidPlans
}

func (s LocalSnapshots) Read(ctx context.Context, orgID, projectID string, now time.Time) (*LocalVisibility, error) {
	if s.Plans != nil {
		available, err := s.Plans.HasPaidPlan(ctx, orgID)
		if err != nil || !available {
			return nil, err
		}
	}
	business, err := s.Store.Business(ctx, projectID)
	if err != nil || business == nil {
		return nil, err
	}
	out := &LocalVisibility{Business: business}
	var reviewRun *localvisibility.Run
	for _, part := range []struct {
		kind   string
		result any
		at     *time.Time
	}{
		{localvisibility.KindCheck, &out.Check, &out.CheckAt},
		{localvisibility.KindGrid, &out.Grid, &out.GridAt},
		{localvisibility.KindReviews, &out.Reviews, &out.ReviewsAt},
	} {
		run, err := s.Store.LatestBusinessCheck(ctx, orgID, projectID, business.CID, part.kind, now)
		if err != nil {
			return nil, err
		}
		if run == nil {
			continue
		}
		if err := decodeLocalCheck(run, part.result, part.at); err != nil {
			return nil, err
		}
		if part.kind == localvisibility.KindReviews {
			reviewRun = run
		}
	}
	if reviewRun != nil {
		previous, err := s.Store.LatestBusinessCheck(ctx, orgID, projectID, business.CID, localvisibility.KindReviews, reviewRun.CreatedAt)
		if err != nil {
			return nil, err
		}
		if previous != nil {
			if err := decodeLocalCheck(previous, &out.PreviousReviews, &out.PreviousReviewsAt); err != nil {
				return nil, err
			}
		}
	}
	if out.Check != nil && !out.Check.MapsAvailable() && out.Check.You.Profile == nil {
		out.Check = nil
	}
	if out.Check == nil && out.Grid == nil && out.Reviews == nil {
		return nil, nil
	}
	return out, nil
}

func decodeLocalCheck(run *localvisibility.Run, result any, at *time.Time) error {
	if err := json.Unmarshal(run.Result, result); err != nil {
		return fmt.Errorf("decode local %s result: %w", run.Kind, err)
	}
	*at = run.CreatedAt
	if run.FinishedAt != nil {
		*at = *run.FinishedAt
	}
	return nil
}
