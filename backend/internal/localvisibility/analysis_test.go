package localvisibility

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"
)

var now = time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)

func ptr[T any](v T) *T { return &v }

func profileFixture(categories []string, reviews, photos int, rating float64, attributes ...string) *Profile {
	return &Profile{Categories: categories, Reviews: reviews, Photos: photos, Rating: ptr(rating), Claimed: true,
		Description: true, Phone: true, Website: true, Hours: true, Attributes: attributes}
}

func titles(fixes []Fix) []string {
	out := make([]string, len(fixes))
	for i, f := range fixes {
		out[i] = f.Status + ": " + f.Title
	}
	return out
}

func TestBuildFixesReportsOnlyGapsTheCompetitorsShow(t *testing.T) {
	you := Contender{Profile: profileFixture([]string{"Dentist"}, 212, 18, 4.6, "wheelchair"), PostsLast30: ptr(0), LastPostAt: ptr(now.AddDate(0, 0, -71))}
	rivals := []Contender{
		{Profile: profileFixture([]string{"Dentist", "Cosmetic dentist"}, 640, 110, 4.8, "wheelchair", "online appointments"), PostsLast30: ptr(5)},
		{Profile: profileFixture([]string{"Dentist", "Cosmetic dentist", "Emergency dental service"}, 388, 64, 4.7, "online appointments"), PostsLast30: ptr(4)},
		{Profile: profileFixture([]string{"Dentist"}, 295, 47, 4.7), PostsLast30: ptr(3)},
	}
	got := strings.Join(titles(buildFixes(you, rivals, now)), "\n")
	for _, want := range []string{
		"missing: Add the category “Cosmetic dentist”",
		"behind: Add photos",
		"behind: Post updates more often",
		"behind: Your rating is the lowest of the group",
		"behind: Turn on the attributes most competitors show",
		"fine: Already complete: business description, opening hours, phone number, website link",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("fixes miss %q:\n%s", want, got)
		}
	}
	for _, notWanted := range []string{"Emergency dental service", "Claim", "review"} {
		if strings.Contains(got, notWanted) {
			t.Errorf("fixes report %q, which fewer than most competitors have or you already match:\n%s", notWanted, got)
		}
	}
}

func TestBuildFixesWithoutCompetitorDataComparesNothing(t *testing.T) {
	you := Contender{Profile: &Profile{Claimed: false, Categories: []string{"Café"}}}
	fixes := buildFixes(you, []Contender{{Profile: nil}}, now)
	if len(fixes) != 1 || fixes[0].Title != "Claim your profile" {
		t.Fatalf("fixes = %v, want only the claim fix", titles(fixes))
	}
	if got := buildFixes(Contender{}, nil, now); len(got) != 0 {
		t.Fatalf("no own profile gives fixes %v", titles(got))
	}
}

func TestCheckSummaryStatesRankAndGaps(t *testing.T) {
	if got := checkSummary("dentist", 5, []Fix{{Status: "missing"}, {Status: "fine"}}); got != "You rank #5 on Google Maps for “dentist” at your address. 1 of the fixes below come from what the top 3 have and you do not." {
		t.Fatalf("summary = %q", got)
	}
	if got := checkSummary("café", 0, nil); !strings.Contains(got, "not in the top 20") || !strings.Contains(got, "everything") {
		t.Fatalf("summary = %q", got)
	}
}

func TestGridPointsAreSpacedAroundTheBusiness(t *testing.T) {
	points := gridPoints(18.5362, 73.8930, 3, 1)
	centre := points[4]
	if centre.Latitude != 18.5362 || centre.Longitude != 73.893 || centre.Row != 1 || centre.Col != 1 {
		t.Fatalf("centre = %+v", centre)
	}
	north := points[1]
	if km := (north.Latitude - centre.Latitude) * 111.32; math.Abs(km-1) > 0.001 {
		t.Fatalf("north point is %.4f km away, want 1", km)
	}
	east := points[5]
	if km := (east.Longitude - centre.Longitude) * 111.32 * math.Cos(centre.Latitude*math.Pi/180); math.Abs(km-1) > 0.001 {
		t.Fatalf("east point is %.4f km away, want 1", km)
	}
}

func TestGridSummaryNamesTheMostFrequentLeader(t *testing.T) {
	points := []GridPoint{
		{Rank: 1, Top: []string{"You"}},
		{Rank: 2, Top: []string{"A", "You"}},
		{Rank: 9, Top: []string{"Bright Smiles"}},
		{Rank: 0, Top: []string{"Bright Smiles"}},
		{Rank: 0, Top: []string{"Aura"}},
		{Missing: true},
	}
	want := "You are in the top 3 at 2 of 5 points and not in the top 20 at 2. Where you are not in the top 3, Bright Smiles is #1 most often (2 points)."
	if got := gridSummary(points); got != want {
		t.Fatalf("summary = %q\nwant      %q", got, want)
	}
	if got := gridSummary([]GridPoint{{Missing: true}}); !strings.HasPrefix(got, "No point") {
		t.Fatalf("summary = %q", got)
	}
}

