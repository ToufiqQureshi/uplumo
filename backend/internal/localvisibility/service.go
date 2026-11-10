// Package localvisibility is the Local Visibility feature: a project's
// Google Business Profile compared with the businesses that beat it on Google
// Maps, from DataForSEO's Maps and Business Data APIs.
package localvisibility

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strconv"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/billing"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/jobs"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

// QueueName is the jobs queue that collects queued DataForSEO tasks.
const QueueName = "local_visibility"

const (
	// cacheFor is how long a finished run answers the same input for free.
	cacheFor = 24 * time.Hour
	// runWindow is how long a run waits for its queued tasks. Priority
	// tasks finish in about a minute; what is not back by then is shown as
	// missing.
	runWindow = 10 * time.Minute
	pollEvery = 15 * time.Second
	jobTimout = runWindow + 2*time.Minute
)

// Errors the handler turns into customer messages.
var (
	ErrNoBusiness   = errors.New("choose your business first")
	ErrNeedsCheck   = errors.New("run a profile check first")
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("not found")
)

// Enqueuer schedules the collect job; *jobs.Queue implements it.
type Enqueuer interface {
	Enqueue(context.Context, jobs.EnqueueInput) (jobs.Job, error)
}

// Service runs Local Visibility checks.
type Service struct {
	Store  Store
	Client Client
	Queue  Enqueuer
	// LLM and Meter write review themes and reply drafts; without them the
	// reviews tab still shows every number, without themes.
	LLM    LLM
	Meter  Meter
	Logger *slog.Logger
	Now    func() time.Time
	Sleep  func(context.Context, time.Duration) error
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s *Service) logger() *slog.Logger {
	if s.Logger != nil {
		return s.Logger
	}
	return slog.Default()
}

