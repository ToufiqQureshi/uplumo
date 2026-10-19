package gsc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/google"
)

// SitemapContent preserves the submitted count for a Google content type.
type SitemapContent struct {
	Type      string `json:"type"`
	Submitted int64  `json:"submitted"`
}

// SitemapStatus is submitted-sitemap metadata, not a URL-level indexing report.
type SitemapStatus struct {
	Path           string           `json:"path"`
	LastSubmitted  string           `json:"lastSubmitted"`
	LastDownloaded string           `json:"lastDownloaded"`
	Pending        bool             `json:"pending"`
	IsIndex        bool             `json:"isIndex"`
	Errors         int64            `json:"errors"`
	Warnings       int64            `json:"warnings"`
	Submitted      int64            `json:"submitted"`
	Contents       []SitemapContent `json:"contents"`
}

type sitemapLister interface {
	ListSitemaps(context.Context, string) ([]SitemapStatus, bool, error)
}

// ListSitemaps reads only the selected Search Console property, never the
// customer-controlled sitemap URLs themselves. Nested files are not fetched.
func (c *Client) ListSitemaps(ctx context.Context, site string) ([]SitemapStatus, bool, error) {
	var response struct {
		Sitemaps []struct {
			Path            string      `json:"path"`
			LastSubmitted   string      `json:"lastSubmitted"`
			LastDownloaded  string      `json:"lastDownloaded"`
			IsPending       bool        `json:"isPending"`
			IsSitemapsIndex bool        `json:"isSitemapsIndex"`
			Errors          json.Number `json:"errors"`
			Warnings        json.Number `json:"warnings"`
			Contents        []struct {
				Type      string      `json:"type"`
				Submitted json.Number `json:"submitted"`
			} `json:"contents"`
		} `json:"sitemap"`
	}
	err := c.API.DoJSON(ctx, google.APIRequest{UserID: c.UserID, Provider: "gsc", AccountID: c.AccountID, Method: http.MethodGet, URL: c.baseURL() + "/sites/" + url.PathEscape(site) + "/sitemaps", Response: &response, Retryable: true, MaxResponseBytes: 2 << 20})
	if err != nil {
		return nil, false, err
	}
	rows := []SitemapStatus{}
	truncated := len(response.Sitemaps) > 200
	for _, item := range response.Sitemaps[:min(len(response.Sitemaps), 200)] {
		parsed, err := url.Parse(item.Path)
		if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || len(item.Path) > 2048 {
			return nil, false, errors.New("invalid sitemap path")
		}
		errorsCount, err := sitemapCount(item.Errors)
		if err != nil {
			return nil, false, err
		}
		warnings, err := sitemapCount(item.Warnings)
		if err != nil {
			return nil, false, err
		}
		var submitted int64
		contents := []SitemapContent{}
		for _, content := range item.Contents {
			count, err := sitemapCount(content.Submitted)
			if err != nil {
				return nil, false, err
			}
			if submitted > 9007199254740991-count {
				return nil, false, errors.New("invalid sitemap count")
			}
			submitted += count
			contents = append(contents, SitemapContent{Type: content.Type, Submitted: count})
		}
		rows = append(rows, SitemapStatus{Path: item.Path, LastSubmitted: item.LastSubmitted, LastDownloaded: item.LastDownloaded, Pending: item.IsPending, IsIndex: item.IsSitemapsIndex, Errors: errorsCount, Warnings: warnings, Submitted: submitted, Contents: contents})
	}
	return rows, truncated, nil
}
func sitemapCount(number json.Number) (int64, error) {
	if number == "" {
		return 0, nil
	}
	value, err := number.Int64()
	if err != nil || value < 0 || value > 9007199254740991 {
		return 0, errors.New("invalid sitemap count")
	}
	return value, nil
}

// GetSitemaps reports bounded submitted entries and preserves unknown download
// dates. A successful sitemap submission does not establish indexing.
func (s *Service) GetSitemaps(ctx context.Context, org, project string) (map[string]any, error) {
	connection, client, err := s.client(ctx, org, project)
	if err != nil {
		return nil, mapProviderError(err)
	}
	lister, ok := client.(sitemapLister)
	if !ok {
		return nil, errors.New("search console sitemap reporting is unavailable")
	}
	work, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	rows, truncated, err := lister.ListSitemaps(work, connection.SiteURL)
	if err != nil {
		return nil, fmt.Errorf("read submitted sitemaps: %w", mapProviderError(err))
	}
	return map[string]any{"siteUrl": connection.SiteURL, "sitemaps": rows, "truncated": truncated, "message": "Submitted sitemap entries, up to 200. Nested sitemap files and their URLs are not crawled here. Errors and warnings are Google's sitemap-level counts, not full page-indexing reasons. Submission, download or submitted entry counts (which may overlap across content types) do not prove pages are indexed. Use URL Inspection for a selected page's stored index status."}, nil
}

// GetPageQueryRows reads returned queries for one exact page. Unreturned queries
// remain unknown and are never manufactured as zero-click searches.
func (s *Service) GetPageQueryRows(ctx context.Context, org, project, start, end, page string) ([]SearchRow, error) {
	parsed, err := url.Parse(page)
	if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || len(page) > 1024 || !validDate(start) || !validDate(end) || start > end {
		return nil, validationError("Choose valid dates and a supported page URL.")
	}
	connection, client, err := s.client(ctx, org, project)
	if err != nil {
		return nil, err
	}
	rows, err := client.QuerySearchAnalytics(ctx, connection.SiteURL, SearchRequest{StartDate: start, EndDate: end, Dimensions: []string{"query"}, RowLimit: 1000, Type: "web", DataState: "final", DimensionFilterGroups: []FilterGroup{{GroupType: "and", Filters: []DimensionFilter{{Dimension: "page", Operator: "equals", Expression: page}}}}})
	if err != nil {
		return nil, fmt.Errorf("read page query evidence: %w", err)
	}
	if len(rows) > 1000 {
		return nil, validationError("Search Console returned too many query rows.")
	}
	return rows, nil
}
