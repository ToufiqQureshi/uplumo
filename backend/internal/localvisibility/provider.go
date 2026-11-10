package localvisibility

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

// DataForSEO endpoints this feature uses. Results of queued tasks are read
// with task_get, which is free: the task was paid for when it was posted.
const (
	mapsLivePath    = "/v3/serp/google/maps/live/advanced"
	mapsPostPath    = "/v3/serp/google/maps/task_post"
	mapsGetPath     = "/v3/serp/google/maps/task_get/advanced/"
	infoPostPath    = "/v3/business_data/google/my_business_info/task_post"
	infoGetPath     = "/v3/business_data/google/my_business_info/task_get/"
	updatesPostPath = "/v3/business_data/google/my_business_updates/task_post"
	updatesGetPath  = "/v3/business_data/google/my_business_updates/task_get/"
	reviewsPostPath = "/v3/business_data/google/reviews/task_post"
	reviewsGetPath  = "/v3/business_data/google/reviews/task_get/"
)

// Endpoints a stored task can be read from, by the name kept in go_local_tasks.
const (
	endpointMaps    = "maps"
	endpointInfo    = "info"
	endpointUpdates = "updates"
	endpointReviews = "reviews"
)

var getPaths = map[string]string{
	endpointMaps: mapsGetPath, endpointInfo: infoGetPath, endpointUpdates: updatesGetPath, endpointReviews: reviewsGetPath,
}

var postPaths = map[string]string{
	endpointMaps: mapsPostPath, endpointInfo: infoPostPath, endpointUpdates: updatesPostPath, endpointReviews: reviewsPostPath,
}

// Request sizes. Every paid task uses priority 2: about a minute in the queue
// instead of five, at twice the queued price and still below the live price.
const (
	priority          = 2
	mapsDepth         = 20 // the top 20 Maps results; "not in top 20" below that
	findDepth         = 20
	postsDepth        = 10
	ownReviewsDepth   = 100
	rivalReviewsDepth = 50
)

var cidPattern = regexp.MustCompile(`^[0-9]{1,40}$`)

// Client is the DataForSEO transport; *dataforseo.Client implements it.
type Client interface {
	Do(ctx context.Context, organizationID string, feature spend.Feature, method, path string, body []byte, retrySafe bool) (json.RawMessage, error)
	DoUnmetered(ctx context.Context, method, path string, body []byte, retrySafe bool) (json.RawMessage, error)
}

// task is one queued request: its key inside the run, its endpoint and body.
type task struct {
	Key      string
	Endpoint string
	Body     map[string]any
}

func (t task) json() []byte {
	raw, _ := json.Marshal([]map[string]any{t.Body})
	return raw
}

// costUSD is the most the task can cost, as the spend guard will hold it.
func (t task) costUSD() float64 {
	return dataforseo.EstimateUSD(postPaths[t.Endpoint], t.json())
}

func tasksCostUSD(tasks []task) float64 {
	total := 0.0
	for _, t := range tasks {
		total += t.costUSD()
	}
	return math.Round(total*1e6) / 1e6
}

// coordinate is DataForSEO's "latitude,longitude,zoom" Maps location.
func coordinate(lat, lng float64, zoom int) string {
	return strconv.FormatFloat(round7(lat), 'f', -1, 64) + "," + strconv.FormatFloat(round7(lng), 'f', -1, 64) + "," + strconv.Itoa(zoom) + "z"
}

func round7(v float64) float64 { return math.Round(v*1e7) / 1e7 }

func mapsTask(key, term, language string, lat, lng float64, zoom int) task {
	return task{Key: key, Endpoint: endpointMaps, Body: map[string]any{
		"keyword": term, "location_coordinate": coordinate(lat, lng, zoom), "language_code": language,
		"depth": mapsDepth, "priority": priority,
	}}
}

// The profile and posts tasks name the business by its Maps CID; the
// coordinate only satisfies the required location.
func infoTask(key, cid, language string, lat, lng float64) task {
	return task{Key: key, Endpoint: endpointInfo, Body: map[string]any{
		"keyword": "cid:" + cid, "location_coordinate": coordinate(lat, lng, 15), "language_code": language, "priority": priority,
	}}
}

