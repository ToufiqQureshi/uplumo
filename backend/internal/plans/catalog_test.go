package plans

import (
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
)

func TestCatalogMatchesTheOwnersPlanTable(t *testing.T) {
	want := []struct {
		id                                   string
		price, usd, sites, keywords, members int
		credits                              int64
	}{
		{Free, 0, 0, 1, 0, 1, 0},
		{Basic, 999, 15, 1, 100, 1, 4000},
		{Pro, 2499, 39, 3, 500, 3, 10000},
		{Business, 4999, 69, 10, 1000, 5, 15000},
	}
	got := Catalog()
	if len(got) != len(want) {
		t.Fatalf("catalog has %d plans, want %d", len(got), len(want))
	}
	for i, w := range want {
		g := got[i]
		if g.ID != w.id || g.PriceINR != w.price || g.PriceUSD != w.usd || g.Sites != w.sites || g.TrackedKeywords != w.keywords || g.Members != w.members || g.MonthlyCredits != w.credits {
			t.Errorf("plan %d = %s ₹%d $%d, %d sites, %d keywords, %d members, %d credits; want %+v",
				i, g.ID, g.PriceINR, g.PriceUSD, g.Sites, g.TrackedKeywords, g.Members, g.MonthlyCredits, w)
		}
		if g.Name == "" || g.Summary == "" || len(g.Highlights) == 0 {
			t.Errorf("plan %s has no name, summary or highlights to show a customer", g.ID)
		}
		if g.TrackedKeywords > 0 && g.RankSchedule != RankScheduleWeekly {
			t.Errorf("plan %s rank schedule = %q, want weekly", g.ID, g.RankSchedule)
		}
	}
	if !slices.Equal(Paid(), []string{Basic, Pro, Business}) {
		t.Errorf("Paid() = %v", Paid())
	}
}

func TestEachPlanIncludesExactlyItsFeatures(t *testing.T) {
	all := []Feature{GoogleData, Keywords, RankTracking, FullAudit, Publishing, Backlinks, DomainResearch, AIVisibility, AIWriting, WhiteLabel, IndexNow, CreditPacks, SiteAddOn}
	want := map[string][]Feature{
		Free:     nil,
		Basic:    {GoogleData, Keywords, RankTracking, FullAudit, Publishing, AIWriting, IndexNow, SiteAddOn},
		Pro:      {GoogleData, Keywords, RankTracking, FullAudit, Publishing, AIWriting, IndexNow, SiteAddOn, Backlinks, DomainResearch, AIVisibility, CreditPacks},
		Business: all,
	}
	for id, included := range want {
		plan, ok := Get(id)
		if !ok {
			t.Fatalf("Get(%q) not found", id)
		}
		for _, feature := range all {
			if got, want := plan.Allows(feature), slices.Contains(included, feature); got != want {
				t.Errorf("%s allows %s = %v, want %v", id, feature, got, want)
			}
		}
	}
	if _, ok := Get("enterprise"); ok {
		t.Error("Get found a plan that is not in the catalog")
	}
}

func TestUnlimitedAllowsEverythingAndHasNoLimits(t *testing.T) {
	for _, feature := range businessFeatures {
		if !Unlimited.Allows(feature) {
			t.Errorf("Unlimited does not allow %s", feature)
		}
	}
	if Unlimited.Sites != 0 || Unlimited.TrackedKeywords != 0 || Unlimited.Members != 0 {
		t.Errorf("Unlimited carries limits: %+v", Unlimited)
	}
}

func TestCallersCannotChangeTheCatalog(t *testing.T) {
	first, _ := Get(Pro)
	first.Highlights[0] = "mutated"
	first.features[0] = "mutated"
	Catalog()[2].Highlights[1] = "mutated"

	second, _ := Get(Pro)
	if second.Highlights[0] == "mutated" || second.Highlights[1] == "mutated" || !second.Allows(GoogleData) {
		t.Fatal("a caller's change reached the shared catalog")
	}
	// Basic's list is the base of Pro's and Business's; they must not share it.
	if basic, _ := Get(Basic); basic.Allows(Backlinks) || basic.Allows(WhiteLabel) || basic.Allows(CreditPacks) {
		t.Fatal("Basic picked up a feature of a higher plan")
	}
}

