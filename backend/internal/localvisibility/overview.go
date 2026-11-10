package localvisibility

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

// Markets reads a project's country and language; *projects.Repository is one.
type Markets interface {
	ProjectMarket(ctx context.Context, organizationID, projectID string) (domain string, locationCode int, languageCode string, err error)
}

// ErrBusy means the same paid search is already running.
var ErrBusy = errors.New("a search is already running")

// WatchState is the weekly competitor watch as the page shows it.
type WatchState struct {
	Enabled        bool       `json:"enabled"`
	NextRunAt      *time.Time `json:"nextRunAt"`
	Competitors    []rival    `json:"competitors"`
	CreditsPerWeek int64      `json:"creditsPerWeek"`
}

// Overview is everything the page shows on load. It makes no paid call.
type Overview struct {
	Business *Business      `json:"business"`
	Runs     map[string]any `json:"runs"`
	Watch    WatchState     `json:"watch"`
	Prices   Prices         `json:"prices"`
	// AIReady is false on a server without an AI key: no themes or drafts.
	AIReady bool `json:"aiReady"`
}

// Overview loads the business, the newest run of each check and the prices.
func (s *Service) Overview(ctx context.Context, projectID string) (Overview, error) {
	prices := PriceList()
	out := Overview{Runs: map[string]any{}, Prices: prices, AIReady: s.LLM != nil && s.Meter != nil}
	business, err := s.Store.Business(ctx, projectID)
	if err != nil {
		return out, err
	}
	out.Business = business
	runs, err := s.Store.LatestRuns(ctx, projectID)
	if err != nil {
		return out, err
	}
	for kind, run := range runs {
		if kind == KindFind {
			continue
		}
		if kind == KindReviews && run.Status == StatusDone {
			if err := s.withDrafts(ctx, run); err != nil {
				return out, err
			}
		}
		out.Runs[kind] = run
	}
	enabled, next, err := s.Store.Watch(ctx, projectID)
	if err != nil {
		return out, err
	}
	out.Watch = WatchState{Enabled: enabled, NextRunAt: next, Competitors: []rival{}}
	if business != nil {
		if out.Watch.Competitors, err = s.rivals(ctx, projectID, business.CID); err != nil {
			return out, err
		}
	}
	out.Watch.CreditsPerWeek = prices.WatchPerCompetitor * int64(len(out.Watch.Competitors))
	return out, nil
}

// RunStatus is one run as the page polls it while it is running.
func (s *Service) RunStatus(ctx context.Context, projectID, runID string) (*Run, error) {
	run, err := s.Store.Run(ctx, projectID, runID)
	if err != nil {
		return nil, err
	}
	if run == nil || run.Kind == KindFind {
		return nil, ErrNotFound
	}
	if run.Kind == KindReviews && run.Status == StatusDone {
		if err := s.withDrafts(ctx, run); err != nil {
			return nil, err
		}
	}
	return run, nil
}

// withDrafts adds the saved reply drafts to a reviews result.
func (s *Service) withDrafts(ctx context.Context, run *Run) error {
	var result ReviewsResult
	if err := json.Unmarshal(run.Result, &result); err != nil {
		return fmt.Errorf("decode reviews result: %w", err)
	}
	ids := make([]string, len(result.Unanswered))
	for i, r := range result.Unanswered {
		ids[i] = r.ID
	}
	drafts, err := s.Store.ReplyDrafts(ctx, run.ProjectID, ids)
	if err != nil {
		return err
	}
	for i, r := range result.Unanswered {
		result.Unanswered[i].ReplyDraft = drafts[r.ID]
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("encode reviews result: %w", err)
	}
	run.Result = raw
	return nil
}

// FindResult is a business search: the listings to pick from.
type FindResult struct {
	Places []Place `json:"places"`
	Cached bool    `json:"cached"`
}

