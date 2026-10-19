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
	billedPerClick := map[spend.Feature]float64{
		spend.KeywordResearch: 0.0461,            // suggestions + ideas + SERP
		spend.DomainResearch:  0.03012,           // domain_rank_overview 0.01212 + ranked_keywords 0.018
		spend.Backlinks:       0.024036 + 0.0258, // summary + backlinks rows
		spend.LocalSEO:        9 * 0.002,         // nine maps searches
	}
	hints := map[spend.Feature]int64{}
	for _, cost := range ActionCosts() {
		hints[cost.Feature] = cost.Credits
	}
	for feature, usd := range billedPerClick {
		billed := CreditsForUSD(usd)
		if hints[feature] < billed || hints[feature] > billed*5/4 {
			t.Errorf("%s hint = %d credits, want %d to %d (what a live click billed, at most 25%% above)", feature, hints[feature], billed, billed*5/4)
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
