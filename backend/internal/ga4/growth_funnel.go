package ga4

import (
	"context"
	"net/http"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/google"
)

// GrowthFunnelInput uses real configured event names, never estimated event counts.
type GrowthFunnelInput struct {
	StartDate string   `json:"startDate"`
	EndDate   string   `json:"endDate"`
	Steps     []string `json:"steps"`
	ByDevice  bool     `json:"byDevice"`
}

// GetGrowthFunnel reads a closed, ordered, user-scoped funnel. It is independent
// of landing-page session attribution and does not prove a cause of abandonment.
func (s *Service) GetGrowthFunnel(ctx context.Context, project string, in GrowthFunnelInput) (map[string]any, error) {
	if len(in.Steps) < 2 || len(in.Steps) > 5 {
		return nil, invalid("Choose 2–5 tracked event steps in order.")
	}
	seen := map[string]bool{}
	for _, event := range in.Steps {
		if !validGrowthEvent(event) || seen[event] {
			return nil, invalid("Each funnel step must be a distinct valid GA4 event name.")
		}
		seen[event] = true
	}
	if s == nil || s.Connections == nil || s.Google == nil {
		return nil, reportErr("ga4_unavailable", "Analytics is unavailable.")
	}
	c, err := s.growthConnection(ctx, project)
	if err != nil {
		return nil, err
	}
	if !validPropertyID(c.PropertyID) {
		return nil, invalid("Choose a valid Analytics property.")
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
	steps := make([]any, 0, len(in.Steps))
	for _, event := range in.Steps {
		steps = append(steps, map[string]any{"name": event, "filterExpression": map[string]any{"funnelEventFilter": map[string]any{"eventName": event}}})
	}
	filter := googleOrganicFilter(map[string]any{"filter": map[string]any{"fieldName": "sessionMedium", "stringFilter": map[string]any{"matchType": "EXACT", "value": "organic"}}})
	body := map[string]any{"dateRanges": []DateRange{dates}, "funnel": map[string]any{"isOpenFunnel": false, "steps": steps}, "dimensionFilter": filter, "limit": "100", "returnPropertyQuota": true}
	if in.ByDevice {
		body["funnelBreakdown"] = map[string]any{"breakdownDimension": Name{Name: "deviceCategory"}, "limit": "10"}
	}
	var response struct {
		FunnelTable *ProviderResponse `json:"funnelTable"`
	}
	work, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	err = s.Google.DoJSON(work, google.APIRequest{UserID: c.ConnectedByUserID, Provider: google.ProviderGA4, AccountID: c.GA4AccountID, Method: http.MethodPost, URL: "https://analyticsdata.googleapis.com/v1alpha/" + c.PropertyID + ":runFunnelReport", Body: body, Response: &response, Retryable: true, MaxResponseBytes: 2 << 20})
	if err != nil {
		return nil, mapProviderError(err)
	}
	if response.FunnelTable == nil || len(response.FunnelTable.Rows) > 100 {
		return nil, reportErr("ga4_malformed_response", "Analytics returned an invalid funnel.")
	}
	table := *response.FunnelTable
	expectedDims := []string{"funnelStepName"}
	if in.ByDevice {
		expectedDims = append(expectedDims, "deviceCategory")
	}
	if !sameStrings(names(table.DimensionHeaders), expectedDims) {
		return nil, reportErr("ga4_malformed_response", "Analytics returned unexpected funnel dimensions.")
	}
	expectedMetrics := []string{"activeUsers", "funnelStepCompletionRate", "funnelStepAbandonments", "funnelStepAbandonmentRate"}
	if !sameStrings(names(table.MetricHeaders), expectedMetrics) {
		return nil, reportErr("ga4_malformed_response", "Analytics returned unexpected funnel metrics.")
	}
	request := APIRequest{Dimensions: table.DimensionHeaders, Metrics: table.MetricHeaders}
	report, err := normalize(table, request)
	if err != nil {
		return nil, reportErr("ga4_malformed_response", "Analytics returned invalid funnel values.")
	}
	for _, row := range report.Rows {
		for _, name := range expectedMetrics {
			value := metricPointer(row, name)
			if value != nil && (*value < 0 || ((name == "funnelStepCompletionRate" || name == "funnelStepAbandonmentRate") && *value > 1)) {
				return nil, reportErr("ga4_malformed_response", "Analytics returned invalid funnel values.")
			}
		}
	}
	return map[string]any{"range": dates, "steps": in.Steps, "rows": report.Rows, "limited": report.Metadata.HasLimitedData || report.TotalRowCount > len(report.Rows), "message": "Ordered user funnel for Google organic traffic, not landing-page session conversion. Users can complete steps across sessions in this date range. Missing or misfiring events can look like abandonment. Rates from fewer than 100 users or limited data need more evidence. Funnel API is alpha."}, nil
}
