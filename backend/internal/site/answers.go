package site

import (
	"fmt"
	"net/http"
	"strings"

	planCatalog "github.com/toufiqqureshi/uplumo/backend/internal/plans"
)

// organizationDescription is the one-sentence answer to "what is Uplumo?" that
// structured data and llms.txt give search engines and AI assistants.
const organizationDescription = "Uplumo is an SEO and AI search visibility platform: it tracks Google rankings, " +
	"audits websites, researches keywords and records how ChatGPT, Gemini, Claude, Perplexity and Google AI Overviews mention and cite a brand."

// plansWith names the plans that include feature, in catalog order ("Pro and
// Business"), so a page can never name a plan the feature check does not allow.
func plansWith(feature planCatalog.Feature) string {
	var names []string
	for _, plan := range planCatalog.Catalog() {
		if plan.Allows(feature) {
			names = append(names, plan.Name)
		}
	}
	return joinNames(names)
}

// pageName is the page's title without the brand suffix, as a breadcrumb
// shows it.
func pageName(title string) string {
	name, _, _ := strings.Cut(title, " | ")
	return name
}

// breadcrumbLD is the Home > page trail for an info page.
func breadcrumbLD(origin string, p infoPage) map[string]any {
	return map[string]any{
		"@type": "BreadcrumbList",
		"itemListElement": []map[string]any{
			{"@type": "ListItem", "position": 1, "name": "Home", "item": origin + "/"},
			{"@type": "ListItem", "position": 2, "name": pageName(p.title), "item": origin + p.path},
		},
	}
}

// faqLD repeats a page's visible questions as FAQPage structured data. It is
// nil when the page has none, so the graph carries no empty FAQ.
func faqLD(pageURL string, faq []QA) map[string]any {
	if len(faq) == 0 {
		return nil
	}
	questions := make([]map[string]any, 0, len(faq))
	for _, qa := range faq {
		questions = append(questions, map[string]any{
			"@type":          "Question",
			"name":           qa.Question,
			"acceptedAnswer": map[string]string{"@type": "Answer", "text": qa.Answer},
		})
	}
	return map[string]any{"@type": "FAQPage", "@id": pageURL + "#faq", "mainEntity": questions}
}

// aiEngines is the list of answer engines saved AI tracking can ask, as the
// pages name them.
const aiEngines = "ChatGPT, Gemini, Google AI Overviews, Claude and Perplexity"

func aiVisibilityFAQ() []QA {
	return []QA{
		{"What is an AI visibility tracker?", "An AI visibility tracker asks AI assistants the questions your customers ask, saves each answer " +
			"and records whether it mentions your brand and which pages it cites. Uplumo runs your saved questions on " + aiEngines + "."},
		{"Which AI engines does Uplumo track?", aiEngines + ". Choose up to five engines and one market for each set of questions. " +
			"Some engines are not available in every market."},
		{"How often are my questions checked?", "Daily, weekly or monthly, as you choose. Before you start, Uplumo shows the credits each check uses."},
		{"What is the difference between a mention and a citation?", "A mention means the answer names your brand. " +
			"A citation means the answer links to one of your pages as a source. An answer can have one, both or neither."},
		{"Will my results match what every user sees?", "No. AI answers change with the account, location, time and conversation. " +
			"Uplumo records sampled answers to the questions you choose, so compare the same questions, engines and market over time."},
		{"Which plans include AI visibility tracking?", plansWith(planCatalog.AIVisibility) + ". Each check uses workspace credits."},
	}
}

func aiOverviewFAQ() []QA {
	return []QA{
		{"What is a Google AI Overview tracker?", "A Google AI Overview tracker runs the searches you choose on Google in a chosen market, " +
			"saves the AI Overview when Google shows one, and records the brands it names and the pages it cites."},
		{"How is AI Overview tracking different from rank tracking?", "Rank tracking records where your page sits in the organic results. " +
			"AI Overview tracking records whether the generated summary above those results names your brand or links to your pages."},
		{"What happens when Google shows no AI Overview?", "Google does not show an overview for every search. " +
			"When it shows none, that check has no overview text or citations to record."},
		{"Can I track ChatGPT and other assistants at the same time?", "Yes. The same saved questions can run on " + aiEngines + ", " +
			"so you can compare Google's overview with other assistants' answers."},
		{"Which plans include Google AI Overview tracking?", plansWith(planCatalog.AIVisibility) + ". Checks run daily, weekly or monthly and use workspace credits."},
	}
}

func aiReferralFAQ() []QA {
	return []QA{
		{"What is AI referral tracking?", "AI referral tracking counts the visits that reach your website from AI assistants " +
			"such as ChatGPT, Perplexity and Gemini, and shows which pages they land on."},
		{"Can it count every visit from an AI assistant?", "No. A visit is recognised only when the AI app passes a referrer or the link carries a campaign tag. " +
			"Without either, the visit stays Direct or unknown."},
		{"Does the tracker use cookies?", "No. The tracker sets no cookies and stores no IP addresses."},
		{"Is AI referral tracking free?", "Yes. Uplumo Analytics, including AI referral detection, is on every plan, Free included."},
	}
}

func seoAuditFAQ() []QA {
	return []QA{
		{"What does an SEO audit tool check?", "It crawls your pages and reports issues that affect how search engines read them: " +
			"broken links, crawl problems and on-page elements such as missing or duplicate titles. Uplumo shows the affected URLs for each issue."},
		{"Is the Uplumo SEO audit free?", "Every plan, Free included, has a starter audit with page limits. " +
			"The full audit is on " + plansWith(planCatalog.FullAudit) + "."},
		{"Can the audit render JavaScript?", "Optional JavaScript rendering depends on your plan and credits."},
		{"How do I check that a fix worked?", "Run the audit again and compare the affected URLs. Use Search Console to see how Google handles the pages you changed."},
	}
}

