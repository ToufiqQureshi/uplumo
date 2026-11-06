package gsc

import (
	"context"
	"errors"
	"fmt"
)

// GetQueryRows returns up to 1,000 final web-search query rows for a bounded
// date range, sorted by clicks as Search Console returns them.
func (s *Service) GetQueryRows(ctx context.Context, organizationID, projectID, startDate, endDate string) ([]SearchRow, error) {
	connection, client, err := s.client(ctx, organizationID, projectID)
	if err != nil {
		return nil, err
	}
	if !validDate(startDate) || !validDate(endDate) || startDate > endDate {
		return nil, validationError("Dates must be valid YYYY-MM-DD values with startDate on or before endDate.")
	}
	request := SearchRequest{StartDate: startDate, EndDate: endDate, Dimensions: []string{"query"}, RowLimit: 1000, Type: "web", DataState: "final", AggregationType: "auto"}
	rows, err := client.QuerySearchAnalytics(ctx, connection.SiteURL, request)
	if err != nil {
		return nil, fmt.Errorf("query Search Console query rows: %w", mapProviderError(err))
	}
	if len(rows) > request.RowLimit {
		return nil, errors.New("search console returned too many query rows")
	}
	if rows == nil {
		rows = []SearchRow{}
	}
	return rows, nil
}
