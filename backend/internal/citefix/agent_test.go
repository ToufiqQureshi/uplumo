package citefix

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/toufiqqureshi/uplumo/backend/internal/audit"
	"github.com/toufiqqureshi/uplumo/backend/internal/billing"
	"github.com/toufiqqureshi/uplumo/backend/internal/gsc"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/openrouter"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

const analystJSON = `{"summary":"The page never answers when to visit.","targetQuestion":"When is the best time to visit Goa?","recommendations":[{"title":"Answer first","detail":"Open with the months.","check":"answer-first"},{"title":"Made up","detail":"x","check":"not-a-check"}],"answerFirst":"November to February is the best time to visit Goa.","faq":[{"question":"Is Goa safe in monsoon?","answer":"Beaches close from June to September."}],"titleTag":"Best time to visit Goa","metaDescription":"When to go.","schemaJsonLd":{"@type":"Article","headline":"Goa"},"verify":["average hotel price"]}`

var writerJSON = `{"draftMarkdown":"` + strings.Repeat("November to February is the best time to visit Goa. ", 6) + `"}`

type fakeReply struct {
	text string
	cost float64
	err  error
}

// fakeLLM answers by model and records every request. When gate is set, a
// call waits on it, so a test can hold a run open.
type fakeLLM struct {
	mu       sync.Mutex
	requests []openrouter.Request
	replies  map[string]fakeReply
	gate     chan struct{}
	started  chan struct{}
}

func (f *fakeLLM) Stream(_ context.Context, req openrouter.Request, _ openrouter.Handler) (openrouter.Result, error) {
	f.mu.Lock()
	f.requests = append(f.requests, req)
	reply := f.replies[req.Model]
	f.mu.Unlock()
	if f.started != nil {
		f.started <- struct{}{}
	}
	if f.gate != nil {
		<-f.gate
	}
	return openrouter.Result{Text: reply.text, Usage: openrouter.Usage{CostUSD: reply.cost}}, reply.err
}

func (f *fakeLLM) calls() []openrouter.Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]openrouter.Request(nil), f.requests...)
}

type settlement struct{ provider, customer float64 }

type fakeMeter struct {
	mu      sync.Mutex
	holds   []float64
	keys    []string
	settled []settlement
	err     error
}

type fakeHold struct{ meter *fakeMeter }

func (h fakeHold) Settle(context.Context, float64) error {
	return errors.New("the agent must record provider cost separately")
}

func (h fakeHold) SettleProviderCost(_ context.Context, provider, customer float64, _ string) error {
	h.meter.mu.Lock()
	defer h.meter.mu.Unlock()
	h.meter.settled = append(h.meter.settled, settlement{provider, customer})
	return nil
}

func (m *fakeMeter) HoldWithIdempotencyKey(_ context.Context, _ string, ceiling float64, key string) (spend.Hold, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	m.holds = append(m.holds, ceiling)
	m.keys = append(m.keys, key)
	return fakeHold{m}, nil
}

type fakePlans struct{ allowed bool }

func (p fakePlans) HasPaidPlan(context.Context, string) (bool, error) { return p.allowed, nil }

type fakeQueries struct{ rows []gsc.SearchRow }

func (q fakeQueries) GetPageQueryRows(context.Context, string, string, string, string, string) ([]gsc.SearchRow, error) {
	return q.rows, nil
}

// agentFixture is a project whose latest scan holds one page served by a
// local site, and an agent with fakes around it.
type agentFixture struct {
	agent          *Agent
	llm            *fakeLLM
	meter          *fakeMeter
	pool           *pgxpool.Pool
	organizationID string
	projectID      string
	pageURL        string
	pageText       atomic.Value
	fetches        atomic.Int32
}

