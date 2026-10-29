package aisearch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toufiqqureshi/uplumo/backend/internal/billing"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/market"
)

// TrackingConfig is snapshotted into every run; edits never rewrite evidence.
type TrackingConfig struct {
	Brand            string          `json:"brand"`
	Domain           string          `json:"domain"`
	Competitors      []TrackedBrand  `json:"competitors"`
	Prompts          []TrackedPrompt `json:"prompts"`
	Engines          []string        `json:"engines"`
	LocationCode     int             `json:"locationCode"`
	LanguageCode     string          `json:"languageCode"`
	Interval         string          `json:"interval"`
	Enabled          bool            `json:"enabled"`
	MaxCreditsPerRun int64           `json:"maxCreditsPerRun"`
}

// TrackedBrand identifies an owned or competing public domain.
type TrackedBrand struct {
	Name   string `json:"name"`
	Domain string `json:"domain"`
}

// TrackedPrompt is a question with an editable topic and collection state.
type TrackedPrompt struct {
	Text   string `json:"text"`
	Topic  string `json:"topic"`
	Paused bool   `json:"paused"`
}

// TrackingEstimate quotes the maximum reserved credits for a full check.
type TrackingEstimate struct {
	Answers        int   `json:"answers"`
	Credits        int64 `json:"credits"`
	MonthlyCredits int64 `json:"monthlyCredits"`
	// Credits is a pre-call ceiling, not a claim of typical provider usage.
	Ceiling bool `json:"ceiling"`
}

// TrackingRun summarizes one immutable collection scope and its coverage.
type TrackingRun struct {
	ID            string         `json:"id"`
	Configuration TrackingConfig `json:"configuration"`
	ScopeKey      string         `json:"scopeKey"`
	Trigger       string         `json:"trigger"`
	Status        string         `json:"status"`
	ErrorCode     string         `json:"errorCode"`
	CreatedAt     time.Time      `json:"createdAt"`
	FinishedAt    *time.Time     `json:"finishedAt"`
	Completed     int            `json:"completed"`
	Failed        int            `json:"failed"`
	Answered      int            `json:"answered"`
	Mentioned     int            `json:"mentioned"`
	Cited         int            `json:"cited"`
}

// BrandObservation separates textual mentions from website citations.
type BrandObservation struct {
	Name      string `json:"name"`
	Domain    string `json:"domain"`
	Mentioned bool   `json:"mentioned"`
	Cited     bool   `json:"cited"`
}

// RecommendedBrand is a business the AI itself named in its answer
// (DataForSEO brand_entities), in the order it named them. It shows who the AI
// recommends instead of the tracked brand, without the customer having to
// guess their competitors first.
type RecommendedBrand struct {
	Name     string `json:"name"`
	Category string `json:"category,omitempty"`
	Position int    `json:"position"`
}

// storedBrand is one element of the brands column: a tracked brand's
// observation, or (Recommended set) a brand the AI named. Both live in one
// column so no migration is needed; rows written before recommendations were
// kept simply have none.
type storedBrand struct {
	BrandObservation
	Recommended bool   `json:"recommended,omitempty"`
	Category    string `json:"category,omitempty"`
	Position    int    `json:"position,omitempty"`
}

// TrackingAnswer is saved evidence for one question and engine.
type TrackingAnswer struct {
	ID          string             `json:"id"`
	Prompt      string             `json:"prompt"`
	Topic       string             `json:"topic"`
	Engine      string             `json:"engine"`
	Status      string             `json:"status"`
	TaskID      string             `json:"-"`
	Answer      string             `json:"answer"`
	Citations   []Citation         `json:"citations"`
	Brands      []BrandObservation `json:"brands"`
	Recommended []RecommendedBrand `json:"recommended"`
	ErrorCode   string             `json:"errorCode"`
	CollectedAt *time.Time         `json:"collectedAt"`
	// Sentiment is how the answer speaks about the brand ("" until the
	// customer classifies it; see Sentiment). Quote is the deciding words,
	// always found in the answer; Reason is a short plain explanation.
	Sentiment       string `json:"sentiment"`
	SentimentQuote  string `json:"sentimentQuote"`
	SentimentReason string `json:"sentimentReason"`
}

