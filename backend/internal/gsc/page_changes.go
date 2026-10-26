package gsc

import (
	"cmp"
	"slices"
)

// pageChangeLimit is how many losing pages the report explains.
const pageChangeLimit = 5

// pageRowLimit bounds each period's page query; losers outside the top 1,000
// pages by clicks are too small to lead the list.
const pageRowLimit = 1000

// Why a page lost clicks, from its own Search Console numbers.
const (
	ReasonRankFell      = "rank_fell"      // average position worsened by 1 or more
	ReasonFewerSearches = "fewer_searches" // shown at least 20% less at a similar rank
	ReasonCTRFell       = "ctr_fell"       // shown as often, clicked at least 20% less
	ReasonMixed         = "mixed"          // small changes in several places
)

// PageChange is one page that lost clicks, before and after.
type PageChange struct {
	Page            string  `json:"page"`
	Clicks          float64 `json:"clicks"`
	PrevClicks      float64 `json:"prevClicks"`
	Impressions     float64 `json:"impressions"`
	PrevImpressions float64 `json:"prevImpressions"`
	Position        float64 `json:"position"`
	PrevPosition    float64 `json:"prevPosition"`
	CTR             float64 `json:"ctr"`
	PrevCTR         float64 `json:"prevCtr"`
	Reason          string  `json:"reason"`
}

// pageReason explains one page's drop. Rank is checked first: a lower rank
// also cuts impressions and CTR, so it is the root cause when present.
func pageReason(now, before SearchRow) string {
	switch {
	case now.Impressions == 0 || before.Position > 0 && now.Position-before.Position >= 1:
		return ReasonRankFell
	case before.Impressions > 0 && now.Impressions <= before.Impressions*0.8:
		return ReasonFewerSearches
	case before.CTR > 0 && now.CTR <= before.CTR*0.8:
		return ReasonCTRFell
	default:
		return ReasonMixed
	}
}

// buildPageChanges returns the pages that lost the most clicks between two
// periods (rows keyed by page), biggest loss first. A page missing from the
// current period lost all its clicks.
func buildPageChanges(current, previous []SearchRow, limit int) []PageChange {
	now := make(map[string]SearchRow, len(current))
	for _, row := range current {
		if len(row.Keys) == 1 {
			now[row.Keys[0]] = row
		}
	}
	changes := []PageChange{}
	for _, before := range previous {
		if len(before.Keys) != 1 {
			continue
		}
		page := before.Keys[0]
		after := now[page]
		if after.Clicks >= before.Clicks {
			continue
		}
		changes = append(changes, PageChange{
			Page: page, Clicks: after.Clicks, PrevClicks: before.Clicks,
			Impressions: after.Impressions, PrevImpressions: before.Impressions,
			Position: after.Position, PrevPosition: before.Position,
			CTR: after.CTR, PrevCTR: before.CTR,
			Reason: pageReason(after, before),
		})
	}
	slices.SortStableFunc(changes, func(a, b PageChange) int {
		return cmp.Or(
			cmp.Compare(b.PrevClicks-b.Clicks, a.PrevClicks-a.Clicks),
			cmp.Compare(a.Page, b.Page),
		)
	})
	return changes[:min(len(changes), limit)]
}
