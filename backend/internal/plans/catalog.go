// Package plans is the one list of what each Uplumo plan costs, limits and
// includes. The pricing page, the billing API and every plan check read it,
// so a plan cannot promise one thing and enforce another.
package plans

import (
	"fmt"
	"slices"
)

// Plan ids, as stored on a subscription. They predate the customer-facing
// names and stay as they are: subscriptions and Razorpay variables use them.
const (
	Free     = "free"
	Basic    = "basic"
	Pro      = "pro"
	Business = "business"
)

// RankScheduleWeekly is the default recurring promise on every paid plan.
// Daily checks are an opt-in that must pass the rank-tracking credit preflight.
const RankScheduleWeekly = "weekly"

// Cite Fix paid actions: the most one click can cost a workspace, shown on
// the button before it is clicked. Owner decisions: the agent's caps on
// 27 Oct 2026, the outreach pitch on 4 Nov 2026.
const (
	CiteFixQuickMaxCredits int64 = 25
	CiteFixDeepMaxCredits  int64 = 60
	CiteFixPitchMaxCredits int64 = 25
)

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
	// WhiteLabel is the workspace's own name, logo and colour on reports.
	WhiteLabel Feature = "white_label"
	// IndexNow is telling Bing and the other IndexNow search engines about
	// new and changed pages, by hand or after a WordPress publish.
	IndexNow Feature = "indexnow"
	// CreditPacks is buying one-time credit packs on top of the monthly grant.
	CreditPacks Feature = "credit_packs"
	// SiteAddOn is buying more websites than the plan includes.
	SiteAddOn Feature = "site_add_on"
)

// Definition is one plan. A limit of zero means the plan has none of that
// thing; Unlimited lifts every limit.
type Definition struct {
	ID       string
	Name     string
	PriceINR int
	// PriceUSD is the monthly price in whole US dollars for customers outside
	// India. It is a price of its own, not a conversion of PriceINR.
	PriceUSD int
	// Summary says who the plan is for, on the pricing page.
	Summary string
	// Sites is how many active websites (projects) a workspace may have
	// before any site add-on.
	Sites int
	// PauseAboveSites, when set, is the site count above which paid work
	// pauses; it is higher than Sites only where Sites was lowered after
	// customers bought the plan (Pro 10 -> 5, Business 20 -> 10 on 3 Nov
	// 2026; Pro 5 -> 3 on 6 Nov 2026), so those customers keep every site
	// working. New sites still stop at Sites.
	PauseAboveSites int
	// TrackedKeywords is how many keywords rank tracking may follow across
	// the whole workspace.
	TrackedKeywords int
	// Members is how many people a workspace may hold, pending invitations
	// included.
	Members int
	// MonthlyCredits is granted on every paid month. Every paid data lookup
	// and AI call spends credits at provider cost times the markup, so the
	// grant plus bought packs is the most a workspace can cost Uplumo.
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

// Currencies a plan is sold in.
const (
	INR = "INR"
	USD = "USD"
)

// CurrencyFor is the currency a visitor from country (ISO 3166-1 alpha-2)
// pays in: rupees in India, US dollars elsewhere. An unknown country pays in
// rupees, as every customer did before dollar prices existed.
func CurrencyFor(country string) string {
	if country == "" || country == "IN" {
		return INR
	}
	return USD
}

// Price is the monthly price in currency's whole units; 0 when the plan is
// not sold in it.
func (d Definition) Price(currency string) int {
	switch currency {
	case INR:
		return d.PriceINR
	case USD:
		return d.PriceUSD
	}
	return 0
}

// Allows reports whether the plan includes feature.
func (d Definition) Allows(feature Feature) bool {
	return slices.Contains(d.features, feature)
}

// Pack is a one-time purchase on top of a plan: credits or more websites.
type Pack struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	PriceINR int64  `json:"priceInr"`
	// PriceUSD is the price in whole US dollars for customers outside India.
	PriceUSD int64 `json:"priceUsd"`
	// Credits is what a credit pack adds. Pack credits last while the
	// workspace's paid plan does and expire when the plan ends.
	Credits int64 `json:"credits"`
	// Sites is what a site add-on adds, for SiteAddOnDays after payment.
	Sites int `json:"sites"`
	// Feature is what the workspace's plan must include to buy the pack.
	Feature Feature `json:"-"`
}

// SiteAddOnPackID is the one-website add-on every paid plan can buy, as
// many times as it needs websites.
const SiteAddOnPackID = "sites_1"

// SiteAddOnDays is how long a site add-on lasts. It counts from the payment,
// not to the next renewal, so an add-on bought the day before a renewal
// still gives the month that was paid for.
const SiteAddOnDays = 30

var packs = []Pack{
	{ID: "credits_5000", Name: "5,000 credits", PriceINR: 1499, PriceUSD: 19, Credits: 5000, Feature: CreditPacks},
	{ID: "credits_15000", Name: "15,000 credits", PriceINR: 3999, PriceUSD: 59, Credits: 15000, Feature: CreditPacks},
	{ID: SiteAddOnPackID, Name: "1 more website", PriceINR: 999, PriceUSD: 12, Sites: 1, Feature: SiteAddOn},
}

// Packs lists every one-time purchase, credit packs first. The caller owns
// the result.
func Packs() []Pack {
	return slices.Clone(packs)
}

