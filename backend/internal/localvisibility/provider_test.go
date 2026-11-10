package localvisibility

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// The prices on the buttons, from DataForSEO's Oct 2026 prices at priority 2
// and the 1.28 credit markup. A change to a task body changes these.
func TestPriceListMatchesTheTasksEachActionPosts(t *testing.T) {
	got := PriceList()
	want := Prices{Find: 3, Check: 40, Grid3: 14, Grid5: 39, Reviews: 61, WatchPerCompetitor: 10, Reply: 1}
	if got != want {
		t.Fatalf("PriceList() = %+v, want %+v", got, want)
	}
}

func TestTaskBodiesUsePriorityAndTheBusinessCID(t *testing.T) {
	in := runInput{CID: "123", Term: "dentist", Language: "pl", Latitude: 52.2297, Longitude: 21.0122, Rivals: []rival{{CID: "9"}}}
	maps := firstTasks(KindCheck, in)[0].Body
	if maps["location_coordinate"] != "52.2297,21.0122,15z" || maps["priority"] != 2 || maps["depth"] != 20 || maps["language_code"] != "pl" {
		t.Fatalf("maps task = %v", maps)
	}
	profile := profileTasks(in, []Place{{CID: "9"}})
	if len(profile) != 4 || profile[0].Body["keyword"] != "cid:123" || profile[2].Body["keyword"] != "cid:9" {
		t.Fatalf("profile tasks = %+v", profile)
	}
	reviews := firstTasks(KindReviews, in)
	if reviews[0].Body["cid"] != "123" || reviews[0].Body["depth"] != 100 || reviews[1].Body["depth"] != 50 || reviews[0].Body["sort_by"] != "newest" {
		t.Fatalf("reviews tasks = %+v", reviews)
	}
	if _, hasKeyword := reviews[0].Body["keyword"]; hasKeyword {
		t.Fatal("reviews must name the business by cid, not keyword")
	}
	grid := firstTasks(KindGrid, runInput{CID: "1", Term: "x", Language: "en", Size: 5, SpacingKm: 2})
	if len(grid) != 25 || grid[0].Body["location_coordinate"] == grid[24].Body["location_coordinate"] {
		t.Fatalf("grid tasks = %d, corners equal", len(grid))
	}
}

func TestCoordinateKeepsSevenDecimals(t *testing.T) {
	if got := coordinate(-33.868819912345, 151.20929987654, 13); got != "-33.8688199,151.2092999,13z" {
		t.Fatalf("coordinate = %q", got)
	}
}

const mapsFixture = `{"items":[
 {"type":"maps_paid_item","cid":"5","title":"Ad"},
 {"type":"maps_search","rank_group":1,"cid":"11","place_id":"p11","title":"Bright Smiles","category":"Dentist","additional_categories":["Cosmetic dentist","dentist"],"rating":{"value":4.8,"votes_count":640},"latitude":1.5,"longitude":2.5},
 {"type":"maps_search","rank_group":2,"cid":"22","title":"Zahnarzt Müller","category":"Zahnarzt","rating":null},
 {"type":"maps_search","rank_group":3,"cid":"not-a-cid","title":"Bad"}
]}`

func TestParseMapsKeepsOrganicListingsWithAValidCID(t *testing.T) {
	places, err := parseMaps(json.RawMessage(mapsFixture))
	if err != nil {
		t.Fatal(err)
	}
	if len(places) != 2 || places[0].CID != "11" || places[1].Title != "Zahnarzt Müller" {
		t.Fatalf("places = %+v", places)
	}
	if got := places[0].Categories; len(got) != 2 || got[1] != "Cosmetic dentist" {
		t.Fatalf("categories = %v, want duplicates removed", got)
	}
	if *places[0].Rating != 4.8 || places[0].Reviews != 640 || places[1].Rating != nil {
		t.Fatalf("ratings = %+v", places)
	}
	if rankOf(places, "22") != 2 || rankOf(places, "404") != 0 {
		t.Fatal("rankOf")
	}
}

func TestParseProfileAndPostsAndReviews(t *testing.T) {
	profile, err := parseProfile(json.RawMessage(`{"items":[{"cid":"1","title":"T","description":" ","category":"Dentist","phone":"+48 1",
		"url":"https://x.example","is_claimed":true,"total_photos":18,"rating":{"value":4.6,"votes_count":212},
		"work_time":{"work_hours":{"timetable":{}}},"attributes":{"available_attributes":{"accessibility":["has_wheelchair_accessible_entrance"],"planning":["accepts_reservations"]}}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if profile.Description || !profile.Phone || !profile.Website || !profile.Hours || !profile.Claimed || profile.Photos != 18 || len(profile.Attributes) != 2 || profile.Reviews != 212 {
		t.Fatalf("profile = %+v", profile)
	}
	if _, err := parseProfile(json.RawMessage(`{"items":[]}`)); !errors.Is(err, errNoResult) {
		t.Fatalf("empty profile err = %v", err)
	}
	posts, err := parsePosts(json.RawMessage(`{"items":[{"post_text":"New","timestamp":"2026-11-01 10:00:00 +00:00"},{"post_text":"bad date","timestamp":"yesterday"}]}`))
	if err != nil || len(posts) != 1 || posts[0].At != time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC) {
		t.Fatalf("posts = %+v err = %v", posts, err)
	}
	reviews, err := parseReviews(json.RawMessage(`{"reviews_count":212,"rating":{"value":4.6},"items":[
		{"review_id":"a","review_text":"Świetnie","profile_name":"Ola","timestamp":"2026-11-05 09:00:00 +00:00","rating":{"value":5},"owner_answer":"Dzięki","owner_timestamp":"2026-11-06 09:00:00 +00:00"},
		{"review_id":"b","review_text":"Waited 50 minutes","profile_name":"Rohan","timestamp":"2026-10-29 09:00:00 +00:00","rating":{"value":2},"owner_answer":null},
		{"review_id":"","rating":{"value":1},"timestamp":"2026-10-29 09:00:00 +00:00"}]}`))
	if err != nil || reviews.Total != 212 || len(reviews.Reviews) != 2 || !reviews.Reviews[0].Answered || reviews.Reviews[1].Answered {
		t.Fatalf("reviews = %+v err = %v", reviews, err)
	}
}

func TestTruncateBytesNeverSplitsACharacter(t *testing.T) {
	for _, tc := range []struct {
		in    string
		limit int
		want  string
	}{
		{"hello", 10, "hello"},
		{"हिन्दी", 7, "हि"},
		{"żółw", 3, "ż"},
		{"", 3, ""},
	} {
		if got := truncateBytes(tc.in, tc.limit); got != tc.want || len(got) > tc.limit {
			t.Errorf("truncateBytes(%q, %d) = %q, want %q", tc.in, tc.limit, got, tc.want)
		}
	}
}
