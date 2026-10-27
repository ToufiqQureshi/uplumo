package gsc

import (
	"cmp"
	"slices"
)

// pageChangeLimit is how many losing pages and searches the report explains.
const pageChangeLimit = 5

// gainLimit is how many growing searches the report shows next to the losses.
const gainLimit = 3

// pageRowLimit bounds each period's page and query reads; losers outside the
// top 1,000 by clicks are too small to lead the list.
const pageRowLimit = 1000

// Why a page or search lost clicks, from its own Search Console numbers.
const (
	ReasonRankFell      = "rank_fell"      // average position worsened by 1 or more
	ReasonFewerSearches = "fewer_searches" // shown at least 20% less at a similar rank
	ReasonCTRFell       = "ctr_fell"       // shown as often, clicked at least 20% less
	ReasonMixed         = "mixed"          // small changes in several places
)

// How urgent a loss is for this site.
const (
	SeverityUrgent = "urgent" // lost half or more, and it mattered to the site
	SeverityWatch  = "watch"  // lost 20% or more
	SeverityMinor  = "minor"  // a small dip
)

// ChangeMetrics are one row's numbers in both periods, why it changed and how
// urgent the change is. Gains carry an empty Reason and Severity.
type ChangeMetrics struct {
	Clicks          float64 `json:"clicks"`
	PrevClicks      float64 `json:"prevClicks"`
	Impressions     float64 `json:"impressions"`
	PrevImpressions float64 `json:"prevImpressions"`
	Position        float64 `json:"position"`
	PrevPosition    float64 `json:"prevPosition"`
	CTR             float64 `json:"ctr"`
	PrevCTR         float64 `json:"prevCtr"`
	Reason          string  `json:"reason,omitempty"`
	Severity        string  `json:"severity,omitempty"`
}

// PageChange is one page that changed clicks, before and after.
type PageChange struct {
	Page string `json:"page"`
	ChangeMetrics
}

// QueryChange is one search that changed clicks, before and after.
type QueryChange struct {
	Query string `json:"query"`
	ChangeMetrics
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

// severity rates a loss against the whole site's previous clicks. A loss is
// urgent when the row lost at least half its clicks and those clicks were a
// real part of the site: 5 or more, and at least 5% of the site's clicks. So
// 2 -> 0 on a big site is minor, and 40 -> 10 on any site is urgent.
func severity(before, after, sitePrevClicks float64) string {
	if before <= 0 || after >= before {
		return ""
	}
	drop := (before - after) / before
	switch {
	case drop >= 0.5 && before >= 5 && before >= sitePrevClicks*0.05:
		return SeverityUrgent
	case drop >= 0.2:
		return SeverityWatch
	default:
		return SeverityMinor
	}
}

type keyedChange struct {
	key string
	ChangeMetrics
}

// buildChanges compares two periods of single-key rows. Losses are the rows
// that lost the most clicks, biggest first; a row missing from the current
// period lost all its clicks. Gains are the rows that won the most clicks,
// including new ones.
func buildChanges(current, previous []SearchRow, sitePrevClicks float64, lossLimit, gainCount int) (losses, gains []keyedChange) {
	now := make(map[string]SearchRow, len(current))
	for _, row := range current {
		if len(row.Keys) == 1 {
			now[row.Keys[0]] = row
		}
	}
	before := make(map[string]SearchRow, len(previous))
	for _, row := range previous {
		if len(row.Keys) == 1 {
			before[row.Keys[0]] = row
		}
	}
	metrics := func(after, prior SearchRow) ChangeMetrics {
		return ChangeMetrics{
			Clicks: after.Clicks, PrevClicks: prior.Clicks,
			Impressions: after.Impressions, PrevImpressions: prior.Impressions,
			Position: after.Position, PrevPosition: prior.Position,
			CTR: after.CTR, PrevCTR: prior.CTR,
		}
	}
	losses, gains = []keyedChange{}, []keyedChange{}
	for key, prior := range before {
		after := now[key]
		if after.Clicks >= prior.Clicks {
			continue
		}
		m := metrics(after, prior)
		m.Reason, m.Severity = pageReason(after, prior), severity(prior.Clicks, after.Clicks, sitePrevClicks)
		losses = append(losses, keyedChange{key, m})
	}
	for key, after := range now {
		prior := before[key]
		if after.Clicks > prior.Clicks {
			gains = append(gains, keyedChange{key, metrics(after, prior)})
		}
	}
	byLoss := func(a, b keyedChange) int {
		return cmp.Or(cmp.Compare(b.PrevClicks-b.Clicks, a.PrevClicks-a.Clicks), cmp.Compare(a.key, b.key))
	}
	byGain := func(a, b keyedChange) int {
		return cmp.Or(cmp.Compare(b.Clicks-b.PrevClicks, a.Clicks-a.PrevClicks), cmp.Compare(a.key, b.key))
	}
	slices.SortFunc(losses, byLoss)
	slices.SortFunc(gains, byGain)
	return losses[:min(len(losses), lossLimit)], gains[:min(len(gains), gainCount)]
}

// buildPageChanges returns the pages that lost the most clicks between two
// periods (rows keyed by page), biggest loss first.
func buildPageChanges(current, previous []SearchRow, sitePrevClicks float64, limit int) []PageChange {
	losses, _ := buildChanges(current, previous, sitePrevClicks, limit, 0)
	out := make([]PageChange, 0, len(losses))
	for _, c := range losses {
		out = append(out, PageChange{Page: c.key, ChangeMetrics: c.ChangeMetrics})
	}
	return out
}

// buildQueryChanges returns the searches that lost and won the most clicks.
// Anonymized searches are missing from both periods, so these rows explain
// named searches only; the site totals stay the source of truth.
func buildQueryChanges(current, previous []SearchRow, sitePrevClicks float64) (losses, gains []QueryChange) {
	lost, won := buildChanges(current, previous, sitePrevClicks, pageChangeLimit, gainLimit)
	losses, gains = make([]QueryChange, 0, len(lost)), make([]QueryChange, 0, len(won))
	for _, c := range lost {
		losses = append(losses, QueryChange{Query: c.key, ChangeMetrics: c.ChangeMetrics})
	}
	for _, c := range won {
		gains = append(gains, QueryChange{Query: c.key, ChangeMetrics: c.ChangeMetrics})
	}
	return losses, gains
}