func rankTrackerFAQ() []QA {
	var limits []string
	for _, plan := range planCatalog.Catalog() {
		if plan.Allows(planCatalog.RankTracking) {
			limits = append(limits, fmt.Sprintf("%s %s keywords (top %d)", plan.Name, formatNumber(int64(plan.TrackedKeywords)), plan.MaxSerpDepth))
		}
	}
	return []QA{
		{"What is a keyword rank tracker?", "A keyword rank tracker checks where your pages appear in Google for the searches you choose, " +
			"on a schedule, and keeps the history so you can see positions move."},
		{"How often are rankings checked?", "Weekly on every paid plan. Daily checks are optional and use more credits."},
		{"How many keywords can I track?", "By plan: " + joinNames(limits) + "."},
		{"Can I see clicks and impressions too?", "Yes. Connect Google Search Console to compare rankings with the clicks and impressions Google reports."},
	}
}

func keywordResearchFAQ() []QA {
	return []QA{
		{"What is a keyword research tool?", "A keyword research tool finds the searches people make about a topic " +
			"and estimates how often they happen and how hard they are to rank for, so you can choose which pages to write."},
		{"Can I research keywords for my own country?", "Yes. Choose the market and language you sell in; volumes and results are estimated for that market."},
		{"Does Uplumo show what currently ranks?", "Yes. SERP analysis shows the pages and result types Google returns for a keyword before you write."},
		{"Which plans include keyword research?", plansWith(planCatalog.Keywords) + ". Research actions use workspace credits."},
	}
}

func localSEOFAQ() []QA {
	return []QA{
		{"What do Uplumo's local SEO tools do?", "They check Google Maps for a search at your address and compare your Google Business Profile " +
			"with the top three businesses there: ratings, review counts, categories, photos, posts and missing details."},
		{"Does it analyse reviews?", "Yes. It reads your latest reviews and your competitors', shows your reply rate and the themes customers mention, " +
			"and lists reviews without a reply with a draft you can copy."},
		{"Can I monitor competitors over time?", "Yes. Competitor watch is off until you turn it on. It then checks the top three every week and shows what changed."},
		{"Which plans include local SEO tools?", plansWith(planCatalog.LocalVisibility) + ". Profile checks and monitoring use workspace credits."},
	}
}

func whiteLabelFAQ() []QA {
	return []QA{
		{"What is a white label SEO report?", "A white label SEO report is a client report that carries your agency's name, logo and colour " +
			"instead of the software vendor's."},
		{"What can an Uplumo report include?", "Google Search performance, organic traffic from Google Analytics, keyword rankings, site health and backlinks " +
			"for the last 7, 28 or 90 days, plus a work log of completed work, outcomes and next steps. Each section needs its data source connected."},
		{"How do clients see the report?", "Share a link. The shared page shows your brand name, logo and accent colour, " +
			"says it was prepared by your brand and can link to your website."},
		{"Which plans include white label reports?", plansWith(planCatalog.WhiteLabel) + ". Reports on other plans are shared under the Uplumo name."},
	}
}

// llmsTxt is llms.txt (llmstxt.org): what Uplumo is and where each product,
// plan and guide page lives, in plain text an AI assistant can read whole.
// With usd, plans carry their dollar price too; without, only the rupee price
// checkout can charge.
func llmsTxt(origin string, posts []Post, usd bool) []byte {
	var b strings.Builder
	b.WriteString("# Uplumo\n\n> " + organizationDescription + "\n\n")
	prices := "Prices are in Indian rupees."
	if usd {
		prices = "Prices are in US dollars, or Indian rupees for customers in India."
	}
	b.WriteString("Uplumo serves websites in every country. " + prices + " " +
		"Results come from Google, AI answer engines and the customer's own connected data; Uplumo does not guarantee rankings or AI mentions.\n\n")

	b.WriteString("## Product\n\n")
	for _, p := range infoPages {
		if p.product {
			fmt.Fprintf(&b, "- [%s](%s%s): %s\n", pageName(p.title), origin, p.path, p.description)
		}
	}

	b.WriteString("\n## Plans\n\n")
	for _, plan := range planCatalog.Catalog() {
		inr := formatMoney(int64(plan.Price(planCatalog.INR)), planCatalog.INR)
		if usd {
			fmt.Fprintf(&b, "- %s: %s a month (%s in India). %s\n", plan.Name,
				formatMoney(int64(plan.Price(planCatalog.USD)), planCatalog.USD), inr, plan.Summary)
		} else {
			fmt.Fprintf(&b, "- %s: %s a month. %s\n", plan.Name, inr, plan.Summary)
		}
	}
	fmt.Fprintf(&b, "- [Compare plans](%s/pricing)\n", origin)

	if len(posts) > 0 {
		b.WriteString("\n## Guides\n\n")
		for _, post := range posts {
			fmt.Fprintf(&b, "- [%s](%s/blog/%s): %s\n", post.Title, origin, post.Slug, post.Description)
		}
	}

	b.WriteString("\n## Optional\n\n")
	for _, p := range infoPages {
		if !p.product {
			fmt.Fprintf(&b, "- [%s](%s%s): %s\n", pageName(p.title), origin, p.path, p.description)
		}
	}
	return []byte(b.String())
}

// ServeLLMs writes llms.txt, with dollar prices only once ShowUSD is on.
func (s *Site) ServeLLMs(w http.ResponseWriter, _ *http.Request) {
	serveText(w, "text/plain; charset=utf-8", s.llms[s.country != nil])
}