// Credits are worth 1/1280 of a US dollar of provider cost (1,000 per dollar
// with a 1.28 markup). A plan's grant is the most one workspace can cost
// Uplumo in a month, so it must stay well under the price.
func TestNoPlanCanCostMoreThanAThirdOfItsPrice(t *testing.T) {
	const inrPerUSD = 90.0 // a rupee that has weakened, to keep a margin
	for _, plan := range Catalog() {
		providerINR := float64(plan.MonthlyCredits) / 1280 * inrPerUSD
		if providerINR > float64(plan.PriceINR)/3 {
			t.Errorf("%s: credits can cost ₹%.0f a month at the provider, more than a third of ₹%d", plan.ID, providerINR, plan.PriceINR)
		}
		// The USD price is set on its own, so it needs the same margin.
		if providerUSD := float64(plan.MonthlyCredits) / 1280; providerUSD > float64(plan.PriceUSD)/3 {
			t.Errorf("%s: credits can cost $%.2f a month at the provider, more than a third of $%d", plan.ID, providerUSD, plan.PriceUSD)
		}
	}
}

// A plan's tracked keywords, checked weekly at its deepest allowed depth with
// the scheduled (queued) price, must fit in its monthly credits, or the rank
// tracking it sells runs out of credits mid-month.
func TestWeeklyRankTrackingFitsInTheMonthlyCredits(t *testing.T) {
	const checksPerMonth = 4.35
	for _, plan := range Catalog() {
		if plan.TrackedKeywords == 0 {
			continue
		}
		if plan.MaxSerpDepth < 10 || plan.MaxSerpDepth > 100 || plan.MaxSerpDepth%10 != 0 {
			t.Fatalf("%s: MaxSerpDepth %d is not 10 to 100 in steps of 10", plan.ID, plan.MaxSerpDepth)
		}
		perCheckUSD := dataforseo.SerpCostUSD(plan.MaxSerpDepth, true)
		credits := math.Ceil(float64(plan.TrackedKeywords) * checksPerMonth * perCheckUSD * 1280)
		if credits > float64(plan.MonthlyCredits) {
			t.Errorf("%s: weekly rank tracking costs %.0f credits a month, over the %d it grants", plan.ID, credits, plan.MonthlyCredits)
		}
	}
}

func TestPacksMatchTheOwnersPrices(t *testing.T) {
	want := []Pack{
		{ID: "credits_5000", PriceINR: 1499, Credits: 5000, Feature: CreditPacks},
		{ID: "credits_15000", PriceINR: 3999, Credits: 15000, Feature: CreditPacks},
		{ID: "sites_1", PriceINR: 999, PriceUSD: 12, Sites: 1, Feature: SiteAddOn},
	}
	got := Packs()
	if len(got) != len(want) {
		t.Fatalf("Packs() has %d packs, want %d", len(got), len(want))
	}
	for i, w := range want {
		g := got[i]
		if g.ID != w.ID || g.PriceINR != w.PriceINR || (w.PriceUSD != 0 && g.PriceUSD != w.PriceUSD) || g.Credits != w.Credits || g.Sites != w.Sites || g.Feature != w.Feature || g.Name == "" {
			t.Errorf("pack %d = %+v, want %+v", i, g, w)
		}
		if (g.Credits > 0) == (g.Sites > 0) {
			t.Errorf("pack %s must add credits or sites, not both or neither", g.ID)
		}
		if p, ok := GetPack(g.ID); !ok || p != g {
			t.Errorf("GetPack(%q) = %+v, %v", g.ID, p, ok)
		}
	}
	if _, ok := GetPack("agency"); ok {
		t.Error("GetPack found a pack that is not sold")
	}
	free, _ := Get(Free)
	if free.Allows(CreditPacks) || free.Allows(SiteAddOn) {
		t.Error("Free may buy packs; only paid plans may")
	}
	if basic, _ := Get(Basic); basic.Allows(CreditPacks) {
		t.Error("Basic may buy credit packs; only Pro and Business may")
	}
	for _, id := range Paid() {
		if plan, _ := Get(id); !plan.Allows(SiteAddOn) {
			t.Errorf("%s cannot buy extra websites; every paid plan may", id)
		}
	}
}

