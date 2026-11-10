package localvisibility

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toufiqqureshi/uplumo/backend/internal/database"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/jobs"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/openrouter"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/pgdb"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend/spendtest"
)

// fakeDataForSEO answers task_post and task_get like DataForSEO and counts
// every paid post by endpoint.
type fakeDataForSEO struct {
	mu     sync.Mutex
	posts  map[string]int
	bodies map[string]map[string]any // task id -> posted body
	n      int
}

func (f *fakeDataForSEO) postCount(path string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.posts[path]
}

func (f *fakeDataForSEO) totalPosts() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	total := 0
	for _, n := range f.posts {
		total += n
	}
	return total
}

func (f *fakeDataForSEO) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case strings.HasSuffix(r.URL.Path, "/task_post"):
		var tasks []map[string]any
		_ = json.NewDecoder(r.Body).Decode(&tasks)
		f.posts[r.URL.Path]++
		f.n++
		id := fmt.Sprintf("task-%d", f.n)
		f.bodies[id] = tasks[0]
		cost := dataforseo.EstimateUSD(r.URL.Path, mustJSON(tasks))
		reply(w, id, 20100, cost, "null")
	case strings.Contains(r.URL.Path, "/task_get/"):
		id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		body := f.bodies[id]
		reply(w, id, 20000, 0, "["+fixtureFor(r.URL.Path, body)+"]")
	case r.URL.Path == mapsLivePath:
		f.posts[r.URL.Path]++
		reply(w, "live", 20000, 0.002, "["+mapsResult+"]")
	default:
		http.NotFound(w, r)
	}
}

// reply writes a DataForSEO envelope with one task.
func reply(w http.ResponseWriter, id string, status int, cost float64, result string) {
	_ = json.NewEncoder(w).Encode(map[string]any{"status_code": 20000, "tasks": []map[string]any{{
		"id": id, "status_code": status, "path": []string{"v3"}, "cost": cost, "result": json.RawMessage(result),
	}}})
}

func mustJSON(v any) []byte {
	raw, _ := json.Marshal(v)
	return raw
}

// Our business (cid 100) is 4th; 11, 22 and 33 are above it.
const mapsResult = `{"items":[
 {"type":"maps_search","rank_group":1,"cid":"11","title":"Bright Smiles","category":"Dentist","latitude":18.53,"longitude":73.89},
 {"type":"maps_search","rank_group":2,"cid":"22","title":"Pune Dental Care","category":"Dentist","latitude":18.53,"longitude":73.89},
 {"type":"maps_search","rank_group":3,"cid":"33","title":"Aura Dental","category":"Dentist","latitude":18.53,"longitude":73.89},
 {"type":"maps_search","rank_group":4,"cid":"100","place_id":"own","title":"Smile Studio","address":"Koregaon Park","category":"Dentist","latitude":18.5362,"longitude":73.893}
]}`

func fixtureFor(path string, body map[string]any) string {
	cid := strings.TrimPrefix(fmt.Sprint(body["keyword"]), "cid:")
	if c, ok := body["cid"].(string); ok {
		cid = c
	}
	switch {
	case strings.Contains(path, "/maps/"):
		return mapsResult
	case strings.Contains(path, "my_business_info"):
		categories := `["Cosmetic dentist"]`
		if cid == "100" {
			categories = `[]`
		}
		return fmt.Sprintf(`{"items":[{"cid":%q,"title":"Business %s","category":"Dentist","additional_categories":%s,"is_claimed":true,"total_photos":40,
			"rating":{"value":4.7,"votes_count":300},"phone":"1","url":"https://x.example","description":"d","work_time":{"a":1}}]}`, cid, cid, categories)
	case strings.Contains(path, "my_business_updates"):
		return `{"items":[{"post_text":"Offer","timestamp":"` + time.Now().UTC().Add(-48*time.Hour).Format("2006-01-02 15:04:05 -07:00") + `"}]}`
	case strings.Contains(path, "/reviews/"):
		return fmt.Sprintf(`{"reviews_count":2,"rating":{"value":4.5},"items":[
			{"review_id":"%s-a","review_text":"Waited an hour","profile_name":"Rohan M","timestamp":"2026-11-01 10:00:00 +00:00","rating":{"value":2}},
			{"review_id":"%s-b","review_text":"Painless","profile_name":"Ananya","timestamp":"2026-10-01 10:00:00 +00:00","rating":{"value":5},"owner_answer":"Thanks"}]}`, cid, cid)
	}
	return `{}`
}

type fakeQueue struct {
	mu   sync.Mutex
	jobs []jobs.EnqueueInput
}

func (q *fakeQueue) Enqueue(_ context.Context, in jobs.EnqueueInput) (jobs.Job, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.jobs = append(q.jobs, in)
	return jobs.Job{}, nil
}

