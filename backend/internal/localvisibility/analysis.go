package localvisibility

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
)

// Every rule below compares the customer's profile with the businesses that
// rank above or around it for the same search, at the same place, from the
// same DataForSEO data. Nothing is a fixed industry benchmark: a gap is only
// reported when the competitors' own numbers show it.

// Business is the project's Google Business Profile.
type Business struct {
	CID          string    `json:"cid"`
	PlaceID      string    `json:"placeId"`
	Title        string    `json:"title"`
	Address      string    `json:"address"`
	Category     string    `json:"category"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	SearchTerm   string    `json:"searchTerm"`
	LanguageCode string    `json:"languageCode"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// Contender is one business in the profile check.
type Contender struct {
	Place       Place      `json:"place"`
	Profile     *Profile   `json:"profile"`
	PostsLast30 *int       `json:"postsLast30"`
	LastPostAt  *time.Time `json:"lastPostAt"`
}

// Fix is one thing to change on the profile, with the numbers behind it.
type Fix struct {
	// Status is missing (the others have it, you do not), behind (you have
	// less) or fine.
	Status string `json:"status"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	You    string `json:"you"`
	Others string `json:"others"`
}

// CheckResult is the profile vs top 3 check.
type CheckResult struct {
	SearchTerm string      `json:"searchTerm"`
	YourRank   int         `json:"yourRank"`
	You        Contender   `json:"you"`
	Rivals     []Contender `json:"rivals"`
	Fixes      []Fix       `json:"fixes"`
	Summary    string      `json:"summary"`
	Missing    []string    `json:"missing"`
}

const maxRivals = 3

// pickRivals are the top businesses of the Maps search other than the
// customer's own.
func pickRivals(places []Place, ownCID string) []Place {
	out := []Place{}
	for _, p := range places {
		if p.CID != ownCID && len(out) < maxRivals {
			out = append(out, p)
		}
	}
	return out
}

func rankOf(places []Place, cid string) int {
	for _, p := range places {
		if p.CID == cid {
			return p.Rank
		}
	}
	return 0
}

func postStats(posts []Post, now time.Time) (int, *time.Time) {
	count := 0
	var last *time.Time
	for _, p := range posts {
		if now.Sub(p.At) <= 30*24*time.Hour {
			count++
		}
		if last == nil || p.At.After(*last) {
			at := p.At
			last = &at
		}
	}
	return count, last
}

// buildFixes turns the side-by-side numbers into fixes, the biggest gaps
// first. A comparison is skipped when the competitors' data is missing.
func buildFixes(you Contender, rivals []Contender, now time.Time) []Fix {
	var profiles []Profile
	for _, r := range rivals {
		if r.Profile != nil {
			profiles = append(profiles, *r.Profile)
		}
	}
	if you.Profile == nil {
		return []Fix{}
	}
	me := *you.Profile
	n := len(profiles)
	fixes := []Fix{}
	var fine []string
	if !me.Claimed {
		fixes = append(fixes, Fix{Status: "missing", Title: "Claim your profile",
			Detail: "Google shows this profile as not claimed by the owner, so you cannot reply to reviews, post updates or fix details.",
			You:    "not claimed", Others: fmt.Sprintf("%d of %d claimed", count(profiles, func(p Profile) bool { return p.Claimed }), n)})
	}
	if n == 0 {
		return fixes
	}
	// A category, attribute or field counts when most competitors have it.
	most := max(1, (n+1)/2)
	for _, category := range commonValues(profiles, func(p Profile) []string { return p.Categories }, most) {
		if !containsFold(me.Categories, category.value) {
			fixes = append(fixes, Fix{Status: "missing", Title: fmt.Sprintf("Add the category “%s”", category.value),
				Detail: fmt.Sprintf("%d of %d competitors use it. You list: %s.", category.count, n, strings.Join(me.Categories, ", ")),
				You:    "not listed", Others: fmt.Sprintf("%d of %d", category.count, n)})
		}
	}
	for _, field := range []struct {
		name string
		has  func(Profile) bool
	}{
		{"business description", func(p Profile) bool { return p.Description }},
		{"opening hours", func(p Profile) bool { return p.Hours }},
		{"phone number", func(p Profile) bool { return p.Phone }},
		{"website link", func(p Profile) bool { return p.Website }},
	} {
		others := count(profiles, field.has)
		switch {
		case field.has(me):
			fine = append(fine, field.name)
		case others >= most:
			fixes = append(fixes, Fix{Status: "missing", Title: "Add your " + field.name,
				Detail: fmt.Sprintf("%d of %d competitors show one; your profile does not.", others, n),
				You:    "none", Others: fmt.Sprintf("%d of %d", others, n)})
		}
	}
	reviews := median(mapSlice(profiles, func(p Profile) float64 { return float64(p.Reviews) }))
	if reviews > 0 && float64(me.Reviews) < reviews/2 {
		fixes = append(fixes, Fix{Status: "behind", Title: "Ask more customers for a review",
			Detail: fmt.Sprintf("You have %d reviews; the competitors' median is %.0f. Review count is one of the signals Maps ranks on.", me.Reviews, reviews),
			You:    fmt.Sprint(me.Reviews), Others: fmt.Sprintf("median %.0f", reviews)})
	}
	if me.Rating != nil {
		ratings := []float64{}
		for _, p := range profiles {
			if p.Rating != nil {
				ratings = append(ratings, *p.Rating)
			}
		}
		if len(ratings) > 0 && *me.Rating < slices.Min(ratings)-0.1 {
			fixes = append(fixes, Fix{Status: "behind", Title: "Your rating is the lowest of the group",
				Detail: fmt.Sprintf("You are at %.1f; every competitor shown is at %.1f or higher. The Reviews tab shows what lowers it.", *me.Rating, slices.Min(ratings)),
				You:    fmt.Sprintf("%.1f", *me.Rating), Others: fmt.Sprintf("%.1f–%.1f", slices.Min(ratings), slices.Max(ratings))})
		}
	}
	photos := median(mapSlice(profiles, func(p Profile) float64 { return float64(p.Photos) }))
	if photos > 0 && float64(me.Photos) < photos/2 {
		fixes = append(fixes, Fix{Status: "behind", Title: "Add photos",
			Detail: fmt.Sprintf("You have %d photos; the competitors' median is %.0f.", me.Photos, photos),
			You:    fmt.Sprint(me.Photos), Others: fmt.Sprintf("median %.0f", photos)})
	}
	var rivalPosts []float64
	for _, r := range rivals {
		if r.PostsLast30 != nil {
			rivalPosts = append(rivalPosts, float64(*r.PostsLast30))
		}
	}
	if you.PostsLast30 != nil && len(rivalPosts) > 0 {
		avg := mean(rivalPosts)
		if avg >= 1 && float64(*you.PostsLast30) < avg/2 {
			last := "You have no posts on the profile."
			if you.LastPostAt != nil {
				last = fmt.Sprintf("Your last post was %d days ago.", int(now.Sub(*you.LastPostAt).Hours()/24))
			}
			fixes = append(fixes, Fix{Status: "behind", Title: "Post updates more often",
				Detail: fmt.Sprintf("%s Competitors posted %.1f times on average in the last 30 days.", last, avg),
				You:    fmt.Sprintf("%d / 30 days", *you.PostsLast30), Others: fmt.Sprintf("avg %.1f / 30 days", avg)})
		}
	}
	var missingAttributes []string
	for _, attribute := range commonValues(profiles, func(p Profile) []string { return p.Attributes }, max(2, most)) {
		if !containsFold(me.Attributes, attribute.value) && len(missingAttributes) < 5 {
			missingAttributes = append(missingAttributes, "“"+attribute.value+"”")
		}
	}
	if len(missingAttributes) > 0 {
		fixes = append(fixes, Fix{Status: "behind", Title: "Turn on the attributes most competitors show",
			Detail: strings.Join(missingAttributes, ", ") + ". Google answers questions about a business from these fields.",
			You:    "off", Others: fmt.Sprintf("%d+ of %d have each", max(2, most), n)})
	}
	if len(fine) > 0 {
		fixes = append(fixes, Fix{Status: "fine", Title: "Already complete: " + strings.Join(fine, ", "),
			Detail: "Nothing to do here.", You: "ok", Others: ""})
	}
	return fixes
}

func checkSummary(term string, rank int, fixes []Fix) string {
	where := fmt.Sprintf("You are not in the top %d on Google Maps for “%s” at your address.", mapsDepth, term)
	if rank > 0 {
		where = fmt.Sprintf("You rank #%d on Google Maps for “%s” at your address.", rank, term)
	}
	gaps := 0
	for _, f := range fixes {
		if f.Status != "fine" {
			gaps++
		}
	}
	if gaps == 0 {
		return where + " Your profile has everything the top 3 have."
	}
	return fmt.Sprintf("%s %d of the fixes below come from what the top 3 have and you do not.", where, gaps)
}

type valueCount struct {
	value string
	count int
}

// commonValues are the values at least atLeast profiles have, most common first.
func commonValues(profiles []Profile, values func(Profile) []string, atLeast int) []valueCount {
	counts := map[string]*valueCount{}
	var order []string
	for _, p := range profiles {
		seen := map[string]bool{}
		for _, v := range values(p) {
			key := strings.ToLower(v)
			if seen[key] {
				continue
			}
			seen[key] = true
			if counts[key] == nil {
				counts[key] = &valueCount{value: v}
				order = append(order, key)
			}
			counts[key].count++
		}
	}
	out := []valueCount{}
	for _, key := range order {
		if counts[key].count >= atLeast {
			out = append(out, *counts[key])
		}
	}
	slices.SortStableFunc(out, func(a, b valueCount) int { return b.count - a.count })
	return out
}

// GridPoint is one Maps search of the rank grid.
type GridPoint struct {
	Row       int     `json:"row"`
	Col       int     `json:"col"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	// Rank is the customer's Maps position there; 0 is not in the top 20.
	Rank int `json:"rank"`
	// Top are the first three businesses at that point.
	Top []string `json:"top"`
	// Missing means the search did not finish, so the point has no rank.
	Missing bool `json:"missing"`
}

// GridResult is a map rank grid scan.
type GridResult struct {
	SearchTerm string      `json:"searchTerm"`
	Size       int         `json:"size"`
	SpacingKm  float64     `json:"spacingKm"`
	Points     []GridPoint `json:"points"`
	Summary    string      `json:"summary"`
}

// gridPoints lays out size x size points spacingKm apart, centred on the
// business. Row 0 is north.
func gridPoints(lat, lng float64, size int, spacingKm float64) []GridPoint {
	half := size / 2
	points := make([]GridPoint, 0, size*size)
	for row := range size {
		for col := range size {
			north := float64(half-row) * spacingKm
			east := float64(col-half) * spacingKm
			points = append(points, GridPoint{
				Row: row, Col: col,
				Latitude:  round7(lat + north/111.32),
				Longitude: round7(lng + east/(111.32*math.Cos(lat*math.Pi/180))),
			})
		}
	}
	return points
}

// gridZoom shows roughly the area around one point: closer points need a
// closer map, or every point would search the same area.
func gridZoom(spacingKm float64) int {
	switch {
	case spacingKm <= 1:
		return 15
	case spacingKm <= 2:
		return 14
	}
	return 13
}

func gridSummary(points []GridPoint) string {
	var top3, outside, measured int
	leaders := map[string]int{}
	for _, p := range points {
		if p.Missing {
			continue
		}
		measured++
		switch {
		case p.Rank >= 1 && p.Rank <= 3:
			top3++
		case p.Rank == 0:
			outside++
		}
		if (p.Rank == 0 || p.Rank > 3) && len(p.Top) > 0 {
			leaders[p.Top[0]]++
		}
	}
	if measured == 0 {
		return "No point of the grid finished. Nothing was charged for the points that did not finish."
	}
	text := fmt.Sprintf("You are in the top 3 at %d of %d points and not in the top %d at %d.", top3, measured, mapsDepth, outside)
	leader, most := "", 0
	for name, n := range leaders {
		if n > most || n == most && name < leader {
			leader, most = name, n
		}
	}
	if most > 0 {
		text += fmt.Sprintf(" Where you are not in the top 3, %s is #1 most often (%d points).", leader, most)
	}
	return text
}

// Theme is a topic customers mention, counted by review.
type Theme struct {
	Name     string `json:"name"`
	Positive int    `json:"positive"`
	Negative int    `json:"negative"`
}

// RivalReviews is a competitor's recent reviews in short.
type RivalReviews struct {
	CID       string  `json:"cid"`
	Title     string  `json:"title"`
	Fetched   int     `json:"fetched"`
	ReplyRate *int    `json:"replyRate"`
	Themes    []Theme `json:"themes"`
}

// ReviewsResult is the reviews analysis.
type ReviewsResult struct {
	Total        int            `json:"total"`
	Rating       *float64       `json:"rating"`
	Fetched      int            `json:"fetched"`
	ReplyRate    *int           `json:"replyRate"`
	Distribution [5]int         `json:"distribution"`
	Unanswered   []Review       `json:"unanswered"`
	Themes       []Theme        `json:"themes"`
	Rivals       []RivalReviews `json:"rivals"`
	Summary      string         `json:"summary"`
	Missing      []string       `json:"missing"`
}

const maxUnanswered = 25

func replyRate(reviews []Review) *int {
	if len(reviews) == 0 {
		return nil
	}
	answered := count(reviews, func(r Review) bool { return r.Answered })
	rate := int(math.Round(float64(answered) * 100 / float64(len(reviews))))
	return &rate
}

func reviewStats(result *ReviewsResult, own reviewsResult) {
	result.Total, result.Rating, result.Fetched = own.Total, own.Rating, len(own.Reviews)
	result.ReplyRate = replyRate(own.Reviews)
	result.Unanswered = []Review{}
	for _, r := range own.Reviews {
		if r.Rating >= 1 && r.Rating <= 5 {
			result.Distribution[r.Rating-1]++
		}
		if !r.Answered && len(result.Unanswered) < maxUnanswered {
			result.Unanswered = append(result.Unanswered, r)
		}
	}
}

func reviewsSummary(result ReviewsResult) string {
	if result.Fetched == 0 {
		return "Google returned no reviews for your business."
	}
	text := fmt.Sprintf("You replied to %d%% of your last %d reviews.", *result.ReplyRate, result.Fetched)
	var rates []float64
	for _, r := range result.Rivals {
		if r.ReplyRate != nil {
			rates = append(rates, float64(*r.ReplyRate))
		}
	}
	if len(rates) > 0 {
		text += fmt.Sprintf(" The top 3 reply to %.0f%% on average.", mean(rates))
	}
	var worst *Theme
	for i, t := range result.Themes {
		if t.Negative > 0 && (worst == nil || t.Negative > worst.Negative) {
			worst = &result.Themes[i]
		}
	}
	if worst != nil {
		text += fmt.Sprintf(" The most common complaint is “%s” (%d reviews).", worst.Name, worst.Negative)
	}
	return text
}

// Snapshot is a competitor's profile on one week's watch.
type Snapshot struct {
	CID          string     `json:"cid"`
	Title        string     `json:"title"`
	Rating       *float64   `json:"rating"`
	Reviews      int        `json:"reviews"`
	Photos       int        `json:"photos"`
	Categories   []string   `json:"categories"`
	HoursKey     string     `json:"hoursKey"`
	PostsLast7   int        `json:"postsLast7"`
	LatestPostAt *time.Time `json:"latestPostAt"`
	Missing      bool       `json:"missing"`
}

// Change is what a competitor changed since the previous watch.
type Change struct {
	CID          string   `json:"cid"`
	Title        string   `json:"title"`
	Rating       *float64 `json:"rating"`
	RatingDelta  *float64 `json:"ratingDelta"`
	NewReviews   *int     `json:"newReviews"`
	NewPosts     int      `json:"newPosts"`
	PhotosDelta  *int     `json:"photosDelta"`
	Added        []string `json:"addedCategories"`
	Removed      []string `json:"removedCategories"`
	HoursChanged bool     `json:"hoursChanged"`
}

// WatchResult is one week's competitor watch.
type WatchResult struct {
	Snapshots []Snapshot `json:"snapshots"`
	Changes   []Change   `json:"changes"`
	// Since is when the previous watch ran; nil on the first one.
	Since   *time.Time `json:"since"`
	Summary string     `json:"summary"`
}

func compareWatch(current []Snapshot, previous []Snapshot) []Change {
	before := map[string]Snapshot{}
	for _, s := range previous {
		if !s.Missing {
			before[s.CID] = s
		}
	}
	changes := []Change{}
	for _, now := range current {
		if now.Missing {
			continue
		}
		c := Change{CID: now.CID, Title: now.Title, Rating: now.Rating, NewPosts: now.PostsLast7, Added: []string{}, Removed: []string{}}
		if old, ok := before[now.CID]; ok {
			newReviews := now.Reviews - old.Reviews
			c.NewReviews = &newReviews
			photos := now.Photos - old.Photos
			c.PhotosDelta = &photos
			if now.Rating != nil && old.Rating != nil {
				delta := math.Round((*now.Rating-*old.Rating)*10) / 10
				c.RatingDelta = &delta
			}
			for _, cat := range now.Categories {
				if !containsFold(old.Categories, cat) {
					c.Added = append(c.Added, cat)
				}
			}
			for _, cat := range old.Categories {
				if !containsFold(now.Categories, cat) {
					c.Removed = append(c.Removed, cat)
				}
			}
			c.HoursChanged = old.HoursKey != "" && now.HoursKey != "" && old.HoursKey != now.HoursKey
		}
		changes = append(changes, c)
	}
	return changes
}

func watchSummary(changes []Change, first bool) string {
	if len(changes) == 0 {
		return "No competitor profile could be read this week. Nothing was charged for the ones that failed."
	}
	if first {
		return fmt.Sprintf("First snapshot of %d competitors. Changes show from next week's watch.", len(changes))
	}
	var busiest *Change
	for i, c := range changes {
		if c.NewReviews != nil && (busiest == nil || *c.NewReviews > *busiest.NewReviews) {
			busiest = &changes[i]
		}
	}
	parts := []string{}
	if busiest != nil && *busiest.NewReviews > 0 {
		parts = append(parts, fmt.Sprintf("%s got the most new reviews (%d)", busiest.Title, *busiest.NewReviews))
	}
	for _, c := range changes {
		if len(c.Added) > 0 {
			parts = append(parts, fmt.Sprintf("%s added the category “%s”", c.Title, strings.Join(c.Added, "”, “")))
		}
	}
	if len(parts) == 0 {
		return "No competitor changed its rating, categories or hours this week."
	}
	return strings.Join(parts, "; ") + "."
}

func count[T any](items []T, pred func(T) bool) int {
	n := 0
	for _, item := range items {
		if pred(item) {
			n++
		}
	}
	return n
}

func mapSlice[T any](items []T, f func(T) float64) []float64 {
	out := make([]float64, len(items))
	for i, item := range items {
		out[i] = f(item)
	}
	return out
}

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total / float64(len(values))
}

func hashText(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:8])
}
