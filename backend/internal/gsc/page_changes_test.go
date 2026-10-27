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
	got := buildPageChanges(current, previous, 340, 10)
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
	if limited := buildPageChanges(current, previous, 340, 2); len(limited) != 2 {
		t.Fatalf("limit ignored: %d", len(limited))
	}
	if empty := buildPageChanges(nil, nil, 0, 5); empty == nil || len(empty) != 0 {
		t.Fatalf("no data must be an empty list, not null: %#v", empty)
	}
}

func TestSeverityScalesWithTheSite(t *testing.T) {
	tests := []struct {
		name                   string
		before, after, siteWas float64
		want                   string
	}{
		{"half lost on a big part of the site", 40, 10, 400, SeverityUrgent},
		{"all lost", 30, 0, 300, SeverityUrgent},
		{"exactly half at the size floor", 5, 2.5, 100, SeverityUrgent},
		{"tiny row on a big site is not urgent", 4, 0, 10000, SeverityWatch},
		{"under 5% of a big site is not urgent", 40, 0, 1000, SeverityWatch},
		{"20% drop", 100, 80, 1000, SeverityWatch},
		{"small dip", 100, 90, 1000, SeverityMinor},
		{"no loss", 10, 10, 100, ""},
		{"growth", 10, 20, 100, ""},
		{"nothing before", 0, 0, 100, ""},
	}
	for _, tt := range tests {
		if got := severity(tt.before, tt.after, tt.siteWas); got != tt.want {
			t.Errorf("%s: severity(%v, %v, %v) = %q, want %q", tt.name, tt.before, tt.after, tt.siteWas, got, tt.want)
		}
	}
}

func TestBuildQueryChangesListsLossesAndGains(t *testing.T) {
	previous := []SearchRow{
		pageRow("hotel lonavala", 50, 1000, .05, 3),
		pageRow("resort pool", 8, 200, .04, 6),
		pageRow("steady", 10, 100, .1, 4),
	}
	current := []SearchRow{
		pageRow("hotel lonavala", 10, 900, .011, 7),
		pageRow("steady", 10, 100, .1, 4),
		pageRow("new search", 12, 300, .04, 5),
		pageRow("resort pool", 9, 220, .041, 6),
	}
	losses, gains := buildQueryChanges(current, previous, 68)
	if len(losses) != 1 || losses[0].Query != "hotel lonavala" || losses[0].PrevClicks != 50 || losses[0].Clicks != 10 ||
		losses[0].Reason != ReasonRankFell || losses[0].Severity != SeverityUrgent {
		t.Fatalf("losses = %+v", losses)
	}
	if len(gains) != 2 || gains[0].Query != "new search" || gains[0].PrevClicks != 0 || gains[1].Query != "resort pool" ||
		gains[0].Reason != "" || gains[0].Severity != "" {
		t.Fatalf("gains = %+v; want new and growing searches, biggest first, without a loss reason", gains)
	}
	if l, g := buildQueryChanges(nil, nil, 0); l == nil || g == nil || len(l)+len(g) != 0 {
		t.Fatalf("no data must give empty lists, not null: %#v %#v", l, g)
	}
}
