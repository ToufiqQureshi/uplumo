// Package plans is the one list of what each Uplumo plan costs, limits and
// includes. The pricing page, the billing API and every plan check read it,
// so a plan cannot promise one thing and enforce another.
package plans

import "slices"

// Plan ids, as stored on a subscription.
const (
	Free  = "free"
	Basic = "basic"
	Pro   = "pro"
	// Business sits between Growth and Agency: more sites, no white-label.
	Business = "business"
	Agency   = "agency"
)

// RankScheduleWeekly is the default recurring promise on every paid plan.
// Daily checks are an opt-in that must pass the rank-tracking credit preflight.
const RankScheduleWeekly = "weekly"

// Feature is something a plan either includes or does not.
type Feature string

// The features a plan can include. Uplumo Analytics and the limited site
// audit are on every plan, so they are not listed.
const (
	// GoogleData is connecting Google Search Console and Google Analytics.
	GoogleData Feature = "google_data"
	// Keywords is keyword research, SERP analysis and metric refreshes.
	Keywords Feature = "keywords"
	// RankTracking is tracking keyword positions on a schedule.
	RankTracking Feature = "rank_tracking"
	// FullAudit is the site audit without the free plan's page limits.
	FullAudit Feature = "full_audit"
	// Publishing is connecting the customer's website and publishing
	// content-editor articles to it.
	Publishing Feature = "publishing"
	// Backlinks is backlink and referring-domain research.
	Backlinks Feature = "backlinks"
	// DomainResearch is a domain's organic footprint, including competitors.
	DomainResearch Feature = "domain_research"
	// AIVisibility is Brand Lookup and Prompt Explorer.
	AIVisibility Feature = "ai_visibility"
	// AIWriting is the writing assistant in the content editor.
	AIWriting Feature = "ai_writing"
	// WhiteLabel is the agency's own name, logo and colour on reports.
	WhiteLabel Feature = "white_label"
	// IndexNow is telling Bing and the other IndexNow search engines about
	// new and changed pages, by hand or after a WordPress publish.
	IndexNow Feature = "indexnow"
)

// Definition is one plan. A limit of zero means the plan has none of that
// thing; Unlimited lifts every limit.
type Definition struct {
	ID       string
	Name     string
	PriceINR int
	// Summary says who the plan is for, on the pricing page.
	Summary string
	// Sites is how many active websites (projects) a workspace may have.
	Sites int
	// TrackedKeywords is how many keywords rank tracking may follow across
	// the whole workspace.
	TrackedKeywords int
	// Members is how many people a workspace may hold, pending invitations
	// included.
	Members int
	// MonthlyCredits is granted on every paid month. Every paid data lookup
	// and AI call spends credits at provider cost times the markup, so the
	// grant is the most a workspace can cost Uplumo in a month.
	MonthlyCredits int64
	// MaxSerpDepth is the deepest rank tracking may look (top 30, top 40...).
	// Deeper checks cost more per keyword, so it keeps a plan's tracked
	// keywords, checked weekly, inside its monthly credits.
	MaxSerpDepth int
	// RankSchedule is the recurring frequency advertised for the plan.
	RankSchedule string
	// Highlights is the plan in the customer's words. Never list here what
	// Features and the limits do not enforce.
	Highlights []string
	features   []Feature
}

// Allows reports whether the plan includes feature.
func (d Definition) Allows(feature Feature) bool {
	return slices.Contains(d.features, feature)
}

var (
	basicFeatures  = []Feature{GoogleData, Keywords, RankTracking, FullAudit, Publishing, AIWriting, IndexNow}
	proFeatures    = append(slices.Clone(basicFeatures), Backlinks, DomainResearch, AIVisibility)
	agencyFeatures = append(slices.Clone(proFeatures), WhiteLabel)
)