func newAgentFixture(t *testing.T) *agentFixture {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	f := &agentFixture{
		llm: &fakeLLM{replies: map[string]fakeReply{
			AnalystModel: {text: analystJSON, cost: 0.002},
			WriterModel:  {text: writerJSON, cost: 0.02},
		}},
		meter: &fakeMeter{},
		pool:  testPool(ctx, t),
	}
	f.pageText.Store("<h1>Goa</h1><p>Goa is a state in India with beaches.</p>")
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.fetches.Add(1)
		if r.URL.Path != "/guide" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, "<html><head><title>Goa guide</title></head><body>%s</body></html>", f.pageText.Load())
	}))
	t.Cleanup(site.Close)
	f.pageURL = site.URL + "/guide"
	f.organizationID, f.projectID = seedProject(ctx, t, f.pool)
	crawl(ctx, t, audit.NewRepository(f.pool), f.projectID, true, []audit.CrawledPageResult{crawledPage(f.pageURL, 600, audit.AIPageSignals{})}, nil)
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.WithoutCancel(ctx), `DELETE FROM go_citefix_agent_runs WHERE project_id = $1`, f.projectID)
	})
	f.agent = &Agent{
		Store: Store{DB: f.pool}, LLM: f.llm, Meter: f.meter, Plans: fakePlans{allowed: true},
		Queries: fakeQueries{rows: []gsc.SearchRow{{Keys: []string{"best time to visit goa"}, Impressions: 900, Clicks: 12, Position: 7.2}}},
		HTTP:    site.Client(),
	}
	return f
}

func (f *agentFixture) run(t *testing.T, mode, key string) (AgentResult, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return f.agent.Run(ctx, f.organizationID, f.projectID, AgentInput{URL: f.pageURL, Mode: mode, IdempotencyKey: key})
}

func TestAgentQuickMakesOneCappedCallAndChargesItsCost(t *testing.T) {
	f := newAgentFixture(t)
	result, err := f.run(t, ModeQuick, "key-1")
	if err != nil {
		t.Fatal(err)
	}
	calls := f.llm.calls()
	if len(calls) != 1 || calls[0].Model != AnalystModel {
		t.Fatalf("calls = %d (%v), want one analyst call", len(calls), calls)
	}
	req := calls[0]
	if !req.JSONObject || req.Provider == nil || req.Provider.DataCollection != "deny" || *req.Provider.MaxPrice != analystPrice || req.MaxTokens != analystMaxTokens {
		t.Fatalf("analyst request is not price-capped: %+v provider %+v", req, req.Provider)
	}
	// The prompt carries the page, its failed checks and Search Console.
	prompt := req.Messages[1].Content
	for _, want := range []string{"Goa is a state in India", "The answer is not in the first paragraph", `"best time to visit goa": 900 impressions`} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt is missing %q", want)
		}
	}
	if len(f.meter.holds) != 1 || f.meter.holds[0] > creditsToUSD(QuickMaxCredits) || f.meter.keys[0] != "citefix:key-1" {
		t.Fatalf("holds = %v keys = %v, want one hold within the 25-credit cap", f.meter.holds, f.meter.keys)
	}
	if len(f.meter.settled) != 1 || f.meter.settled[0] != (settlement{0.002, 0.002}) {
		t.Fatalf("settled = %v, want the reported $0.002 charged once", f.meter.settled)
	}
	if result.Credits != billing.CreditsForUSD(0.002) || result.Cached || result.Partial {
		t.Fatalf("result credits %d cached %v partial %v", result.Credits, result.Cached, result.Partial)
	}
	// Model output is checked: an unknown check is dropped, the schema object
	// becomes text, and the draft starts with the new opening.
	if result.Recommendations[1].Check != "" || !strings.Contains(result.SchemaJSONLD, `"@type": "Article"`) || !strings.HasPrefix(result.Draft, "November to February") {
		t.Fatalf("output not validated: %+v", result.AgentOutput)
	}
	saved, err := f.agent.Results(context.Background(), f.organizationID, f.projectID)
	if err != nil || len(saved) != 1 || saved[0].Summary != result.Summary {
		t.Fatalf("saved results = %+v, %v", saved, err)
	}
}

func TestAgentDeepCallsAnalystThenWriterUnderOneHold(t *testing.T) {
	f := newAgentFixture(t)
	result, err := f.run(t, ModeDeep, "key-1")
	if err != nil {
		t.Fatal(err)
	}
	calls := f.llm.calls()
	if len(calls) != 2 || calls[0].Model != AnalystModel || calls[1].Model != WriterModel || *calls[1].Provider.MaxPrice != writerPrice {
		t.Fatalf("calls = %+v, want analyst then price-capped writer", calls)
	}
	if !strings.Contains(calls[1].Messages[1].Content, "Answer first") {
		t.Fatal("the writer did not get the analyst's plan")
	}
	if len(f.meter.holds) != 1 || f.meter.holds[0] > creditsToUSD(DeepMaxCredits) {
		t.Fatalf("holds = %v, want one within the 60-credit cap", f.meter.holds)
	}
	if got := f.meter.settled[0]; got.provider != 0.022 || got.customer != 0.022 {
		t.Fatalf("settled = %+v, want both calls ($0.022)", got)
	}
	if !strings.HasPrefix(result.Draft, "November to February is the best time") || result.Partial {
		t.Fatalf("draft = %q partial %v, want the writer's rewrite", result.Draft, result.Partial)
	}
}