// TrackingState contains configuration and free-to-read recent history.
type TrackingState struct {
	Configuration      *TrackingConfig    `json:"configuration"`
	NextCheckAt        *time.Time         `json:"nextCheckAt"`
	Runs               []TrackingRun      `json:"runs"`
	ProviderConfigured bool               `json:"providerConfigured"`
	Visibility         *VisibilityMetrics `json:"visibility"`
}

// Tracking reuses the existing project middleware, paid provider and jobs queue.
type Tracking struct {
	DB     *pgxpool.Pool
	Search *Service
	Plans  PaidPlans
}

func (c *TrackingConfig) validate() error {
	c.Brand, c.Domain = strings.TrimSpace(c.Brand), strings.TrimSpace(c.Domain)
	if utf16Len(c.Brand) < 2 || utf16Len(c.Brand) > 100 {
		return badRequest("Brand name must contain 2–100 characters.")
	}
	target, err := parseResearchTarget(c.Domain, ScopeDomain)
	if err != nil {
		return badRequest("Enter a valid public website domain.")
	}
	c.Domain = target.Hostname
	if len(c.Prompts) < 1 || len(c.Prompts) > 30 {
		return badRequest("Save 1–30 questions per project.")
	}
	seen := map[string]bool{}
	active := 0
	for i := range c.Prompts {
		p := &c.Prompts[i]
		p.Text = strings.Join(strings.Fields(p.Text), " ")
		p.Topic = strings.TrimSpace(p.Topic)
		if p.Topic == "" {
			p.Topic = "General"
		}
		if utf16Len(p.Text) < 3 || utf16Len(p.Text) > maxPromptChars || utf16Len(p.Topic) > 80 || seen[p.Text] {
			return badRequest("Questions must be distinct, 3–500 characters, with topics up to 80 characters.")
		}
		seen[p.Text] = true
		if !p.Paused {
			active++
		}
	}
	if active == 0 && c.Enabled {
		return badRequest("Enable at least one question before scheduling.")
	}
	if len(c.Engines) < 1 || len(c.Engines) > 5 {
		return badRequest("Choose 1–5 engines.")
	}
	c.Engines = slices.Sorted(slices.Values(c.Engines))
	c.Engines = slices.Compact(c.Engines)
	if _, ok := market.Lookup(c.LocationCode); !ok || !market.IsSupportedLanguageCode(c.LanguageCode) {
		return badRequest("Choose a supported country and language.")
	}
	if c.Interval != "daily" && c.Interval != "weekly" && c.Interval != "monthly" {
		return badRequest("Choose daily, weekly or monthly checks.")
	}
	for _, engine := range c.Engines {
		if !slices.Contains([]string{"chat_gpt", "gemini", "google_ai_overview", "claude", "perplexity"}, engine) {
			return badRequest("Unknown answer engine.")
		}
		if engine == "chat_gpt" && c.LocationCode == 2275 {
			return badRequest("ChatGPT collection is unavailable in this market.")
		}
		if (engine == "claude" || engine == "perplexity") && !supportsWebSearchCountry(engine, strings.ToUpper(market.ISOCode(c.LocationCode))) {
			return badRequest("One selected model does not support this country.")
		}
	}
	if len(c.Competitors) > 5 {
		return badRequest("Compare at most five competitors.")
	}
	domains := map[string]bool{c.Domain: true}
	for i := range c.Competitors {
		b := &c.Competitors[i]
		b.Name = strings.TrimSpace(b.Name)
		target, e := parseResearchTarget(b.Domain, ScopeDomain)
		if e != nil || utf16Len(b.Name) < 2 || utf16Len(b.Name) > 100 || domains[target.Hostname] {
			return badRequest("Competitors need distinct public domains and names of 2–100 characters.")
		}
		b.Domain = target.Hostname
		domains[b.Domain] = true
	}
	if c.Competitors == nil {
		c.Competitors = []TrackedBrand{}
	}
	return nil
}

