package gsc

import (
	"context"
	"fmt"
)

// GetGrowthRows pages a bounded, explicit growth analysis without changing
// the legacy report's row limits. Even a complete API page is not all GSC data.
func (s *Service) GetGrowthRows(ctx context.Context, org, project, start, end string, queries bool, limit int) ([]SearchRow, error) {
	if limit != 1000 && limit != 5000 {
		return nil, validationError("Choose standard or expanded coverage.")
	}
	if !validDate(start) || !validDate(end) || start > end {
		return nil, validationError("Choose valid dates.")
	}
	connection, client, err := s.client(ctx, org, project)
	if err != nil {
		return nil, err
	}
	dimensions := []string{"page"}
	if queries {
		dimensions = []string{"query", "page"}
	}
	rows := []SearchRow{}
	for offset := 0; offset < limit; offset += 1000 {
		request := SearchRequest{StartDate: start, EndDate: end, Dimensions: dimensions, RowLimit: 1000, StartRow: offset, Type: "web", DataState: "final", AggregationType: "auto"}
		page, err := client.QuerySearchAnalytics(ctx, connection.SiteURL, request)
		if err != nil {
			return nil, fmt.Errorf("query growth rows: %w", err)
		}
		if len(page) > 1000 {
			return nil, fmt.Errorf("search console exceeded the growth row bound")
		}
		rows = append(rows, page...)
		if len(page) < 1000 {
			break
		}
	}
	return rows, nil
}
