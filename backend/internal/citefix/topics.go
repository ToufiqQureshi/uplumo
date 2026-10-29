package citefix

import (
	"cmp"
	"fmt"
	"math"
	"net/url"
	"slices"
	"strings"
	"unicode"
)

const (
	maxTopicPages        = 500
	maxTopicClusters     = 20
	maxTopicTermsPerPage = 60
	maxTopicExamples     = 20
)

// TopicReport describes a bounded sample of the pages from one completed scan.
// It contains estimates from HTML, not a claim about a search engine's ranking.
type TopicReport struct {
	Score           *int           `json:"score"`
	PagesAnalyzed   int            `json:"pagesAnalyzed"`
	Limited         bool           `json:"limited"`
	ClustersLimited bool           `json:"clustersLimited"`
	Clusters        []TopicCluster `json:"clusters"`
	Recommendations []string       `json:"recommendations"`
}

// TopicCluster groups pages sharing title and heading terms.
type TopicCluster struct {
	Topic          string    `json:"topic"`
	PageCount      int       `json:"pageCount"`
	InterlinkRatio float64   `json:"interlinkRatio"`
	Pillar         *PageRef  `json:"pillar"`
	Pages          []PageRef `json:"pages"`
	Orphans        []PageRef `json:"orphans"`
	Thin           []PageRef `json:"thin"`
	Duplicates     []PageRef `json:"duplicates"`
}

var topicStopwords = map[string]bool{
	"the": true, "and": true, "for": true, "with": true, "from": true, "your": true,
	"this": true, "that": true, "how": true, "what": true, "why": true, "are": true,
	"our": true, "you": true, "all": true, "about": true, "home": true, "contact": true,
	"page": true, "services": true, "best": true, "more": true, "में": true, "और": true,
	"का": true, "की": true, "के": true, "से": true, "को": true, "है": true, "पर": true,
}

func topicTerms(page Page) []string {
	parts := []string{page.Title}
	parts = append(parts, page.Signals.TopicHeadings...)
	seen := map[string]bool{}
	terms := make([]string, 0, maxTopicTermsPerPage)
	for _, part := range parts {
		words := strings.FieldsFunc(strings.ToLower(part), func(r rune) bool {
			return !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsMark(r)
		})
		previous := ""
		for _, word := range words {
			length := len([]rune(word))
			if length < 2 || length > 64 || topicStopwords[word] {
				previous = ""
				continue
			}
			for _, term := range []string{word, strings.TrimSpace(previous + " " + word)} {
				if term == "" || seen[term] {
					continue
				}
				seen[term] = true
				terms = append(terms, term)
				if len(terms) == maxTopicTermsPerPage {
					return terms
				}
			}
			previous = word
		}
	}
	return terms
}

func topicURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	if u.Path != "/" {
		u.Path = strings.TrimRight(u.Path, "/")
	}
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String()
}

func topicRef(page Page) PageRef { return PageRef{URL: page.URL, Title: page.Title} }