func (s *Service) sleep(ctx context.Context, d time.Duration) error {
	if s.Sleep != nil {
		return s.Sleep(ctx, d)
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type rival struct {
	CID   string `json:"cid"`
	Title string `json:"title"`
}

// runInput is everything a run's tasks are built from; its hash is the cache key.
type runInput struct {
	CID          string  `json:"cid,omitempty"`
	Term         string  `json:"term,omitempty"`
	Latitude     float64 `json:"lat,omitempty"`
	Longitude    float64 `json:"lng,omitempty"`
	Language     string  `json:"language"`
	Size         int     `json:"size,omitempty"`
	SpacingKm    float64 `json:"spacingKm,omitempty"`
	Rivals       []rival `json:"rivals,omitempty"`
	Query        string  `json:"query,omitempty"`
	LocationCode int     `json:"locationCode,omitempty"`
}

func (in runInput) hash() string {
	raw, _ := json.Marshal(in)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// StartRequest is what the customer chose for a run.
type StartRequest struct {
	Kind      string  `json:"kind"`
	Size      int     `json:"size"`
	SpacingKm float64 `json:"spacingKm"`
}

// Grid sizes and spacings the customer can pick.
var (
	gridSizes    = []int{3, 5}
	gridSpacings = []float64{0.5, 1, 2, 5}
)

// Start begins a paid run, or answers with the run already doing the same
// work or a finished run of the same input from the last 24 hours.
func (s *Service) Start(ctx context.Context, org, projectID string, req StartRequest) (*Run, error) {
	business, err := s.Store.Business(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if business == nil {
		return nil, ErrNoBusiness
	}
	in, err := s.input(ctx, projectID, *business, req)
	if err != nil {
		return nil, err
	}
	return s.start(ctx, org, projectID, req.Kind, in)
}

func (s *Service) input(ctx context.Context, projectID string, b Business, req StartRequest) (runInput, error) {
	in := runInput{CID: b.CID, Latitude: b.Latitude, Longitude: b.Longitude, Language: b.LanguageCode}
	switch req.Kind {
	case KindCheck:
		in.Term = b.SearchTerm
	case KindGrid:
		if !slices.Contains(gridSizes, req.Size) || !slices.Contains(gridSpacings, req.SpacingKm) {
			return runInput{}, fmt.Errorf("%w: pick a 3 x 3 or 5 x 5 grid, 0.5 to 5 km apart", ErrInvalidInput)
		}
		in.Term, in.Size, in.SpacingKm = b.SearchTerm, req.Size, req.SpacingKm
	case KindReviews, KindWatch:
		rivals, err := s.rivals(ctx, projectID, b.CID)
		if err != nil {
			return runInput{}, err
		}
		if req.Kind == KindWatch && len(rivals) == 0 {
			return runInput{}, ErrNeedsCheck
		}
		in.Rivals = rivals
	default:
		return runInput{}, fmt.Errorf("%w: unknown check", ErrInvalidInput)
	}
	return in, nil
}

// rivals are the competitors of the newest profile check of this business.
func (s *Service) rivals(ctx context.Context, projectID, cid string) ([]rival, error) {
	run, err := s.Store.LatestDone(ctx, projectID, KindCheck, s.now().Add(time.Minute))
	if err != nil || run == nil {
		return []rival{}, err
	}
	var result CheckResult
	if err := json.Unmarshal(run.Result, &result); err != nil {
		return nil, fmt.Errorf("decode check result: %w", err)
	}
	out := []rival{}
	if result.You.Place.CID != cid {
		return out, nil
	}
	for _, r := range result.Rivals {
		out = append(out, rival{CID: r.Place.CID, Title: r.Place.Title})
	}
	return out, nil
}

func (s *Service) start(ctx context.Context, org, projectID, kind string, in runInput) (*Run, error) {
	hash := in.hash()
	cached, err := s.Store.Cached(ctx, projectID, kind, hash, s.now().Add(-cacheFor))
	if err != nil {
		return nil, err
	}
	if cached != nil && kind != KindWatch {
		cached.Cached = true
		return cached, nil
	}
	rawInput, _ := json.Marshal(in)
	run, created, err := s.Store.CreateRun(ctx, Run{ProjectID: projectID, OrganizationID: org, Kind: kind, InputHash: hash, Input: rawInput})
	if err != nil || !created {
		return run, err
	}
	posted, err := s.postAll(ctx, run, firstTasks(kind, in))
	if posted == 0 {
		message := "Google data could not be requested. Nothing was charged."
		if text := spend.Message(err); text != "" {
			message = text
		}
		if finishErr := s.Store.FinishRun(context.WithoutCancel(ctx), run.ID, StatusFailed, nil, message); finishErr != nil {
			return nil, errors.Join(err, finishErr)
		}
		return nil, err
	}
	if _, err := s.Queue.Enqueue(ctx, jobs.EnqueueInput{
		Queue: QueueName, IdempotencyKey: run.ID, Payload: json.RawMessage(`{"runId":"` + run.ID + `"}`),
		MaxAttempts: 3, Timeout: jobTimout,
	}); err != nil {
		// The tasks are paid; the tick finishes runs whose job never ran.
		s.logger().ErrorContext(ctx, "enqueue local visibility run", "run", run.ID, "err", err)
	}
	return run, nil
}

// firstTasks are what a run posts when it starts. A check posts the
// competitors' profiles once the Maps search says who they are.
func firstTasks(kind string, in runInput) []task {
	switch kind {
	case KindCheck:
		return []task{mapsTask("maps", in.Term, in.Language, in.Latitude, in.Longitude, 15)}
	case KindGrid:
		var tasks []task
		for i, p := range gridPoints(in.Latitude, in.Longitude, in.Size, in.SpacingKm) {
			tasks = append(tasks, mapsTask("p"+strconv.Itoa(i), in.Term, in.Language, p.Latitude, p.Longitude, gridZoom(in.SpacingKm)))
		}
		return tasks
	case KindReviews:
		tasks := []task{reviewsTask("rev:you", in.CID, in.Language, in.Latitude, in.Longitude, ownReviewsDepth)}
		for i, r := range in.Rivals {
			tasks = append(tasks, reviewsTask("rev:r"+strconv.Itoa(i), r.CID, in.Language, in.Latitude, in.Longitude, rivalReviewsDepth))
		}
		return tasks
	case KindWatch:
		var tasks []task
		for i, r := range in.Rivals {
			key := "r" + strconv.Itoa(i)
			tasks = append(tasks, infoTask("info:"+key, r.CID, in.Language, in.Latitude, in.Longitude),
				updatesTask("posts:"+key, r.CID, in.Language, in.Latitude, in.Longitude))
		}
		return tasks
	}
	return nil
}

// profileTasks are the check's second step: the profile and posts of the
// business and of each competitor.
func profileTasks(in runInput, rivals []Place) []task {
	tasks := []task{
		infoTask("info:you", in.CID, in.Language, in.Latitude, in.Longitude),
		updatesTask("posts:you", in.CID, in.Language, in.Latitude, in.Longitude),
	}
	for i, r := range rivals {
		key := "r" + strconv.Itoa(i)
		tasks = append(tasks, infoTask("info:"+key, r.CID, in.Language, in.Latitude, in.Longitude),
			updatesTask("posts:"+key, r.CID, in.Language, in.Latitude, in.Longitude))
	}
	return tasks
}

// postAll posts each task once. Each post carries an idempotency key made
// of the run and the task, so a retried job can never buy a task twice; a
// task whose outcome is unknown is marked failed instead of retried. It
// stops at the first refusal by the spend guard (no credits, paused).
func (s *Service) postAll(ctx context.Context, run *Run, tasks []task) (int, error) {
	existing, err := s.Store.tasks(ctx, run.ID)
	if err != nil {
		return 0, err
	}
	posted := 0
	var firstErr error
	for i, t := range tasks {
		if _, ok := existing[t.Key]; ok {
			posted++
			continue
		}
		postCtx := spend.WithIdempotencyKey(ctx, "local:"+run.ID+":"+t.Key)
		id, err := post(postCtx, s.Client, run.OrganizationID, t)
		if err != nil {
			firstErr = cmpErr(firstErr, err)
			if spend.Refused(err) || errors.Is(err, dataforseo.ErrBillingIssue) {
				for _, rest := range tasks[i:] {
					if _, ok := existing[rest.Key]; !ok {
						if failErr := s.Store.FailTask(context.WithoutCancel(ctx), run.ID, rest); failErr != nil {
							return posted, errors.Join(err, failErr)
						}
					}
				}
				return posted, err
			}
			s.logger().WarnContext(ctx, "local visibility task_post failed", "run", run.ID, "task", t.Key, "err", err)
			if failErr := s.Store.FailTask(context.WithoutCancel(ctx), run.ID, t); failErr != nil {
				return posted, errors.Join(err, failErr)
			}
			continue
		}
		if err := s.Store.RecordTask(context.WithoutCancel(ctx), run.ID, t, id); err != nil {
			return posted, err
		}
		posted++
	}
	return posted, firstErr
}

func cmpErr(first, next error) error {
	if first != nil {
		return first
	}
	return next
}

// JobHandler collects a run's queued tasks until all are back or the run's
// window closes, then stores the result.
func (s *Service) JobHandler() jobs.Handler {
	return func(ctx context.Context, job jobs.Job) error {
		var payload struct {
			RunID string `json:"runId"`
		}
		if err := json.Unmarshal(job.Payload, &payload); err != nil || payload.RunID == "" {
			return fmt.Errorf("%w: local visibility job payload", jobs.ErrInvalidJob)
		}
		return s.Process(ctx, payload.RunID)
	}
}

// Process collects and finishes one run. It is safe to call more than once.
func (s *Service) Process(ctx context.Context, runID string) error {
	run, err := s.Store.runByID(ctx, runID)
	if err != nil || run == nil || run.Status != StatusRunning {
		return err
	}
	var in runInput
	if err := json.Unmarshal(run.Input, &in); err != nil {
		return fmt.Errorf("decode local run input: %w", err)
	}
	deadline := run.CreatedAt.Add(runWindow)
	for {
		tasks, err := s.Store.tasks(ctx, run.ID)
		if err != nil {
			return err
		}
		pending, err := s.collectRound(ctx, run.ID, tasks)
		if err != nil {
			return err
		}
		if run.Kind == KindCheck {
			more, err := s.secondStep(ctx, run, in, tasks)
			if err != nil {
				return err
			}
			pending += more
		}
		if pending == 0 || !s.now().Before(deadline) {
			tasks, err := s.Store.tasks(ctx, run.ID)
			if err != nil {
				return err
			}
			return s.finish(ctx, run, in, tasks)
		}
		if err := s.sleep(ctx, pollEvery); err != nil {
			return err
		}
	}
}

// taskFailure is a task the provider reported as failed.
type taskFailure struct{ err error }

func (f taskFailure) Error() string { return f.err.Error() }

func isTaskFailure(err error) bool {
	_, ok := errors.AsType[taskFailure](err)
	return ok
}

// collectRound reads every posted task once and returns how many are still
// running. A read that fails for a network reason is tried next round.
func (s *Service) collectRound(ctx context.Context, runID string, tasks map[string]storedTask) (int, error) {
	pending := 0
	for key, t := range tasks {
		if t.State != taskPosted {
			continue
		}
		result, err := collect(ctx, s.Client, t.Endpoint, t.TaskID)
		state := taskDone
		switch {
		case errors.Is(err, errNotReady):
			pending++
			continue
		case errors.Is(err, errNoResult):
			result = nil
		case isTaskFailure(err):
			s.logger().WarnContext(ctx, "local visibility task failed", "run", runID, "task", key, "err", err)
			state, result = taskFailed, nil
		case err != nil:
			if ctx.Err() != nil {
				return pending, ctx.Err()
			}
			s.logger().WarnContext(ctx, "local visibility task_get failed", "run", runID, "task", key, "err", err)
			pending++
			continue
		}
		if err := s.Store.SetTask(ctx, runID, key, state, result); err != nil {
			return pending, err
		}
		// The check's second step reads the Maps result from this map.
		t.State, t.Result = state, result
		tasks[key] = t
	}
	return pending, nil
}

// secondStep posts the profiles once the check's Maps search is back.
func (s *Service) secondStep(ctx context.Context, run *Run, in runInput, tasks map[string]storedTask) (int, error) {
	maps, ok := tasks["maps"]
	if !ok || maps.State == taskPosted {
		return 0, nil
	}
	if _, posted := tasks["info:you"]; posted {
		return 0, nil
	}
	var places []Place
	if maps.State == taskDone && len(maps.Result) > 0 {
		var err error
		if places, err = parseMaps(maps.Result); err != nil {
			s.logger().WarnContext(ctx, "local visibility maps result unreadable", "run", run.ID, "err", err)
		}
	}
	posted, err := s.postAll(ctx, run, profileTasks(in, pickRivals(places, in.CID)))
	if err != nil && posted == 0 {
		s.logger().WarnContext(ctx, "local visibility profiles not requested", "run", run.ID, "err", err)
	}
	return posted, nil
}

// finish builds the run's result from whatever came back. A run where
// nothing came back fails.
func (s *Service) finish(ctx context.Context, run *Run, in runInput, tasks map[string]storedTask) error {
	result, ok, err := s.build(ctx, run, in, tasks)
	if err != nil {
		return err
	}
	if !ok {
		return s.Store.FinishRun(ctx, run.ID, StatusFailed, nil, "Google did not return data for this check in time. Try again later.")
	}
	return s.Store.FinishRun(ctx, run.ID, StatusDone, result, "")
}

func (s *Service) build(ctx context.Context, run *Run, in runInput, tasks map[string]storedTask) (any, bool, error) {
	done := func(key string) (json.RawMessage, bool) {
		t, ok := tasks[key]
		return t.Result, ok && t.State == taskDone
	}
	switch run.Kind {
	case KindCheck:
		result, ok := buildCheck(in, done, run.CreatedAt)
		return result, ok, nil
	case KindGrid:
		result, ok := buildGrid(in, done)
		return result, ok, nil
	case KindReviews:
		return s.buildReviews(ctx, run, in, done)
	case KindWatch:
		return s.buildWatch(ctx, run, in, done)
	}
	return nil, false, fmt.Errorf("unknown local run kind %q", run.Kind)
}

type doneFunc func(key string) (json.RawMessage, bool)

func buildCheck(in runInput, done doneFunc, now time.Time) (CheckResult, bool) {
	result := CheckResult{SearchTerm: in.Term, Rivals: []Contender{}, Missing: []string{}}
	raw, ok := done("maps")
	if !ok {
		result.Missing = append(result.Missing, "Google Maps results")
	}
	places, _ := parseMaps(raw)
	result.YourRank = rankOf(places, in.CID)
	contender := func(key string, place Place) Contender {
		c := Contender{Place: place}
		if raw, ok := done("info:" + key); ok {
			if profile, err := parseProfile(raw); err == nil {
				c.Profile = &profile
			}
		}
		if raw, ok := done("posts:" + key); ok {
			posts := []Post{}
			if len(raw) > 0 {
				posts, _ = parsePosts(raw)
			}
			n, last := postStats(posts, now)
			c.PostsLast30, c.LastPostAt = &n, last
		}
		return c
	}
	own := Place{CID: in.CID, Rank: result.YourRank}
	for _, p := range places {
		if p.CID == in.CID {
			own = p
		}
	}
	result.You = contender("you", own)
	if result.You.Profile == nil {
		result.Missing = append(result.Missing, "your profile")
	} else if own.Title == "" {
		result.You.Place.Title = result.You.Profile.Title
	}
	for i, p := range pickRivals(places, in.CID) {
		c := contender("r"+strconv.Itoa(i), p)
		if c.Profile == nil {
			result.Missing = append(result.Missing, p.Title+"'s profile")
		}
		result.Rivals = append(result.Rivals, c)
	}
	result.Fixes = buildFixes(result.You, result.Rivals, now)
	result.Summary = checkSummary(in.Term, result.YourRank, result.Fixes)
	return result, ok || result.You.Profile != nil
}

func buildGrid(in runInput, done doneFunc) (GridResult, bool) {
	points := gridPoints(in.Latitude, in.Longitude, in.Size, in.SpacingKm)
	found := false
	for i := range points {
		raw, ok := done("p" + strconv.Itoa(i))
		if !ok {
			points[i].Missing = true
			continue
		}
		found = true
		places, _ := parseMaps(raw)
		points[i].Rank = rankOf(places, in.CID)
		points[i].Top = []string{}
		for _, p := range places[:min(3, len(places))] {
			points[i].Top = append(points[i].Top, p.Title)
		}
	}
	return GridResult{SearchTerm: in.Term, Size: in.Size, SpacingKm: in.SpacingKm, Points: points, Summary: gridSummary(points)}, found
}

func (s *Service) buildReviews(ctx context.Context, run *Run, in runInput, done doneFunc) (ReviewsResult, bool, error) {
	result := ReviewsResult{Themes: []Theme{}, Rivals: []RivalReviews{}, Missing: []string{}, Unanswered: []Review{}}
	raw, ok := done("rev:you")
	if !ok {
		return result, false, nil
	}
	own, err := parseReviews(orEmpty(raw))
	if err != nil {
		return result, false, err
	}
	reviewStats(&result, own)
	sets := []reviewSet{{Key: "you", Reviews: own.Reviews}}
	for i, r := range in.Rivals {
		raw, ok := done("rev:r" + strconv.Itoa(i))
		if !ok {
			result.Missing = append(result.Missing, r.Title+"'s reviews")
			continue
		}
		parsed, err := parseReviews(orEmpty(raw))
		if err != nil {
			result.Missing = append(result.Missing, r.Title+"'s reviews")
			continue
		}
		result.Rivals = append(result.Rivals, RivalReviews{CID: r.CID, Title: r.Title, Fetched: len(parsed.Reviews), ReplyRate: replyRate(parsed.Reviews), Themes: []Theme{}})
		sets = append(sets, reviewSet{Key: "r" + strconv.Itoa(len(result.Rivals)-1), Reviews: parsed.Reviews})
	}
	themes, err := s.themes(ctx, run, sets)
	if err != nil {
		s.logger().WarnContext(ctx, "local visibility review themes failed", "run", run.ID, "err", err)
		result.Missing = append(result.Missing, "review themes")
	} else {
		result.Themes = orThemes(themes["you"])
		for i := range result.Rivals {
			result.Rivals[i].Themes = orThemes(themes["r"+strconv.Itoa(i)])
		}
	}
	result.Summary = reviewsSummary(result)
	return result, true, nil
}

func (s *Service) buildWatch(ctx context.Context, run *Run, in runInput, done doneFunc) (WatchResult, bool, error) {
	result := WatchResult{Snapshots: []Snapshot{}}
	found := false
	for i, r := range in.Rivals {
		key := "r" + strconv.Itoa(i)
		snap := Snapshot{CID: r.CID, Title: r.Title, Categories: []string{}, Missing: true}
		if raw, ok := done("info:" + key); ok && len(raw) > 0 {
			if profile, err := parseProfile(raw); err == nil {
				found = true
				snap = Snapshot{CID: r.CID, Title: r.Title, Rating: profile.Rating, Reviews: profile.Reviews, Photos: profile.Photos,
					Categories: profile.Categories, HoursKey: profile.HoursKey}
				if raw, ok := done("posts:" + key); ok && len(raw) > 0 {
					posts, _ := parsePosts(raw)
					for _, p := range posts {
						if run.CreatedAt.Sub(p.At) <= 7*24*time.Hour {
							snap.PostsLast7++
						}
						if snap.LatestPostAt == nil || p.At.After(*snap.LatestPostAt) {
							at := p.At
							snap.LatestPostAt = &at
						}
					}
				}
			}
		}
		result.Snapshots = append(result.Snapshots, snap)
	}
	previous, err := s.Store.LatestDone(ctx, run.ProjectID, KindWatch, run.CreatedAt)
	if err != nil {
		return result, false, err
	}
	var before []Snapshot
	if previous != nil {
		var prior WatchResult
		if err := json.Unmarshal(previous.Result, &prior); err == nil {
			before = prior.Snapshots
			at := previous.CreatedAt
			result.Since = &at
		}
	}
	result.Changes = compareWatch(result.Snapshots, before)
	result.Summary = watchSummary(result.Changes, previous == nil)
	return result, found, nil
}

func orEmpty(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}

func orThemes(themes []Theme) []Theme {
	if themes == nil {
		return []Theme{}
	}
	return themes
}

// Tick starts the weekly watches that are due and finishes runs whose
// collect job was lost. It makes no paid call except the due watches the
// customer turned on.
func (s *Service) Tick(ctx context.Context, plans PaidPlans) error {
	due, err := s.Store.claimDueWatches(ctx, s.now(), 50)
	if err != nil {
		return err
	}
	for _, w := range due {
		if plans != nil {
			ok, err := plans.HasPaidPlan(ctx, w.OrganizationID)
			if err != nil || !ok {
				continue
			}
		}
		if _, err := s.Start(ctx, w.OrganizationID, w.ProjectID, StartRequest{Kind: KindWatch}); err != nil {
			s.logger().WarnContext(ctx, "weekly local watch did not start", "project", w.ProjectID, "err", err)
		}
	}
	rows, err := s.Store.DB.Query(ctx, `SELECT id::text FROM go_local_runs WHERE status='running' AND created_at < $1 LIMIT 20`, s.now().Add(-jobTimout-time.Minute))
	if err != nil {
		return fmt.Errorf("list stale local runs: %w", err)
	}
	var stale []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("scan stale local run: %w", err)
		}
		stale = append(stale, id)
	}
	rows.Close()
	for _, id := range stale {
		if err := s.Process(ctx, id); err != nil {
			s.logger().ErrorContext(ctx, "finish stale local run", "run", id, "err", err)
		}
	}
	return nil
}

// PaidPlans reports whether a workspace's plan includes Local Visibility.
type PaidPlans interface {
	HasPaidPlan(context.Context, string) (bool, error)
}

// Prices are the most each action can charge, shown on its button.
type Prices struct {
	Find               int64 `json:"find"`
	Check              int64 `json:"check"`
	Grid3              int64 `json:"grid3"`
	Grid5              int64 `json:"grid5"`
	Reviews            int64 `json:"reviews"`
	WatchPerCompetitor int64 `json:"watchPerCompetitor"`
	Reply              int64 `json:"reply"`
}

// PriceList prices every action from the exact task bodies it posts.
func PriceList() Prices {
	in := runInput{CID: "1", Term: "x", Language: "en", Rivals: []rival{{CID: "2"}, {CID: "3"}, {CID: "4"}}}
	grid := func(size int) int64 {
		g := in
		g.Size, g.SpacingKm = size, 1
		return billing.CreditsForUSD(tasksCostUSD(firstTasks(KindGrid, g)))
	}
	rivals := []Place{{CID: "2"}, {CID: "3"}, {CID: "4"}}
	return Prices{
		Find:               billing.CreditsForUSD(FindCostUSD()),
		Check:              billing.CreditsForUSD(tasksCostUSD(firstTasks(KindCheck, in)) + tasksCostUSD(profileTasks(in, rivals))),
		Grid3:              grid(3),
		Grid5:              grid(5),
		Reviews:            billing.CreditsForUSD(tasksCostUSD(firstTasks(KindReviews, in)) + themesCeilingUSD),
		WatchPerCompetitor: billing.CreditsForUSD(tasksCostUSD(firstTasks(KindWatch, runInput{Language: "en", Rivals: []rival{{CID: "2"}}}))),
		Reply:              billing.CreditsForUSD(replyCeilingUSD),
	}
}
