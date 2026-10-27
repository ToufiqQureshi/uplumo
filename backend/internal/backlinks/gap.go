package backlinks

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	pathGap = "/v3/backlinks/domain_intersection/live"
	// GapRows is how many sites one gap lookup buys: $0.024 + 100 x $0.000036.
	GapRows        = 100
	maxCompetitors = 3
	gapTTL         = 7 * 24 * time.Hour
)

// GapRow is a site that links to at least one competitor and never to the
// customer's own domain.
type GapRow struct {
	Domain string `json:"domain"`
	// Rank is the linking site's own authority, 0-100.
	Rank float64 `json:"rank"`
	// Links[i] is how many links the site has to competitor i (0 = none).
	Links []int `json:"links"`
}

// GapResult is one gap lookup, with when it was bought.
type GapResult struct {
	Competitors []string  `json:"competitors"`
	Rows        []GapRow  `json:"rows"`
	FetchedAt   time.Time `json:"fetchedAt"`
}

type gapRequest struct {
	Own         string   `json:"own"`
	Competitors []string `json:"competitors"`
	// All asks only for sites that link to every competitor.
	All bool `json:"all"`
}

type providerGapSide struct {
	Target    string   `json:"target"`
	Rank      *float64 `json:"rank"`
	Backlinks *float64 `json:"backlinks"`
}

type providerGapItem struct {
	DomainIntersection map[string]*providerGapSide `json:"domain_intersection"`
}

// Gap lists sites that link to the competitors but not to own. The result is
// cached for seven days per workspace, project inputs and mode, so repeating
// the same lookup is free.
func (s *Service) Gap(ctx context.Context, org string, in gapRequest) (GapResult, error) {
	own, err := normalizeTarget(in.Own, string(ScopeDomain))
	if err != nil {
		return GapResult{}, inputError("Enter your own domain.")
	}
	if len(in.Competitors) == 0 || len(in.Competitors) > maxCompetitors {
		return GapResult{}, inputError("Enter 1 to 3 competitor domains.")
	}
	var competitors []string
	for _, c := range in.Competitors {
		t, err := normalizeTarget(c, string(ScopeDomain))
		if err != nil {
			return GapResult{}, inputError(fmt.Sprintf("%q is not a valid competitor domain.", c))
		}
		if t.APITarget == own.APITarget || slices.Contains(competitors, t.APITarget) {
			return GapResult{}, inputError("Competitors must differ from each other and from your domain.")
		}
		competitors = append(competitors, t.APITarget)
	}
	sorted := slices.Sorted(slices.Values(competitors))
	digest := sha256.Sum256([]byte(strings.Join(append([]string{own.APITarget, strconv.FormatBool(in.All)}, sorted...), "\n")))
	key := "backlinks:gap:" + org + ":" + hex.EncodeToString(digest[:16])
	if raw, err := s.cache.Get(ctx, key); err == nil {
		var cached GapResult
		if json.Unmarshal(raw, &cached) == nil && slices.Equal(slices.Sorted(slices.Values(cached.Competitors)), sorted) {
			return reorderGap(cached, competitors), nil
		}
	}
	// Two identical clicks in one process share one purchase.
	v, err, _ := s.flights.Do(key, func() (any, error) {
		return s.buyGap(context.WithoutCancel(ctx), org, own.APITarget, competitors, in.All)
	})
	if err != nil {
		return GapResult{}, err
	}
	got, _ := v.(GapResult)
	if raw, err := json.Marshal(got); err == nil {
		if err := s.cache.Set(ctx, key, raw, gapTTL); err != nil {
			s.logger.WarnContext(ctx, "cache backlink gap", "err", err)
		}
	}
	return reorderGap(got, competitors), nil
}

func (s *Service) buyGap(ctx context.Context, org, own string, competitors []string, all bool) (GapResult, error) {
	targets := map[string]string{}
	for i, c := range competitors {
		targets[strconv.Itoa(i+1)] = c
	}
	mode := "partial"
	if all {
		mode = "all"
	}
	payload := map[string]any{
		"targets":                    targets,
		"exclude_targets":            []string{own},
		"intersection_mode":          mode,
		"limit":                      GapRows,
		"order_by":                   []string{"1.rank,desc"},
		"include_subdomains":         true,
		"exclude_internal_backlinks": true,
		"backlinks_status_type":      "live",
		"rank_scale":                 "one_hundred",
	}
	results, err := s.provider.post(ctx, org, pathGap, []map[string]any{payload})
	if err != nil {
		return GapResult{}, err
	}
	items, err := decodeItems[providerGapItem](pathGap, results)
	if err != nil {
		return GapResult{}, err
	}
	return GapResult{Competitors: competitors, Rows: shapeGap(items, len(competitors)), FetchedAt: s.now().UTC()}, nil
}

// shapeGap turns provider items into rows, ordered by the linking site's
// authority. A side the site does not link to is null in partial mode.
func shapeGap(items []providerGapItem, n int) []GapRow {
	rows := make([]GapRow, 0, len(items))
	for _, item := range items {
		row := GapRow{Links: make([]int, n)}
		for i := range n {
			side := item.DomainIntersection[strconv.Itoa(i+1)]
			if side == nil {
				continue
			}
			if row.Domain == "" {
				row.Domain = side.Target
			}
			if side.Rank != nil && *side.Rank > row.Rank {
				row.Rank = *side.Rank
			}
			if side.Backlinks != nil {
				row.Links[i] = int(*side.Backlinks)
			}
		}
		if row.Domain != "" {
			rows = append(rows, row)
		}
	}
	slices.SortStableFunc(rows, func(a, b GapRow) int {
		return cmp.Compare(b.Rank, a.Rank)
	})
	return rows
}

// reorderGap returns the cached result in the caller's competitor order.
func reorderGap(r GapResult, order []string) GapResult {
	if slices.Equal(r.Competitors, order) {
		return r
	}
	index := make([]int, len(order))
	for i, c := range order {
		index[i] = slices.Index(r.Competitors, c)
	}
	rows := make([]GapRow, len(r.Rows))
	for j, row := range r.Rows {
		links := make([]int, len(order))
		for i, k := range index {
			if k >= 0 {
				links[i] = row.Links[k]
			}
		}
		rows[j] = GapRow{Domain: row.Domain, Rank: row.Rank, Links: links}
	}
	return GapResult{Competitors: order, Rows: rows, FetchedAt: r.FetchedAt}
}

func gapHandler(d Deps) http.HandlerFunc {
	return handle(d, func(w http.ResponseWriter, r *http.Request, org string) (any, error) {
		var in gapRequest
		if err := decode(w, r, &in); err != nil {
			return nil, err
		}
		return d.Service.Gap(r.Context(), org, in)
	})
}