func updatesTask(key, cid, language string, lat, lng float64) task {
	return task{Key: key, Endpoint: endpointUpdates, Body: map[string]any{
		"keyword": "cid:" + cid, "location_coordinate": coordinate(lat, lng, 15), "language_code": language,
		"depth": postsDepth, "priority": priority,
	}}
}

func reviewsTask(key, cid, language string, lat, lng float64, depth int) task {
	return task{Key: key, Endpoint: endpointReviews, Body: map[string]any{
		"cid": cid, "location_coordinate": coordinate(lat, lng, 15), "language_code": language,
		"depth": depth, "sort_by": "newest", "priority": priority,
	}}
}

// errNotReady means a queued task is still running at the provider.
var errNotReady = errors.New("task not ready")

// errNoResult means the provider finished the task and found nothing, such
// as a business without posts.
var errNoResult = errors.New("task has no result")

type envelope struct {
	StatusCode    int    `json:"status_code"`
	StatusMessage string `json:"status_message"`
	Tasks         []struct {
		ID            string            `json:"id"`
		StatusCode    int               `json:"status_code"`
		StatusMessage string            `json:"status_message"`
		Result        []json.RawMessage `json:"result"`
	} `json:"tasks"`
}

// post sends one task and returns its provider id. A refusal by the spend
// guard is returned as is, so the caller can tell the customer.
func post(ctx context.Context, c Client, org string, t task) (string, error) {
	raw, err := c.Do(ctx, org, spend.LocalVisibility, http.MethodPost, postPaths[t.Endpoint], t.json(), false)
	if err != nil {
		return "", err
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return "", fmt.Errorf("decode %s task_post: %w", t.Endpoint, err)
	}
	if len(env.Tasks) == 0 || env.Tasks[0].StatusCode != 20100 || env.Tasks[0].ID == "" {
		message := env.StatusMessage
		if len(env.Tasks) > 0 {
			message = env.Tasks[0].StatusMessage
		}
		return "", fmt.Errorf("DataForSEO did not accept the %s task: %s", t.Endpoint, message)
	}
	return env.Tasks[0].ID, nil
}

// collect reads a posted task. It returns errNotReady while the task runs,
// errNoResult when it finished empty, and the first result otherwise.
func collect(ctx context.Context, c Client, endpoint, id string) (json.RawMessage, error) {
	path, ok := getPaths[endpoint]
	if !ok {
		return nil, fmt.Errorf("unknown endpoint %q", endpoint)
	}
	raw, err := c.DoUnmetered(ctx, http.MethodGet, path+url.PathEscape(id), nil, true)
	if err != nil {
		return nil, err
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("decode %s task_get: %w", endpoint, err)
	}
	if len(env.Tasks) == 0 {
		return nil, fmt.Errorf("DataForSEO returned no %s task", endpoint)
	}
	t := env.Tasks[0]
	switch {
	case t.StatusCode == 20100 || t.StatusCode == 40601 || t.StatusCode == 40602:
		return nil, errNotReady
	case t.StatusCode == 40102:
		return nil, errNoResult
	case t.StatusCode != 20000:
		return nil, taskFailure{fmt.Errorf("DataForSEO %s task failed: %d %s", endpoint, t.StatusCode, t.StatusMessage)}
	case len(t.Result) == 0 || string(t.Result[0]) == "null":
		return nil, errNoResult
	}
	return t.Result[0], nil
}

// findBusinesses is the one live call: a Maps search for the customer's own
// business by name, so they can pick it from the list.
func findBusinesses(ctx context.Context, c Client, org, query string, locationCode int, language string) ([]Place, error) {
	body, _ := json.Marshal([]map[string]any{{
		"keyword": query, "location_code": locationCode, "language_code": language, "depth": findDepth,
	}})
	raw, err := c.Do(ctx, org, spend.LocalVisibility, http.MethodPost, mapsLivePath, body, false)
	if err != nil {
		return nil, err
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("decode Maps search: %w", err)
	}
	if len(env.Tasks) == 0 {
		return nil, errors.New("DataForSEO returned no Maps task")
	}
	if env.Tasks[0].StatusCode == 40102 || len(env.Tasks[0].Result) == 0 {
		return []Place{}, nil
	}
	if env.Tasks[0].StatusCode != 20000 {
		return nil, fmt.Errorf("DataForSEO Maps search failed: %s", env.Tasks[0].StatusMessage)
	}
	return parseMaps(env.Tasks[0].Result[0])
}

