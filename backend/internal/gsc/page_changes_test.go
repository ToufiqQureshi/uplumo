package gsc

import (
	"slices"
	"testing"
)

func pageRow(page string, clicks, impressions, ctr, position float64) SearchRow {
	return SearchRow{Keys: []string{page}, Clicks: clicks, Impressions: impressions, CTR: ctr, Position: position}
}

func TestBuildPageChangesExplainsTheBiggestLosses(t *testing.T) {
	previous := []SearchRow{
		pageRow("/rank", 100, 1000, .10, 3),
		pageRow("/searches", 80, 1000, .08, 5),
		pageRow("/ctr", 60, 1000, .06, 4),
		pageRow("/mixed", 40, 1000, .04, 6),
		pageRow("/gone", 30, 300, .10, 8),
		pageRow("/grew", 10, 100, .10, 9),
		pageRow("/same", 20, 200, .10, 5),
		{Keys: []string{"bad", "row"}, Clicks: 999},
	}
	current := []SearchRow{
		pageRow("/rank", 40, 700, .057, 4.5),
		pageRow("/searches", 50, 600, .083, 5.2),
		pageRow("/ctr", 42, 1000, .042, 4.1), // CTR down 30%
		pageRow("/mixed", 35, 900, .039, 6.3),
		pageRow("/grew", 20, 150, .13, 7),
		pageRow("/same", 20, 300, .07, 9),
	}
	got := buildPageChanges(current, previous, 10)
	// Equal losses (30 clicks) are ordered by page for a stable list.
	var order, reasons []string
	for _, change := range got {
		order = append(order, change.Page)
		reasons = append(reasons, change.Reason)
	}
	if !slices.Equal(order, []string{"/rank", "/gone", "/searches", "/ctr", "/mixed"}) {
		t.Fatalf("order = %v", order)
	}
	if !slices.Equal(reasons, []string{ReasonRankFell, ReasonRankFell, ReasonFewerSearches, ReasonCTRFell, ReasonMixed}) {
		t.Fatalf("reasons = %v", reasons)
	}
	if got[1].Clicks != 0 || got[1].PrevClicks != 30 {
		t.Fatalf("a page that vanished lost all its clicks: %+v", got[1])
	}
	if limited := buildPageChanges(current, previous, 2); len(limited) != 2 {
		t.Fatalf("limit ignored: %d", len(limited))
	}
	if empty := buildPageChanges(nil, nil, 5); empty == nil || len(empty) != 0 {
		t.Fatalf("no data must be an empty list, not null: %#v", empty)
	}
}
