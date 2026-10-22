package billing

import (
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

// The hint next to a button must not promise much less than the click costs.
// The figures are what the provider billed for one live click on 16-19 Oct 2026
// (database rows in go_dataforseo_usage), summed over the tasks of that click.
func TestTypicalCostHintsAreNotBelowWhatALiveClickBilled(t *testing.T) {
	billedPerClick := map[string]float64{
		string(spend.KeywordResearch): 0.0461,  // suggestions + ideas + SERP
		string(spend.DomainResearch):  0.03012, // domain_rank_overview 0.01212 + ranked_keywords 0.018
		// 21 Oct: summary 0.024036 + history 0.024432 + a 1,000-row table at the
		// measured $0.024 + $0.000036 a row (200 rows billed 0.0312).
		string(spend.Backlinks): 0.024036 + 0.024432 + 0.024 + 1000*0.000036,
		// Brand lookup, 22 Oct (dellaresorts.com, India): target metrics 0.101 +
		// the Google prompt sample of 82 rows 0.182.
		string(spend.AIVisibility) + ":" + ActionBrandLookup: 0.101 + 0.182,
	}
	hints := map[string]int64{}
	for _, cost := range ActionCosts() {
		key := string(cost.Feature)
		if cost.Action != "" {
			key += ":" + cost.Action
		}
		hints[key] = cost.Credits
	}
	for key, usd := range billedPerClick {
		billed := CreditsForUSD(usd)
		if hints[key] < billed || hints[key] > billed*5/4 {
			t.Errorf("%s hint = %d credits, want %d to %d (what a live click billed, at most 25%% above)", key, hints[key], billed, billed*5/4)
		}
	}
}

func TestRankTrackingTypicalCostUsesThePublishedTop30QueuedRate(t *testing.T) {
	for _, cost := range ActionCosts() {
		if cost.Feature == spend.RankTracking {
			if cost.Credits != CreditsForUSD(dataforseo.SerpCostUSD(30, true)) {
				t.Fatalf("rank tracking typical cost = %d credits, want the published top-30 Standard rate", cost.Credits)
			}
			return
		}
	}
	t.Fatal("rank tracking typical cost is missing")
}