func TestAgentReplaysTheSameKeyAndCachesTheSameContent(t *testing.T) {
	f := newAgentFixture(t)
	first, err := f.run(t, ModeQuick, "key-1")
	if err != nil {
		t.Fatal(err)
	}
	// A retried request (same key): its saved answer, no call, no hold.
	again, err := f.run(t, ModeQuick, "key-1")
	if err != nil || again.Credits != first.Credits || again.Cached {
		t.Fatalf("replay = %+v, %v", again, err)
	}
	// A new click on the unchanged page: the cached answer for 0 credits.
	cached, err := f.run(t, ModeQuick, "key-2")
	if err != nil || !cached.Cached || cached.Credits != 0 || cached.Summary != first.Summary {
		t.Fatalf("cached = %+v, %v", cached, err)
	}
	if calls := len(f.llm.calls()); calls != 1 || len(f.meter.holds) != 1 {
		t.Fatalf("calls %d holds %d, want 1 and 1", calls, len(f.meter.holds))
	}
	// Deep is a different answer, and a changed page is new work.
	if _, err := f.run(t, ModeDeep, "key-3"); err != nil {
		t.Fatal(err)
	}
	f.pageText.Store("<h1>Goa</h1><p>Goa is a state in India with beaches and forts.</p>")
	if r, err := f.run(t, ModeQuick, "key-4"); err != nil || r.Cached {
		t.Fatalf("changed page = %+v, %v; want a new run", r, err)
	}
	if calls := len(f.llm.calls()); calls != 4 {
		t.Fatalf("calls = %d, want 4 (quick, deep x2, quick on new content)", calls)
	}
}

func TestAgentRefusesBeforeSpending(t *testing.T) {
	f := newAgentFixture(t)
	f.agent.Plans = fakePlans{allowed: false}
	if _, err := f.run(t, ModeQuick, "key-1"); !errors.Is(err, ErrAgentPlanRequired) {
		t.Fatalf("free plan: err = %v", err)
	}
	f.agent.Plans = fakePlans{allowed: true}
	ctx := context.Background()
	// A page outside the latest scan, and the same page asked for by another
	// workspace, are never fetched.
	if _, err := f.agent.Run(ctx, f.organizationID, f.projectID, AgentInput{URL: f.pageURL + "-other", Mode: ModeQuick, IdempotencyKey: "key-2"}); !errors.Is(err, ErrAgentPageNotFound) {
		t.Fatalf("unknown page: err = %v", err)
	}
	if _, err := f.agent.Run(ctx, "org-someone-else", f.projectID, AgentInput{URL: f.pageURL, Mode: ModeQuick, IdempotencyKey: "key-3"}); !errors.Is(err, ErrAgentPageNotFound) {
		t.Fatalf("foreign workspace: err = %v", err)
	}
	for name, input := range map[string]AgentInput{
		"mode":   {URL: f.pageURL, Mode: "max", IdempotencyKey: "k"},
		"no key": {URL: f.pageURL, Mode: ModeQuick},
		"scheme": {URL: "file:///etc/passwd", Mode: ModeQuick, IdempotencyKey: "k"},
	} {
		if _, err := f.agent.Run(ctx, f.organizationID, f.projectID, input); err == nil {
			t.Errorf("%s: accepted", name)
		} else if _, ok := errors.AsType[*AgentValidationError](err); !ok {
			t.Errorf("%s: err = %v, want a validation error", name, err)
		}
	}
	f.meter.err = spend.ErrNoCredits
	if _, err := f.run(t, ModeQuick, "key-4"); !errors.Is(err, spend.ErrNoCredits) {
		t.Fatalf("no credits: err = %v", err)
	}
	if calls := len(f.llm.calls()); calls != 0 || f.fetches.Load() != 1 {
		t.Fatalf("calls %d fetches %d, want no model call and one fetch (the no-credit run)", calls, f.fetches.Load())
	}
}

