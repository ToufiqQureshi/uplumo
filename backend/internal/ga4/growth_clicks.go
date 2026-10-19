package ga4

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/gsc"
)

// GrowthClickQuery compares only returned query evidence; missing is unknown.
type GrowthClickQuery struct {
	Query    string         `json:"query"`
	Current  *GrowthMetrics `json:"current"`
	Previous *GrowthMetrics `json:"previous"`
}

// DiagnoseGrowthClicks uses a trusted report snapshot and two exact-page query
// reads. Ranking, demand and CTR are observations, not confirmed causes.
func (s *Service) DiagnoseGrowthClicks(ctx context.Context, search *gsc.Service, org, project, reportID, page string) (map[string]any, error) {
	report, ok, err := s.Growth.Cached(ctx, org, project, reportID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, invalid("Refresh the analysis before investigating clicks.")
	}
	var selected *GrowthPage
	for i := range report.Pages {
		if report.Pages[i].Page == page {
			selected = &report.Pages[i]
			break
		}
	}
	if selected == nil {
		return nil, invalid("Choose a page from this analysis.")
	}
	if search == nil || search.Connections == nil {
		return nil, reportErr("gsc_unavailable", "Connect Search Console first.")
	}
	scope, err := search.Connections.GetByProjectID(ctx, org, project)
	if err != nil {
		return nil, mapOpportunityGSCErr(err)
	}
	if scope.SiteURL != report.SiteURL {
		return nil, invalid("Search Console property changed. Refresh the analysis.")
	}
	work, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	current, err := search.GetPageQueryRows(work, org, project, report.Range.StartDate, report.Range.EndDate, page)
	if err != nil {
		return nil, mapOpportunityGSCErr(err)
	}
	previous, err := search.GetPageQueryRows(work, org, project, report.PreviousRange.StartDate, report.PreviousRange.EndDate, page)
	if err != nil {
		return nil, mapOpportunityGSCErr(err)
	}
	queries, err := growthClickQueries(current, previous)
	if err != nil {
		return nil, err
	}
	findings := clickFindings(*selected)
	return map[string]any{"page": page, "range": report.Range, "previousRange": report.PreviousRange, "findings": findings, "queries": queries, "truncated": len(current) >= 1000 || len(previous) >= 1000, "message": "Possible explanations from returned Web Search data, not confirmed causes. Query rows can be withheld or truncated; an absent query is unknown. Check country/device and branded query mix in the detailed GSC report. Demand seasonality, competitors and search-result layout need separate evidence; GA4 cannot prove why a Google result was not clicked."}, nil
}

func clickFindings(p GrowthPage) []GrowthFinding {
	out := []GrowthFinding{}
	add := func(kind, title, evidence, action string) {
		out = append(out, GrowthFinding{Kind: kind, Title: title, Evidence: evidence, Action: action, Confidence: "Evidence-backed investigation; cause unverified"})
	}
	if p.Current.Impressions < 100 {
		add("low_visibility", "Few search impressions were recorded", fmt.Sprintf("%.0f impressions and %.0f clicks in this window.", p.Current.Impressions, p.Current.Clicks), "Inspect Google's stored index status, then check query relevance and search demand. Low impressions alone do not prove an indexing problem.")
	}
	if p.Current.Impressions >= 100 && p.Current.Position > 10 {
		add("weak_position", "Most visibility may be below the first results", fmt.Sprintf("Average position %.1f across %.0f impressions. This average mixes queries and devices.", p.Current.Position, p.Current.Impressions), "Review the highest-demand returned queries and their individual positions. Compare intent, useful coverage and internal links before rewriting.")
	}
	if p.Previous != nil {
		old := p.Previous
		if old.Impressions >= 100 && p.Current.Impressions < old.Impressions*.8 {
			add("visibility_drop", "Search visibility has fallen", fmt.Sprintf("Impressions %.0f → %.0f; average position %.1f → %.1f.", old.Impressions, p.Current.Impressions, old.Position, p.Current.Position), "Check affected queries and URL Inspection. Compare seasonal demand before concluding that content or a Google update caused the change.")
		}
		if old.Impressions >= 100 && p.Current.Impressions >= 100 && p.Current.Position >= old.Position+2 {
			add("position_drop", "Average search position moved lower", fmt.Sprintf("Average position %.1f → %.1f.", old.Position, p.Current.Position), "Inspect query-level changes. Ranking loss, new low-ranking queries or a changed country/device mix can all move the average.")
		}
		if old.Clicks >= 10 && old.Impressions >= 100 && p.Current.Impressions >= 100 && p.Current.CTR < old.CTR*.7 {
			add("ctr_drop", "Fewer impressions are becoming clicks", fmt.Sprintf("CTR %.1f%% → %.1f%%; position %.1f → %.1f.", old.CTR*100, p.Current.CTR*100, old.Position, p.Current.Position), "Compare the same queries and devices, then review the title, snippet, intent and current search results. AI features or competing results are hypotheses until checked.")
		}
	}
	if len(out) == 0 {
		add("no_clear_cause", "No clear click-loss pattern is established", fmt.Sprintf("%.0f clicks, %.0f impressions, %.1f%% CTR, average position %.1f.", p.Current.Clicks, p.Current.Impressions, p.Current.CTR*100, p.Current.Position), "Review individual queries and index status. Compare a longer complete window if the sample is small; there is no universal CTR target for every query.")
	}
	return out
}

func growthClickQueries(current, previous []gsc.SearchRow) ([]GrowthClickQuery, error) {
	rows := map[string]*GrowthClickQuery{}
	for period, list := range [][]gsc.SearchRow{current, previous} {
		seen := map[string]bool{}
		for _, r := range list {
			if len(r.Keys) != 1 || !validGrowthRow(r) || seen[r.Keys[0]] {
				return nil, reportErr("ga4_malformed_response", "Search Console returned invalid query evidence.")
			}
			query := r.Keys[0]
			seen[query] = true
			if rows[query] == nil {
				rows[query] = &GrowthClickQuery{Query: query}
			}
			m := growthMetrics(r, nil)
			if period == 0 {
				rows[query].Current = &m
			} else {
				rows[query].Previous = &m
			}
		}
	}
	result := make([]GrowthClickQuery, 0, len(rows))
	for _, row := range rows {
		result = append(result, *row)
	}
	// Show observed losses first, then queries needing more evidence. Do not rank
	// an absent current row as a measured total loss.
	delta := func(q GrowthClickQuery) float64 {
		if q.Current != nil && q.Previous != nil {
			return q.Current.Clicks - q.Previous.Clicks
		}
		return 0
	}
	sort.Slice(result, func(i, j int) bool {
		a, b := delta(result[i]), delta(result[j])
		if a != b {
			return a < b
		}
		return result[i].Query < result[j].Query
	})
	if len(result) > 20 {
		result = result[:20]
	}
	return result, nil
}
