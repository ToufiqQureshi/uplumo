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
		string(spend.KeywordResearch): 0.0461,            // suggestions + ideas + SERP
		string(spend.DomainResearch):  0.03012,           // domain_rank_overview 0.01212 + ranked_keywords 0.018
		string(spend.Backlinks):       0.024036 + 0.0258, // summary + backlinks rows
		string(spend.LocalSEO):        9 * 0.002,         // nine maps searches
		// Brand lookup, 16 Oct, per platform: mentions search 0.151, aggregated 0.101, top pages 0.101.
		string(spend.AIVisibility) + ":" + ActionBrandLookup: 0.151 + 0.101 + 0.101,
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