// Only plans that include CreditPacks may buy one, so a pack must not sell a
// credit for less than the cheapest such plan's grant does, and its credits
// must never cost Uplumo more than a third of its price at the provider.
func TestCreditPacksKeepTheMargin(t *testing.T) {
	const inrPerUSD = 90.0
	cheapest := math.Inf(1)
	for _, plan := range Catalog() {
		if plan.Allows(CreditPacks) {
			cheapest = min(cheapest, float64(plan.PriceINR)/float64(plan.MonthlyCredits))
		}
	}
	for _, pack := range Packs() {
		if pack.Credits == 0 {
			continue
		}
		if perCredit := float64(pack.PriceINR) / float64(pack.Credits); perCredit < cheapest {
			t.Errorf("%s costs ₹%.3f a credit, less than a plan's ₹%.3f", pack.ID, perCredit, cheapest)
		}
		if providerINR := float64(pack.Credits) / 1280 * inrPerUSD; providerINR > float64(pack.PriceINR)/3 {
			t.Errorf("%s can cost ₹%.0f at the provider, more than a third of ₹%d", pack.ID, providerINR, pack.PriceINR)
		}
	}
}

// Every paid card states the per-website add-on price; it must be the pack's,
// and each card's website line must match the limit the plan enforces.
func TestPaidCardsStateTheSiteAddOnPrice(t *testing.T) {
	addOn, _ := GetPack(SiteAddOnPackID)
	want := fmt.Sprintf("Add more websites for ₹%d a month each", addOn.PriceINR)
	usd := fmt.Sprintf("Add more websites for $%d a month each", addOn.PriceUSD)
	for _, plan := range Catalog() {
		sites := fmt.Sprintf("%d websites", plan.Sites)
		if plan.Sites == 1 {
			sites = "1 website"
		}
		if !slices.Contains(plan.Highlights, sites) {
			t.Errorf("%s highlights do not state %q", plan.ID, sites)
		}
		if got := slices.Contains(plan.Highlights, want); got != plan.Allows(SiteAddOn) {
			t.Errorf("%s highlights contain %q = %v, want %v", plan.ID, want, got, plan.Allows(SiteAddOn))
		}
		if !plan.Allows(SiteAddOn) {
			continue
		}
		if got := plan.HighlightsIn(USD); !slices.Contains(got, usd) || slices.Contains(got, want) {
			t.Errorf("%s highlights in dollars = %q, want %q and no rupee price", plan.ID, got, usd)
		}
		if got := plan.HighlightsIn(INR); !slices.Equal(got, plan.Highlights) {
			t.Errorf("%s highlights in rupees = %q, want the catalog's", plan.ID, got)
		}
	}
}

func TestCurrencyFor(t *testing.T) {
	for country, want := range map[string]string{"IN": INR, "": INR, "US": USD, "GB": USD, "AE": USD} {
		if got := CurrencyFor(country); got != want {
			t.Errorf("CurrencyFor(%q) = %s, want %s", country, got, want)
		}
	}
	pro, _ := Get(Pro)
	if pro.Price(INR) != 2499 || pro.Price(USD) != 39 || pro.Price("EUR") != 0 {
		t.Errorf("Pro prices = %d INR, %d USD, %d EUR; want 2499, 39, 0", pro.Price(INR), pro.Price(USD), pro.Price("EUR"))
	}
}