// FindCostUSD is the most one business search can cost.
func FindCostUSD() float64 {
	body, _ := json.Marshal([]map[string]any{{"depth": findDepth}})
	return dataforseo.EstimateUSD(mapsLivePath, body)
}

// Place is one Google Maps listing.
type Place struct {
	Rank       int      `json:"rank"`
	CID        string   `json:"cid"`
	PlaceID    string   `json:"placeId"`
	Title      string   `json:"title"`
	Address    string   `json:"address"`
	Category   string   `json:"category"`
	Categories []string `json:"categories"`
	Rating     *float64 `json:"rating"`
	Reviews    int      `json:"reviews"`
	Latitude   float64  `json:"latitude"`
	Longitude  float64  `json:"longitude"`
	Domain     string   `json:"domain"`
}

type rating struct {
	Value      *float64 `json:"value"`
	VotesCount *int     `json:"votes_count"`
}

func parseMaps(raw json.RawMessage) ([]Place, error) {
	var result struct {
		Items []struct {
			Type                 string   `json:"type"`
			RankGroup            int      `json:"rank_group"`
			CID                  string   `json:"cid"`
			PlaceID              string   `json:"place_id"`
			Title                string   `json:"title"`
			Address              string   `json:"address"`
			Category             string   `json:"category"`
			AdditionalCategories []string `json:"additional_categories"`
			Rating               *rating  `json:"rating"`
			Latitude             float64  `json:"latitude"`
			Longitude            float64  `json:"longitude"`
			Domain               string   `json:"domain"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("decode Maps results: %w", err)
	}
	places := []Place{}
	for _, item := range result.Items {
		if item.Type != "maps_search" || !cidPattern.MatchString(item.CID) {
			continue
		}
		p := Place{
			Rank: item.RankGroup, CID: item.CID, PlaceID: item.PlaceID, Title: item.Title, Address: item.Address,
			Category: item.Category, Categories: categories(item.Category, item.AdditionalCategories),
			Latitude: item.Latitude, Longitude: item.Longitude, Domain: item.Domain,
		}
		if item.Rating != nil {
			p.Rating = item.Rating.Value
			if item.Rating.VotesCount != nil {
				p.Reviews = *item.Rating.VotesCount
			}
		}
		places = append(places, p)
	}
	return places, nil
}

func categories(main string, extra []string) []string {
	out := []string{}
	for _, c := range append([]string{main}, extra...) {
		if c = strings.TrimSpace(c); c != "" && !containsFold(out, c) {
			out = append(out, c)
		}
	}
	return out
}

func containsFold(list []string, value string) bool {
	for _, item := range list {
		if strings.EqualFold(item, value) {
			return true
		}
	}
	return false
}

// Profile is the public Google Business Profile of one business.
type Profile struct {
	CID         string   `json:"cid"`
	Title       string   `json:"title"`
	Categories  []string `json:"categories"`
	Description bool     `json:"hasDescription"`
	Phone       bool     `json:"hasPhone"`
	Website     bool     `json:"hasWebsite"`
	Hours       bool     `json:"hasHours"`
	HoursKey    string   `json:"hoursKey"`
	Claimed     bool     `json:"claimed"`
	Photos      int      `json:"photos"`
	Rating      *float64 `json:"rating"`
	Reviews     int      `json:"reviews"`
	Attributes  []string `json:"attributes"`
}

func parseProfile(raw json.RawMessage) (Profile, error) {
	var result struct {
		Items []struct {
			Type                 string          `json:"type"`
			CID                  string          `json:"cid"`
			Title                string          `json:"title"`
			Description          string          `json:"description"`
			Category             string          `json:"category"`
			AdditionalCategories []string        `json:"additional_categories"`
			Phone                string          `json:"phone"`
			URL                  string          `json:"url"`
			IsClaimed            bool            `json:"is_claimed"`
			TotalPhotos          int             `json:"total_photos"`
			Rating               *rating         `json:"rating"`
			WorkTime             json.RawMessage `json:"work_time"`
			Attributes           *struct {
				Available map[string][]string `json:"available_attributes"`
			} `json:"attributes"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return Profile{}, fmt.Errorf("decode business profile: %w", err)
	}
	if len(result.Items) == 0 {
		return Profile{}, errNoResult
	}
	item := result.Items[0]
	p := Profile{
		CID: item.CID, Title: item.Title, Categories: categories(item.Category, item.AdditionalCategories),
		Description: strings.TrimSpace(item.Description) != "", Phone: strings.TrimSpace(item.Phone) != "",
		Website: strings.TrimSpace(item.URL) != "", Claimed: item.IsClaimed, Photos: item.TotalPhotos, Attributes: []string{},
	}
	if len(item.WorkTime) > 0 && string(item.WorkTime) != "null" {
		p.Hours = true
		p.HoursKey = hashText(string(item.WorkTime))
	}
	if item.Rating != nil {
		p.Rating = item.Rating.Value
		if item.Rating.VotesCount != nil {
			p.Reviews = *item.Rating.VotesCount
		}
	}
	if item.Attributes != nil {
		for _, values := range item.Attributes.Available {
			for _, v := range values {
				if v = strings.TrimSpace(v); v != "" && !containsFold(p.Attributes, v) {
					p.Attributes = append(p.Attributes, v)
				}
			}
		}
	}
	return p, nil
}

// Post is one Google Business Profile update.
type Post struct {
	Text string    `json:"text"`
	At   time.Time `json:"at"`
}

func parsePosts(raw json.RawMessage) ([]Post, error) {
	var result struct {
		Items []struct {
			PostText  string `json:"post_text"`
			Timestamp string `json:"timestamp"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("decode business posts: %w", err)
	}
	posts := []Post{}
	for _, item := range result.Items {
		at, ok := parseTime(item.Timestamp)
		if !ok {
			continue
		}
		posts = append(posts, Post{Text: truncate(item.PostText, 280), At: at})
	}
	return posts, nil
}

// Review is one Google review.
type Review struct {
	ID         string    `json:"id"`
	Author     string    `json:"author"`
	Rating     int       `json:"rating"`
	Text       string    `json:"text"`
	At         time.Time `json:"at"`
	Answered   bool      `json:"answered"`
	AnsweredAt time.Time `json:"answeredAt,omitzero"`
	URL        string    `json:"url"`
	ReplyDraft string    `json:"replyDraft,omitempty"`
}

type reviewsResult struct {
	Rating  *float64
	Total   int
	Reviews []Review
}

func parseReviews(raw json.RawMessage) (reviewsResult, error) {
	var result struct {
		ReviewsCount int     `json:"reviews_count"`
		Rating       *rating `json:"rating"`
		Items        []struct {
			ReviewID       string  `json:"review_id"`
			ReviewText     string  `json:"review_text"`
			ProfileName    string  `json:"profile_name"`
			Timestamp      string  `json:"timestamp"`
			OwnerAnswer    *string `json:"owner_answer"`
			OwnerTimestamp string  `json:"owner_timestamp"`
			ReviewURL      string  `json:"review_url"`
			Rating         *rating `json:"rating"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return reviewsResult{}, fmt.Errorf("decode reviews: %w", err)
	}
	out := reviewsResult{Total: result.ReviewsCount, Reviews: []Review{}}
	if result.Rating != nil {
		out.Rating = result.Rating.Value
	}
	for _, item := range result.Items {
		at, ok := parseTime(item.Timestamp)
		if !ok || item.ReviewID == "" || item.Rating == nil || item.Rating.Value == nil {
			continue
		}
		r := Review{
			ID: item.ReviewID, Author: truncate(item.ProfileName, 80), Rating: int(math.Round(*item.Rating.Value)),
			Text: truncate(item.ReviewText, 1500), At: at, URL: item.ReviewURL,
		}
		if item.OwnerAnswer != nil && strings.TrimSpace(*item.OwnerAnswer) != "" {
			r.Answered = true
			r.AnsweredAt, _ = parseTime(item.OwnerTimestamp)
		}
		out.Reviews = append(out.Reviews, r)
	}
	return out, nil
}

// parseTime reads DataForSEO's "yyyy-mm-dd hh:mm:ss +00:00".
func parseTime(value string) (time.Time, bool) {
	at, err := time.Parse("2006-01-02 15:04:05 -07:00", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, false
	}
	return at.UTC(), true
}

func truncate(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len([]rune(value)) <= limit {
		return value
	}
	return string([]rune(value)[:limit]) + "…"
}
