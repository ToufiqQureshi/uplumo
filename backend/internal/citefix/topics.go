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

// topicStopwords are function words and generic web words that never name a
// topic on their own. Template text ("FAQ", "| Brand") is removed by
// boilerplateSegments instead, so this list stays small.
var topicStopwords = map[string]bool{
	"the": true, "and": true, "for": true, "with": true, "from": true, "your": true,
	"this": true, "that": true, "how": true, "what": true, "why": true, "are": true,
	"our": true, "you": true, "all": true, "about": true, "home": true, "contact": true,
	"page": true, "services": true, "best": true, "more": true, "an": true, "as": true,
	"at": true, "be": true, "by": true, "can": true, "do": true, "does": true, "if": true,
	"in": true, "into": true, "is": true, "it": true, "its": true, "of": true, "on": true,
	"or": true, "so": true, "to": true, "up": true, "no": true, "not": true, "vs": true,
	"these": true, "those": true, "they": true, "them": true, "their": true, "there": true,
	"when": true, "where": true, "which": true, "who": true, "will": true, "should": true,
	"before": true, "after": true, "than": true, "then": true, "just": true, "also": true,
	"get": true, "make": true, "use": true, "tell": true, "need": true, "want": true,
	"new": true, "one": true, "but": true, "better": true, "find": true, "way": true,
	"ways": true, "tips": true, "here": true, "now": true,
	// Generic title words: a guide, a site, a fix or a plan is not a subject.
	"guide": true, "guides": true, "site": true, "sites": true, "website": true,
	"websites": true, "pages": true, "show": true, "shows": true, "remove": true,
	"diagnose": true, "fix": true, "fixes": true, "plan": true, "step": true,
	"steps": true, "check": true, "start": true, "work": true, "works": true,
	"help": true, "improve": true, "simple": true, "practical": true, "complete": true,
	"ultimate": true, "quick": true, "easy": true, "realistic": true, "useful": true,
	"keep": true, "turn": true, "choose": true, "explained": true, "inside": true, "faq": true, "faqs": true, "source": true, "sources": true,
	"related": true, "में": true, "और": true, "का": true, "की": true, "के": true, "से": true,
	"को": true, "है": true, "पर": true,
}

// titleSeparators split "Topic | Brand" titles so a repeated brand part is
// treated like any other repeated template text.
var titleSeparators = strings.NewReplacer(" | ", "\x00", " - ", "\x00", " – ", "\x00", " — ", "\x00", " · ", "\x00")

func normalizeSegment(text string) string {
	return strings.Join(strings.Fields(strings.ToLower(text)), " ")
}

// pageSegments is a page's title parts and headings, normalized.
type pageSegments struct {
	title, headings []string
}

func topicSegments(page Page) pageSegments {
	var out pageSegments
	for _, part := range strings.Split(titleSeparators.Replace(page.Title), "\x00") {
		if segment := normalizeSegment(part); segment != "" && !slices.Contains(out.title, segment) {
			out.title = append(out.title, segment)
		}
	}
	for _, heading := range page.Signals.TopicHeadings {
		if segment := normalizeSegment(heading); segment != "" && !slices.Contains(out.headings, segment) {
			out.headings = append(out.headings, segment)
		}
	}
	return out
}

// boilerplateSegments finds text that repeats word for word across pages: a
// site name in titles, "FAQ", "Related guides", footer headings. It comes
// from a template and says nothing about what a page is about. brandWords
// are the words of repeated title parts ("uplumo" from "... | Uplumo"), so
// "Uplumo AI Writing" is read as "ai writing".
func boilerplateSegments(pages []pageSegments) (boilerplate, brandWords map[string]bool) {
	counts := map[string]int{}
	inTitle := map[string]bool{}
	for _, page := range pages {
		seen := map[string]bool{}
		for _, segment := range page.title {
			inTitle[segment] = true
			seen[segment] = true
		}
		for _, segment := range page.headings {
			seen[segment] = true
		}
		for segment := range seen {
			counts[segment]++
		}
	}
	boilerplate, brandWords = map[string]bool{}, map[string]bool{}
	for segment, count := range counts {
		if count < 3 || count*20 < len(pages) {
			continue
		}
		boilerplate[segment] = true
		if inTitle[segment] {
			for _, word := range topicWords(segment) {
				brandWords[word] = true
			}
		}
	}
	return boilerplate, brandWords
}

func topicWords(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && !unicode.IsMark(r)
	})
}

// topicTerms reads a page's subject from its title. Section headings are
// generic ("Why this happens", "Start here") and only stand in when the title
// says nothing beyond template text.
func topicTerms(page pageSegments, boilerplate, brandWords map[string]bool) []string {
	if terms := segmentTerms(page.title, boilerplate, brandWords); len(terms) > 0 {
		return terms
	}
	for _, heading := range page.headings {
		if terms := segmentTerms([]string{heading}, boilerplate, brandWords); len(terms) > 0 {
			return terms
		}
	}
	return nil
}

func segmentTerms(segments []string, boilerplate, brandWords map[string]bool) []string {
	seen := map[string]bool{}
	terms := make([]string, 0, maxTopicTermsPerPage)
	for _, part := range segments {
		if boilerplate[part] {
			continue
		}
		previous := ""
		for _, word := range topicWords(part) {
			length := len([]rune(word))
			if length < 2 || length > 64 || topicStopwords[word] || brandWords[word] {
				previous = ""
				continue
			}
			candidates := []string{word, strings.TrimSpace(previous + " " + word)}
			if allDigits(word) {
				// "2026" alone is not a topic; "web 3" may be.
				candidates[0] = ""
				if previous == "" {
					candidates[1] = ""
				}
			}
			for _, term := range candidates {
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

func allDigits(word string) bool {
	return strings.IndexFunc(word, func(r rune) bool { return !unicode.IsDigit(r) }) == -1
}

func topicURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	u.Fragment = ""
	u.Host = strings.ToLower(u.Host)
	// A trailing slash is kept: /guide and /guide/ can be different pages,
	// as the crawler's normalizeURL also treats them.
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
	segments := make([]pageSegments, len(eligible))
	for i, page := range eligible {
		segments[i] = topicSegments(page)
	}
	boilerplate, brandWords := boilerplateSegments(segments)
	terms := make(map[string][]int)
	for i := range eligible {
		for _, term := range topicTerms(segments[i], boilerplate, brandWords) {
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