type fakeLLM struct {
	mu    sync.Mutex
	calls int
	text  func(openrouter.Request) string
}

func (l *fakeLLM) Stream(_ context.Context, req openrouter.Request, _ openrouter.Handler) (openrouter.Result, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls++
	if req.ReasoningEffort != "none" {
		// A thinking model spends the whole budget before answering.
		return openrouter.Result{FinishReason: "length", Usage: openrouter.Usage{CompletionTokens: req.MaxTokens, CostUSD: 0.004}}, nil
	}
	return openrouter.Result{Text: l.text(req), Usage: openrouter.Usage{CostUSD: 0.0004}}, nil
}

type fakeMeter struct {
	guard spendtest.Guard
	keys  []string
}

func (m *fakeMeter) HoldWithIdempotencyKey(ctx context.Context, org string, ceiling float64, key string) (spend.Hold, error) {
	m.keys = append(m.keys, key)
	return m.guard.Hold(ctx, spend.Request{OrganizationID: org, Feature: spend.LocalVisibility, EstimateUSD: ceiling, IdempotencyKey: key})
}

type fixture struct {
	svc     *Service
	dfs     *fakeDataForSEO
	guard   *spendtest.Guard
	queue   *fakeQueue
	llm     *fakeLLM
	meter   *fakeMeter
	org     string
	project string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("TEST_DATABASE_URL required")
		}
		t.Skip("TEST_DATABASE_URL required")
	}
	ctx := t.Context()
	db, err := pgdb.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	if err := database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	f := fixture{dfs: &fakeDataForSEO{posts: map[string]int{}, bodies: map[string]map[string]any{}}, guard: &spendtest.Guard{}, queue: &fakeQueue{},
		meter: &fakeMeter{}, org: "org-" + rand.Text(), project: "project-" + rand.Text()}
	f.llm = &fakeLLM{text: func(req openrouter.Request) string {
		if req.JSONObject {
			return `{"themes":[{"name":"Waiting time","neg":["y0","a0"]}]}`
		}
		return "Hi Rohan, sorry about the wait."
	}}
	server := httptest.NewServer(f.dfs)
	t.Cleanup(server.Close)
	client, err := dataforseo.NewClient(dataforseo.Options{BaseURL: server.URL, APIKey: base64.StdEncoding.EncodeToString([]byte("fixture-user:fixture-pass")),
		Recorder: dataforseo.NewUsageRecorder(db), Guard: f.guard})
	if err != nil {
		t.Fatal(err)
	}
	f.svc = &Service{Store: Store{DB: db}, Client: client, Queue: f.queue, LLM: f.llm, Meter: f.meter,
		Sleep: func(context.Context, time.Duration) error { return nil }}
	seed(t, db, f.org, f.project)
	if err := f.svc.Store.SaveBusiness(ctx, f.org, f.project, Business{CID: "100", Title: "Smile Studio", Category: "Dentist",
		Latitude: 18.5362, Longitude: 73.893, SearchTerm: "dentist", LanguageCode: "en"}); err != nil {
		t.Fatal(err)
	}
	return f
}

func seed(t *testing.T, db *pgxpool.Pool, org, project string) {
	t.Helper()
	t.Cleanup(func() {
		if _, err := db.Exec(context.Background(), `DELETE FROM organization WHERE id=$1`, org); err != nil {
			t.Error(err)
		}
	})
	if _, err := db.Exec(t.Context(), `INSERT INTO organization(id,name,slug,created_at) VALUES($1,'Local',$1,now())`, org); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(t.Context(), `INSERT INTO projects(id,organization_id,name) VALUES($1,$2,'Local')`, project, org); err != nil {
		t.Fatal(err)
	}
}

func decodeResult[T any](t *testing.T, run *Run) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(run.Result, &out); err != nil {
		t.Fatalf("decode %s result: %v (%s)", run.Kind, err, run.Result)
	}
	return out
}

