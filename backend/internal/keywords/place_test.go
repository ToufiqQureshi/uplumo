package keywords

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func indiaPlaces() []SerpLocation {
	return []SerpLocation{
		{LocationName: "Lonavala,Maharashtra,India", LocationType: "City"},
		{LocationName: "Navi Mumbai,Maharashtra,India", LocationType: "City"},
		{LocationName: "Mumbai,Maharashtra,India", LocationType: "City"},
		{LocationName: "Goa,India", LocationType: "State"},
		{LocationName: "Pen,Maharashtra,India", LocationType: "City"},
		{LocationName: "Best,Noord-Brabant,Netherlands", LocationType: "Postal Code"},
		{LocationName: "Bandra West,Mumbai,Maharashtra,India", LocationType: "Neighborhood"},
		{LocationName: "Un,Gujarat,India", LocationType: "State"},
	}
}

func TestPlaceIn(t *testing.T) {
	tests := []struct {
		seed, want string
	}{
		{"best resort in lonavala", "lonavala"},
		{"Best Resort In LONAVALA", "lonavala"},
		{"plumber navi mumbai", "navi mumbai"}, // the longest name wins over "mumbai"
		{"villas in goa", "goa"},               // a three-letter state is a place
		{"best pen drive", ""},                 // a three-letter city is too often a word
		{"best seo tool", ""},                  // a postal-code name is not a place
		{"cafe in bandra west", ""},            // neighbourhoods are left out
		{"un keyword", ""},                     // two letters never match
		{"lonavalaresort", ""},                 // whole words only
		{"", ""},
	}
	for _, tt := range tests {
		if got := placeIn(tt.seed, indiaPlaces()); got != tt.want {
			t.Errorf("placeIn(%q) = %q, want %q", tt.seed, got, tt.want)
		}
	}
	if got := placeIn("resort in lonavala", []SerpLocation{{LocationName: "Lona%vala,India", LocationType: "City"}}); got != "" {
		t.Errorf("a name with a LIKE wildcard must never become a filter, got %q", got)
	}
}

func TestResearchLimitsIdeasToTheSeedPlaceForTheSamePaidCalls(t *testing.T) {
	f := newResearchFixture(t)
	f.registry.locations = indiaPlaces()
	f.data.labsFn = func(req LabsRequest) ([]LabsItem, error) {
		return items(string(req.Source)+" lonavala", 10), nil
	}
	got, err := f.research(t, func(in *ResearchInput) { in.Keywords, in.LocationCode = []string{"best resort in lonavala"}, 2356 })
	if err != nil {
		t.Fatal(err)
	}
	suggestions, ideas := f.data.labsCalls(SourceSuggestions), f.data.labsCalls(SourceIdeas)
	if len(f.data.labs) != 2 || len(suggestions) != 1 || len(ideas) != 1 {
		t.Fatalf("paid Labs calls = %+v, want one suggestions and one ideas task, as without the filter", f.data.labs)
	}
	if ideas[0].MustContain != "lonavala" {
		t.Errorf("ideas MustContain = %q, want the place named in the seed", ideas[0].MustContain)
	}
	if suggestions[0].MustContain != "" {
		t.Errorf("suggestions MustContain = %q, want none: they already contain the seed", suggestions[0].MustContain)
	}
	if got.PlaceFilter == nil || *got.PlaceFilter != "lonavala" {
		t.Errorf("PlaceFilter = %v, want the customer told which place the ideas were limited to", got.PlaceFilter)
	}

	// Asking again is a cache hit; the same seed without the place is a different purchase.
	if _, err := f.research(t, func(in *ResearchInput) { in.Keywords, in.LocationCode = []string{"best resort in lonavala"}, 2356 }); err != nil {
		t.Fatal(err)
	}
	if len(f.data.labs) != 2 {
		t.Fatalf("a repeated search bought %d more tasks, want a cache hit", len(f.data.labs)-2)
	}
	broad, err := f.research(t, func(in *ResearchInput) { in.Keywords, in.LocationCode = []string{"best resort"}, 2356 })
	if err != nil {
		t.Fatal(err)
	}
	if ideas := f.data.labsCalls(SourceIdeas); len(ideas) != 2 || ideas[1].MustContain != "" {
		t.Errorf("ideas calls = %+v, want an unfiltered task for a seed without a place", ideas)
	}
	if broad.PlaceFilter != nil {
		t.Errorf("PlaceFilter = %q, want nil without a place", *broad.PlaceFilter)
	}
}

func TestResearchFiltersTheRelatedFallbackToo(t *testing.T) {
	f := newResearchFixture(t)
	f.registry.locations = indiaPlaces()
	f.data.labsFn = func(req LabsRequest) ([]LabsItem, error) {
		if req.Source == SourceRelated {
			return items("related goa", 10), nil
		}
		return nil, nil // thin results trigger the related fallback
	}
	if _, err := f.research(t, func(in *ResearchInput) { in.Keywords, in.LocationCode = []string{"villas in goa"}, 2356 }); err != nil {
		t.Fatal(err)
	}
	related := f.data.labsCalls(SourceRelated)
	if len(related) != 1 || related[0].MustContain != "goa" {
		t.Errorf("related calls = %+v, want one task limited to the place", related)
	}
}

func TestResearchRunsUnfilteredWhenTheRegistryFails(t *testing.T) {
	f := newResearchFixture(t)
	f.registry.err = errors.New("registry down")
	f.data.labsFn = func(req LabsRequest) ([]LabsItem, error) { return items(string(req.Source), 10), nil }
	got, err := f.research(t, func(in *ResearchInput) { in.Keywords, in.LocationCode = []string{"best resort in lonavala"}, 2356 })
	if err != nil {
		t.Fatalf("research failed with the registry down: %v", err)
	}
	if ideas := f.data.labsCalls(SourceIdeas); len(ideas) != 1 || ideas[0].MustContain != "" || got.PlaceFilter != nil {
		t.Errorf("ideas = %+v, filter = %v, want one unfiltered task", ideas, got.PlaceFilter)
	}
}

func TestResearchNeverFiltersGoogleAdsMarkets(t *testing.T) {
	f := newResearchFixture(t)
	f.registry.locations = []SerpLocation{{LocationName: "Nassau,Bahamas", LocationType: "City"}}
	f.data.ideasFn = func() ([]AdsItem, error) { return []AdsItem{{Keyword: "hotel nassau", SearchVolume: intp(10)}}, nil }
	const adsMarket = 2044 // a Google Ads-only market (see market_test.go)
	got, err := f.research(t, func(in *ResearchInput) { in.Keywords, in.LocationCode = []string{"hotel nassau"}, adsMarket })
	if err != nil {
		t.Fatal(err)
	}
	if f.registry.calls != 0 || got.PlaceFilter != nil {
		t.Errorf("registry calls = %d, filter = %v: Google Ads takes no filter, so no place lookup", f.registry.calls, got.PlaceFilter)
	}
	if !slices.Equal(got.Rows[0].SerpFeatures, []string{}) {
		t.Errorf("Ads row SerpFeatures = %v, want an empty list", got.Rows[0].SerpFeatures)
	}
}

func TestSeedPlaceWithoutRegistry(t *testing.T) {
	s := &ResearchService{}
	if got := s.seedPlace(context.Background(), "org", 2356, "resort in lonavala"); got != "" {
		t.Errorf("seedPlace without a registry = %q, want none", got)
	}
}