func TestAgentDoesNotChargeForUnusableAnswers(t *testing.T) {
	f := newAgentFixture(t)
	f.llm.replies[AnalystModel] = fakeReply{text: "Sorry, I cannot help with that.", cost: 0.001}
	if _, err := f.run(t, ModeQuick, "key-1"); !errors.Is(err, ErrAgentFailed) {
		t.Fatalf("err = %v, want ErrAgentFailed", err)
	}
	if len(f.meter.settled) != 1 || f.meter.settled[0] != (settlement{0.001, 0}) {
		t.Fatalf("settled = %v, want provider cost recorded and the customer charged 0", f.meter.settled)
	}
	// Retrying the same request does not buy it again.
	if _, err := f.run(t, ModeQuick, "key-1"); !errors.Is(err, ErrAgentFailed) || len(f.llm.calls()) != 1 {
		t.Fatalf("retry err = %v calls = %d", err, len(f.llm.calls()))
	}

	// Deep: a failed rewrite still delivers the analysis, charging only it.
	f.llm.replies[AnalystModel] = fakeReply{text: analystJSON, cost: 0.002}
	f.llm.replies[WriterModel] = fakeReply{text: `{"draftMarkdown":"too short"}`, cost: 0.02}
	result, err := f.run(t, ModeDeep, "key-2")
	if err != nil || !result.Partial || result.Credits != billing.CreditsForUSD(0.002) {
		t.Fatalf("partial = %+v, %v", result, err)
	}
	if got := f.meter.settled[1]; got != (settlement{0.022, 0.002}) {
		t.Fatalf("settled = %+v, want provider $0.022, customer $0.002", got)
	}
}

func TestAgentNeverChargesAboveItsHold(t *testing.T) {
	f := newAgentFixture(t)
	f.llm.replies[AnalystModel] = fakeReply{text: analystJSON, cost: 5}
	result, err := f.run(t, ModeQuick, "key-1")
	if err != nil {
		t.Fatal(err)
	}
	hold := f.meter.holds[0]
	if got := f.meter.settled[0]; got.provider != 5 || got.customer != hold {
		t.Fatalf("settled = %+v, want the customer charge capped at the hold %v", got, hold)
	}
	if result.Credits > QuickMaxCredits {
		t.Fatalf("charged %d credits, above the %d cap", result.Credits, QuickMaxCredits)
	}
}