var catalog = []Definition{
	{
		ID: Free, Name: "Free", PriceINR: 0, Sites: 1, Members: 1,
		Summary: "See who visits your site, and which AI assistants send them.",
		Highlights: []string{
			"1 website",
			"Uplumo Analytics with AI traffic detection",
			"A starter site audit",
		},
	},
	{
		ID: Basic, Name: "Solo", PriceINR: 999, Sites: 1, TrackedKeywords: 100, Members: 1, MonthlyCredits: 4000, MaxSerpDepth: 30, RankSchedule: RankScheduleWeekly,
		Summary: "For one business website that wants to rank on Google.",
		Highlights: []string{
			"1 website",
			"1 team member",
			"Rank tracking for 100 keywords, checked weekly",
			"Uplumo Analytics with AI traffic detection",
			"Keyword research and SERP analysis",
			"Local SEO and Google Business research",
			"Google Search Console and Google Analytics",
			"Full site audit",
			"Content editor with SEO and AI-answer scores",
			"AI writing assistant",
			"Publish articles straight to your WordPress site",
			"Tell Bing and other IndexNow search engines about new pages instantly",
		},
		features: basicFeatures,
	},
	{
		ID: Pro, Name: "Growth", PriceINR: 2499, Sites: 2, TrackedKeywords: 500, Members: 3, MonthlyCredits: 10000, MaxSerpDepth: 40, RankSchedule: RankScheduleWeekly,
		Summary: "For a growing business: beat competitors on Google and in AI answers.",
		Highlights: []string{
			"2 websites",
			"3 team members",
			"Rank tracking for 500 keywords, checked weekly",
			"AI Visibility across ChatGPT, Perplexity, Gemini, AI Overviews and Claude",
			"Backlink research",
			"Competitor and domain research",
			"Uplumo Analytics with AI traffic detection",
			"Keyword research and SERP analysis",
			"Local SEO and Google Business research",
			"Google Search Console and Google Analytics",
			"Full site audit",
			"Content editor with SEO and AI-answer scores",
			"AI writing assistant",
			"Publish articles straight to your WordPress site",
			"Tell Bing and other IndexNow search engines about new pages instantly",
		},
		features: proFeatures,
	},
	{
		ID: Business, Name: "Business", PriceINR: 4999, Sites: 5, TrackedKeywords: 1000, Members: 5, MonthlyCredits: 15000, MaxSerpDepth: 40, RankSchedule: RankScheduleWeekly,
		Summary: "For several brands or a freelancer's clients, without agency branding.",
		Highlights: []string{
			"5 websites",
			"5 team members",
			"Rank tracking for 1,000 keywords, checked weekly",
			"AI Visibility across ChatGPT, Perplexity, Gemini, AI Overviews and Claude",
			"Backlink research",
			"Competitor and domain research",
			"Uplumo Analytics with AI traffic detection",
			"Keyword research and SERP analysis",
			"Local SEO and Google Business research",
			"Google Search Console and Google Analytics",
			"Full site audit",
			"Content editor with SEO and AI-answer scores",
			"AI writing assistant",
			"Publish articles straight to your WordPress site",
			"Tell Bing and other IndexNow search engines about new pages instantly",
		},
		features: proFeatures,
	},
	{
		ID: Agency, Name: "Agency", PriceINR: 9999, Sites: 10, TrackedKeywords: 1500, Members: 10, MonthlyCredits: 25000, MaxSerpDepth: 40, RankSchedule: RankScheduleWeekly,
		Summary: "For agencies: client sites and reports under your own brand.",
		Highlights: []string{
			"10 websites",
			"10 team members",
			"Rank tracking for 1,500 keywords, checked weekly",
			"White-label reports with your name, logo and colour",
			"AI Visibility across ChatGPT, Perplexity, Gemini, AI Overviews and Claude",
			"Backlink research",
			"Competitor and domain research",
			"Uplumo Analytics with AI traffic detection",
			"Keyword research and SERP analysis",
			"Local SEO and Google Business research",
			"Google Search Console and Google Analytics",
			"Full site audit",
			"Content editor with SEO and AI-answer scores",
			"AI writing assistant",
			"Publish articles straight to your WordPress site",
			"Tell Bing and other IndexNow search engines about new pages instantly",
		},
		features: agencyFeatures,
	},
}

// Unlimited is what Uplumo's own workspaces get: every feature and no limit.
// It is not sold and is not part of Catalog.
var Unlimited = Definition{ID: Agency, Name: "Agency", features: agencyFeatures}

// Paid lists the ids of the plans that are sold, cheapest first.
func Paid() []string {
	return []string{Basic, Pro, Business, Agency}
}

// Catalog returns every plan, cheapest first. The caller owns the result.
func Catalog() []Definition {
	result := make([]Definition, len(catalog))
	for i, definition := range catalog {
		result[i] = definition.clone()
	}
	return result
}

// Get returns the plan with id.
func Get(id string) (Definition, bool) {
	for _, definition := range catalog {
		if definition.ID == id {
			return definition.clone(), true
		}
	}
	return Definition{}, false
}

func (d Definition) clone() Definition {
	d.Highlights = slices.Clone(d.Highlights)
	d.features = slices.Clone(d.features)
	return d
}
