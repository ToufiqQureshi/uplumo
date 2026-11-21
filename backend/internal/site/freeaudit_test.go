package site

import (
	"bytes"
	"context"
	"net/http/httptest"
	"net/url"
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
		kind     string
		scores   map[string]int
		currency string
		want     []string
	}{
		{"seo weak page", audit.KindSEO, map[string]int{audit.CategoryOnPage: 50, audit.CategoryContent: 90}, planCatalog.INR,
			[]string{"free_audit_offer_site_audit", "free_audit_offer_rank_tracking", "free_audit_offer_content_editor"}},
		{"seo strong page", audit.KindSEO, map[string]int{audit.CategoryOnPage: 95, audit.CategoryContent: 80}, planCatalog.USD,
			[]string{"free_audit_offer_site_audit", "free_audit_offer_rank_tracking"}},
		{"aeo weak page", audit.KindAEO, map[string]int{audit.CategoryAnswers: 40, audit.CategoryTrust: 90}, planCatalog.INR,
			[]string{"free_audit_offer_ai_visibility", "free_audit_offer_content_editor"}},
		{"aeo strong page", audit.KindAEO, map[string]int{audit.CategoryAnswers: 85, audit.CategoryTrust: 80}, planCatalog.USD,
			nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			offers := auditOffers(tc.kind, tc.scores, tc.currency)
			if len(offers) != len(tc.want) {
				t.Fatalf("offers = %+v", offers)
			}
			if len(offers) == 0 {
				return
			}
			for i, offer := range offers {
				if offer.Event != tc.want[i] {
					t.Errorf("offer %d = %s, want %s", i, offer.Event, tc.want[i])
				}
				if offer.Price == "" || offer.Learn == "" {
					t.Errorf("offer %s has no price or link", offer.Event)
				}
			}
			if tc.kind == audit.KindSEO && offers[1].Event == "free_audit_offer_rank_tracking" &&
				!strings.Contains(offers[1].Price, cheapest(planCatalog.RankTracking, tc.currency)) {
				t.Errorf("rank tracking price = %q, want %q", offers[1].Price, cheapest(planCatalog.RankTracking, tc.currency))
			}
			if tc.kind == audit.KindAEO && !strings.Contains(offers[0].Price, cheapest(planCatalog.AIVisibility, tc.currency)) {
				t.Errorf("AI visibility price = %q, want %q", offers[0].Price, cheapest(planCatalog.AIVisibility, tc.currency))
			}
		})
	}
}

func TestAuditViewOrdersWorstFirst(t *testing.T) {
	report := audit.PageReport{URL: "https://shop.example/", Reachable: true, Score: 55,
		Categories: []audit.CategoryScore{{ID: audit.CategoryOnPage, Score: 55}},
		Checks: []audit.PageCheck{
			{ID: "a", Category: audit.CategoryOnPage, Status: audit.CheckPass, Weight: 3},
			{ID: "b", Category: audit.CategoryOnPage, Status: audit.CheckWarn, Weight: 3},
			{ID: "c", Category: audit.CategoryOnPage, Status: audit.CheckFail, Weight: 1},
			{ID: "d", Category: audit.CategoryOnPage, Status: audit.CheckFail, Weight: 2},
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

func TestReportLogoIsEmbeddedAsPNG(t *testing.T) {
	logo := ReportLogoPNG()
	if !bytes.HasPrefix(logo, []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatalf("report logo is not a PNG (%d bytes)", len(logo))
	}
}

func TestGradeBandsMatchTheReportLabels(t *testing.T) {
	cases := map[int]struct{ label, tone string }{
		95: {"Excellent", "good"}, 90: {"Excellent", "good"}, 89: {"Good", "good"}, 80: {"Good", "good"},
		79: {"Needs work", "warn"}, 50: {"Needs work", "warn"}, 49: {"Poor", "bad"}, 0: {"Poor", "bad"},
	}
	for score, want := range cases {
		if label, tone := grade(score); label != want.label || tone != want.tone {
			t.Errorf("grade(%d) = %s/%s, want %s/%s", score, label, tone, want.label, want.tone)
		}
	}
}

func TestAuditReportFragmentSaysWhatTheScoreIs(t *testing.T) {
	pages, err := New(&url.URL{Scheme: "https", Host: "uplumo.com"})
	if err != nil {
		t.Fatal(err)
	}
	report := audit.PageReport{Kind: audit.KindSEO, URL: "https://shop.example/?a=1", Reachable: true, Score: 59,
		Caps:       []string{"Score capped at 59 because a critical check failed (HTTPS)."},
		Categories: []audit.CategoryScore{{ID: audit.CategoryOnPage, Name: "On-page SEO", Score: 40}},
		Checks: []audit.PageCheck{{ID: "https", Category: audit.CategoryOnPage, Status: audit.CheckFail, Title: "HTTPS",
			Found: "The page loads over plain HTTP.", Fix: "Install a TLS certificate."}}}
	rec := httptest.NewRecorder()
	pages.ServeAuditReport(rec, httptest.NewRequestWithContext(context.Background(), "GET", "/free-seo-audit/report", nil), report)
	body := rec.Body.String()
	for _, want := range []string{
		"SEO score: Needs work", "Checks how well this page is built for Google. It does not measure rankings or backlinks.",
		"Why the score is limited", "Score capped at 59 because a critical check failed (HTTPS).", "data-print-report", "Download PDF",
		"/free-seo-audit/report.docx?url=https%3A%2F%2Fshop.example%2F%3Fa%3D1&amp;type=seo", "Download Word (.docx)",
		"/site/logo.svg?v=", "Prepared by Uplumo · uplumo.com",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("fragment does not contain %q", want)
		}
	}
	// Scripts from the page are not allowed by the CSP; the fragment has no inline JavaScript.
	for _, banned := range []string{"<script", " onclick=", "javascript:"} {
		if strings.Contains(body, banned) {
			t.Errorf("fragment contains %q", banned)
		}
	}
}

func TestAEOReportFragmentSaysItsOwnThing(t *testing.T) {
	pages, err := New(&url.URL{Scheme: "https", Host: "uplumo.com"})
	if err != nil {
		t.Fatal(err)
	}
	report := audit.PageReport{Kind: audit.KindAEO, URL: "https://shop.example/", Reachable: true, Score: 55,
		Categories: []audit.CategoryScore{{ID: audit.CategoryAnswers, Name: "Answers AI can quote", Score: 31}},
		Checks: []audit.PageCheck{{ID: "question-headings", Category: audit.CategoryAnswers, Status: audit.CheckFail, Title: "Questions as headings",
			Found: "No headings phrased as a question.", Fix: "Add questions as headings."}}}
	rec := httptest.NewRecorder()
	pages.ServeAuditReport(rec, httptest.NewRequestWithContext(context.Background(), "GET", "/free-aeo-audit/report", nil), report)
	body := rec.Body.String()
	for _, want := range []string{
		"AEO score: Needs work", "Checks how ready this page is to be read, trusted and quoted by AI assistants",
		"does not measure whether ChatGPT or other assistants mention you today", "See if ChatGPT recommends you",
		"&amp;type=aeo", "AEO audit report",
	} {
		if !strings.Contains(body, want) && !strings.Contains(body, strings.ReplaceAll(want, "&amp;", "&")) {
			t.Errorf("AEO fragment does not contain %q", want)
		}
	}
	if strings.Contains(body, "SEO score") {
		t.Errorf("AEO fragment says SEO score")
	}
}
