package ga4

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// GetGrowthTracking surfaces observed referral/host evidence and a setup
// checklist. Admin stream configuration cannot verify browser consent or tags.
func (s *Service) GetGrowthTracking(ctx context.Context, project string, in GrowthInput) (map[string]any, error) {
	if err := validateGrowthInput(in); err != nil {
		return nil, err
	}
	if s == nil || s.Connections == nil {
		return nil, reportErr("ga4_unavailable", "Analytics is unavailable.")
	}
	c, err := s.growthConnection(ctx, project)
	if err != nil {
		return nil, err
	}
	dates, err := resolveOpportunityDates(SearchOpportunityInput{StartDate: in.StartDate, EndDate: in.EndDate}, c.PropertyTimeZone, s.now())
	if err != nil {
		return nil, err
	}
	start, _ := parseDate(dates.StartDate)
	end, _ := parseDate(dates.EndDate)
	if end.Sub(start) > 89*24*time.Hour {
		return nil, invalid("Choose up to 90 days.")
	}
	work, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	request := buildRequest(ReportInput{Kind: LandingPages, Channel: "all"}, dates, definitions[LandingPages], 100, 0)
	request.DimensionFilter = nil
	request.Dimensions = []Name{{Name: "hostName"}, {Name: "sessionSource"}, {Name: "sessionMedium"}}
	request.Metrics = []Name{{Name: "sessions"}, {Name: "keyEvents"}}
	response, err := s.runProvider(work, c, request)
	if err != nil {
		return nil, mapProviderError(err)
	}
	report, err := normalize(response, request)
	if err != nil {
		return nil, reportErr("ga4_malformed_response", "Analytics returned invalid tracking evidence.")
	}
	findings := []GrowthFinding{}
	for _, row := range report.Rows {
		host, _ := row["hostName"].(string)
		source, _ := row["sessionSource"].(string)
		medium, _ := row["sessionMedium"].(string)
		n := metricPointer(row, "sessions")
		if n == nil || *n < 20 || report.Metadata.HasLimitedData {
			continue
		}
		parsed, parseErr := url.Parse("https://" + source)
		if parseErr != nil || parsed.Hostname() == "" {
			continue
		}
		if medium == "referral" && strings.EqualFold(strings.TrimPrefix(parsed.Hostname(), "www."), strings.TrimPrefix(host, "www.")) {
			findings = append(findings, GrowthFinding{Kind: "self_referral", Title: "Your own host appears as a referral", Evidence: fmt.Sprintf("%s → %s: %.0f sessions in returned rows.", source, host, *n), Action: "Test the journey in Tag Assistant and DebugView. Review cross-domain measurement and unwanted referrals before changing content.", Confidence: "Investigation needed"})
		}
		if len(findings) >= 5 {
			break
		}
	}
	return map[string]any{"range": dates, "rows": report.Rows, "findings": findings, "limited": report.Metadata.HasLimitedData || report.TotalRowCount > len(report.Rows), "checklist": []string{"Confirm the selected success event fires once after a real completed lead, purchase or booking, not merely a click.", "Test your checkout/booking across domains; verify the same GA4 measurement ID and that the _gl linker survives redirects when cross-domain measurement is required.", "Review payment/booking referrals and consent behavior before interpreting changes. This read-only check cannot validate tag or consent configuration.", "Compare dates, time zones and channel definitions. GSC clicks and GA4 sessions are different measures; a mismatch is not proof that tracking is broken.", "Direct traffic has an unknown source. Do not label it AI traffic without referral or other evidence."}}, nil
}
