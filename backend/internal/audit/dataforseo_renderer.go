package audit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"golang.org/x/net/publicsuffix"

	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

const (
	onPageInstantPagesPath = "/v3/on_page/instant_pages"
	onPageRawHTMLPath      = "/v3/on_page/raw_html"
)

// DataForSEORenderer runs the paid On-Page JavaScript task and reads the raw
// HTML through DataForSEO's free seven-day retrieval endpoint.
type DataForSEORenderer struct {
	Client *dataforseo.Client
}

type renderTaskInput struct {
	URL              string `json:"url"`
	EnableJavaScript bool   `json:"enable_javascript"`
	StoreRawHTML     bool   `json:"store_raw_html"`
}

type rawHTMLInput struct {
	ID string `json:"id"`
}

type renderTaskResponse struct {
	StatusCode int `json:"status_code"`
	Tasks      []struct {
		ID         string  `json:"id"`
		StatusCode int     `json:"status_code"`
		Cost       float64 `json:"cost"`
		Data       struct {
			URL string `json:"url"`
		} `json:"data"`
		Result []struct {
			Items []struct {
				ResourceType string `json:"resource_type"`
				StatusCode   int    `json:"status_code"`
				URL          string `json:"url"`
			} `json:"items"`
		} `json:"result"`
	} `json:"tasks"`
}

type rawHTMLResponse struct {
	StatusCode int `json:"status_code"`
	Tasks      []struct {
		ID         string `json:"id"`
		StatusCode int    `json:"status_code"`
		Result     []struct {
			Items struct {
				HTML string `json:"html"`
			} `json:"items"`
		} `json:"result"`
	} `json:"tasks"`
}

const (
	onPageMaxTasksPerRequest = 20
	onPageMaxTasksPerDomain  = 5
)

