package ga4

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/toufiqqureshi/uplumo/backend/internal/gsc"
)

// GrowthInput selects a bounded first-party analysis. Brand filters apply only
// to query evidence: GA4 cannot attribute outcomes to individual GSC queries.
type GrowthInput struct {
	StartDate  string   `json:"startDate,omitempty"`
	EndDate    string   `json:"endDate,omitempty"`
	BrandTerms []string `json:"brandTerms,omitempty"`
}

// GrowthMetrics separates Search Console totals from nullable Analytics outcomes.
type GrowthMetrics struct {
	Clicks         float64  `json:"clicks"`
	Impressions    float64  `json:"impressions"`
	CTR            float64  `json:"ctr"`
	Position       float64  `json:"position"`
	Sessions       *float64 `json:"sessions"`
	KeyEvents      *float64 `json:"keyEvents"`
	KeyEventRate   *float64 `json:"keyEventRate"`
	EngagementRate *float64 `json:"engagementRate"`
	Revenue        *float64 `json:"revenue"`
}

// GrowthQuery carries bounded query evidence without conversion attribution.
type GrowthQuery struct {
	Query          string   `json:"query"`
	Clicks         float64  `json:"clicks"`
	Impressions    float64  `json:"impressions"`
	Position       float64  `json:"position"`
	Branded        bool     `json:"branded"`
	CompetingPages []string `json:"competingPages"`
}

// GrowthFinding explains an observation and a next investigation step.
type GrowthFinding struct {
	Kind       string `json:"kind"`
	Title      string `json:"title"`
	Evidence   string `json:"evidence"`
	Action     string `json:"action"`
	Confidence string `json:"confidence"`
}

// GrowthPage is one page and its source-backed review candidates.
type GrowthPage struct {
	Page     string          `json:"page"`
	Current  GrowthMetrics   `json:"current"`
	Previous *GrowthMetrics  `json:"previous"`
	Queries  []GrowthQuery   `json:"queries"`
	Findings []GrowthFinding `json:"findings"`
	Priority float64         `json:"priority"`
}

// GrowthReport preserves source identity and coverage alongside every finding.
type GrowthReport struct {
	ID            string       `json:"id"`
	GeneratedAt   time.Time    `json:"generatedAt"`
	Range         DateRange    `json:"range"`
	PreviousRange DateRange    `json:"previousRange"`
	SiteURL       string       `json:"siteUrl"`
	PropertyID    string       `json:"propertyId"`
	Currency      string       `json:"currency"`
	TimeZone      string       `json:"timeZone"`
	BrandTerms    []string     `json:"brandTerms"`
	Pages         []GrowthPage `json:"pages"`
	Warnings      []string     `json:"warnings"`
	Limited       bool         `json:"limited"`
	Truncated     bool         `json:"truncated"`
	MatchedPages  int          `json:"matchedPages"`
	GSCPageCount  int          `json:"gscPageCount"`
	GA4PageCount  int          `json:"ga4PageCount"`
}

func validateGrowthInput(in GrowthInput) error {
	if len(in.BrandTerms) > 10 {
		return invalid("Use at most 10 brand terms.")
	}
	for _, term := range in.BrandTerms {
		if strings.TrimSpace(term) == "" || utf8.RuneCountInString(term) > 100 || strings.ContainsRune(term, 0) {
			return invalid("Brand terms must contain 1–100 characters.")
		}
	}
	return nil
}

