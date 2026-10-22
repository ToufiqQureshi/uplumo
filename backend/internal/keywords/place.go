package keywords

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/toufiqqureshi/uplumo/backend/internal/platform/market"
)

// placeTypes are the registry location types a seed can name for its ideas to
// be limited to. Postal codes, neighbourhoods and sub-districts are left out:
// their names collide with ordinary words far more often than they help.
var placeTypes = map[string]bool{
	"City": true, "State": true, "Union Territory": true, "District": true,
	"Region": true, "County": true, "Municipality": true,
}

// shortPlaceTypes may match with a three-letter name ("Goa"). A three-letter
// city or district ("Pen", "Ten") is too often an ordinary word.
var shortPlaceTypes = map[string]bool{"State": true, "Union Territory": true}

// maxPlaceWords is the longest place name matched in a seed ("navi mumbai",
// "new york city").
const maxPlaceWords = 3

// seedPlace returns the place a seed names ("lonavala" in "best resort in
// lonavala"), lowercased, or "" when it names none. Without a place, Labs ideas
// drift from the customer's town to the whole category (resorts in Goa and
// Jaipur for a Lonavala resort), and every one of those rows is billed. The
// registry is free and cached; when it fails, research runs unfiltered rather
// than failing the search.
func (s *ResearchService) seedPlace(ctx context.Context, org string, locationCode int, seed string) string {
	if s.Registry == nil {
		return ""
	}
	locations, err := s.Registry.Locations(ctx, org, market.ISOCode(locationCode))
	if err != nil {
		s.Logger.WarnContext(ctx, "load locations for the keyword place filter", "err", err)
		return ""
	}
	return placeIn(seed, locations)
}

// placeIn returns the longest registry place name that appears as whole words
// in seed, or "".
func placeIn(seed string, locations []SerpLocation) string {
	names := make(map[string]bool)
	for _, l := range locations {
		if !placeTypes[l.LocationType] {
			continue
		}
		name, _, _ := strings.Cut(l.LocationName, ",")
		name = strings.ToLower(strings.TrimSpace(name))
		// The name goes into a LIKE pattern; skip any that carry its wildcards.
		if strings.ContainsAny(name, `%_\`) {
			continue
		}
		if n := utf8.RuneCountInString(name); n < 3 || (n == 3 && !shortPlaceTypes[l.LocationType]) {
			continue
		}
		names[name] = true
	}
	words := strings.Fields(strings.ToLower(seed))
	for size := min(maxPlaceWords, len(words)); size > 0; size-- {
		for i := 0; i+size <= len(words); i++ {
			if candidate := strings.Join(words[i:i+size], " "); names[candidate] {
				return candidate
			}
		}
	}
	return ""
}