// Render executes each unique app-shell URL once. The stable audit key and the
// audit repository's render claim prevent a restarted job from buying a repeat.
func (r DataForSEORenderer) Render(ctx context.Context, organizationID, auditID string, urls []string) (map[string]RenderedHTML, error) {
	if r.Client == nil || strings.TrimSpace(organizationID) == "" || strings.TrimSpace(auditID) == "" {
		return nil, fmt.Errorf("DataForSEO renderer is not configured")
	}
	if len(urls) == 0 {
		return map[string]RenderedHTML{}, nil
	}
	if len(urls) > MaxRenderedPagesPerAudit {
		return nil, fmt.Errorf("at most %d pages may be rendered in one audit", MaxRenderedPagesPerAudit)
	}
	unique := make([]string, 0, len(urls))
	seen := map[string]bool{}
	for _, rawURL := range urls {
		parsed, err := url.Parse(rawURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || seen[rawURL] {
			if seen[rawURL] {
				continue
			}
			return nil, fmt.Errorf("invalid or duplicate renderer URL")
		}
		seen[rawURL] = true
		unique = append(unique, rawURL)
	}
	batches := batchRenderURLs(unique)
	ctx = spend.WithStableIdempotencyKey(ctx, "site-audit-javascript:"+auditID)
	pages := map[string]RenderedHTML{}
	err := r.Client.DoBudgetedWithDelivery(ctx, organizationID, spend.SiteAudit, dataforseo.OnPageJavaScriptCostUSD(len(unique)), func(request dataforseo.DoBudgetedRequest) (float64, error) {
		// Each batch's raw HTML is read right after it is rendered, so a later
		// batch failing never throws away pages that were already paid for.
		var stopErr error
		for _, batch := range batches {
			if stopErr = renderBatch(request, batch, pages); stopErr != nil {
				break
			}
		}
		if len(pages) == 0 {
			return 0, errors.Join(dataforseo.ErrNothingDelivered, stopErr)
		}
		var deliveredUSD float64
		for _, page := range pages {
			deliveredUSD += page.CostUSD
		}
		return deliveredUSD, stopErr
	})
	if err != nil {
		return pages, err
	}
	return pages, nil
}

// renderBatch buys one Instant Pages request and reads the raw HTML of its
// successful tasks through the free retrieval endpoint, adding every usable
// page to pages. An error means no further batch should be bought.
func renderBatch(request dataforseo.DoBudgetedRequest, batch []string, pages map[string]RenderedHTML) error {
	inputs := make([]renderTaskInput, len(batch))
	for i, pageURL := range batch {
		inputs[i] = renderTaskInput{URL: pageURL, EnableJavaScript: true, StoreRawHTML: true}
	}
	body, err := json.Marshal(inputs)
	if err != nil {
		return fmt.Errorf("encode On-Page JavaScript request: %w", err)
	}
	paidResponse, err := request(http.MethodPost, onPageInstantPagesPath, body, false)
	if err != nil {
		return err
	}
	var taskResult renderTaskResponse
	if err := json.Unmarshal(paidResponse, &taskResult); err != nil {
		return fmt.Errorf("decode On-Page JavaScript response: %w", err)
	}
	if taskResult.StatusCode != 20000 {
		return nil
	}
	taskIDs := map[string]string{}
	taskCosts := map[string]float64{}
	pageStatuses := map[string]int{}
	for _, task := range taskResult.Tasks {
		if task.ID == "" || task.StatusCode != 20000 || !slices.Contains(batch, task.Data.URL) {
			continue
		}
		pageURL := task.Data.URL
		for _, result := range task.Result {
			for _, item := range result.Items {
				if item.ResourceType != "html" || item.StatusCode < 200 || item.StatusCode >= 300 || task.Cost <= 0 {
					continue
				}
				taskIDs[task.ID] = pageURL
				taskCosts[pageURL] = task.Cost
				pageStatuses[pageURL] = item.StatusCode
				break
			}
			if taskIDs[task.ID] != "" {
				break
			}
		}
	}
	if len(taskIDs) == 0 {
		return nil
	}
	rawInputs := make([]rawHTMLInput, 0, len(taskIDs))
	for id := range taskIDs {
		rawInputs = append(rawInputs, rawHTMLInput{ID: id})
	}
	slices.SortFunc(rawInputs, func(a, b rawHTMLInput) int { return strings.Compare(a.ID, b.ID) })
	rawBody, err := json.Marshal(rawInputs)
	if err != nil {
		return fmt.Errorf("encode raw HTML request: %w", err)
	}
	rawResponse, err := request(http.MethodPost, onPageRawHTMLPath, rawBody, false)
	if err != nil {
		return err
	}
	var htmlResult rawHTMLResponse
	if err := json.Unmarshal(rawResponse, &htmlResult); err != nil {
		return fmt.Errorf("decode raw HTML response: %w", err)
	}
	if htmlResult.StatusCode != 20000 {
		return fmt.Errorf("DataForSEO raw HTML request failed with status %d", htmlResult.StatusCode)
	}
	for _, task := range htmlResult.Tasks {
		pageURL := taskIDs[task.ID]
		if pageURL == "" || task.StatusCode != 20000 || len(task.Result) == 0 || strings.TrimSpace(task.Result[0].Items.HTML) == "" {
			continue
		}
		html := task.Result[0].Items.HTML
		if len(html) > MaxHTMLBytes {
			html = html[:MaxHTMLBytes]
		}
		status := pageStatuses[pageURL]
		if AnalyzeHTML(html, pageURL, status, 0, "").JavaScriptShell {
			continue
		}
		pages[pageURL] = RenderedHTML{URL: pageURL, StatusCode: status, HTML: html, CostUSD: taskCosts[pageURL]}
	}
	return nil
}

// batchRenderURLs respects DataForSEO's Instant Pages limit of 20 tasks and
// five URLs per domain in one request. A site's twenty same-domain shells are
// therefore four paid requests, each followed by one free raw HTML retrieval.
func batchRenderURLs(urls []string) [][]string {
	batches := make([][]string, 0, (len(urls)+onPageMaxTasksPerRequest-1)/onPageMaxTasksPerRequest)
	batch := make([]string, 0, onPageMaxTasksPerRequest)
	domains := map[string]int{}
	flush := func() {
		if len(batch) == 0 {
			return
		}
		batches = append(batches, batch)
		batch = make([]string, 0, onPageMaxTasksPerRequest)
		domains = map[string]int{}
	}
	for _, rawURL := range urls {
		parsed, err := url.Parse(rawURL)
		if err != nil {
			continue
		}
		domain := strings.ToLower(parsed.Hostname())
		if registrable, suffixErr := publicsuffix.EffectiveTLDPlusOne(domain); suffixErr == nil {
			domain = registrable
		}
		if len(batch) == onPageMaxTasksPerRequest || domains[domain] == onPageMaxTasksPerDomain {
			flush()
		}
		batch = append(batch, rawURL)
		domains[domain]++
	}
	flush()
	return batches
}