func TestCheckPaysForEachTaskOnceAndCachesTheAnswer(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	// A double click: two starts at once make one run and one Maps search.
	var wg sync.WaitGroup
	runs := make([]*Run, 2)
	for i := range runs {
		wg.Go(func() {
			run, err := f.svc.Start(ctx, f.org, f.project, StartRequest{Kind: KindCheck})
			if err != nil {
				t.Error(err)
			}
			runs[i] = run
		})
	}
	wg.Wait()
	if runs[0] == nil || runs[1] == nil || runs[0].ID != runs[1].ID {
		t.Fatalf("runs = %+v, want one shared run", runs)
	}
	if got := f.dfs.postCount(mapsPostPath); got != 1 {
		t.Fatalf("Maps posts = %d, want 1", got)
	}
	// The job and the tick may both process the run; it is paid once.
	for range 2 {
		if err := f.svc.Process(ctx, runs[0].ID); err != nil {
			t.Fatal(err)
		}
	}
	if info, posts := f.dfs.postCount(infoPostPath), f.dfs.postCount(updatesPostPath); info != 4 || posts != 4 {
		t.Fatalf("profile posts = %d, update posts = %d, want 4 and 4", info, posts)
	}
	if held := len(f.guard.Held()); held != 9 {
		t.Fatalf("held %d requests, want 9 (1 map + 4 profiles + 4 post lists)", held)
	}
	run, err := f.svc.RunStatus(ctx, f.project, runs[0].ID)
	if err != nil || run.Status != StatusDone {
		t.Fatalf("run = %+v err = %v", run, err)
	}
	result := decodeResult[CheckResult](t, run)
	if result.YourRank != 4 || len(result.Rivals) != 3 || result.Rivals[0].Place.CID != "11" || len(result.Missing) != 0 {
		t.Fatalf("result = %+v", result)
	}
	if !strings.Contains(fmt.Sprint(titles(result.Fixes)), "Cosmetic dentist") {
		t.Fatalf("fixes = %v, want the category all three competitors use", titles(result.Fixes))
	}
	again, err := f.svc.Start(ctx, f.org, f.project, StartRequest{Kind: KindCheck})
	if err != nil || !again.Cached || again.ID != run.ID {
		t.Fatalf("second check = %+v err = %v, want the cached run", again, err)
	}
	if got := f.dfs.totalPosts(); got != 9 {
		t.Fatalf("posts after the cached answer = %d, want 9", got)
	}
}

func TestNoCreditsFailsTheRunBeforeAnythingIsBought(t *testing.T) {
	f := newFixture(t)
	f.guard.Refuse = spend.ErrNoCredits
	_, err := f.svc.Start(t.Context(), f.org, f.project, StartRequest{Kind: KindGrid, Size: 3, SpacingKm: 1})
	if !errors.Is(err, spend.ErrNoCredits) {
		t.Fatalf("err = %v, want no credits", err)
	}
	if f.dfs.totalPosts() != 0 || len(f.queue.jobs) != 0 {
		t.Fatalf("posts = %d jobs = %d, want none", f.dfs.totalPosts(), len(f.queue.jobs))
	}
	runs, err := f.svc.Store.LatestRuns(t.Context(), f.project)
	if err != nil || runs[KindGrid].Status != StatusFailed || !strings.Contains(runs[KindGrid].Error, "credits") {
		t.Fatalf("grid run = %+v err = %v", runs[KindGrid], err)
	}
	// The failed run does not block the next attempt.
	f.guard.Refuse = nil
	if _, err := f.svc.Start(t.Context(), f.org, f.project, StartRequest{Kind: KindGrid, Size: 3, SpacingKm: 1}); err != nil {
		t.Fatal(err)
	}
	if got := f.dfs.postCount(mapsPostPath); got != 9 {
		t.Fatalf("grid posts = %d, want 9", got)
	}
}

func TestGridRejectsSizesItDoesNotPrice(t *testing.T) {
	f := newFixture(t)
	for _, req := range []StartRequest{{Kind: KindGrid, Size: 7, SpacingKm: 1}, {Kind: KindGrid, Size: 3, SpacingKm: 0.1}, {Kind: "everything"}} {
		if _, err := f.svc.Start(t.Context(), f.org, f.project, req); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("Start(%+v) err = %v, want invalid input", req, err)
		}
	}
	if f.dfs.totalPosts() != 0 {
		t.Fatal("an invalid request bought data")
	}
}