func (c TrackingConfig) estimate() TrackingEstimate {
	n := 0
	for _, p := range c.Prompts {
		if !p.Paused {
			n++
		}
	}
	usd := 0.0
	for _, engine := range c.Engines {
		cost := 0.0012
		if engine == "claude" || engine == "perplexity" {
			cost = dataforseo.AIResponseCeilingUSD
		}
		usd += float64(n) * cost
	}
	checks := 4.35
	if c.Interval == "daily" {
		checks = 30
	}
	if c.Interval == "monthly" {
		checks = 1
	}
	credits := billing.CreditsForUSD(usd)
	// One hold covers the complete run, including every engine and question.
	return TrackingEstimate{n * len(c.Engines), credits, int64(math.Ceil(float64(credits) * checks)), true}
}
func trackingScope(c TrackingConfig) string {
	// Topics and schedules do not change the questions or market being measured.
	c.Enabled = false
	c.Interval = ""
	c.MaxCreditsPerRun = 0
	c.Engines = slices.Clone(c.Engines)
	slices.Sort(c.Engines)
	c.Prompts = slices.Clone(c.Prompts)
	for i := range c.Prompts {
		c.Prompts[i].Topic = ""
	}
	slices.SortFunc(c.Prompts, func(a, b TrackedPrompt) int { return strings.Compare(a.Text, b.Text) })
	c.Competitors = slices.Clone(c.Competitors)
	slices.SortFunc(c.Competitors, func(a, b TrackedBrand) int { return strings.Compare(a.Domain, b.Domain) })
	raw, _ := json.Marshal(c)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func nextTrackingCheck(t time.Time, interval string) time.Time {
	switch interval {
	case "daily":
		return t.AddDate(0, 0, 1)
	case "monthly":
		first := time.Date(t.Year(), t.Month()+1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
		last := first.AddDate(0, 1, -1).Day()
		return first.AddDate(0, 0, min(t.Day(), last)-1)
	default:
		return t.AddDate(0, 0, 7)
	}
}
func (s *Tracking) allowed(ctx context.Context, org string) error {
	if s.Plans == nil {
		return errTrackingPlan
	}
	ok, err := s.Plans.HasPaidPlan(ctx, org)
	if err != nil {
		return err
	}
	if !ok {
		return errTrackingPlan
	}
	return nil
}

var errTrackingPlan = errors.New("AI Visibility is included in Growth and above")
var errTrackingBusy = errors.New("a check is already running")
var errTrackingMissing = errors.New("tracking not found")

// ScheduledCredits estimates checks due before the next credit refill.
func (s *Tracking) ScheduledCredits(ctx context.Context, org string, until time.Time) (int64, error) {
	rows, err := s.DB.Query(ctx, `SELECT configuration,next_check_at FROM go_ai_trackers t JOIN projects p ON p.id=t.project_id WHERE t.organization_id=$1 AND p.archived_at IS NULL AND next_check_at IS NOT NULL AND next_check_at<$2`, org, until)
	if err != nil {
		return 0, fmt.Errorf("load AI scheduled budget: %w", err)
	}
	defer rows.Close()
	var total int64
	for rows.Next() {
		var raw []byte
		var next time.Time
		if err := rows.Scan(&raw, &next); err != nil {
			return 0, err
		}
		var c TrackingConfig
		if err := json.Unmarshal(raw, &c); err != nil {
			return 0, err
		}
		for n := 0; next.Before(until) && n < 366; n++ {
			total += c.estimate().Credits
			next = nextTrackingCheck(next, c.Interval)
		}
	}
	return total, rows.Err()
}