// GetGrowthReport makes five bounded Google reads on a cache miss; no paid AI
// or SEO provider is invoked. Concurrent identical requests share those reads.
func (s *Service) GetGrowthReport(ctx context.Context, searchConsole *gsc.Service, org, project string, in GrowthInput) (GrowthReport, error) {
	if err := validateGrowthInput(in); err != nil {
		return GrowthReport{}, err
	}
	if s == nil || s.Growth == nil || s.Connections == nil {
		return GrowthReport{}, reportErr("ga4_unavailable", "Search growth reporting is unavailable.")
	}
	if searchConsole == nil || searchConsole.Connections == nil {
		return GrowthReport{}, reportErr("gsc_unavailable", "Connect Search Console to analyze search growth.")
	}
	connection, err := s.Connections.GetByProjectID(ctx, project)
	if err != nil {
		if errors.Is(err, ErrConnectionNotFound) {
			return GrowthReport{}, reportErr("ga4_not_connected", "Connect Google Analytics to analyze search growth.")
		}
		return GrowthReport{}, fmt.Errorf("load growth connection: %w", err)
	}
	dates, err := resolveOpportunityDates(SearchOpportunityInput{StartDate: in.StartDate, EndDate: in.EndDate}, connection.PropertyTimeZone, s.now())
	if err != nil {
		return GrowthReport{}, err
	}
	now := s.now().UTC()
	pacific, zoneErr := time.LoadLocation(searchConsoleTimeZone)
	if zoneErr != nil {
		return GrowthReport{}, zoneErr
	}
	latest := now.In(pacific).AddDate(0, 0, -3).Format("2006-01-02")
	if in.StartDate == "" && dates.EndDate > latest {
		end, _ := parseDate(latest)
		dates = DateRange{StartDate: end.AddDate(0, 0, -27).Format("2006-01-02"), EndDate: latest}
	}
	// Both periods must fit inside GSC's available final-data window.
	previous, err := previousPeriod(dates)
	if err != nil {
		return GrowthReport{}, err
	}
	if dates.EndDate > latest || previous.StartDate < now.AddDate(0, -16, 1).Format("2006-01-02") {
		return GrowthReport{}, invalid("Choose complete dates ending at least three days ago, with both periods inside the last 16 months.")
	}
	start, _ := parseDate(dates.StartDate)
	end, _ := parseDate(dates.EndDate)
	if end.Sub(start) > 89*24*time.Hour {
		return GrowthReport{}, invalid("Analyze up to 90 days at a time.")
	}
	in.BrandTerms = append([]string{}, in.BrandTerms...)
	for i := range in.BrandTerms {
		in.BrandTerms[i] = strings.ToLower(strings.TrimSpace(in.BrandTerms[i]))
	}
	sort.Strings(in.BrandTerms)
	// Re-reading the selected GSC connection prevents serving a cached report
	// after a property/account change. No Google request is made here.
	scope, err := searchConsole.Connections.GetByProjectID(ctx, org, project)
	if err != nil {
		return GrowthReport{}, mapOpportunityGSCErr(err)
	}
	keyBytes, err := json.Marshal([]any{org, project, connection, scope, dates, in.BrandTerms})
	if err != nil {
		return GrowthReport{}, fmt.Errorf("encode growth cache key: %w", err)
	}
	hash := sha256.Sum256(keyBytes)
	cacheKey := hex.EncodeToString(hash[:])
	if cached, ok, err := s.Growth.Cached(ctx, org, project, cacheKey); err != nil {
		return GrowthReport{}, err
	} else if ok {
		return cached, nil
	}
	result, err, _ := s.growthFlights.Do(cacheKey, func() (any, error) {
		if cached, ok, err := s.Growth.Cached(ctx, org, project, cacheKey); err != nil {
			return nil, err
		} else if ok {
			return cached, nil
		}
		work, cancel := context.WithTimeout(ctx, 90*time.Second)
		defer cancel()
		_, currentGSC, err := searchConsole.GetPageRows(work, org, project, dates.StartDate, dates.EndDate)
		if err != nil {
			return nil, mapOpportunityGSCErr(err)
		}
		_, previousGSC, err := searchConsole.GetPageRows(work, org, project, previous.StartDate, previous.EndDate)
		if err != nil {
			return nil, mapOpportunityGSCErr(err)
		}
		queries, err := searchConsole.GetQueryPageRows(work, org, project, dates.StartDate, dates.EndDate)
		if err != nil {
			return nil, mapOpportunityGSCErr(err)
		}
		currentGA, err := s.growthLandingReport(work, connection, dates)
		if err != nil {
			return nil, err
		}
		previousGA, err := s.growthLandingReport(work, connection, previous)
		if err != nil {
			return nil, err
		}
		report, err := buildGrowthReport(currentGSC, previousGSC, queries, currentGA, previousGA, in.BrandTerms)
		if err != nil {
			return nil, err
		}
		report.ID = cacheKey
		report.GeneratedAt = s.now().UTC()
		report.Range = dates
		report.PreviousRange = previous
		report.SiteURL = scope.SiteURL
		report.PropertyID = connection.PropertyID
		report.Currency = connection.PropertyCurrencyCode
		report.TimeZone = connection.PropertyTimeZone
		if connection.PropertyTimeZone != searchConsoleTimeZone {
			report.Warnings = append(report.Warnings, "Search Console uses Pacific time; Analytics uses "+connection.PropertyTimeZone+". Daily totals can differ.")
		}
		if err := s.Growth.StoreReport(work, org, project, report); err != nil {
			return nil, err
		}
		return report, nil
	})
	if err != nil {
		return GrowthReport{}, err
	}
	return result.(GrowthReport), nil
}

