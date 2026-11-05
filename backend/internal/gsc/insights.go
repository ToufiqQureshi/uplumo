package gsc

import (
	"context"
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	// insightQueryPageLimit is how many query+page rows one analysis reads.
	// Google allows 25,000 in one request; this keeps the answer near 3 MB.
	insightQueryPageLimit = 10_000
	// insightPageLimit bounds each window's page read.
	insightPageLimit = 5_000
	// insightListCap bounds the page-change lists returned.
	insightListCap = 100
)

// InsightsInput selects the range and filters, like the report, plus the
// owner's brand names. Brand names are never guessed.
type InsightsInput struct {
	PerformanceInput
	BrandTerms []string `json:"brandTerms,omitempty"`
}

// InsightsCoverage says how much of Google's data the analysis read, so the
// page can say plainly when it is not the whole picture.
type InsightsCoverage struct {
	QueryPageRows int  `json:"queryPageRows"`
	Truncated     bool `json:"truncated"`
	// DecayAvailable is false when three windows do not fit in the 16 months
	// Search Console keeps.
	DecayAvailable bool `json:"decayAvailable"`
	// ComparisonAvailable is false when the previous window is out of range.
	ComparisonAvailable bool `json:"comparisonAvailable"`
	BrandTermsUsed      bool `json:"brandTermsUsed"`
}

// Insights is the free analysis of the site's own Search Console data.
type Insights struct {
	Connected       bool             `json:"connected"`
	Range           map[string]any   `json:"range,omitempty"`
	Coverage        InsightsCoverage `json:"coverage"`
	Segments        []Segment        `json:"segments"`
	Actions         []Action         `json:"actions"`
	LosingPages     []PageChange     `json:"losingPages"`
	QueryLosses     []QueryChange    `json:"queryLosses"`
	QueryGains      []QueryChange    `json:"queryGains"`
	Decaying        []Decay          `json:"decaying"`
	CTRGaps         []CTRGap         `json:"ctrGaps"`
	Cannibalization []Cannibalized   `json:"cannibalization"`
	// Totals are how many items each list holds before it is cut to the cap.
	Totals map[string]int `json:"totals"`
}

func (s *Service) insightRules() InsightRules {
	if s.Rules != nil {
		return *s.Rules
	}
	return DefaultInsightRules()
}

func validateBrandTerms(terms []string) error {
	if len(terms) > maxBrandTerms {
		return validationError("Give at most 20 brand names.")
	}
	for _, term := range terms {
		term = strings.TrimSpace(term)
		if utf8.RuneCountInString(term) > maxBrandTermRune {
			return validationError("Each brand name must be at most 100 characters.")
		}
		for _, r := range term {
			if unicode.IsControl(r) {
				return validationError("Brand names cannot contain control characters.")
			}
		}
	}
	return nil
}

func capList[T any](items []T, limit int) []T {
	if len(items) > limit {
		return items[:limit]
	}
	return items
}

