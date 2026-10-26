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
	// MaxCredits is the most one unit can charge (its hold), shown before the
	// action where it is far above the typical cost. Zero when not shown.
	MaxCredits int64 `json:"maxCredits,omitempty"`
}

// typicalUSD is the provider cost of one typical action, per feature. The
// lookups were measured on the live site (what the provider billed for one
// click), not taken from a price list, because one click buys several tasks:
// a keyword search is suggestions, ideas and a SERP; a domain report is a rank
// overview and its keyword rows; a backlink report is a summary and its rows;
// a brand lookup is the totals task
// ($0.101) and one prompt sample ($0.10 + $0.001 a row; $0.143-0.182 measured
// for an 80-mention brand). A brand AI never mentions stops after the totals
// ($0.101); the US market adds ChatGPT's sample. maxUSD is set for the AI
// visibility actions, whose holds are several times their typical cost; it is
// the worst case (US market, competitors, web search).
var typicalUSD = []struct {
	feature spend.Feature
	action  string
	unit    string
	usd     float64
	maxUSD  float64
}{
	{spend.RankTracking, "", "keyword check (top 30, scheduled)", dataforseo.SerpCostUSD(30, true), 0},
	{spend.KeywordResearch, "", "keyword search", 0.047, 0},
	{spend.DomainResearch, "", "domain report tab", 0.031, 0},
	// Summary $0.024 + history $0.024 + the first 1,000 table rows $0.06.
	{spend.Backlinks, "", "backlink report", 0.108, 0},
	{spend.AIVisibility, ActionPromptExplorer, "prompt on 4 AI models", 0.1, dataforseo.PromptExplorerCeilingUSD(4, true)},
	{spend.AIVisibility, ActionBrandLookup, "brand lookup (less when AI has no data, more with competitors)", 0.29, dataforseo.BrandLookupCeilingUSD(2, true)},
	{spend.AIAssistant, "", "AI assistant reply", 0.03, 0},
	{spend.AIWriting, "", "AI writing action", 0.01, 0},
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
		out[i] = ActionCost{Feature: c.feature, Action: c.action, Unit: c.unit, Credits: CreditsForUSD(c.usd), MaxCredits: CreditsForUSD(c.maxUSD)}
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