func (s *Service) growthLandingReport(ctx context.Context, c Connection, dates DateRange) (normalizedReport, error) {
	request := buildRequest(ReportInput{Kind: LandingPages, Channel: "organic_search"}, dates, definitions[LandingPages], 1000, 0)
	// Match Google organic traffic, rather than combining Bing with GSC clicks.
	request.DimensionFilter = googleOrganicFilter(request.DimensionFilter)
	response, err := s.runProvider(ctx, c, request)
	if err != nil {
		return normalizedReport{}, mapProviderError(err)
	}
	normalized, err := normalize(response, request)
	if err != nil {
		return normalizedReport{}, reportErr("ga4_malformed_response", "Google Analytics returned an invalid report.")
	}
	return normalized, nil
}

func metricPointer(row map[string]any, key string) *float64 {
	if v, ok := row[key].(float64); ok && !math.IsNaN(v) && !math.IsInf(v, 0) {
		return &v
	}
	return nil
}
func positive(v *float64) bool { return v != nil && *v > 0 }

func growthMetrics(row gsc.SearchRow, ga map[string]any) GrowthMetrics {
	return GrowthMetrics{Clicks: row.Clicks, Impressions: row.Impressions, CTR: row.CTR, Position: row.Position, Sessions: metricPointer(ga, "sessions"), KeyEvents: metricPointer(ga, "keyEvents"), KeyEventRate: metricPointer(ga, "sessionKeyEventRate"), EngagementRate: metricPointer(ga, "engagementRate"), Revenue: metricPointer(ga, "purchaseRevenue")}
}

// Duplicate normalized GA4 rows are deliberately unmatched. Rates and users
// cannot safely be summed when URL variants collapse into the same join key.
func growthAnalytics(rows []map[string]any) map[string]map[string]any {
	result := map[string]map[string]any{}
	ambiguous := map[string]bool{}
	for _, row := range rows {
		host, _ := row["hostName"].(string)
		path, _ := row["landingPage"].(string)
		if path == "(not set)" || !strings.HasPrefix(path, "/") {
			continue
		}
		key, ok := normalizePageKey(host + path)
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			ambiguous[key] = true
		}
		result[key] = row
	}
	for key := range ambiguous {
		delete(result, key)
	}
	return result
}

func growthSearchRows(rows []gsc.SearchRow) (map[string]gsc.SearchRow, error) {
	result := map[string]gsc.SearchRow{}
	ambiguous := map[string]bool{}
	for _, row := range rows {
		if len(row.Keys) != 1 || !validGrowthRow(row) {
			return nil, reportErr("ga4_malformed_response", "Search Console returned an invalid report.")
		}
		key, ok := normalizePageKey(row.Keys[0])
		if !ok {
			continue
		}
		if _, exists := result[key]; exists {
			ambiguous[key] = true
		}
		result[key] = row
	}
	// A summed baseline across distinct GSC URLs cannot be compared with a
	// later exact-URL read. Leave ambiguous normalized pages out of findings.
	for key := range ambiguous {
		delete(result, key)
	}
	return result, nil
}

