package sitereport

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/toufiqqureshi/uplumo/backend/internal/ga4"
	"github.com/toufiqqureshi/uplumo/backend/internal/gsc"
	"github.com/toufiqqureshi/uplumo/backend/internal/ranktracking"
)

const day = "2006-01-02"

// GSC reads Search Console through the project's own connection. Google does
// not bill these reads, and one report makes at most 10 of them.
type GSC struct{ Service *gsc.Service }

// Search returns the period's totals, trend, top rows and movers.
func (g GSC) Search(ctx context.Context, organizationID, projectID string, p Period) (*Search, error) {
	if g.Service == nil {
		return nil, ErrNotConnected
	}
	start, end := p.Start.Format(day), p.End.Format(day)
	raw, err := g.Service.GetPerformance(ctx, organizationID, projectID, gsc.PerformanceInput{StartDate: start, EndDate: end})
	if err != nil {
		return nil, err
	}
	var perf struct {
		Connected bool `json:"connected"`
		Range     struct {
			ComparisonAvailable bool `json:"comparisonAvailable"`
		} `json:"range"`
		Totals           Totals            `json:"totals"`
		PrevTotals       Totals            `json:"prevTotals"`
		Daily            []SearchDay       `json:"daily"`
		StrikingDistance []Opportunity     `json:"strikingDistance"`
		Countries        []Row             `json:"countries"`
		QueryGains       []gsc.QueryChange `json:"queryGains"`
		QueryLosses      []gsc.QueryChange `json:"queryLosses"`
	}
	if err := roundTrip(raw, &perf); err != nil {
		return nil, fmt.Errorf("decode search performance: %w", err)
	}
	if !perf.Connected {
		return nil, ErrNotConnected
	}
	_, pages, err := g.Service.GetPageRows(ctx, organizationID, projectID, start, end)
	if err != nil {
		return nil, fmt.Errorf("load top pages: %w", err)
	}
	queries, err := g.Service.GetQueryRows(ctx, organizationID, projectID, start, end)
	if err != nil {
		return nil, fmt.Errorf("load top queries: %w", err)
	}
	out := &Search{
		// A previous period with no impressions is a site that was not in
		// Search yet: "was 0" there would read as growth from nothing.
		Current: perf.Totals, Previous: perf.PrevTotals, HasPrevious: perf.Range.ComparisonAvailable && perf.PrevTotals.Impressions > 0,
		Daily: perf.Daily, TopQueries: topRows(queries), TopPages: topRows(pages),
		Gains: changes(perf.QueryGains), Losses: changes(perf.QueryLosses),
		Opportunities: firstN(perf.StrikingDistance, 15), Countries: firstN(perf.Countries, 10),
	}
	if !out.HasPrevious {
		// Every search is "rising" against an empty period: say nothing.
		out.Gains, out.Losses = nil, nil
	}
	out.Daily = fillDays(p, out.Daily, func(d SearchDay) string { return d.Date }, func(date string) SearchDay { return SearchDay{Date: date} })
	return out, nil
}

func topRows(rows []gsc.SearchRow) []Row {
	out := make([]Row, 0, len(rows))
	for _, r := range rows {
		if len(r.Keys) == 0 || r.Keys[0] == "" {
			continue
		}
		out = append(out, Row{Key: r.Keys[0], Clicks: r.Clicks, Impressions: r.Impressions, CTR: r.CTR, Position: r.Position})
	}
	slices.SortStableFunc(out, func(a, b Row) int {
		if c := cmp.Compare(b.Clicks, a.Clicks); c != 0 {
			return c
		}
		return cmp.Compare(b.Impressions, a.Impressions)
	})
	return firstN(out, maxTableRows)
}

func changes(rows []gsc.QueryChange) []Change {
	out := make([]Change, 0, len(rows))
	for _, r := range rows {
		if r.Query == "" {
			continue
		}
		out = append(out, Change{Key: r.Query, Clicks: r.Clicks, PrevClicks: r.PrevClicks, Position: r.Position, PrevPosition: r.PrevPosition})
	}
	return firstN(out, 10)
}

// GA4 reads Google Analytics organic-search traffic through the project's own
// connection: 3 free reads per report.
type GA4 struct{ Service *ga4.Service }