// GetInsights reads the project's Search Console data once and finds what is
// worth acting on: segments, losing pages and searches, decaying pages,
// searches clicked less than the site's own pages usually are, and searches
// several pages compete for. It makes only free Search Console reads (at most
// six, in parallel) and never calls a paid provider.
func (s *Service) GetInsights(ctx context.Context, organizationID, projectID string, input InsightsInput) (*Insights, error) {
	if err := validateBrandTerms(input.BrandTerms); err != nil {
		return nil, err
	}
	connection, client, err := s.client(ctx, organizationID, projectID)
	if err != nil {
		if errors.Is(err, ErrConnectionNotFound) {
			return &Insights{Connected: false}, nil
		}
		return nil, err
	}
	rules := s.insightRules()
	now := s.now()
	window, err := resolveDateRange(input.PerformanceInput, now)
	if err != nil {
		return nil, err
	}
	middle, err := previousPeriod(window.StartDate, window.EndDate)
	if err != nil {
		return nil, err
	}
	oldest, err := previousPeriod(middle.StartDate, middle.EndDate)
	if err != nil {
		return nil, err
	}
	floor := subtractUTCMonths(now, 16).Format("2006-01-02")
	comparison := middle.StartDate >= floor
	decay := comparison && oldest.StartDate >= floor

	in := func(w dateWindow) PerformanceInput {
		p := input.PerformanceInput
		p.StartDate, p.EndDate, p.DateRange = w.StartDate, w.EndDate, ""
		return p
	}
	var requests []SearchRequest
	add := func(w dateWindow, dimensions []string, limit int) (int, error) {
		request, e := BuildSearchAnalyticsRequest(in(w), dimensions, limit, 0, now)
		if e != nil {
			return 0, e
		}
		requests = append(requests, request)
		return len(requests) - 1, nil
	}
	queryPagesAt, err := add(window, []string{"query", "page"}, insightQueryPageLimit)
	if err != nil {
		return nil, err
	}
	var pagesNowAt, pagesMiddleAt, pagesOldestAt, queriesNowAt, queriesMiddleAt int
	if comparison {
		for _, step := range []struct {
			at         *int
			w          dateWindow
			dimensions []string
		}{
			{&pagesNowAt, window, []string{"page"}},
			{&pagesMiddleAt, middle, []string{"page"}},
			{&queriesNowAt, window, []string{"query"}},
			{&queriesMiddleAt, middle, []string{"query"}},
		} {
			if *step.at, err = add(step.w, step.dimensions, insightPageLimit); err != nil {
				return nil, err
			}
		}
	}
	if decay {
		if pagesOldestAt, err = add(oldest, []string{"page"}, insightPageLimit); err != nil {
			return nil, err
		}
	}
	rows, err := s.fetchSearchAnalyticsRows(ctx, client, connection.SiteURL, requests...)
	if err != nil {
		mapped := mapProviderError(err)
		if expectedGrantFailure(mapped) {
			return &Insights{Connected: false}, nil
		}
		return nil, mapped
	}

	terms := normalizeBrandTerms(input.BrandTerms)
	stats := aggregateQueries(rows[queryPagesAt])
	out := &Insights{
		Connected: true,
		Range:     map[string]any{"startDate": window.StartDate, "endDate": window.EndDate, "prevStartDate": middle.StartDate, "prevEndDate": middle.EndDate},
		Coverage: InsightsCoverage{
			QueryPageRows:       len(rows[queryPagesAt]),
			Truncated:           len(rows[queryPagesAt]) >= insightQueryPageLimit,
			DecayAvailable:      decay,
			ComparisonAvailable: comparison,
			BrandTermsUsed:      len(terms) > 0,
		},
		Segments:    buildSegments(stats, terms, rules),
		LosingPages: []PageChange{}, QueryLosses: []QueryChange{}, QueryGains: []QueryChange{}, Decaying: []Decay{},
	}
	gaps := findCTRGaps(stats, terms, rules)
	cannibal := findCannibalization(stats, terms, rules)
	var losing []PageChange
	var decaying []Decay
	if comparison {
		prevClicks := sumSearchTotals(rows[pagesMiddleAt]).Clicks
		losing = buildPageChanges(rows[pagesNowAt], rows[pagesMiddleAt], prevClicks, insightListCap)
		lost, won := buildChanges(rows[queriesNowAt], rows[queriesMiddleAt], prevClicks, insightListCap, insightListCap)
		for _, c := range lost {
			out.QueryLosses = append(out.QueryLosses, QueryChange{Query: c.key, ChangeMetrics: c.ChangeMetrics})
		}
		for _, c := range won {
			out.QueryGains = append(out.QueryGains, QueryChange{Query: c.key, ChangeMetrics: c.ChangeMetrics})
		}
	}
	if decay {
		decaying = findDecay(rows[pagesOldestAt], rows[pagesMiddleAt], rows[pagesNowAt], rules)
	}
	out.Totals = map[string]int{"decaying": len(decaying), "ctrGaps": len(gaps), "cannibalization": len(cannibal)}
	out.Actions = pickActions(losing, decaying, gaps, cannibal)
	out.LosingPages = append(out.LosingPages, losing...)
	out.Decaying = append(out.Decaying, capList(decaying, rules.ListLimit)...)
	out.CTRGaps = capList(gaps, rules.ListLimit)
	out.Cannibalization = capList(cannibal, rules.ListLimit)
	return out, nil
}