func TestCountThemesCountsOnlyKnownReviewsOnce(t *testing.T) {
	known := map[string]string{"you:0": "you", "you:1": "you", "r0:0": "r0"}
	answer := `{"themes":[
		{"name":"Waiting time","mentions":[{"id":"you:0","sentiment":"negative"},{"id":"you:0","sentiment":"negative"},{"id":"r0:0","sentiment":"positive"},{"id":"you:99","sentiment":"negative"}]},
		{"name":"  ","mentions":[{"id":"you:1","sentiment":"positive"}]},
		{"name":"Price","mentions":[{"id":"you:1","sentiment":"maybe"}]}
	]}`
	got, err := countThemes(answer, known)
	if err != nil {
		t.Fatal(err)
	}
	if len(got["you"]) != 1 || got["you"][0] != (Theme{Name: "Waiting time", Negative: 1}) {
		t.Fatalf("your themes = %+v, want one negative mention of waiting time", got["you"])
	}
	if len(got["r0"]) != 1 || got["r0"][0].Positive != 1 {
		t.Fatalf("rival themes = %+v", got["r0"])
	}
	if _, err := countThemes("not json", known); err == nil {
		t.Fatal("a non-JSON answer must fail, so it is not charged")
	}
}

func TestReviewStatsAndSummary(t *testing.T) {
	own := reviewsResult{Total: 212, Rating: ptr(4.6), Reviews: []Review{
		{ID: "a", Rating: 5, Answered: true}, {ID: "b", Rating: 2}, {ID: "c", Rating: 2}, {ID: "d", Rating: 4, Answered: true},
	}}
	var result ReviewsResult
	reviewStats(&result, own)
	if *result.ReplyRate != 50 || result.Distribution != [5]int{0, 2, 0, 1, 1} || len(result.Unanswered) != 2 {
		t.Fatalf("stats = %+v", result)
	}
	result.Rivals = []RivalReviews{{ReplyRate: ptr(80)}, {ReplyRate: ptr(100)}}
	result.Themes = []Theme{{Name: "Price", Negative: 1}, {Name: "Waiting time", Negative: 2, Positive: 1}}
	want := "You replied to 50% of your last 4 reviews. The top 3 reply to 90% on average. The most common complaint is “Waiting time” (2 reviews)."
	if got := reviewsSummary(result); got != want {
		t.Fatalf("summary = %q", got)
	}
}

func TestCompareWatchFindsWhatChanged(t *testing.T) {
	before := []Snapshot{{CID: "1", Title: "A", Rating: ptr(4.6), Reviews: 369, Photos: 60, Categories: []string{"Dentist"}, HoursKey: "h1"}}
	current := []Snapshot{
		{CID: "1", Title: "A", Rating: ptr(4.7), Reviews: 388, Photos: 64, Categories: []string{"Dentist", "Emergency dental service"}, HoursKey: "h2", PostsLast7: 2},
		{CID: "2", Title: "B", Reviews: 10, Categories: []string{}},
		{CID: "3", Missing: true},
	}
	changes := compareWatch(current, before)
	if len(changes) != 2 {
		t.Fatalf("changes = %+v, want the two read profiles", changes)
	}
	a := changes[0]
	if *a.NewReviews != 19 || *a.RatingDelta != 0.1 || *a.PhotosDelta != 4 || a.NewPosts != 2 || !a.HoursChanged || len(a.Added) != 1 {
		t.Fatalf("change = %+v", a)
	}
	if changes[1].NewReviews != nil {
		t.Fatal("a competitor without a previous snapshot has no delta")
	}
	if got := watchSummary(changes, false); got != "A got the most new reviews (19); A added the category “Emergency dental service”." {
		t.Fatalf("summary = %q", got)
	}
}

// A business outside the Maps top 20 has no listing in the results; its
// result must still encode lists as [] for the page's schema.
func TestCheckResultOfABusinessOutsideTheTop20EncodesEmptyLists(t *testing.T) {
	done := func(key string) (json.RawMessage, bool) {
		if key == "maps" {
			return json.RawMessage(`{"items":[]}`), true
		}
		return nil, false
	}
	result, _ := buildCheck(runInput{CID: "100", Term: "dentist"}, done, now)
	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), `"categories":null`) {
		t.Fatalf("result has a null list: %s", raw)
	}
}
