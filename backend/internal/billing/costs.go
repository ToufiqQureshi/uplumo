package billing

import (
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

// ActionCost is about what one action of a feature costs, for the Billing
// screen and the hints next to each action. It is a typical figure from
// DataForSEO's and the AI models' published prices: the real cost is charged
// per request, and a request is held at its maximum before it is sent.
type ActionCost struct {
	Feature spend.Feature `json:"feature"`
	// Unit is the action the credits are for, in the customer's words.
	Unit string `json:"unit"`
	// Credits is the typical cost of one unit.
	Credits int64 `json:"credits"`
	// Action tells apart actions of one feature that cost very differently,
	// such as a brand lookup and a prompt explorer run. It is empty when the
	// feature has one action.
	Action string `json:"action,omitempty"`
}

// typicalUSD is the provider cost of one typical action, per feature. The
// lookups were measured on the live site (what the provider billed for one
// click), not taken from a price list, because one click buys several tasks:
// a keyword search is suggestions, ideas and a SERP; a domain report is a rank
// overview and its keyword rows; a backlink report is a summary and its rows;
// a brand lookup is six AI-mentions
// tasks (two platforms of search $0.151, aggregated $0.101, top pages $0.101).
var typicalUSD = []struct {
	feature spend.Feature
	action  string
	unit    string
	usd     float64
}{
	{spend.RankTracking, "", "keyword check (top 30, scheduled)", dataforseo.SerpCostUSD(30, true)},
	{spend.KeywordResearch, "", "keyword search", 0.047},
	{spend.DomainResearch, "", "domain report tab", 0.031},
	// Summary $0.024 + history $0.024 + the first 1,000 table rows $0.06.
	{spend.Backlinks, "", "backlink report", 0.108},
	{spend.AIVisibility, ActionPromptExplorer, "prompt on 4 AI models", 0.1},
	{spend.AIVisibility, ActionBrandLookup, "brand lookup (more with competitors)", 0.71},
	{spend.AIAssistant, "", "AI assistant reply", 0.03},
	{spend.AIWriting, "", "AI writing action", 0.01},
}

// Actions of the AI visibility feature that ActionCost tells apart.
const (
	ActionPromptExplorer = "prompt_explorer"
	ActionBrandLookup    = "brand_lookup"
)

// ActionCosts lists the typical cost of every feature that spends credits.
func ActionCosts() []ActionCost {
	out := make([]ActionCost, len(typicalUSD))
	for i, c := range typicalUSD {
		out[i] = ActionCost{Feature: c.feature, Action: c.action, Unit: c.unit, Credits: CreditsForUSD(c.usd)}
	}
	return out
}

func featureNames() []string {
	features := spend.Features()
	names := make([]string, len(features))
	for i, f := range features {
		names[i] = string(f)
	}
	return names
}
