package ga4

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/google"
)

var growthEventPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{0,39}$`)

func validGrowthEvent(event string) bool { return growthEventPattern.MatchString(event) }

func growthComparisonPeriod(dates DateRange, mode string) (DateRange, error) {
	if mode != "year" {
		return previousPeriod(dates)
	}
	start, a := parseDate(dates.StartDate)
	end, b := parseDate(dates.EndDate)
	if !a || !b || end.Before(start) {
		return DateRange{}, invalid("Choose valid comparison dates.")
	}
	return DateRange{StartDate: start.AddDate(0, 0, -364).Format("2006-01-02"), EndDate: end.AddDate(0, 0, -364).Format("2006-01-02")}, nil
}

// GetGrowthGoals discovers only metrics actually supported by this property.
// Configuration is not proof the underlying event fires after a real success.
func (s *Service) GetGrowthGoals(ctx context.Context, project string) ([]string, error) {
	if s == nil || s.Connections == nil {
		return nil, reportErr("ga4_unavailable", "Connect Analytics first.")
	}
	c, err := s.growthConnection(ctx, project)
	if err != nil {
		return nil, err
	}
	return s.growthGoals(ctx, c)
}

func (s *Service) growthGoals(ctx context.Context, c Connection) ([]string, error) {
	if s.Google == nil || !validPropertyID(c.PropertyID) {
		return nil, reportErr("ga4_unavailable", "Analytics metadata is unavailable.")
	}
	work, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var metadata struct {
		Metrics []struct {
			APIName string `json:"apiName"`
		} `json:"metrics"`
	}
	err := s.Google.DoJSON(work, google.APIRequest{UserID: c.ConnectedByUserID, Provider: google.ProviderGA4, AccountID: c.GA4AccountID, Method: http.MethodGet, URL: "https://analyticsdata.googleapis.com/v1beta/" + c.PropertyID + "/metadata", Response: &metadata, Retryable: true, MaxResponseBytes: 2 << 20})
	if err != nil {
		return nil, mapProviderError(err)
	}
	goals := []string{}
	seen := map[string]bool{}
	for _, metric := range metadata.Metrics {
		event, ok := strings.CutPrefix(metric.APIName, "sessionKeyEventRate:")
		if ok && validGrowthEvent(event) && !seen[event] {
			goals = append(goals, event)
			seen[event] = true
		}
	}
	return goals, nil
}

func (s *Service) validateGrowthGoal(ctx context.Context, c Connection, event string) error {
	if !validGrowthEvent(event) {
		return invalid("Choose a valid goal event.")
	}
	goals, err := s.growthGoals(ctx, c)
	if err != nil {
		return err
	}
	for _, goal := range goals {
		if goal == event {
			return nil
		}
	}
	return invalid("This goal is not available as a registered key event for the selected property. Refresh goals or review GA4 setup.")
}

func applyGrowthGoal(request *APIRequest, event string) {
	if event != "" {
		request.Metrics = append(request.Metrics, Name{Name: "sessionKeyEventRate:" + event})
	}
}

func labelGrowthGoal(rows []map[string]any, goal string) error {
	if goal == "" {
		return nil
	}
	for _, row := range rows {
		rate := metricPointer(row, "sessionKeyEventRate:"+goal)
		if rate != nil && (*rate < 0 || *rate > 1) {
			return reportErr("ga4_malformed_response", "Analytics returned an invalid goal rate.")
		}
		row["goalRate"] = row["sessionKeyEventRate:"+goal]
	}
	return nil
}

// The decision view uses estimated sessions reaching the selected goal for
// minimum-sample checks. The raw key-event count remains unchanged in the API.
func growthOutcomeRows(rows map[string]map[string]any) map[string]map[string]any {
	result := make(map[string]map[string]any, len(rows))
	for key, row := range rows {
		copyRow := make(map[string]any, len(row))
		for name, value := range row {
			copyRow[name] = value
		}
		if rate, selected := row["goalRate"]; selected {
			copyRow["sessionKeyEventRate"] = rate
			copyRow["keyEvents"] = nil
			sessions, r := metricPointer(row, "sessions"), metricPointer(row, "goalRate")
			if sessions != nil && r != nil {
				copyRow["keyEvents"] = *sessions * *r
			}
		}
		result[key] = copyRow
	}
	return result
}

func (s *Service) growthConnection(ctx context.Context, project string) (Connection, error) {
	if s == nil || s.Connections == nil {
		return Connection{}, reportErr("ga4_unavailable", "Analytics reporting is unavailable.")
	}
	c, err := s.Connections.GetByProjectID(ctx, project)
	if errors.Is(err, ErrConnectionNotFound) {
		return Connection{}, reportErr("ga4_not_connected", "Connect Analytics before investigating this project.")
	}
	if err != nil {
		return Connection{}, fmt.Errorf("load growth connection: %w", err)
	}
	return c, nil
}
