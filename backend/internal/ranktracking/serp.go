package ranktracking

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

// CheckRequest is one live SERP lookup: where a domain ranks for a keyword.
type CheckRequest struct {
	Keyword      string
	Device       string // desktop or mobile
	LocationCode int
	LanguageCode string
	LocationName *string // a city; wins over LocationCode when set
	TargetDomain string
	Depth        int
}

// CompetitorResult is one organic result in the SERP, kept so the customer can
// see who ranks above them, not just their own position.
type CompetitorResult struct {
	Rank   int    `json:"rank"`
	URL    string `json:"url"`
	Domain string `json:"domain"`
	Title  string `json:"title"`
}

// CheckResult is the organic position of the target, nil when it is not in the depth.
type CheckResult struct {
	Position     *int
	URL          *string
	SerpFeatures []string
	// Competitors are the top organic results (including the target) the crawl
	// returned, so a rank check keeps the whole SERP it already paid for.
	Competitors []CompetitorResult
}

// SerpProvider runs live SERP lookups for an organization.
type SerpProvider interface {
	CheckLive(ctx context.Context, organizationID string, req CheckRequest) (CheckResult, error)
}

const (
	serpLivePath = "/v3/serp/google/organic/live/advanced"
	// noResultsStatus is a billed, valid empty SERP, common for new or obscure
	// keywords. It means "not ranking", not a failure.
	noResultsStatus = 40501
)

// DataForSEOSerp implements SerpProvider on the shared DataForSEO client, which
// also records each call's cost against the organization.
type DataForSEOSerp struct{ Client *dataforseo.Client }

// CheckLive asks Google for the keyword at the config's depth. It stops the
// crawl once the target is found among organic results, because the provider
// bills only the pages it crawled.
func (p DataForSEOSerp) CheckLive(ctx context.Context, organizationID string, req CheckRequest) (CheckResult, error) {
	if req.Depth < 10 || req.Depth > 100 || req.Depth%10 != 0 {
		return CheckResult{}, fmt.Errorf("rank check depth %d is not a provider depth", req.Depth)
	}
	task := map[string]any{
		"keyword":       req.Keyword,
		"language_code": req.LanguageCode,
		"device":        req.Device,
		"os":            map[string]string{"desktop": "windows", "mobile": "android"}[req.Device],
		"depth":         req.Depth,
		"stop_crawl_on_match": []map[string]string{
			{"match_value": req.TargetDomain, "match_type": "with_subdomains"},
		},
		"find_targets_in": []string{"organic"},
	}
	if req.LocationName != nil && *req.LocationName != "" {
		task["location_name"] = *req.LocationName
	} else {
		task["location_code"] = req.LocationCode
	}
	body, err := json.Marshal([]map[string]any{task})
	if err != nil {
		return CheckResult{}, fmt.Errorf("encode rank check request: %w", err)
	}
	// Not retry-safe: a repeated billed call would be charged twice.
	response, err := p.Client.Do(ctx, organizationID, spend.RankTracking, http.MethodPost, serpLivePath, body, false)
	if err != nil {
		return CheckResult{}, fmt.Errorf("DataForSEO %s: %w", serpLivePath, err)
	}
	results, err := dataforseo.Results(response)
	if taskErr, ok := errors.AsType[*dataforseo.TaskError](err); ok && taskErr.StatusCode == noResultsStatus {
		return CheckResult{SerpFeatures: []string{}}, nil
	}
	if err != nil {
		return CheckResult{}, fmt.Errorf("DataForSEO %s: %w", serpLivePath, err)
	}
	return parseSerp(results, req.TargetDomain)
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func parseSerp(results []json.RawMessage, targetDomain string) (CheckResult, error) {
	out := CheckResult{SerpFeatures: []string{}}
	if len(results) == 0 {
		return out, nil
	}
	var result struct {
		Items []struct {
			Type         string  `json:"type"`
			RankGroup    *int    `json:"rank_group"`
			RankAbsolute *int    `json:"rank_absolute"`
			Domain       *string `json:"domain"`
			URL          *string `json:"url"`
			Title        *string `json:"title"`
		} `json:"items"`
	}
	if err := json.Unmarshal(results[0], &result); err != nil {
		return CheckResult{}, fmt.Errorf("decode SERP result: %w", err)
	}
	target := strings.ToLower(targetDomain)
	seen, matched := map[string]bool{}, false
	for _, item := range result.Items {
		if item.Type != "" && !seen[item.Type] {
			seen[item.Type] = true
			out.SerpFeatures = append(out.SerpFeatures, item.Type)
		}
		// Keep the top organic results (including the target) so the paid crawl
		// is not thrown away; capped at 20.
		if item.Type == "organic" && len(out.Competitors) < 20 {
			rank := 0
			if item.RankGroup != nil {
				rank = *item.RankGroup
			} else if item.RankAbsolute != nil {
				rank = *item.RankAbsolute
			}
			out.Competitors = append(out.Competitors, CompetitorResult{
				Rank: rank, URL: derefStr(item.URL), Domain: derefStr(item.Domain), Title: derefStr(item.Title),
			})
		}
		if matched || item.Type != "organic" || item.Domain == nil {
			continue
		}
		domain := strings.ToLower(*item.Domain)
		if domain != target && !strings.HasSuffix(domain, "."+target) {
			continue
		}
		// rank_group counts organic results only, which is what people mean by
		// "my ranking"; rank_absolute also counts local packs, PAA and AI boxes.
		matched = true
		out.Position, out.URL = item.RankGroup, item.URL
		if out.Position == nil {
			out.Position = item.RankAbsolute
		}
	}
	return out, nil
}
