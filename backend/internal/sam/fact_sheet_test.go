package sam

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/plans"
)

// numberWords spells the site counts the fact sheet writes out in prose.
var numberWords = map[string]int{"one": 1, "two": 2, "three": 3, "four": 4, "five": 5, "ten": 10, "twenty": 20}

// TestFactSheetMatchesCatalog keeps Sam from quoting plan limits the catalog
// no longer sells: every plan line and the agency site-allowance sentence
// must agree with plans.Catalog().
func TestFactSheetMatchesCatalog(t *testing.T) {
	for _, d := range plans.Catalog() {
		line := regexp.MustCompile(`(?m)^- ` + d.Name + ` \(₹([\d,]+)\): (\d+) websites?`).FindStringSubmatch(productFactSheet)
		if line == nil {
			t.Errorf("fact sheet has no plan line for %s", d.Name)
			continue
		}
		if got := strings.ReplaceAll(line[1], ",", ""); got != strconv.Itoa(d.PriceINR) {
			t.Errorf("%s price = ₹%s, catalog ₹%d", d.Name, line[1], d.PriceINR)
		}
		if got, _ := strconv.Atoi(line[2]); got != d.Sites {
			t.Errorf("%s websites = %d, catalog %d", d.Name, got, d.Sites)
		}
	}

	allowance := regexp.MustCompile(`site allowance: Free and Basic include (\w+) site, Pro (\w+) and Business (\w+)\b`).FindStringSubmatch(productFactSheet)
	if allowance == nil {
		t.Fatal("fact sheet has no agency site-allowance sentence")
	}
	for i, id := range []string{plans.Basic, plans.Pro, plans.Business} {
		d, _ := plans.Get(id)
		if got, ok := numberWords[allowance[i+1]]; !ok || got != d.Sites {
			t.Errorf("agency sentence says %s for %s, catalog %d", allowance[i+1], d.Name, d.Sites)
		}
	}
}