// Price is the pack's price in currency's whole units; 0 when it is not sold
// in it.
func (p Pack) Price(currency string) int64 {
	switch currency {
	case INR:
		return p.PriceINR
	case USD:
		return p.PriceUSD
	}
	return 0
}

// GetPack returns the pack with id.
func GetPack(id string) (Pack, bool) {
	i := slices.IndexFunc(packs, func(p Pack) bool { return p.ID == id })
	if i < 0 {
		return Pack{}, false
	}
	return packs[i], true
}

var (
	basicFeatures    = []Feature{GoogleData, Keywords, RankTracking, FullAudit, Publishing, AIWriting, IndexNow, SiteAddOn}
	proFeatures      = append(slices.Clone(basicFeatures), Backlinks, DomainResearch, AIVisibility, CreditPacks)
	businessFeatures = append(slices.Clone(proFeatures), WhiteLabel)
)

// sharedHighlights close every paid plan's list.
var sharedHighlights = []string{
	"Uplumo Analytics with AI traffic detection",
	"Keyword research and SERP analysis",
	"Google Search Console and Google Analytics",
	"Full site audit",
	"Content editor with SEO and AI-answer scores",
	"AI writing assistant",
	"Publish articles straight to your WordPress site",
	"Tell Bing and other IndexNow search engines about new pages instantly",
}

var researchHighlights = []string{
	"AI Visibility across ChatGPT, Perplexity, Gemini, AI Overviews and Claude",
	"Backlink research",
	"Competitor and domain research",
}

const creditPackHighlight = "Buy extra credits any time; they last while your plan is active"

func highlights(lines ...[]string) []string {
	return slices.Concat(append(lines, sharedHighlights)...)
}

var catalog = []Definition{
	{
		ID: Free, Name: "Free", PriceINR: 0, Sites: 1, Members: 1,
		Summary: "See who visits your site, and which AI assistants send them.",
		Highlights: []string{
			websites(1),
			"Uplumo Analytics with AI traffic detection",
			"A starter site audit",
		},
	},
	{
		ID: Basic, Name: "Basic", PriceINR: 999, PriceUSD: 15, Sites: 1, TrackedKeywords: 100, Members: 1, MonthlyCredits: 4000, MaxSerpDepth: 30, RankSchedule: RankScheduleWeekly,
		Summary: "For one business website that wants to rank on Google.",
		Highlights: highlights([]string{
			websites(1),
			siteAddOnHighlight(INR),
			"1 team member",
			"4,000 credits every month",
			"Rank tracking for 100 keywords, checked weekly",
		}),
		features: basicFeatures,
	},
	{
		ID: Pro, Name: "Pro", PriceINR: 2499, PriceUSD: 39, Sites: 3, PauseAboveSites: 10, TrackedKeywords: 500, Members: 3, MonthlyCredits: 10000, MaxSerpDepth: 40, RankSchedule: RankScheduleWeekly,
		Summary: "For a growing business: beat competitors on Google and in AI answers.",
		Highlights: highlights([]string{
			websites(3),
			siteAddOnHighlight(INR),
			"3 team members",
			"10,000 credits every month",
			creditPackHighlight,
			"Rank tracking for 500 keywords, checked weekly",
		}, researchHighlights),
		features: proFeatures,
	},
	{
		ID: Business, Name: "Business", PriceINR: 4999, PriceUSD: 69, Sites: 10, PauseAboveSites: 20, TrackedKeywords: 1000, Members: 5, MonthlyCredits: 15000, MaxSerpDepth: 40, RankSchedule: RankScheduleWeekly,
		Summary: "For several brands, freelancers and agencies, with reports under your own brand.",
		Highlights: highlights([]string{
			websites(10),
			siteAddOnHighlight(INR),
			"5 team members",
			"15,000 credits every month",
			creditPackHighlight,
			"Rank tracking for 1,000 keywords, checked weekly",
			"White-label reports with your name, logo and colour",
		}, researchHighlights),
		features: businessFeatures,
	},
}

// websites is a plan card's website line, written from the same number the
// plan's Sites field holds.
func websites(n int) string {
	if n == 1 {
		return "1 website"
	}
	return fmt.Sprintf("%d websites", n)
}

// siteAddOnHighlight states the site add-on's price in currency on each paid
// plan card, from the pack itself so the two cannot disagree.
func siteAddOnHighlight(currency string) string {
	pack, _ := GetPack(SiteAddOnPackID)
	symbol := "₹"
	if currency == USD {
		symbol = "$"
	}
	return fmt.Sprintf("Add more websites for %s%d a month each", symbol, pack.Price(currency))
}

// HighlightsIn is Highlights with every price in currency.
func (d Definition) HighlightsIn(currency string) []string {
	out := slices.Clone(d.Highlights)
	if i := slices.Index(out, siteAddOnHighlight(INR)); i >= 0 {
		out[i] = siteAddOnHighlight(currency)
	}
	return out
}

// Unlimited is what Uplumo's own workspaces get: every feature and no limit.
// It is not sold and is not part of Catalog.
var Unlimited = Definition{ID: Business, Name: "Business", features: businessFeatures}

// Paid lists the ids of the plans that are sold, cheapest first.
func Paid() []string {
	return []string{Basic, Pro, Business}
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
