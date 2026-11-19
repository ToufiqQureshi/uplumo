package site

import (
	"bytes"
	"cmp"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"slices"

	"github.com/toufiqqureshi/uplumo/backend/internal/audit"
	planCatalog "github.com/toufiqqureshi/uplumo/backend/internal/plans"
)

// FreeAuditPath is the public free SEO and AEO audit page. Its report is
// rendered by ServeAuditReport and loaded into the page by free-audit.js.
const FreeAuditPath = "/free-seo-audit"

// passingScore is the category score at or above which the report stops
// suggesting the paid tool for that category.
const passingScore = 80

// scoreRing is the circumference of the report's score circle (r = 52).
var scoreRing = 2 * math.Pi * 52

// auditView is what the report fragment template receives.
type auditView struct {
	Report     audit.PageReport
	Host       string
	Grade      string
	Tone       string
	Dash       string
	Gap        string
	Fails      int
	Warns      int
	Passes     int
	Priorities []audit.PageCheck
	Groups     []auditGroup
	Offers     []auditOffer
	SignUp     string
}

type auditGroup struct {
	audit.CategoryScore
	Tone   string
	Checks []audit.PageCheck
}

// auditOffer is a paid tool the report suggests because of what it found.
type auditOffer struct {
	Eyebrow, Title, Body, Price, Learn, Event string
}

// ServeAuditReport writes the free audit report as an HTML fragment for the
// free audit page. It is never cached by shared caches: prices follow the
// visitor's country.
func (s *Site) ServeAuditReport(w http.ResponseWriter, r *http.Request, report audit.PageReport) {
	s.serveFragment(w, http.StatusOK, "audit-report", auditViewOf(report, s.currency(r)))
}

// ServeAuditError writes a free audit failure as an HTML fragment.
func (s *Site) ServeAuditError(w http.ResponseWriter, status int, message string) {
	s.serveFragment(w, status, "audit-error", message)
}

func (s *Site) serveFragment(w http.ResponseWriter, status int, name string, data any) {
	var buf bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		http.Error(w, "Something went wrong. Please try again.", http.StatusInternalServerError)
		return
	}
	h := w.Header()
	h.Set("Content-Type", contentType)
	h.Set("Cache-Control", "private, no-store")
	h.Set("Content-Security-Policy", contentSecurityPolicy)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("X-Robots-Tag", "noindex")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}

func auditViewOf(report audit.PageReport, currency string) auditView {
	view := auditView{Report: report, SignUp: signUpPath}
	if parsed, err := url.Parse(report.URL); err == nil {
		view.Host = parsed.Host
	}
	if !report.Reachable {
		return view
	}
	view.Grade, view.Tone = grade(report.Score)
	dash := scoreRing * float64(report.Score) / 100
	view.Dash, view.Gap = fmt.Sprintf("%.1f", dash), fmt.Sprintf("%.1f", scoreRing-dash)

	rank := map[audit.CheckStatus]int{audit.CheckFail: 0, audit.CheckWarn: 1, audit.CheckPass: 2, audit.CheckInfo: 3}
	byPriority := func(a, b audit.PageCheck) int {
		return cmp.Or(cmp.Compare(rank[a.Status], rank[b.Status]), cmp.Compare(b.Weight, a.Weight))
	}
	scores := map[string]int{}
	for _, category := range report.Categories {
		scores[category.ID] = category.Score
		group := auditGroup{CategoryScore: category}
		_, group.Tone = grade(category.Score)
		for _, check := range report.Checks {
			if check.Category == category.ID {
				group.Checks = append(group.Checks, check)
			}
		}
		slices.SortStableFunc(group.Checks, byPriority)
		view.Groups = append(view.Groups, group)
	}
	for _, check := range report.Checks {
		switch check.Status {
		case audit.CheckFail:
			view.Fails++
		case audit.CheckWarn:
			view.Warns++
		case audit.CheckPass:
			view.Passes++
		case audit.CheckInfo:
		}
		if check.Status == audit.CheckFail || check.Status == audit.CheckWarn {
			view.Priorities = append(view.Priorities, check)
		}
	}
	slices.SortStableFunc(view.Priorities, byPriority)
	view.Priorities = view.Priorities[:min(len(view.Priorities), 5)]
	view.Offers = auditOffers(scores, currency)
	return view
}

