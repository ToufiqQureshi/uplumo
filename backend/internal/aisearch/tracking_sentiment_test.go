package aisearch

import (
	"context"
	"crypto/rand"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/contenteditor"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/openrouter"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

type sentimentLLM struct {
	calls atomic.Int32
	reply string
	req   openrouter.Request
}

func (l *sentimentLLM) Stream(_ context.Context, req openrouter.Request, _ openrouter.Handler) (openrouter.Result, error) {
	l.calls.Add(1)
	l.req = req
	return openrouter.Result{Text: l.reply, Usage: openrouter.Usage{CostUSD: 0.002}}, nil
}

type sentimentMeter struct {
	holds    int
	ceiling  float64
	charged  float64
	provider float64
	noCredit bool
}

func (m *sentimentMeter) HoldWithIdempotencyKey(_ context.Context, _ string, ceiling float64, _ string) (spend.Hold, error) {
	if m.noCredit {
		return nil, spend.ErrNoCredits
	}
	m.holds++
	m.ceiling = ceiling
	return sentimentHold{m}, nil
}

type sentimentHold struct{ m *sentimentMeter }

func (h sentimentHold) Settle(_ context.Context, usd float64) error { h.m.charged += usd; return nil }
func (h sentimentHold) SettleProviderCost(ctx context.Context, provider, customer float64, _ string) error {
	h.m.provider += provider
	return h.Settle(ctx, customer)
}

type allowPlans bool

func (p allowPlans) HasPaidPlan(context.Context, string) (bool, error) { return bool(p), nil }

func TestSentimentPricesMatchTheWriter(t *testing.T) {
	if SentimentModel != contenteditor.WriterModel || MaxSentimentCredits != contenteditor.MaxWriterCreditsPerAction {
		t.Fatalf("sentiment model/ceiling %q/%d drifted from the writer's %q/%d", SentimentModel, MaxSentimentCredits, contenteditor.WriterModel, contenteditor.MaxWriterCreditsPerAction)
	}
}

func TestBrandExcerptAndParsing(t *testing.T) {
	if got := brandExcerpt("Try Brew Lab for coffee.", "Sunrise Café", "sunrisecafe.in"); got != "" {
		t.Fatalf("excerpt without the brand = %q, want none (no model call)", got)
	}
	if got := brandExcerpt("Locals like sunrisecafe.in for views.", "Sunrise Café", "www.sunrisecafe.in"); !strings.Contains(got, "sunrisecafe.in") {
		t.Fatalf("domain mention not found: %q", got)
	}
	long := strings.Repeat("मौसम अच्छा है। ", 400) + "Sunrise Café is slow on weekends."
	if got := brandExcerpt(long, "sunrise café", ""); !strings.Contains(got, "Sunrise Café is slow") || len([]rune(got)) > sentimentExcerptRunes {
		t.Fatalf("excerpt = %d runes, must contain the mention and stay bounded", len([]rune(got)))
	}
	batch := []sentimentRow{{id: "a1", excerpt: "Sunrise Café has slow service at night"}}
	got, ok := parseSentiment("```json\n"+`[{"id":"a1","sentiment":"negative","quote":"slow service","reason":"complaint"},{"id":"a1","sentiment":"positive"},{"id":"zz","sentiment":"negative"},{"id":"a2","sentiment":"angry"}]`+"\n```", batch)
	if !ok || len(got) != 1 || got[0].Sentiment != "negative" || got[0].Quote != "slow service" {
		t.Fatalf("parse = %+v, %v; want one negative verdict, unknown ids and duplicates dropped", got, ok)
	}
	got, _ = parseSentiment(`[{"id":"a1","sentiment":"negative","quote":"terrible food"}]`, batch)
	if got[0].Quote != "" {
		t.Fatalf("an invented quote %q must not be shown as the AI's words", got[0].Quote)
	}
	if _, ok := parseSentiment("I think it's negative", batch); ok {
		t.Fatal("prose instead of JSON must be unusable")
	}
}

func TestSentimentClassifiesOnceChargesOnceAndSkipsAnswersWithoutTheBrand(t *testing.T) {
	db := trackingDB(t)
	ctx := t.Context()
	org, otherOrg, project, run := "org-"+rand.Text(), "org-"+rand.Text(), "project-"+rand.Text(), "run-"+rand.Text()
	for _, o := range []string{org, otherOrg} {
		if _, err := db.Exec(ctx, `INSERT INTO organization(id,name,slug,created_at) VALUES($1,'S',$1,now())`, o); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		if _, err := db.Exec(context.Background(), `DELETE FROM organization WHERE id IN ($1,$2)`, org, otherOrg); err != nil {
			t.Error(err)
		}
	})
	cfg := `{"brand":"Sunrise Café","domain":"sunrisecafe.in"}`
	for _, q := range []string{
		`INSERT INTO projects(id,organization_id,name) VALUES($1,$2,'S')`,
		`INSERT INTO go_ai_trackers(project_id,organization_id,configuration) VALUES($1,$2,'` + cfg + `')`,
	} {
		if _, err := db.Exec(ctx, q, project, org); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(ctx, `INSERT INTO go_ai_tracking_runs(id,project_id,organization_id,configuration,scope_key,trigger,status,request_key) VALUES($1,$2,$3,'`+cfg+`','s','manual','completed','k')`, run, project, org); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO go_ai_tracking_answers(id,run_id,prompt,topic,engine,status,answer) VALUES
		('a1',$1,'p1','t','chat_gpt','completed','Sunrise Café has great lake views.'),
		('a2',$1,'p2','t','gemini','completed','Brew Lab is the best café here.'),
		('a3',$1,'p3','t','claude','failed','')`, run); err != nil {
		t.Fatal(err)
	}
	llm := &sentimentLLM{reply: `[{"id":"a1","sentiment":"positive","quote":"great lake views","reason":"praises the view"}]`}
	meter := &sentimentMeter{}
	s := &Sentiment{DB: db, LLM: llm, Meter: meter, Plans: allowPlans(true)}

	if _, err := s.Classify(ctx, otherOrg, project, run, "key-other-1"); err == nil || llm.calls.Load() != 0 || meter.holds != 0 {
		t.Fatalf("another workspace classified this run: err=%v calls=%d", err, llm.calls.Load())
	}
	if _, err := (&Sentiment{DB: db, LLM: llm, Meter: meter, Plans: allowPlans(false)}).Classify(ctx, org, project, run, "key-plan-1"); err == nil || llm.calls.Load() != 0 {
		t.Fatalf("a plan without AI Visibility reached the model: err=%v", err)
	}
	if _, err := (&Sentiment{DB: db, LLM: llm, Meter: &sentimentMeter{noCredit: true}, Plans: allowPlans(true)}).Classify(ctx, org, project, run, "key-poor-1"); !errors.Is(err, errSentimentNoCredits) || llm.calls.Load() != 0 {
		t.Fatalf("no credits: err=%v calls=%d, want refusal before the model", err, llm.calls.Load())
	}

	got, err := s.Classify(ctx, org, project, run, "key-first-1")
	if err != nil {
		t.Fatal(err)
	}
	// a2 never names the brand: the refused no-credit click already marked it
	// for free, so this click only classifies a1.
	if got.Classified != 1 || got.Remaining != 0 {
		t.Fatalf("result = %+v, want 1 classified, none remaining", got)
	}
	if llm.calls.Load() != 1 || meter.holds != 1 || meter.charged != 0.002 || meter.ceiling > float64(MaxSentimentCredits)/1280 {
		t.Fatalf("calls=%d holds=%d charged=%v ceiling=%v, want one call charged its real cost under the ceiling", llm.calls.Load(), meter.holds, meter.charged, meter.ceiling)
	}
	if body := llm.req.Messages[1].Content; strings.Contains(body, "Brew Lab") || !strings.Contains(body, "untrusted-data") {
		t.Fatalf("model input = %q; an answer without the brand must not be sent, and answers are fenced as data", body)
	}
	answers, err := (&Tracking{DB: db}).Answers(ctx, org, project, run)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"a1": "positive", "a2": "not_mentioned", "a3": ""}
	for _, a := range answers {
		if a.Sentiment != want[a.ID] {
			t.Errorf("answer %s sentiment = %q, want %q", a.ID, a.Sentiment, want[a.ID])
		}
		if a.ID == "a1" && a.SentimentQuote != "great lake views" {
			t.Errorf("quote = %q", a.SentimentQuote)
		}
	}
	// A second click finds nothing left: no model call, no hold.
	if got, err := s.Classify(ctx, org, project, run, "key-second-1"); err != nil || got.Classified != 0 || llm.calls.Load() != 1 || meter.holds != 1 {
		t.Fatalf("second click = %+v, %v; calls=%d holds=%d, want nothing bought", got, err, llm.calls.Load(), meter.holds)
	}
}