func validGrowthRow(r gsc.SearchRow) bool {
	for _, v := range []float64{r.Clicks, r.Impressions, r.CTR, r.Position} {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
			return false
		}
	}
	return r.CTR <= 1 && r.Clicks <= r.Impressions
}

func buildGrowthReport(current, previous, queryRows []gsc.SearchRow, ga, prevGA normalizedReport, brands []string) (GrowthReport, error) {
	report := GrowthReport{Pages: []GrowthPage{}, Warnings: []string{}, BrandTerms: append([]string{}, brands...), Limited: ga.Metadata.HasLimitedData || prevGA.Metadata.HasLimitedData, Truncated: len(current) >= 1000 || len(previous) >= 1000 || len(queryRows) >= 1000 || ga.TotalRowCount > len(ga.Rows) || prevGA.TotalRowCount > len(prevGA.Rows), GSCPageCount: len(current), GA4PageCount: len(ga.Rows)}
	curr, err := growthSearchRows(current)
	if err != nil {
		return report, err
	}
	prev, err := growthSearchRows(previous)
	if err != nil {
		return report, err
	}
	currentAnalytics, previousAnalytics := growthAnalytics(ga.Rows), growthAnalytics(prevGA.Rows)
	queries := map[string][]GrowthQuery{}
	queryPages := map[string]map[string]bool{}
	for _, row := range queryRows {
		if len(row.Keys) != 2 || !validGrowthRow(row) {
			return report, reportErr("ga4_malformed_response", "Search Console returned invalid query evidence.")
		}
		query, page := row.Keys[0], row.Keys[1]
		key, ok := normalizePageKey(page)
		if !ok {
			continue
		}
		branded := false
		for _, brand := range brands {
			if strings.Contains(strings.ToLower(query), brand) {
				branded = true
				break
			}
		}
		queries[key] = append(queries[key], GrowthQuery{Query: query, Clicks: row.Clicks, Impressions: row.Impressions, Position: row.Position, Branded: branded, CompetingPages: []string{}})
		if row.Impressions >= 20 {
			if queryPages[query] == nil {
				queryPages[query] = map[string]bool{}
			}
			queryPages[query][key] = true
		}
	}
	// Baseline uses only joined, sufficiently observed pages; rate comparisons
	// are investigation candidates, never statistical or causal conclusions.
	var sessions, successfulSessions float64
	for key := range curr {
		row := currentAnalytics[key]
		n, rate := metricPointer(row, "sessions"), metricPointer(row, "sessionKeyEventRate")
		if n != nil && *n >= 100 && rate != nil {
			sessions += *n
			successfulSessions += *n * *rate
		}
	}
	baseline := 0.0
	if sessions > 0 {
		baseline = successfulSessions / sessions
	}
	for key, row := range curr {
		p := GrowthPage{Page: row.Keys[0], Current: growthMetrics(row, currentAnalytics[key]), Queries: queries[key], Findings: []GrowthFinding{}}
		if p.Queries == nil {
			p.Queries = []GrowthQuery{}
		}
		sort.SliceStable(p.Queries, func(i, j int) bool { return p.Queries[i].Impressions > p.Queries[j].Impressions })
		for i := range p.Queries {
			q := &p.Queries[i]
			for other := range queryPages[q.Query] {
				if other != key {
					q.CompetingPages = append(q.CompetingPages, "https://"+other)
				}
			}
			sort.Strings(q.CompetingPages)
		}
		if len(p.Queries) > 10 {
			p.Queries = p.Queries[:10]
		}
		if old, ok := prev[key]; ok {
			v := growthMetrics(old, previousAnalytics[key])
			p.Previous = &v
		}
		if p.Current.Sessions != nil {
			report.MatchedPages++
		}
		add := func(kind, title, evidence, action string) {
			p.Findings = append(p.Findings, GrowthFinding{Kind: kind, Title: title, Evidence: evidence, Action: action, Confidence: "Review candidate"})
		}
		if p.Previous != nil && p.Previous.Clicks >= 30 && p.Current.Clicks <= p.Previous.Clicks*.8 {
			add("traffic_loss", "Search clicks are falling", fmt.Sprintf("%.0f → %.0f Google clicks across equal-length periods.", p.Previous.Clicks, p.Current.Clicks), "Review lost queries and search intent; check indexing before rewriting. Seasonality and search-result changes may contribute.")
		}
		enough := p.Current.Sessions != nil && *p.Current.Sessions >= 100
		if !report.Limited && enough && positive(p.Current.KeyEvents) && *p.Current.KeyEvents >= 5 && positive(p.Current.KeyEventRate) && *p.Current.KeyEventRate >= baseline && row.Position >= 8 && row.Position <= 20 && row.Impressions >= 100 {
			add("grow_winner", "A performing page could reach more searchers", fmt.Sprintf("%.0f Google organic sessions, %.0f key events, %.1f%% session key-event rate; average position %.1f.", *p.Current.Sessions, *p.Current.KeyEvents, *p.Current.KeyEventRate*100, row.Position), "Review the related query themes and relevant internal links. Confirm that these key events represent a useful business outcome.")
		}
		if !report.Limited && enough && baseline > 0 && p.Current.KeyEventRate != nil && *p.Current.KeyEventRate < baseline*.5 && len(currentAnalytics) >= 3 {
			add("outcome_gap", "Traffic is arriving; review the next step", fmt.Sprintf("%.0f Google organic sessions; key-event rate %.1f%% versus %.1f%% across observed site pages.", *p.Current.Sessions, *p.Current.KeyEventRate*100, baseline*100), "Verify event tracking first, then review intent, offer and CTA. Different page types can have different expected rates.")
		}
		if !report.Limited && enough && p.Previous != nil && p.Previous.Sessions != nil && *p.Previous.Sessions >= 100 && p.Previous.KeyEvents != nil && *p.Previous.KeyEvents >= 5 && positive(p.Previous.KeyEventRate) && p.Current.KeyEventRate != nil && *p.Current.KeyEventRate < *p.Previous.KeyEventRate*.7 {
			add("rate_decline", "Key-event rate has dropped", fmt.Sprintf("%.1f%% → %.1f%% session key-event rate.", *p.Previous.KeyEventRate*100, *p.Current.KeyEventRate*100), "Check event setup and audience mix, then inspect the page and conversion path. This does not establish a cause.")
		}
		if !report.Limited && p.Previous != nil && positive(p.Previous.Revenue) && p.Current.Revenue != nil && numberField(previousAnalytics[key], "transactions") >= 5 && *p.Current.Revenue < *p.Previous.Revenue*.8 {
			add("revenue_decline", "Recorded purchase revenue has fallen", fmt.Sprintf("GA4 recorded purchase revenue %.2f → %.2f in the property's currency across equal-length periods.", *p.Previous.Revenue, *p.Current.Revenue), "Verify purchase/refund tracking and review traffic mix and the purchase path before attributing this change to content.")
		}
		if p.Previous != nil && p.Previous.Impressions >= 100 && row.Impressions >= 100 && math.Abs(row.Position-p.Previous.Position) <= 1 && row.CTR < p.Previous.CTR*.7 && p.Previous.Clicks >= 10 {
			add("ctr_decline", "Similar position, fewer clicks per impression", fmt.Sprintf("CTR %.1f%% → %.1f%%; average position %.1f → %.1f.", p.Previous.CTR*100, row.CTR*100, p.Previous.Position, row.Position), "Review title, snippet and current search results. Brand, country and device mix can change aggregate CTR.")
		}
		for _, q := range p.Queries {
			if len(q.CompetingPages) > 0 {
				add("competing_pages", "Multiple pages appear for a query", fmt.Sprintf("%q has impressions for more than one returned page.", q.Query), "Compare intent and canonical pages before consolidating. Multiple ranking pages are not proof of cannibalization.")
				break
			}
		}
		if len(p.Findings) == 0 && row.Position >= 8 && row.Position <= 20 && row.Impressions >= 100 {
			add("near_top", "Queries are within reach of the top results", fmt.Sprintf("%.0f impressions; average position %.1f.", row.Impressions, row.Position), "Review query intent and content coverage before starting a new article.")
		}
		p.Priority = float64(len(p.Findings))*10 + math.Log1p(row.Impressions)
		if !report.Limited && p.Previous != nil && positive(p.Previous.Revenue) {
			p.Priority += math.Min(math.Log1p(*p.Previous.Revenue), 10) * 2
		}
		if !report.Limited && p.Previous != nil && positive(p.Previous.KeyEvents) {
			p.Priority += math.Min(*p.Previous.KeyEvents, 100)
		}
		report.Pages = append(report.Pages, p)
	}
	sort.SliceStable(report.Pages, func(i, j int) bool {
		if report.Pages[i].Priority != report.Pages[j].Priority {
			return report.Pages[i].Priority > report.Pages[j].Priority
		}
		return report.Pages[i].Page < report.Pages[j].Page
	})
	if report.Limited {
		report.Warnings = append(report.Warnings, "Analytics reports are limited, sampled, thresholded or restricted. Business-rate recommendations are suppressed.")
	}
	if report.Truncated {
		report.Warnings = append(report.Warnings, "Reports contain at most 1,000 rows per source. Missing rows are unknown, not zero; opportunities are not exhaustive.")
	}
	if len(brands) == 0 {
		report.Warnings = append(report.Warnings, "No brand terms were supplied; query branding is unclassified.")
	}
	report.Warnings = append(report.Warnings, "Ambiguous URL variants are excluded rather than combining incompatible page baselines or Analytics rates.")
	report.Warnings = append(report.Warnings, "Google clicks and Analytics sessions measure different things. Query evidence cannot be attributed to individual conversions.", "GA4 key events reflect your configuration; they are not automatically verified leads or purchases.")
	return report, nil
}

