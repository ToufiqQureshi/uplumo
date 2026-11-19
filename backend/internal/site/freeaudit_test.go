package site

import (
	"strings"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/audit"
	planCatalog "github.com/toufiqqureshi/uplumo/backend/internal/plans"
)

func TestAuditOffersFollowTheReportAndTheCatalog(t *testing.T) {
	cheapest := func(feature planCatalog.Feature, currency string) string {
		for _, plan := range planCatalog.Catalog() {
			if plan.Allows(feature) {
				return formatMoney(int64(plan.Price(currency)), currency) + "/month on " + plan.Name
			}
		}
		t.Fatalf("no plan has %s", feature)
		return ""
	}
	cases := []struct {
		name     string
		scores   map[string]int
		currency string
		want     []string
	}{
		{"weak everywhere", map[string]int{audit.CategoryAI: 40, audit.CategorySEO: 50, audit.CategoryContent: 90}, planCatalog.INR,
			[]string{"free_audit_offer_ai_visibility", "free_audit_offer_content_editor", "free_audit_offer_site_audit", "free_audit_offer_rank_tracking"}},
		{"strong page", map[string]int{audit.CategoryAI: 80, audit.CategorySEO: 95, audit.CategoryContent: 80}, planCatalog.USD,
			[]string{"free_audit_offer_site_audit", "free_audit_offer_rank_tracking"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			offers := auditOffers(tc.scores, tc.currency)
			if len(offers) != len(tc.want) {
				t.Fatalf("offers = %+v", offers)
			}
			for i, offer := range offers {
				if offer.Event != tc.want[i] {
					t.Errorf("offer %d = %s, want %s", i, offer.Event, tc.want[i])
				}
				if offer.Price == "" || offer.Learn == "" {
					t.Errorf("offer %s has no price or link", offer.Event)
				}
			}
			last := offers[len(offers)-1]
			if !strings.Contains(last.Price, cheapest(planCatalog.RankTracking, tc.currency)) {
				t.Errorf("rank tracking price = %q, want %q", last.Price, cheapest(planCatalog.RankTracking, tc.currency))
			}
		})
	}
}

func TestAuditViewOrdersWorstFirst(t *testing.T) {
	report := audit.PageReport{URL: "https://shop.example/", Reachable: true, Score: 55,
		Categories: []audit.CategoryScore{{ID: audit.CategorySEO, Score: 55}},
		Checks: []audit.PageCheck{
			{ID: "a", Category: audit.CategorySEO, Status: audit.CheckPass, Weight: 3},
			{ID: "b", Category: audit.CategorySEO, Status: audit.CheckWarn, Weight: 3},
			{ID: "c", Category: audit.CategorySEO, Status: audit.CheckFail, Weight: 1},
			{ID: "d", Category: audit.CategorySEO, Status: audit.CheckFail, Weight: 2},
		}}
	view := auditViewOf(report, planCatalog.INR)
	var order []string
	for _, check := range view.Groups[0].Checks {
		order = append(order, check.ID)
	}
	if strings.Join(order, "") != "dcba" {
		t.Fatalf("order = %v, want fails (heaviest first), then warnings, then passes", order)
	}
	if len(view.Priorities) != 3 || view.Fails != 2 || view.Warns != 1 || view.Passes != 1 || view.Host != "shop.example" {
		t.Fatalf("view = %+v", view)
	}
}