// Traffic returns organic sessions, users, engagement and key events.
func (g GA4) Traffic(ctx context.Context, projectID string, p Period) (*Traffic, error) {
	if g.Service == nil || g.Service.Connections == nil {
		return nil, ErrNotConnected
	}
	if _, err := g.Service.Connections.GetByProjectID(ctx, projectID); err != nil {
		if errors.Is(err, ga4.ErrConnectionNotFound) {
			return nil, ErrNotConnected
		}
		return nil, fmt.Errorf("load analytics connection: %w", err)
	}
	raw, err := g.Service.GetOrganicOverview(ctx, projectID, ga4.OrganicOverviewInput{StartDate: p.Start.Format(day), EndDate: p.End.Format(day), Trend: "daily"})
	if err != nil {
		return nil, err
	}
	var overview struct {
		Request struct {
			CurrencyCode string `json:"currencyCode"`
		} `json:"request"`
		Current  map[string]*float64 `json:"current"`
		Previous map[string]*float64 `json:"previous"`
		Trend    []map[string]any    `json:"trend"`
	}
	if err := roundTrip(raw, &overview); err != nil {
		return nil, fmt.Errorf("decode analytics overview: %w", err)
	}
	metric := func(name string) Metric {
		return Metric{Value: overview.Current[name], Previous: overview.Previous[name]}
	}
	out := &Traffic{
		Currency: overview.Request.CurrencyCode,
		Sessions: metric("sessions"), Users: metric("activeUsers"), EngagedSessions: metric("engagedSessions"),
		EngagementRate: metric("engagementRate"), KeyEvents: metric("keyEvents"), Revenue: metric("purchaseRevenue"),
	}
	for _, row := range overview.Trend {
		date, _ := row["date"].(string)
		sessions, _ := row["sessions"].(float64)
		if len(date) == 8 {
			date = date[:4] + "-" + date[4:6] + "-" + date[6:]
		}
		if date != "" {
			out.Daily = append(out.Daily, TrafficDay{Date: date, Sessions: sessions})
		}
	}
	out.Daily = fillDays(p, out.Daily, func(d TrafficDay) string { return d.Date }, func(date string) TrafficDay { return TrafficDay{Date: date} })
	return out, nil
}

// RankTracking reads stored rank snapshots; it never starts a paid check.
type RankTracking struct{ Service *ranktracking.Service }

// Rankings returns the tracked keywords of the config for the project
// domain, or nil when nothing is tracked yet.
func (r RankTracking) Rankings(ctx context.Context, projectID, domain string, p Period) (*Rankings, error) {
	if r.Service == nil {
		return nil, nil
	}
	configs, err := r.Service.ListConfigSummaries(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list rank tracking configs: %w", err)
	}
	var chosen *ranktracking.ConfigSummary
	for i := range configs {
		c := &configs[i]
		if c.KeywordCount == 0 || c.LastRunCompletedAt == nil {
			continue
		}
		if chosen == nil || strings.EqualFold(c.Domain, domain) && !strings.EqualFold(chosen.Domain, domain) {
			chosen = c
		}
	}
	if chosen == nil {
		return nil, nil
	}
	compare := "30d"
	switch {
	case p.Days <= 7:
		compare = "7d"
	case p.Days > 30:
		compare = "90d"
	}
	latest, err := r.Service.LatestResults(ctx, projectID, chosen.ID, compare)
	if err != nil {
		return nil, fmt.Errorf("load rank results: %w", err)
	}
	device := "desktop"
	if chosen.Devices == "mobile" {
		device = "mobile"
	}
	out := &Rankings{Domain: chosen.Domain, Device: device}
	if chosen.LocationName != nil {
		out.Location = *chosen.LocationName
	}
	if latest.Run != nil {
		out.CheckedAt = latest.Run.LastCheckedAt
	}
	for _, row := range latest.Rows {
		result := row.Desktop
		if device == "mobile" {
			result = row.Mobile
		}
		item := RankRow{Keyword: row.Keyword, Position: result.Position, Previous: result.PreviousPosition, Volume: row.SearchVolume}
		if result.RankingURL != nil {
			item.URL = *result.RankingURL
		}
		out.Rows = append(out.Rows, item)
	}
	summarizeRanks(out)
	return out, nil
}

func summarizeRanks(r *Rankings) {
	r.Total = len(r.Rows)
	for _, row := range r.Rows {
		switch {
		case row.Position == nil:
			r.NotRanking++
		case *row.Position <= 3:
			r.Top3++
			r.Top10++
			r.Top20++
		case *row.Position <= 10:
			r.Top10++
			r.Top20++
		case *row.Position <= 20:
			r.Top20++
		}
		switch change := row.Change(); {
		case change > 0:
			r.Improved++
		case change < 0, row.Position == nil && row.Previous != nil:
			r.Declined++
		}
	}
	slices.SortStableFunc(r.Rows, func(a, b RankRow) int {
		switch {
		case a.Position == nil && b.Position == nil:
			return strings.Compare(a.Keyword, b.Keyword)
		case a.Position == nil:
			return 1
		case b.Position == nil:
			return -1
		}
		return cmp.Compare(*a.Position, *b.Position)
	})
}

func firstN[T any](rows []T, n int) []T {
	if len(rows) > n {
		return rows[:n]
	}
	return rows
}

func roundTrip(in any, out any) error {
	raw, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

// fillDays returns one row per day of the period, in date order. Google
// leaves out days with no data; without them a chart would draw 21 Oct and
// 30 Oct one step apart.
func fillDays[T any](p Period, rows []T, date func(T) string, empty func(string) T) []T {
	byDate := make(map[string]T, len(rows))
	for _, row := range rows {
		byDate[date(row)] = row
	}
	out := make([]T, 0, p.Days)
	for d := p.Start; !d.After(p.End); d = d.AddDate(0, 0, 1) {
		key := d.Format(day)
		if row, ok := byDate[key]; ok {
			out = append(out, row)
		} else {
			out = append(out, empty(key))
		}
	}
	return out
}