// GetGrowthDevices reads a single Google-organic device report for an explicit
// drill-down. It does not infer that device or speed caused an outcome gap.
func (s *Service) GetGrowthDevices(ctx context.Context, project string, in GrowthInput) (map[string]any, error) {
	if err := validateGrowthInput(in); err != nil {
		return nil, err
	}
	if s == nil || s.Connections == nil {
		return nil, reportErr("ga4_unavailable", "Analytics reporting is unavailable.")
	}
	c, err := s.Connections.GetByProjectID(ctx, project)
	if err != nil {
		if errors.Is(err, ErrConnectionNotFound) {
			return nil, reportErr("ga4_not_connected", "Connect Analytics first.")
		}
		return nil, err
	}
	dates, err := resolveOpportunityDates(SearchOpportunityInput{StartDate: in.StartDate, EndDate: in.EndDate}, c.PropertyTimeZone, s.now())
	if err != nil {
		return nil, err
	}
	first, ok1 := parseDate(dates.StartDate)
	last, ok2 := parseDate(dates.EndDate)
	if !ok1 || !ok2 || last.Sub(first) > 89*24*time.Hour {
		return nil, invalid("Choose up to 90 days.")
	}
	request := buildRequest(ReportInput{Kind: LandingPages, Channel: "organic_search"}, dates, definitions[LandingPages], 100, 0)
	request.Dimensions = []Name{{Name: "deviceCategory"}}
	request.DimensionFilter = googleOrganicFilter(request.DimensionFilter)
	work, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	response, err := s.runProvider(work, c, request)
	if err != nil {
		return nil, mapProviderError(err)
	}
	report, err := normalize(response, request)
	if err != nil {
		return nil, reportErr("ga4_malformed_response", "Analytics returned an invalid device report.")
	}
	return map[string]any{"rows": report.Rows, "limited": report.Metadata.HasLimitedData, "range": dates}, nil
}
func googleOrganicFilter(organic any) any {
	return map[string]any{"andGroup": map[string]any{"expressions": []any{organic, map[string]any{"filter": map[string]any{"fieldName": "sessionSource", "stringFilter": map[string]any{"matchType": "EXACT", "value": "google"}}}}}}
}