func TestAgentRunsOnePageOnceAtATime(t *testing.T) {
	f := newAgentFixture(t)
	f.llm.gate, f.llm.started = make(chan struct{}), make(chan struct{}, 2)
	first := make(chan error, 1)
	go func() {
		_, err := f.run(t, ModeQuick, "key-1")
		first <- err
	}()
	<-f.llm.started
	// A second click with a new key while the first is still with the model.
	if _, err := f.run(t, ModeQuick, "key-2"); !errors.Is(err, ErrAgentBusy) {
		t.Fatalf("second run err = %v, want ErrAgentBusy", err)
	}
	close(f.llm.gate)
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	if calls := len(f.llm.calls()); calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestAgentLimitsRunsPerProject(t *testing.T) {
	f := newAgentFixture(t)
	ctx := context.Background()
	for i := range MaxAgentPages {
		if _, err := f.pool.Exec(ctx, `INSERT INTO go_citefix_agent_runs (id, organization_id, project_id, url, mode, idempotency_key, status)
			VALUES ($1, $2, $3, $4, 'quick', $1, 'completed')`, fmt.Sprintf("run-%s-%d", f.projectID, i), f.organizationID, f.projectID, fmt.Sprintf("%s?p=%d", f.pageURL, i)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.run(t, ModeQuick, "key-1"); !errors.Is(err, ErrAgentRateLimited) {
		t.Fatalf("err = %v, want ErrAgentRateLimited", err)
	}
	if len(f.llm.calls()) != 0 || len(f.meter.holds) != 0 {
		t.Fatal("a rate-limited run reached the model or held credits")
	}
	// Free cached replays do not count towards the limit.
	if _, err := f.pool.Exec(ctx, `UPDATE go_citefix_agent_runs SET cached = true WHERE project_id = $1`, f.projectID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.run(t, ModeQuick, "key-2"); err != nil {
		t.Fatalf("after cached rows: %v", err)
	}
}

func TestAgentEmptyPageIsNotPaidFor(t *testing.T) {
	f := newAgentFixture(t)
	f.pageText.Store("<script>render()</script>")
	if _, err := f.run(t, ModeQuick, "key-1"); !errors.Is(err, ErrAgentPageFetch) {
		t.Fatalf("err = %v, want ErrAgentPageFetch", err)
	}
	if len(f.meter.holds) != 0 || len(f.llm.calls()) != 0 {
		t.Fatal("an empty page held credits or called the model")
	}
}

func TestAgentPageThatMovedIsNotPaidFor(t *testing.T) {
	f := newAgentFixture(t)
	f.pageURL += "-moved"
	if _, err := f.pool.Exec(context.Background(), `UPDATE go_audit_pages SET url = $1 WHERE audit_id IN (SELECT id FROM go_audits WHERE project_id = $2)`, f.pageURL, f.projectID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.run(t, ModeQuick, "key-1"); !errors.Is(err, ErrAgentPageFetch) {
		t.Fatalf("err = %v, want ErrAgentPageFetch", err)
	}
	if len(f.meter.holds) != 0 || len(f.llm.calls()) != 0 {
		t.Fatal("an unreadable page held credits or called the model")
	}
}

func TestParseAnalystOutput(t *testing.T) {
	if _, err := parseAnalystOutput("```json\n" + analystJSON + "\n```"); err != nil {
		t.Fatalf("fenced JSON: %v", err)
	}
	for name, text := range map[string]string{
		"empty":         `{}`,
		"not JSON":      `Here are your fixes: ...`,
		"only an FAQ":   `{"faq":[{"question":"q","answer":"a"}]}`,
		"blank summary": `{"summary":"  ","recommendations":[{"title":" ","detail":"x"}]}`,
	} {
		if _, err := parseAnalystOutput(text); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	long := strings.Repeat("ab", maxSummaryChars)
	recs := strings.Repeat(`{"title":"t","detail":"d"},`, 12)
	out, err := parseAnalystOutput(`{"summary":"` + long + `","recommendations":[` + strings.TrimSuffix(recs, ",") + `],"schemaJsonLd":"{not json"}`)
	if err != nil {
		t.Fatal(err)
	}
	if len([]rune(out.Summary)) != maxSummaryChars || len(out.Recommendations) != maxRecommendation || out.SchemaJSONLD != "" {
		t.Fatalf("bounds not applied: %d chars, %d recs, schema %q", len([]rune(out.Summary)), len(out.Recommendations), out.SchemaJSONLD)
	}
	encoded, _ := json.Marshal(out)
	if strings.Contains(string(encoded), "null") {
		t.Fatalf("lists must be arrays: %s", encoded)
	}
}

func TestAgentCeilingsFitTheCaps(t *testing.T) {
	// The largest page text and the fullest evidence still fit each cap, so
	// no page is refused for being long and no hold exceeds what was shown.
	page := fetchedPage{Title: strings.Repeat("t", 300), MetaDescription: strings.Repeat("m", 300), Text: strings.Repeat("क", maxPageTextBytes/3)}
	target := agentTarget{URL: "https://x.example/" + strings.Repeat("p", 1900), Failed: []string{"author", "answer-first", "statistics", "sources", "quotes", "dated", "structured-data", "question-headings", "scannable", "not-thin", "unique"}}
	rows := make([]gsc.SearchRow, maxGSCQueries)
	for i := range rows {
		rows[i] = gsc.SearchRow{Keys: []string{strings.Repeat("q", 200)}, Impressions: 1e6}
	}
	proof := &Proof{Answers: 99, Domain: "x.example"}
	for range maxProofRivals {
		proof.Rivals = append(proof.Rivals, DomainCount{Domain: strings.Repeat("d", 60) + ".com", Count: 99})
	}
	for range maxProofMisses {
		proof.Misses = append(proof.Misses, Miss{Prompt: strings.Repeat("w", 400), Engine: "chatgpt", Cited: []string{"a.com", "b.com", "c.com"}})
	}
	evidence := agentEvidence{Queries: rows, Proof: proof}
	quick, err := holdCeiling(ModeQuick, analystRequest(ModeQuick, target, page, evidence), target, page)
	if err != nil || quick <= 0 {
		t.Fatalf("quick ceiling = %v, %v", quick, err)
	}
	deep, err := holdCeiling(ModeDeep, analystRequest(ModeDeep, target, page, evidence), target, page)
	if err != nil || deep <= quick {
		t.Fatalf("deep ceiling = %v, %v; want above quick %v", deep, err, quick)
	}
	// A page the fetch did not cut (a future change to the limits) is refused
	// rather than held above the cap.
	page.Text = strings.Repeat("क", 30_000)
	if _, err := holdCeiling(ModeDeep, analystRequest(ModeDeep, target, page, evidence), target, page); !errors.Is(err, ErrAgentFailed) {
		t.Fatalf("oversized deep page: err = %v, want a refusal", err)
	}
}
