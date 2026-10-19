package ga4

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/gsc"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/httpx"
)

func (d Deps) growthHandler(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if d.Service == nil || d.Service.Growth == nil {
			httpx.WriteError(w, 503, "ga4_unavailable", "Search growth reporting is unavailable.")
			return
		}
		org, ok := auth.ProjectOrganizationFromContext(r.Context())
		if !ok {
			httpx.WriteError(w, 401, "unauthenticated", "Sign in to continue.")
			return
		}
		project := r.PathValue("projectId")
		var result any
		var err error
		switch action {
		case "report":
			var in GrowthInput
			if err = decodeRequest(w, r, &in); err != nil {
				err = invalid("Choose valid dates and up to 10 brand terms.")
				break
			}
			result, err = d.Service.GetGrowthReport(r.Context(), d.SearchConsole, org, project, in)
		case "devices":
			var in GrowthInput
			if err = decodeRequest(w, r, &in); err != nil {
				err = invalid("Choose valid dates.")
				break
			}
			result, err = d.Service.GetGrowthDevices(r.Context(), project, in)
		case "list":
			var in struct{}
			if err = decodeRequest(w, r, &in); err != nil {
				err = invalid("Expected an empty object.")
				break
			}
			result, err = d.Service.Growth.Actions(r.Context(), org, project)
		case "save":
			var in struct {
				ReportID string `json:"reportId"`
				Page     string `json:"page"`
				Note     string `json:"note"`
			}
			if err = decodeRequest(w, r, &in); err != nil {
				err = invalid("Choose a report page and add a note.")
				break
			}
			var id string
			id, err = d.Service.Growth.SaveAction(r.Context(), org, project, in.ReportID, in.Page, in.Note)
			result = map[string]string{"id": id}
		case "update":
			var in struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			}
			if err = decodeRequest(w, r, &in); err != nil {
				err = invalid("Choose a saved action and status.")
				break
			}
			err = d.Service.Growth.UpdateAction(r.Context(), org, project, in.ID, in.Status)
			result = map[string]bool{"saved": err == nil}
		case "delete":
			var in struct {
				ID string `json:"id"`
			}
			if err = decodeRequest(w, r, &in); err != nil {
				err = invalid("Choose a saved action.")
				break
			}
			err = d.Service.Growth.DeleteAction(r.Context(), org, project, in.ID)
			result = map[string]bool{"deleted": err == nil}
		case "compare":
			var in struct {
				ID   string `json:"id"`
				Days int    `json:"days"`
			}
			if err = decodeRequest(w, r, &in); err != nil {
				err = invalid("Choose a saved action and 14 or 28 days.")
				break
			}
			result, err = d.Service.CompareGrowthAction(r.Context(), d.SearchConsole, org, project, in.ID, in.Days)
		}
		if err != nil {
			writeError(d.Logger, w, r, err)
			return
		}
		httpx.WriteJSON(w, 200, result)
	}
}

