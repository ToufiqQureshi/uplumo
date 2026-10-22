package aisearch

import (
	"fmt"
	"testing"
)

func TestBrandMatcherSplitsBrandedFromUnbrandedQuestions(t *testing.T) {
	della := newBrandMatcher(Target{Type: TargetDomain, Value: "www.dellaresorts.com"})
	keyword := newBrandMatcher(Target{Type: TargetKeyword, Value: "Della Resorts"})
	tests := []struct {
		matcher  brandMatcher
		question string
		want     bool
	}{
		{della, "della resorts lonavala", true},
		{della, "Della adventure park in Lonavala", true}, // a word the domain label starts with
		{della, "is dellaresorts.com safe to book", true},
		{della, "dellaresorts reviews", true},
		{della, "best resort in lonavala", false},
		{della, "data resort", false},
		{della, "del mar hotels", false}, // a three-letter prefix is too short to count
		{keyword, "della resorts lonavala price", true},
		{keyword, "resorts in lonavala", false}, // one word of the phrase is not the brand
		{keyword, "dellaresorts", false},
	}
	for _, tt := range tests {
		if got := tt.matcher.branded(tt.question); got != tt.want {
			t.Errorf("%+v branded(%q) = %v, want %v", tt.matcher, tt.question, got, tt.want)
		}
	}
}

func TestShapeSitesStripsWwwSortsAndCaps(t *testing.T) {
	var groups []groupElement
	for i := range maxSitesPerList + 3 {
		groups = append(groups, groupElement{Key: stringPtr(fmt.Sprintf("www.site%d.com", i)), Mentions: new(float64(i))})
	}
	groups = append(groups, groupElement{Key: stringPtr("")})
	sites := shapeSites(groups)
	if len(sites) != maxSitesPerList || sites[0].Domain != fmt.Sprintf("site%d.com", maxSitesPerList+2) {
		t.Fatalf("sites = %+v, want the %d most mentioned without www. or blanks", sites, maxSitesPerList)
	}
	if got := shapeSites(nil); got == nil || len(got) != 0 {
		t.Errorf("no groups = %v, want an empty list, not null", got)
	}
}

func TestShapeBrandsAlongsideLeavesOutTheBrand(t *testing.T) {
	brand := newBrandMatcher(Target{Type: TargetDomain, Value: "dellaresorts.com"})
	got := shapeBrandsAlongside([]groupElement{
		{Key: stringPtr("Della Resorts"), Mentions: new(4.0)},
		{Key: stringPtr("The Machan"), Mentions: new(1.0)},
		{Key: stringPtr("Fariyas Resort Lonavala"), Mentions: new(2.0)},
	}, brand)
	if len(got) != 2 || got[0].Name != "Fariyas Resort Lonavala" || got[1].Name != "The Machan" {
		t.Errorf("brands = %+v, want the two competitors, most mentioned first", got)
	}
}

func TestTopQueriesKeepUnbrandedPromptsBehindManyBrandedOnes(t *testing.T) {
	var mentions []mentionItem
	for i := range topQueriesPerPlatform + 5 {
		mentions = append(mentions, mentionItem{Question: stringPtr(fmt.Sprintf("della resorts offer %d", i)), AISearchVolume: new(float64(90000 - i))})
	}
	mentions = append(mentions, mentionItem{Question: stringPtr("best resort in lonavala"), AISearchVolume: new(2900.0)})
	brand := newBrandMatcher(Target{Type: TargetDomain, Value: "dellaresorts.com"})

	queries, branded, unbranded := shapeTopQueries([]successBundle{{platform: platformGoogle, bundle: platformBundle{mentions: mentions}}}, nil, brand)

	if branded != topQueriesPerPlatform+5 || unbranded != 1 {
		t.Fatalf("counts = %d branded, %d unbranded; want every sampled prompt counted", branded, unbranded)
	}
	if len(queries) != topQueriesPerPlatform+1 || queries[len(queries)-1].Question != "best resort in lonavala" || queries[len(queries)-1].Branded {
		t.Fatalf("queries = %d, last %+v; want the capped branded prompts plus the unbranded one", len(queries), queries[len(queries)-1])
	}
}
