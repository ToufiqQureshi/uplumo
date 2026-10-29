package citefix

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/audit"
)

func topicPage(url, title string, words int, links ...string) Page {
	return Page{URL: url, Title: title, StatusCode: 200, Indexable: true, WordCount: words, Signals: &audit.AIPageSignals{TopicLinks: links}}
}

func TestTopicReportEmptyAndSinglePage(t *testing.T) {
	for _, tc := range []struct {
		name     string
		pages    []Page
		score    *int
		clusters int
	}{
		{"no pages", nil, nil, 0},
		{"one Hindi page", []Page{topicPage("https://site.example/solar", "सौर पैनल गाइड", 300)}, new(11), 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := buildTopics(tc.pages, nil, nil)
			if (got.Score == nil) != (tc.score == nil) || got.Score != nil && *got.Score != *tc.score {
				t.Fatalf("score = %v, want %v", got.Score, tc.score)
			}
			if len(got.Clusters) != tc.clusters || got.PagesAnalyzed != len(tc.pages) {
				t.Fatalf("topics = %+v", got)
			}
			if len(tc.pages) == 1 && (!strings.Contains(got.Clusters[0].Topic, "पैनल") || got.Clusters[0].PageCount != 1 || !strings.Contains(strings.Join(got.Recommendations, " "), "only 1 page")) {
				t.Fatalf("Hindi cluster = %+v", got)
			}
		})
	}
}

func TestTopicReportLinkedPillarAndIssues(t *testing.T) {
	pages := []Page{
		topicPage("https://site.example/solar", "Solar panels", 900, "https://site.example/cost"),
		topicPage("https://site.example/cost", "Solar panels cost", 200),
	}
	got := buildTopics(pages, map[string]struct{}{pages[1].URL: {}}, nil)
	if len(got.Clusters) != 2 || got.Clusters[0].Topic != "solar panels" {
		t.Fatalf("clusters = %+v", got.Clusters)
	}
	cluster := got.Clusters[0]
	if cluster.PageCount != 2 || cluster.InterlinkRatio != 0.5 || cluster.Pillar == nil || cluster.Pillar.URL != pages[0].URL {
		t.Fatalf("cluster = %+v", cluster)
	}
	if len(cluster.Orphans) != 1 || cluster.Orphans[0].URL != pages[0].URL || len(cluster.Thin) != 1 || len(cluster.Duplicates) != 1 || cluster.Thin[0].URL != pages[1].URL {
		t.Fatalf("issues = %+v", cluster)
	}
	// 40*(2/5) coverage + 30*(0.5/2) links + 20*(1/2) pillar
	// + 10*(2/3) breadth = 40.17, rounded to 40.
	if got.Score == nil || *got.Score != 40 {
		t.Fatalf("score = %v, want 40", got.Score)
	}
	if !slices.ContainsFunc(got.Recommendations, func(s string) bool { return strings.Contains(s, "Write a pillar page") }) {
		t.Fatalf("recommendations = %v", got.Recommendations)
	}
}

func TestTopicReportLimits500PagesAndClusters(t *testing.T) {
	pages := make([]Page, 0, 501)
	for i := range 501 {
		pages = append(pages, topicPage(fmt.Sprintf("https://site.example/%03d", i), fmt.Sprintf("Room service guide %03d", i), 350))
	}
	got := buildTopics(pages, nil, nil)
	if got.PagesAnalyzed != 500 || !got.Limited || !got.ClustersLimited || len(got.Clusters) != 20 || got.Score == nil {
		t.Fatalf("bounds: pages %d limited %v clusters %d", got.PagesAnalyzed, got.Limited, len(got.Clusters))
	}
	pages = pages[:500]
	if got := buildTopics(pages, nil, nil); got.PagesAnalyzed != 500 || got.Limited {
		t.Fatalf("exact limit: %+v", got)
	}
}

