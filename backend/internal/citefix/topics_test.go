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
