package gsc

import (
	"context"
	"fmt"
)

// GetFilteredPageRows reads final web data for one exact URL, so a comparison
// does not silently lose a page outside Google's most popular rows.
func (s *Service) GetFilteredPageRows(ctx context.Context, org, project, start, end, page string) ([]SearchRow, error) {
	if !validDate(start) || !validDate(end) || start > end || len(page) > 2048 {
		return nil, validationError("Choose valid dates and a page URL.")
	}
	connection, client, err := s.client(ctx, org, project)
	if err != nil {
		return nil, err
	}
	rows, err := client.QuerySearchAnalytics(ctx, connection.SiteURL, SearchRequest{StartDate: start, EndDate: end, Dimensions: []string{"page"}, RowLimit: 1, Type: "web", DataState: "final", DimensionFilterGroups: []FilterGroup{{GroupType: "and", Filters: []DimensionFilter{{Dimension: "page", Operator: "equals", Expression: page}}}}})
	if err != nil {
		return nil, fmt.Errorf("read exact page evidence: %w", err)
	}
	if len(rows) > 1 {
		return nil, validationError("Search Console returned too many page rows.")
	}
	return rows, nil
}