// buildTopics is pure: it uses only already-stored crawl results and issues.
func buildTopics(pages []Page, duplicateURLs, shellURLs map[string]struct{}) TopicReport {
	out := TopicReport{Clusters: []TopicCluster{}, Recommendations: []string{}}
	eligible := make([]Page, 0, min(len(pages), maxTopicPages))
	for _, page := range pages {
		_, shell := shellURLs[page.URL]
		if shell || !isContentPage(page) || page.Signals == nil {
			continue
		}
		if len(eligible) == maxTopicPages {
			out.Limited = true
			break
		}
		eligible = append(eligible, page)
	}
	out.PagesAnalyzed = len(eligible)
	if len(eligible) == 0 {
		out.Recommendations = append(out.Recommendations, "Scan some readable content pages to see your topics.")
		return out
	}
	byURL := make(map[string]int, len(eligible))
	for i, page := range eligible {
		byURL[topicURL(page.URL)] = i
	}
	inbound := make(map[string]bool, len(eligible))
	for _, page := range eligible {
		for _, target := range page.Signals.TopicLinks {
			key := topicURL(target)
			if key != topicURL(page.URL) {
				inbound[key] = true
			}
		}
	}
	terms := make(map[string][]int)
	for i, page := range eligible {
		for _, term := range topicTerms(page) {
			terms[term] = append(terms[term], i)
		}
	}
	// Terms with the same member pages describe the same cluster. Prefer the
	// more specific phrase, then a stable lexical tie-breaker.
	type group struct {
		term    string
		members []int
	}
	groups := map[string]group{}
	for term, members := range terms {
		if len(eligible) >= 5 && len(members)*5 > len(eligible)*4 {
			continue
		}
		var key strings.Builder
		for _, i := range members {
			fmt.Fprintf(&key, "%d,", i)
		}
		old, exists := groups[key.String()]
		if !exists || strings.Count(term, " ") > strings.Count(old.term, " ") || strings.Count(term, " ") == strings.Count(old.term, " ") && term < old.term {
			groups[key.String()] = group{term: term, members: members}
		}
	}
	ordered := make([]group, 0, len(groups))
	for _, item := range groups {
		ordered = append(ordered, item)
	}
	slices.SortFunc(ordered, func(a, b group) int {
		return cmp.Or(cmp.Compare(len(b.members), len(a.members)), strings.Compare(a.term, b.term))
	})
	out.ClustersLimited = len(ordered) > maxTopicClusters
	ordered = ordered[:min(len(ordered), maxTopicClusters)]
	for _, item := range ordered {
		cluster := TopicCluster{Topic: item.term, PageCount: len(item.members), Pages: []PageRef{}, Orphans: []PageRef{}, Thin: []PageRef{}, Duplicates: []PageRef{}}
		memberSet := map[int]bool{}
		for _, index := range item.members {
			memberSet[index] = true
		}
		linked := 0
		for _, index := range item.members {
			page := eligible[index]
			ref := topicRef(page)
			if len(cluster.Pages) < maxTopicExamples {
				cluster.Pages = append(cluster.Pages, ref)
			}
			parsed, _ := url.Parse(page.URL)
			if parsed != nil && parsed.Path != "" && parsed.Path != "/" && !inbound[topicURL(page.URL)] && len(cluster.Orphans) < maxTopicExamples {
				cluster.Orphans = append(cluster.Orphans, ref)
			}
			if page.WordCount < minWords && len(cluster.Thin) < maxTopicExamples {
				cluster.Thin = append(cluster.Thin, ref)
			}
			if _, duplicate := duplicateURLs[page.URL]; duplicate && len(cluster.Duplicates) < maxTopicExamples {
				cluster.Duplicates = append(cluster.Duplicates, ref)
			}
			otherLinks := 0
			for _, target := range page.Signals.TopicLinks {
				if targetIndex, ok := byURL[topicURL(target)]; ok && targetIndex != index && memberSet[targetIndex] {
					otherLinks++
				}
			}
			if otherLinks > 0 {
				linked++
			}
			if cluster.Pillar == nil && page.WordCount >= 600 && (len(item.members) == 1 || otherLinks > 0) && strings.Contains(strings.ToLower(page.Title), item.term) {
				pillar := ref
				cluster.Pillar = &pillar
			}
		}
		cluster.InterlinkRatio = math.Round(float64(linked)*100/float64(cluster.PageCount)) / 100
		out.Clusters = append(out.Clusters, cluster)
	}
	if len(out.Clusters) == 0 {
		zero := 0
		out.Score = &zero
		out.Recommendations = append(out.Recommendations, "No clear topic was found in the scanned page titles and headings.")
		return out
	}
	var links, pillars float64
	for _, cluster := range out.Clusters {
		links += cluster.InterlinkRatio
		if cluster.Pillar != nil {
			pillars++
		}
		if cluster.PageCount == 1 {
			out.Recommendations = append(out.Recommendations, fmt.Sprintf("Topic %q has only 1 page; add related pages.", cluster.Topic))
		}
		if cluster.Pillar == nil {
			out.Recommendations = append(out.Recommendations, fmt.Sprintf("Write a pillar page for %s.", cluster.Topic))
		}
		if cluster.PageCount > 1 && cluster.InterlinkRatio < 0.5 {
			out.Recommendations = append(out.Recommendations, fmt.Sprintf("Link these %d %s pages together.", min(cluster.PageCount, 4), cluster.Topic))
		}
	}
	count := float64(len(out.Clusters))
	score := int(math.Round(40*math.Min(float64(out.Clusters[0].PageCount)/5, 1) + 30*links/count + 20*pillars/count + 10*math.Min(count/3, 1)))
	out.Score = &score
	return out
}