func TestGridRanksEveryPoint(t *testing.T) {
	f := newFixture(t)
	run, err := f.svc.Start(t.Context(), f.org, f.project, StartRequest{Kind: KindGrid, Size: 3, SpacingKm: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Process(t.Context(), run.ID); err != nil {
		t.Fatal(err)
	}
	run, _ = f.svc.RunStatus(t.Context(), f.project, run.ID)
	result := decodeResult[GridResult](t, run)
	if len(result.Points) != 9 || result.Points[0].Rank != 4 || result.Points[0].Top[0] != "Bright Smiles" {
		t.Fatalf("grid = %+v", result)
	}
}

func TestReviewsThemesAndReplyDrafts(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	check, err := f.svc.Start(ctx, f.org, f.project, StartRequest{Kind: KindCheck})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Process(ctx, check.ID); err != nil {
		t.Fatal(err)
	}
	run, err := f.svc.Start(ctx, f.org, f.project, StartRequest{Kind: KindReviews})
	if err != nil {
		t.Fatal(err)
	}
	if got := f.dfs.postCount(reviewsPostPath); got != 4 {
		t.Fatalf("review posts = %d, want yours and 3 competitors'", got)
	}
	if err := f.svc.Process(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
	run, _ = f.svc.RunStatus(ctx, f.project, run.ID)
	result := decodeResult[ReviewsResult](t, run)
	if *result.ReplyRate != 50 || len(result.Unanswered) != 1 || len(result.Themes) != 1 || result.Themes[0].Negative != 1 || len(result.Rivals) != 3 {
		t.Fatalf("reviews = %+v", result)
	}
	if f.llm.calls != 1 {
		t.Fatalf("model calls = %d, want 1 for the themes", f.llm.calls)
	}
	reviewID := result.Unanswered[0].ID
	reply, err := f.svc.DraftReply(ctx, f.org, f.project, reviewID, false, "click-0001")
	if err != nil || reply == "" {
		t.Fatalf("reply = %q err = %v", reply, err)
	}
	// Reopening shows the saved draft; it is not written (or paid) again.
	if again, err := f.svc.DraftReply(ctx, f.org, f.project, reviewID, false, "click-0002"); err != nil || again != reply || f.llm.calls != 2 {
		t.Fatalf("second draft = %q err = %v calls = %d", again, err, f.llm.calls)
	}
	if _, err := f.svc.DraftReply(ctx, f.org, f.project, "someone-elses-review", false, "click-0003"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("draft for an unknown review err = %v", err)
	}
	if len(f.meter.guard.Settled()) != 2 {
		t.Fatalf("AI charges = %v, want themes and one reply", f.meter.guard.Settled())
	}
	run, _ = f.svc.RunStatus(ctx, f.project, run.ID)
	if got := decodeResult[ReviewsResult](t, run).Unanswered[0].ReplyDraft; got != reply {
		t.Fatalf("saved draft = %q, want %q", got, reply)
	}
}

func TestWatchNeedsACheckAndSnapshotsWhenTurnedOn(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	if _, err := f.svc.SetWatch(ctx, f.org, f.project, true); !errors.Is(err, ErrNeedsCheck) {
		t.Fatalf("err = %v, want needs check", err)
	}
	check, _ := f.svc.Start(ctx, f.org, f.project, StartRequest{Kind: KindCheck})
	if err := f.svc.Process(ctx, check.ID); err != nil {
		t.Fatal(err)
	}
	before := f.dfs.totalPosts()
	state, err := f.svc.SetWatch(ctx, f.org, f.project, true)
	if err != nil || !state.Enabled || len(state.Competitors) != 3 || state.CreditsPerWeek != 30 {
		t.Fatalf("watch = %+v err = %v", state, err)
	}
	if got := f.dfs.totalPosts() - before; got != 6 {
		t.Fatalf("first watch posted %d tasks, want 6 (profile and posts of 3 competitors)", got)
	}
	// Turning it on again does not buy another snapshot.
	if _, err := f.svc.SetWatch(ctx, f.org, f.project, true); err != nil || f.dfs.totalPosts()-before != 6 {
		t.Fatalf("second turn-on err = %v posts = %d", err, f.dfs.totalPosts()-before)
	}
	due, err := f.svc.Store.claimDueWatches(ctx, time.Now(), 50)
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range due {
		if w.ProjectID == f.project {
			t.Fatal("a watch turned on now is due a week later, not today")
		}
	}
}

func TestARunWithMissingPartsIsNotServedFromCache(t *testing.T) {
	f := newFixture(t)
	ctx := t.Context()
	f.llm.text = func(openrouter.Request) string { return `{"themes":[` } // cut off
	check, _ := f.svc.Start(ctx, f.org, f.project, StartRequest{Kind: KindCheck})
	if err := f.svc.Process(ctx, check.ID); err != nil {
		t.Fatal(err)
	}
	run, err := f.svc.Start(ctx, f.org, f.project, StartRequest{Kind: KindReviews})
	if err != nil {
		t.Fatal(err)
	}
	if err := f.svc.Process(ctx, run.ID); err != nil {
		t.Fatal(err)
	}
	run, _ = f.svc.RunStatus(ctx, f.project, run.ID)
	if missing := decodeResult[ReviewsResult](t, run).Missing; len(missing) != 1 || missing[0] != "review themes" {
		t.Fatalf("missing = %v, want the themes", missing)
	}
	if settled := f.meter.guard.Settled(); len(settled) != 1 || settled[0] != 0 {
		t.Fatalf("AI charges = %v, want the cut-off answer free", settled)
	}
	again, err := f.svc.Start(ctx, f.org, f.project, StartRequest{Kind: KindReviews})
	if err != nil || again.Cached || again.ID == run.ID {
		t.Fatalf("second reviews run = %+v err = %v, want a fresh run", again, err)
	}
}