func TestTopicReportIgnoresTemplateHeadingsAndBrand(t *testing.T) {
	// The live uplumo.com scan: every blog post carries the same H2s and
	// footer headings and ends its title with "| Uplumo". The old report
	// showed "make", "faq", "sources", "in" and "related uplumo" as topics.
	template := []string{
		"Why this problem happens", "Check these signals before changing the site",
		"Fix it in a safe order", "How to tell the fix worked", "Mistakes that make this worse",
		"FAQ", "Related Uplumo guides", "Sources", "Product", "Company", "Legal",
	}
	titles := []string{
		"Google Search Console for small businesses", "Search Console sitemap couldn't fetch",
		"Search Console impressions up but clicks down", "Track AI Overview visibility in Search Console",
		"Keyword research for service pages", "Keyword cannibalization: how to fix it",
		"Keyword research tools show different volume", "Fix redirect chains and loops",
		"Core Web Vitals explained", "Ethical backlinks for small business",
	}
	pages := make([]Page, 0, len(titles))
	for i, title := range titles {
		page := topicPage(fmt.Sprintf("https://site.example/blog/%d", i), title+" | Uplumo", 900)
		page.Signals.TopicHeadings = append([]string{title}, template...)
		if i < 2 {
			// A generic section heading on two posts: below the template
			// threshold, so only the title-first rule keeps it out.
			page.Signals.TopicHeadings = append(page.Signals.TopicHeadings, "Useful next steps")
		}
		pages = append(pages, page)
	}
	// The brand also opens some titles, as in "Uplumo AI Writing".
	for i, title := range []string{"Uplumo AI writing", "Uplumo rank tracker", "Inside Uplumo site crawler"} {
		page := topicPage(fmt.Sprintf("https://site.example/blog/brand-%d", i), title+" | Uplumo", 900)
		page.Signals.TopicHeadings = template
		pages = append(pages, page)
	}
	// Product pages without the blog template, as on the live site, so the
	// template sits on about 60% of pages: below the 80% "everywhere" cut.
	for i, title := range []string{"Pricing plans", "About the team", "Rank tracking tool", "Site audit tool", "AI visibility tool", "Content editor tool", "Contact support"} {
		pages = append(pages, topicPage(fmt.Sprintf("https://site.example/p/%d", i), title, 400))
	}
	got := buildTopics(pages, nil, nil)
	topics := make([]string, 0, len(got.Clusters))
	for _, cluster := range got.Clusters {
		topics = append(topics, cluster.Topic)
	}
	// Every topic must come from a page's own subject, never from the
	// template headings or the "| Uplumo" suffix.
	for _, topic := range topics {
		if !slices.ContainsFunc(pages, func(p Page) bool {
			own := strings.ToLower(strings.TrimSuffix(p.Title, " | Uplumo"))
			return strings.Contains(own, topic)
		}) {
			t.Fatalf("topic %q is not in any page's own title; topics = %v", topic, topics)
		}
		if strings.Contains(topic, "uplumo") {
			t.Fatalf("brand became topic %q; topics = %v", topic, topics)
		}
	}
	for _, want := range []string{"search console", "keyword research"} {
		if !slices.Contains(topics, want) {
			t.Fatalf("topic %q missing; topics = %v", want, topics)
		}
	}
	for _, rec := range got.Recommendations {
		if strings.Contains(rec, "faq") || strings.Contains(rec, "uplumo") {
			t.Fatalf("recommendation names template text: %q", rec)
		}
	}
}

func TestTopicReportFallsBackToHeadingsWhenTitleIsOnlyTheBrand(t *testing.T) {
	// A small site that titles every page "Acme": the title says nothing, so
	// the page's own heading names its topic.
	pages := []Page{
		topicPage("https://site.example/a", "Acme", 700),
		topicPage("https://site.example/b", "Acme", 700),
		topicPage("https://site.example/c", "Acme", 700),
	}
	// "Customer stories" is a template heading on every page and comes first.
	pages[0].Signals.TopicHeadings = []string{"Customer stories", "Rooftop solar subsidy"}
	pages[1].Signals.TopicHeadings = []string{"Customer stories", "Rooftop solar cost"}
	pages[2].Signals.TopicHeadings = []string{"Customer stories", "Wind turbines"}
	got := buildTopics(pages, nil, nil)
	if !slices.ContainsFunc(got.Clusters, func(c TopicCluster) bool { return c.Topic == "rooftop solar" && c.PageCount == 2 }) {
		t.Fatalf("clusters = %+v", got.Clusters)
	}
	if slices.ContainsFunc(got.Clusters, func(c TopicCluster) bool { return strings.Contains(c.Topic, "acme") }) {
		t.Fatalf("brand became a topic: %+v", got.Clusters)
	}
}

func TestTopicURLKeepsTrailingSlashPagesApart(t *testing.T) {
	if topicURL("https://Site.example/guide") == topicURL("https://site.example/guide/") {
		t.Fatal("/guide and /guide/ collapsed into one page; they can be different pages")
	}
	if got := topicURL("https://Site.example/guide/#top"); got != "https://site.example/guide/" {
		t.Fatalf("topicURL = %q, want host lower-cased and fragment dropped", got)
	}
	if got := topicURL("https://site.example"); got != "https://site.example/" {
		t.Fatalf("topicURL(root) = %q", got)
	}
}
