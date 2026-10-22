package aisearch

import (
	"cmp"
	"slices"
	"strings"
	"unicode"
)

const (
	// maxSitesPerList is the provider's cap on each site list of the totals
	// call; it is also how many the page shows.
	maxSitesPerList = 10
	maxBrandsAlong  = 10
	// minBrandWordLength stops a short word ("ai", "the") from marking a
	// question as branded because the domain happens to start with it.
	minBrandWordLength = 4
)

// brandMatcher tells whether a question names the brand being looked up.
type brandMatcher struct {
	label  string // a domain's first label: "dellaresorts" for dellaresorts.com
	host   string // the domain without www.
	phrase string // a brand keyword, lowercased
}

func newBrandMatcher(target Target) brandMatcher {
	value := strings.ToLower(strings.TrimSpace(target.Value))
	if target.Type != TargetDomain {
		return brandMatcher{phrase: strings.Join(words(value), " ")}
	}
	host := strings.TrimPrefix(value, "www.")
	host, _, _ = strings.Cut(host, "/")
	label, _, _ := strings.Cut(host, ".")
	return brandMatcher{label: label, host: host}
}

// words splits s into lowercase letter-and-digit runs.
func words(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// branded reports whether question names the brand. A domain's brand is
// usually its first label written as one or more words ("della resorts" for
// dellaresorts.com), so a question word the label starts with counts. A
// generic word that happens to start the label ("best" for bestbuy.com) is
// counted too: the split is a guide for the customer, not a billed number.
func (m brandMatcher) branded(question string) bool {
	q := strings.ToLower(question)
	if m.phrase != "" {
		return strings.Contains(" "+strings.Join(words(q), " ")+" ", " "+m.phrase+" ")
	}
	if m.label == "" {
		return false
	}
	if strings.Contains(q, m.host) {
		return true
	}
	for _, w := range words(q) {
		if w == m.label || (len(w) >= minBrandWordLength && strings.HasPrefix(m.label, w)) || strings.HasPrefix(w, m.label) {
			return true
		}
	}
	return false
}

// shapeSites turns a site group of the totals call into page rows, most
// mentioned first, without a leading www.
func shapeSites(groups []groupElement) []SiteMentions {
	sites := []SiteMentions{}
	for _, g := range groups {
		domain := strings.TrimPrefix(strings.ToLower(derefOr(g.Key, "")), "www.")
		if domain == "" {
			continue
		}
		sites = append(sites, SiteMentions{Domain: domain, Mentions: roundOrNil(g.Mentions), AISearchVolume: roundOrNil(g.AISearchVolume)})
	}
	slices.SortStableFunc(sites, func(a, b SiteMentions) int { return cmp.Compare(valueOr(b.Mentions, 0), valueOr(a.Mentions, 0)) })
	return sites[:min(len(sites), maxSitesPerList)]
}

// shapeBrandsAlongside lists the other brands named with the target, leaving
// out the target's own name.
func shapeBrandsAlongside(groups []groupElement, brand brandMatcher) []BrandMention {
	brands := []BrandMention{}
	for _, g := range groups {
		name := truncate(strings.TrimSpace(derefOr(g.Key, "")), maxBrandEntityLength)
		if name == "" || brand.branded(name) {
			continue
		}
		brands = append(brands, BrandMention{Name: name, Mentions: roundOrNil(g.Mentions)})
	}
	slices.SortStableFunc(brands, func(a, b BrandMention) int { return cmp.Compare(valueOr(b.Mentions, 0), valueOr(a.Mentions, 0)) })
	return brands[:min(len(brands), maxBrandsAlong)]
}

// pagesFromMentions builds the cited-page rows from a mentions sample that
// holds every answer for the platform, so the top_pages call is not bought:
// each page counts the answers that cited it and their AI search volume.
func pagesFromMentions(platform string, mentions []mentionItem) []topPagesItem {
	type page struct {
		url      string
		mentions float64
		volume   float64
	}
	var order []string
	pages := map[string]*page{}
	for _, m := range mentions {
		seen := map[string]bool{}
		for _, s := range m.Sources {
			u, ok := safeHTTPURL(derefOr(s.URL, ""))
			if !ok || seen[u] {
				continue
			}
			seen[u] = true
			p := pages[u]
			if p == nil {
				p = &page{url: u}
				pages[u] = p
				order = append(order, u)
			}
			p.mentions++
			if m.AISearchVolume != nil {
				p.volume += *m.AISearchVolume
			}
		}
	}
	items := make([]topPagesItem, 0, len(order))
	for _, u := range order {
		p := pages[u]
		items = append(items, topPagesItem{
			Key:      stringPtr(p.url),
			Platform: []groupElement{{Key: stringPtr(platform), Mentions: &p.mentions, AISearchVolume: &p.volume}},
		})
	}
	return items
}

// platformMentions is the provider's mention count for one platform, 0 when
// the totals call did not list it.
func platformMentions(groups []groupElement, platform string) int {
	for _, g := range groups {
		if g.Key != nil && *g.Key == platform && g.Mentions != nil {
			return int(*g.Mentions)
		}
	}
	return 0
}
