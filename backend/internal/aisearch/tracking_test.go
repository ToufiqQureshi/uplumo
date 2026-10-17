package aisearch

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toufiqqureshi/uplumo/backend/internal/database"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/jobs"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/pgdb"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend/spendtest"
)

func validTracking() TrackingConfig {
	return TrackingConfig{Brand: "Acme", Domain: "acme.com", Competitors: []TrackedBrand{{"Other", "other.com"}}, Prompts: []TrackedPrompt{{Text: "Which service should I choose?", Topic: "General"}}, Engines: []string{"chat_gpt", "gemini"}, LocationCode: 2840, LanguageCode: "en", Interval: "weekly", MaxCreditsPerRun: 10}
}
func TestTrackingRejectsInvalidPaidInputs(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*TrackingConfig)
	}{
		{"private domain", func(c *TrackingConfig) { c.Domain = "localhost" }},
		{"unknown engine", func(c *TrackingConfig) { c.Engines = []string{"unknown"} }},
		{"invalid country", func(c *TrackingConfig) { c.LocationCode = 1 }},
		{"invalid language", func(c *TrackingConfig) { c.LanguageCode = "nonsense" }},
		{"empty prompts", func(c *TrackingConfig) { c.Prompts = nil }},
		{"duplicate whitespace", func(c *TrackingConfig) {
			c.Prompts = append(c.Prompts, TrackedPrompt{Text: "Which  service should I choose?"})
		}},
		{"oversize prompt", func(c *TrackingConfig) { c.Prompts[0].Text = strings.Repeat("🙂", 251) }},
		{"paused schedule", func(c *TrackingConfig) { c.Enabled = true; c.Prompts[0].Paused = true }},
		{"own competitor", func(c *TrackingConfig) { c.Competitors[0].Domain = c.Domain }},
		{"unsupported interval", func(c *TrackingConfig) { c.Interval = "hourly" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := validTracking()
			tc.change(&c)
			if c.validate() == nil {
				t.Fatal("invalid paid request accepted")
			}
		})
	}
}
func TestTrackingEstimateAndFairScope(t *testing.T) {
	c := validTracking()
	if err := c.validate(); err != nil {
		t.Fatal(err)
	}
	e := c.estimate()
	if e.Answers != 2 || e.Credits != 4 || e.MonthlyCredits != 18 {
		t.Fatalf("estimate=%+v", e)
	}
	same := validTracking()
	same.Prompts[0].Topic = "Renamed"
	same.Interval = "daily"
	same.Enabled = true
	same.MaxCreditsPerRun = 999
	if trackingScope(c) != trackingScope(same) {
		t.Fatal("topic or schedule changed comparison cohort")
	}
	same.Prompts[0].Text = "Another exact question"
	if trackingScope(c) == trackingScope(same) {
		t.Fatal("different prompts must not be compared")
	}
	c.Engines = []string{"claude"}
	if c.estimate().Credits != 640 {
		t.Fatal("model API ceiling missing")
	}
}
func TestTrackingEvidenceSeparatesCitationsFromMentions(t *testing.T) {
	text, citations, err := parseTrackingAnswer([]byte(`{"markdown":"Compare services [acme.com](https://acme.com).","sources":[{"url":"javascript:alert(1)"},{"url":"https://acme.com/guide"},{"url":"https://acme.com/guide"}]}`), "gemini")
	if err != nil || len(citations) != 1 {
		t.Fatalf("citations=%+v err=%v", citations, err)
	}
	if mentionsBrand(trackingProse(text), "Acme") {
		t.Fatal("citation-only brand counted as prose mention")
	}
	if !mentionsBrand(trackingProse("**Acme** is named here."), "Acme") {
		t.Fatal("genuine mention disappeared")
	}
	text, _, err = parseTrackingAnswer([]byte(`{"items":[{"type":"organic","markdown":"Acme"}]}`), "google_ai_overview")
	if err != nil || text != "" {
		t.Fatal("organic result treated as AI Overview")
	}
}
func trackingDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("TEST_DATABASE_URL required")
		}
		t.Skip("TEST_DATABASE_URL required")
	}
	db, err := pgdb.Open(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	if err = database.Migrate(t.Context(), db); err != nil {
		t.Fatal(err)
	}
	return db
}
func TestTrackingDurableCollectionAndTenantIsolation(t *testing.T) {
	db := trackingDB(t)
	ctx := t.Context()
	org, project := "org-"+rand.Text(), "project-"+rand.Text()
	if _, err := db.Exec(ctx, `INSERT INTO organization(id,name,slug,created_at) VALUES($1,'Tracking',$1,now())`, org); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec(context.Background(), `DELETE FROM organization WHERE id=$1`, org); err != nil {
			t.Error(err)
		}
	})
	if _, err := db.Exec(ctx, `INSERT INTO projects(id,organization_id,name) VALUES($1,$2,'Tracking')`, project, org); err != nil {
		t.Fatal(err)
	}
	var posts atomic.Int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/task_post") {
			posts.Add(1)
			var tasks []struct {
				Tag string `json:"tag"`
			}
			if err := json.NewDecoder(r.Body).Decode(&tasks); err != nil {
				t.Error(err)
			}
			entries := []any{}
			for _, task := range tasks {
				entries = append(entries, map[string]any{"id": task.Tag, "status_code": 20100, "cost": 0.0012, "path": strings.Split(strings.Trim(r.URL.Path, "/"), "/"), "data": map[string]string{"tag": task.Tag}})
			}
			if err := json.NewEncoder(w).Encode(map[string]any{"status_code": 20000, "tasks": entries}); err != nil {
				t.Error(err)
			}
		} else {
			if _, err := io.WriteString(w, `{"status_code":20000,"tasks":[{"status_code":20000,"result":[{"markdown":"Acme is a service.","sources":[{"url":"https://acme.com/guide"}]}]}]}`); err != nil {
				t.Error(err)
			}
		}
	}))
	t.Cleanup(api.Close)
	guard := &spendtest.Guard{}
	client, err := dataforseo.NewClient(dataforseo.Options{APIKey: base64.StdEncoding.EncodeToString([]byte("test:fixture")), BaseURL: api.URL, Guard: guard, Recorder: dataforseo.NewUsageRecorder(db)})
	if err != nil {
		t.Fatal(err)
	}
	svc := &Tracking{DB: db, Plans: trackingPaidPlans{}, Search: NewService(client, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))}
	c := validTracking()
	if err = svc.Save(ctx, org, project, c); err != nil {
		t.Fatal(err)
	}
	if _, err = svc.Start(ctx, org, project, "unapproved", false, 0); err == nil {
		t.Fatal("missing credit approval accepted")
	}
	id, err := svc.Start(ctx, org, project, "request-key-1", false, 10)
	if err != nil {
		t.Fatal(err)
	}
	again, err := svc.Start(ctx, org, project, "request-key-1", false, 10)
	if err != nil || again != id {
		t.Fatal("idempotent start changed run", err)
	}
	if _, err = svc.Start(ctx, org, project, "request-key-2", false, 10); !errors.Is(err, errTrackingBusy) {
		t.Fatal("overlapping run allowed", err)
	}
	raw, _ := json.Marshal(map[string]string{"runId": id})
	if err = svc.Execute(ctx, jobs.Job{Payload: raw, Attempts: 1}); err != nil {
		t.Fatal(err)
	}
	if posts.Load() != 2 || len(guard.Held()) != 1 || guard.Held()[0].OrganizationID != org {
		t.Fatalf("paid calls=%d holds=%+v", posts.Load(), guard.Held())
	}
	if err = svc.Execute(ctx, jobs.Job{Payload: raw, Attempts: 2}); err != nil || posts.Load() != 2 {
		t.Fatal("retry bought duplicate answers", err)
	}
	foreign, err := svc.Answers(ctx, "other-org", project, id)
	if err != nil || len(foreign) != 0 {
		t.Fatal("tenant leak", err)
	}
	state, err := svc.State(ctx, org, project)
	if err != nil || len(state.Runs) != 1 || state.Runs[0].Mentioned != 2 || state.Runs[0].Status != "completed" {
		t.Fatalf("state=%+v err=%v", state, err)
	}
	c.Prompts[0].Text = "Edited question"
	if err = svc.Save(ctx, org, project, c); err != nil {
		t.Fatal(err)
	}
	answers, err := svc.Answers(ctx, org, project, id)
	if err != nil || answers[0].Prompt == c.Prompts[0].Text {
		t.Fatal("editing rewrote past evidence", err)
	}
	guard.Refuse = spend.ErrNoCredits
	id, err = svc.Start(ctx, org, project, "no-credit-check", false, 10)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(map[string]string{"runId": id})
	if err = svc.Execute(ctx, jobs.Job{Payload: raw, Attempts: 1}); err != nil {
		t.Fatal(err)
	}
	if posts.Load() != 2 {
		t.Fatal("provider called after budget refused")
	}
	guard.Refuse = nil
	id, err = svc.Start(ctx, org, project, "interrupted-check", false, 10)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(ctx, `UPDATE go_ai_tracking_answers SET status=CASE WHEN engine='chat_gpt' THEN 'submitted' ELSE 'submitting' END,task_id='accepted-task' WHERE run_id=$1`, id); err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(map[string]string{"runId": id})
	if err = svc.Execute(ctx, jobs.Job{Payload: raw, Attempts: 2}); err != nil {
		t.Fatal(err)
	}
	state, err = svc.State(ctx, org, project)
	if err != nil || state.Runs[0].Status != "partial" || state.Runs[0].Completed != 1 || state.Runs[0].Failed != 1 || posts.Load() != 2 {
		t.Fatalf("uncertain POST replayed or accepted task lost: %+v %v", state, err)
	}
	id, err = svc.Start(ctx, org, project, "exhausted-check", false, 10)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(ctx, `UPDATE go_jobs SET state='failed',finished_at=now() WHERE queue=$1 AND idempotency_key=$2`, trackingQueue, id); err != nil {
		t.Fatal(err)
	}
	if err = svc.Tick(ctx); err != nil {
		t.Fatal(err)
	}
	state, err = svc.State(ctx, org, project)
	if err != nil || state.Runs[0].Status != "failed" || state.Runs[0].ErrorCode != "job_exhausted" {
		t.Fatal("exhausted job blocked future checks", state, err)
	}
}
func TestTrackingScheduleConcurrentClaim(t *testing.T) {
	db := trackingDB(t)
	org, project := "org-"+rand.Text(), "project-"+rand.Text()
	ctx := t.Context()
	if _, err := db.Exec(ctx, `INSERT INTO organization(id,name,slug,created_at) VALUES($1,'Schedule',$1,now())`, org); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, err := db.Exec(context.Background(), `DELETE FROM organization WHERE id=$1`, org)
		if err != nil {
			t.Error(err)
		}
	})
	if _, err := db.Exec(ctx, `INSERT INTO projects(id,organization_id,name) VALUES($1,$2,'Schedule')`, project, org); err != nil {
		t.Fatal(err)
	}
	svc := &Tracking{DB: db, Plans: trackingPaidPlans{}, Search: &Service{}}
	c := validTracking()
	c.Enabled = true
	if err := svc.Save(ctx, org, project, c); err != nil {
		t.Fatal(err)
	}
	forecast, err := svc.ScheduledCredits(ctx, org, time.Now().AddDate(0, 0, 8))
	if err != nil || forecast != 8 {
		t.Fatal("scheduled AI budget", forecast, err)
	}
	foreign, err := svc.ScheduledCredits(ctx, "foreign-org", time.Now().AddDate(0, 0, 8))
	if err != nil || foreign != 0 {
		t.Fatal("foreign schedule counted", foreign, err)
	}
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			_, err := svc.Start(ctx, org, project, "", true, 0)
			if err != nil && !errors.Is(err, errTrackingBusy) {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	state, err := svc.State(ctx, org, project)
	if err != nil || len(state.Runs) != 1 {
		t.Fatalf("duplicate scheduled runs: %+v %v", state, err)
	}
	c.Enabled = false
	if err := svc.Save(ctx, org, project, c); err != nil {
		t.Fatal(err)
	}
	state, err = svc.State(ctx, org, project)
	if err != nil || state.NextCheckAt != nil || state.Runs[0].Status != "failed" {
		t.Fatal("paused schedule kept queued work", err)
	}
}

type trackingPaidPlans struct{}

func (trackingPaidPlans) HasPaidPlan(context.Context, string) (bool, error) { return true, nil }
func TestTrackingMissingPlanGateRefusesAndMonthlyClamps(t *testing.T) {
	if err := (&Tracking{}).allowed(t.Context(), "org"); !errors.Is(err, errTrackingPlan) {
		t.Fatal("missing plan gate accepted", err)
	}
	next := nextTrackingCheck(time.Date(2026, 1, 31, 10, 0, 0, 0, time.UTC), "monthly")
	if next.Month() != time.February || next.Day() != 28 {
		t.Fatal("monthly check skipped February", next)
	}
}