// CompareGrowthAction compares fixed baseline evidence with a complete window
// after the applied date. It never treats missing rows as measured zero.
func (s *Service) CompareGrowthAction(ctx context.Context, searchConsole *gsc.Service, org, project, id string, days int) (map[string]any, error) {
	if days != 14 && days != 28 {
		return nil, invalid("Choose 14 or 28 days.")
	}
	actions, err := s.Growth.Actions(ctx, org, project)
	if err != nil {
		return nil, err
	}
	var selected *GrowthAction
	for i := range actions {
		if actions[i].ID == id {
			selected = &actions[i]
			break
		}
	}
	if selected == nil {
		return nil, reportErr("growth_not_found", "This saved action was not found.")
	}
	if selected.Status != "applied" || selected.AppliedAt == nil {
		return nil, invalid("Mark the page change applied before comparing.")
	}
	connection, err := s.Connections.GetByProjectID(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("load comparison connection: %w", err)
	}
	if searchConsole == nil {
		return nil, reportErr("gsc_unavailable", "Search Console is unavailable.")
	}
	scope, err := searchConsole.Connections.GetByProjectID(ctx, org, project)
	if err != nil {
		return nil, mapOpportunityGSCErr(err)
	}
	if connection.PropertyID != selected.Source.PropertyID || scope.SiteURL != selected.Source.SiteURL || connection.PropertyTimeZone != selected.Source.TimeZone || connection.PropertyCurrencyCode != selected.Source.Currency {
		return nil, invalid("The connected property, timezone or currency changed. Save a new baseline before comparing.")
	}
	location, err := time.LoadLocation(connection.PropertyTimeZone)
	if err != nil {
		return nil, err
	}
	// Exclude the change day in both source time zones.
	first := selected.AppliedAt.In(location).AddDate(0, 0, 1).Format("2006-01-02")
	pacific, err := time.LoadLocation(searchConsoleTimeZone)
	if err != nil {
		return nil, err
	}
	first = max(first, selected.AppliedAt.In(pacific).AddDate(0, 0, 1).Format("2006-01-02"))
	start, _ := parseDate(first)
	last := start.AddDate(0, 0, days-1).Format("2006-01-02")
	latest := min(s.now().In(location).AddDate(0, 0, -3).Format("2006-01-02"), s.now().In(pacific).AddDate(0, 0, -3).Format("2006-01-02"))
	result := map[string]any{"baselineClicksPerDay": nil, "afterClicksPerDay": nil, "status": "waiting", "baseline": selected.Baseline.Current, "baselineRange": selected.BaselineRange, "after": nil, "afterRange": DateRange{StartDate: first, EndDate: last}, "message": "The complete post-change window is not available yet. Allow the selected days plus Google's reporting delay."}
	if last > latest {
		return result, nil
	}
	// Fetch only this page rather than dropping a low-traffic page from top rows.
	work, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	gscRows, err := searchConsole.GetFilteredPageRows(work, org, project, first, last, selected.Page)
	if err != nil {
		return nil, mapOpportunityGSCErr(err)
	}
	key, ok := normalizePageKey(selected.Page)
	if !ok {
		return nil, invalid("The saved page URL is invalid.")
	}
	request := buildRequest(ReportInput{Kind: LandingPages, Channel: "organic_search"}, DateRange{StartDate: first, EndDate: last}, definitions[LandingPages], 1000, 0)
	parsedPage, err := url.Parse(selected.Page)
	if err != nil {
		return nil, invalid("The saved page URL is invalid.")
	}
	pagePath := strings.TrimRight(parsedPage.Path, "/")
	pathExpression := "^" + regexp.QuoteMeta(pagePath) + "/?$"
	request.DimensionFilter = map[string]any{"andGroup": map[string]any{"expressions": []any{
		googleOrganicFilter(request.DimensionFilter),
		map[string]any{"filter": map[string]any{"fieldName": "hostName", "stringFilter": map[string]any{"matchType": "EXACT", "value": pageHost(selected.Page)}}},
		map[string]any{"filter": map[string]any{"fieldName": "landingPage", "stringFilter": map[string]any{"matchType": "FULL_REGEXP", "value": pathExpression, "caseSensitive": true}}},
	}}}

	response, err := s.runProvider(work, connection, request)
	if err != nil {
		return nil, mapProviderError(err)
	}
	ga, err := normalize(response, request)
	if err != nil {
		return nil, reportErr("ga4_malformed_response", "Analytics returned an invalid report.")
	}
	rows, err := growthSearchRows(gscRows)
	if err != nil {
		return nil, err
	}
	row, exists := rows[key]
	baseStart, ok1 := parseDate(selected.BaselineRange.StartDate)
	baseEnd, ok2 := parseDate(selected.BaselineRange.EndDate)
	if !ok1 || !ok2 || baseEnd.Before(baseStart) {
		return nil, invalid("Saved baseline dates are invalid.")
	}
	baselineDays := int(baseEnd.Sub(baseStart).Hours()/24) + 1
	result["baselineClicksPerDay"] = selected.Baseline.Current.Clicks / float64(baselineDays)
	result["afterClicksPerDay"] = nil
	result["status"] = "ready"
	result["message"] = "Observed before/after difference, not proof the edit caused it. Baseline and post-change dates can differ in length; compare daily averages and account for seasonality. Missing rows are unknown."
	if exists {
		result["after"] = growthMetrics(row, growthAnalytics(ga.Rows)[key])
		result["afterClicksPerDay"] = row.Clicks / float64(days)
	}
	result["limited"] = ga.Metadata.HasLimitedData || selected.Source.Limited
	return result, nil
}

func pageHost(page string) string {
	u, err := url.Parse(page)
	if err != nil {
		return ""
	}
	return u.Hostname()
}