// Find searches Google Maps for the customer's business by name, in the
// project's country. The same search within 24 hours is answered free.
func (s *Service) Find(ctx context.Context, markets Markets, org, projectID, query string) (FindResult, error) {
	query = strings.Join(strings.Fields(query), " ")
	if n := utf8.RuneCountInString(query); n < 2 || n > 120 {
		return FindResult{}, fmt.Errorf("%w: type your business name, 2 to 120 characters", ErrInvalidInput)
	}
	_, locationCode, language, err := markets.ProjectMarket(ctx, org, projectID)
	if err != nil {
		return FindResult{}, err
	}
	in := runInput{Query: query, LocationCode: locationCode, Language: language}
	cached, err := s.Store.Cached(ctx, projectID, KindFind, in.hash(), s.now().Add(-cacheFor))
	if err != nil {
		return FindResult{}, err
	}
	if cached != nil {
		var places []Place
		if err := json.Unmarshal(cached.Result, &places); err != nil {
			return FindResult{}, fmt.Errorf("decode business search: %w", err)
		}
		return FindResult{Places: places, Cached: true}, nil
	}
	rawInput, _ := json.Marshal(in)
	run, created, err := s.Store.CreateRun(ctx, Run{ProjectID: projectID, OrganizationID: org, Kind: KindFind, InputHash: in.hash(), Input: rawInput})
	if err != nil {
		return FindResult{}, err
	}
	if !created {
		return FindResult{}, ErrBusy
	}
	places, err := findBusinesses(spend.WithIdempotencyKey(ctx, "local-find:"+run.ID), s.Client, org, query, locationCode, language)
	finishCtx := context.WithoutCancel(ctx)
	if err != nil {
		return FindResult{}, errors.Join(err, s.Store.FinishRun(finishCtx, run.ID, StatusFailed, nil, "The search failed."))
	}
	if err := s.Store.FinishRun(finishCtx, run.ID, StatusDone, places, ""); err != nil {
		return FindResult{}, err
	}
	return FindResult{Places: places}, nil
}

// SaveRequest picks the business and the search customers use to find it.
type SaveRequest struct {
	CID        string `json:"cid"`
	SearchTerm string `json:"searchTerm"`
}

// SaveBusiness sets the project's business to a listing from the newest
// business search, or changes the search term of the current business.
func (s *Service) SaveBusiness(ctx context.Context, markets Markets, org, projectID string, req SaveRequest) (*Business, error) {
	term := strings.Join(strings.Fields(req.SearchTerm), " ")
	if n := utf8.RuneCountInString(term); n < 2 || n > 80 {
		return nil, fmt.Errorf("%w: the search term must be 2 to 80 characters", ErrInvalidInput)
	}
	current, err := s.Store.Business(ctx, projectID)
	if err != nil {
		return nil, err
	}
	var business Business
	switch {
	case current != nil && current.CID == req.CID:
		business = *current
	default:
		place, err := s.searched(ctx, projectID, req.CID)
		if err != nil {
			return nil, err
		}
		_, _, language, err := markets.ProjectMarket(ctx, org, projectID)
		if err != nil {
			return nil, err
		}
		business = Business{CID: place.CID, PlaceID: place.PlaceID, Title: place.Title, Address: place.Address,
			Category: place.Category, Latitude: place.Latitude, Longitude: place.Longitude, LanguageCode: language}
	}
	business.SearchTerm = term
	if err := s.Store.SaveBusiness(ctx, org, projectID, business); err != nil {
		return nil, err
	}
	return s.Store.Business(ctx, projectID)
}

// searched finds a listing in the project's newest business search, so the
// saved coordinates are Google's, not the browser's.
func (s *Service) searched(ctx context.Context, projectID, cid string) (Place, error) {
	run, err := s.Store.LatestDone(ctx, projectID, KindFind, s.now().Add(time.Minute))
	if err != nil {
		return Place{}, err
	}
	if run != nil {
		var places []Place
		if err := json.Unmarshal(run.Result, &places); err != nil {
			return Place{}, fmt.Errorf("decode business search: %w", err)
		}
		for _, p := range places {
			if p.CID == cid && cidPattern.MatchString(cid) {
				return p, nil
			}
		}
	}
	return Place{}, fmt.Errorf("%w: search for your business again and pick it from the list", ErrInvalidInput)
}

// SetWatch turns the weekly competitor watch on or off. It needs a profile
// check, whose top 3 are the competitors watched.
func (s *Service) SetWatch(ctx context.Context, org, projectID string, enabled bool) (WatchState, error) {
	business, err := s.Store.Business(ctx, projectID)
	if err != nil {
		return WatchState{}, err
	}
	if business == nil {
		return WatchState{}, ErrNoBusiness
	}
	if enabled {
		rivals, err := s.rivals(ctx, projectID, business.CID)
		if err != nil {
			return WatchState{}, err
		}
		if len(rivals) == 0 {
			return WatchState{}, ErrNeedsCheck
		}
		// Turning the watch on takes this week's snapshot at once, so the
		// customer sees what they pay for; the next one is a week later.
		if on, _, err := s.Store.Watch(ctx, projectID); err != nil {
			return WatchState{}, err
		} else if !on {
			if _, err := s.Start(ctx, org, projectID, StartRequest{Kind: KindWatch}); err != nil {
				return WatchState{}, err
			}
		}
	}
	if err := s.Store.SetWatch(ctx, org, projectID, enabled, s.now().Add(7*24*time.Hour)); err != nil {
		return WatchState{}, err
	}
	overview, err := s.Overview(ctx, projectID)
	return overview.Watch, err
}