func grade(score int) (label, tone string) {
	switch {
	case score >= 90:
		return "Excellent", "good"
	case score >= passingScore:
		return "Good", "good"
	case score >= 50:
		return "Needs work", "warn"
	}
	return "Poor", "bad"
}

// auditOffers suggests the paid tools that fix what the report found, with
// the cheapest plan that includes each one, priced from the plan catalog.
func auditOffers(scores map[string]int, currency string) []auditOffer {
	from := func(feature planCatalog.Feature) string {
		for _, plan := range planCatalog.Catalog() {
			if plan.Allows(feature) {
				if plan.Price(currency) == 0 {
					return "Included on the Free plan"
				}
				return fmt.Sprintf("From %s/month on %s", formatMoney(int64(plan.Price(currency)), currency), plan.Name)
			}
		}
		return ""
	}
	var offers []auditOffer
	if score := scores[audit.CategoryAI]; score < passingScore {
		offers = append(offers, auditOffer{
			Eyebrow: fmt.Sprintf("AI search score: %d/100", score),
			Title:   "See if ChatGPT recommends you or your competitor",
			Body:    "Track the questions your buyers ask in ChatGPT, Gemini, Perplexity, Claude and Google AI Overviews. See who gets mentioned and which pages get cited.",
			Price:   from(planCatalog.AIVisibility), Learn: "/ai-visibility-tracker", Event: "free_audit_offer_ai_visibility",
		})
	}
	if seo, content := scores[audit.CategorySEO], scores[audit.CategoryContent]; seo < passingScore || content < passingScore {
		offers = append(offers, auditOffer{
			Eyebrow: fmt.Sprintf("On-page SEO %d/100 · Content %d/100", seo, content),
			Title:   "Fix the page in the SEO and AEO content editor",
			Body:    "Open any page in the editor and watch its SEO and AI-answer scores change as you write. The editor is on every plan; search briefs from Google's top pages and AI writing need a paid plan.",
			Price:   "Briefs and AI writing: " + from(planCatalog.AIWriting), Learn: "/pricing", Event: "free_audit_offer_content_editor",
		})
	}
	offers = append(offers,
		auditOffer{
			Eyebrow: "We checked 1 page",
			Title:   "Audit every page of your site",
			Body:    "Broken links, duplicate titles, redirect chains, orphan pages and AI-readiness across the whole site, with the affected URLs for each issue.",
			Price:   "Full site audit: " + from(planCatalog.FullAudit), Learn: "/seo-audit-tool", Event: "free_audit_offer_site_audit",
		},
		auditOffer{
			Eyebrow: "Next step",
			Title:   "Track where you rank on Google",
			Body:    "Follow your keywords' positions every week, by country, and see which fixes moved them.",
			Price:   from(planCatalog.RankTracking), Learn: "/keyword-rank-tracker", Event: "free_audit_offer_rank_tracking",
		},
	)
	return offers
}

func freeAuditFAQ() []QA {
	return []QA{
		{"Is the audit really free?", "Yes. No sign-up and no card. It checks one page at a time and calls no paid data provider."},
		{"What is an AEO audit?", "AEO (answer engine optimisation) is making your pages easy for ChatGPT, Perplexity, Claude and Google AI Overviews to read, trust and quote. " +
			"The audit checks whether AI crawlers may read your site, whether your content is in the HTML without JavaScript, whether schema says who you are, and whether the page answers questions directly."},
		{"Which page is checked?", "The address you enter, after following its redirects. Enter your home page or any important page; each page gets its own report."},
		{"Does a good score mean I will rank or be cited by AI?", "No. The audit checks what you control on the page. Rankings and AI answers also depend on competition, links and the question asked."},
		{"How often can I run it?", "A page's report is kept for 24 hours, so running it again shows the same result. Each visitor can run a few fresh checks a day."},
		{"Why did the check fail?", "Usually the site's firewall or bot protection blocked our checker (user agent \"" + audit.UserAgent + "\"), the site was down, or the address is not a web page."},
	}
}
