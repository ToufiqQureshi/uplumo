package citefix

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/billing"
)

const pitchJSON = `{"subject":"Adding Shop to your list","body":"Hello, could you consider adding Shop to your list? [one result you can show]"}`

func newPitchFixture(t *testing.T) (*agentFixture, string) {
	t.Helper()
	f := newAgentFixture(t)
	f.llm.replies[AnalystModel] = fakeReply{text: pitchJSON, cost: 0.0002}
	ctx := t.Context()
	page := RivalPage{
		URL: "https://dir.example/top", Domain: "dir.example", Title: "Top firms",
		FetchedAt: time.Now(), ContentHash: "h", LinksRead: true,
		LinkedHosts: []string{"rival-one.example", "rival-two.example"},
	}
	if err := f.agent.Store.saveRivalCache(ctx, f.organizationID, f.projectID, page); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.WithoutCancel(ctx), `DELETE FROM go_citefix_pitches WHERE project_id = $1`, f.projectID)
	})
	return f, page.URL
}

func (f *agentFixture) pitch(t *testing.T, url, key string) (Pitch, error) {
	t.Helper()
	return f.agent.DraftPitch(t.Context(), f.organizationID, f.projectID, PitchInput{URL: url, IdempotencyKey: key})
}

func TestPitchChargesOneCappedCallThenServesItFree(t *testing.T) {
	f, url := newPitchFixture(t)
	got, err := f.pitch(t, url+"#:~:text=firms", "k1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Subject == "" || got.Credits != billing.CreditsForUSD(0.0002) || got.Cached {
		t.Fatalf("pitch = %+v", got)
	}
	if len(f.llm.calls()) != 1 || len(f.meter.holds) != 1 || f.meter.holds[0] > creditsToUSD(PitchMaxCredits) {
		t.Fatalf("calls %d, holds %v; want one call under one hold within the cap", len(f.llm.calls()), f.meter.holds)
	}
	if f.meter.settled[0] != (settlement{0.0002, 0.0002}) {
		t.Fatalf("settled %+v; want the provider cost charged", f.meter.settled)
	}
	// The same key replays; a new key for the same page is a free copy.
	again, err := f.pitch(t, url, "k1")
	if err != nil || again.Subject != got.Subject {
		t.Fatalf("replay = %+v, %v", again, err)
	}
	cached, err := f.pitch(t, url, "k2")
	if err != nil || !cached.Cached || cached.Credits != 0 {
		t.Fatalf("second key = %+v, %v; want a free cached copy", cached, err)
	}
	if len(f.llm.calls()) != 1 || len(f.meter.holds) != 1 {
		t.Fatalf("replays made %d calls and %d holds; want none", len(f.llm.calls())-1, len(f.meter.holds)-1)
	}
}

func TestPitchRefusesBeforeSpending(t *testing.T) {
	f, url := newPitchFixture(t)
	if _, err := f.pitch(t, "https://unknown.example/", "k1"); !errors.Is(err, ErrPitchUnknownPage) {
		t.Errorf("unknown page: %v", err)
	}
	if _, err := f.pitch(t, "", "k2"); err == nil {
		t.Error("empty URL accepted")
	}
	f.agent.Plans = fakePlans{allowed: false}
	if _, err := f.pitch(t, url, "k3"); !errors.Is(err, ErrAgentPlanRequired) {
		t.Errorf("free plan: %v", err)
	}
	if len(f.llm.calls()) != 0 || len(f.meter.holds) != 0 {
		t.Fatalf("refusals spent: %d calls, %d holds", len(f.llm.calls()), len(f.meter.holds))
	}
}

func TestPitchDoesNotChargeForAnUnusableAnswer(t *testing.T) {
	f, url := newPitchFixture(t)
	f.llm.replies[AnalystModel] = fakeReply{text: `{"subject":""}`, cost: 0.0002}
	if _, err := f.pitch(t, url, "k1"); !errors.Is(err, ErrAgentFailed) {
		t.Fatalf("err = %v, want ErrAgentFailed", err)
	}
	if f.meter.settled[0] != (settlement{0.0002, 0}) {
		t.Fatalf("settled %+v; the customer must pay nothing", f.meter.settled)
	}
	// The failed draft is not cached: a new key may try again.
	f.llm.replies[AnalystModel] = fakeReply{text: pitchJSON, cost: 0.0002}
	if got, err := f.pitch(t, url, "k2"); err != nil || got.Cached {
		t.Fatalf("retry = %+v, %v", got, err)
	}
}

func TestPitchConcurrentClicksMakeOneCall(t *testing.T) {
	f, url := newPitchFixture(t)
	f.llm.gate, f.llm.started = make(chan struct{}), make(chan struct{}, 4)
	var wg sync.WaitGroup
	errs := make([]error, 3)
	for i := range errs {
		wg.Go(func() { _, errs[i] = f.pitch(t, url, fmt.Sprintf("k%d", i)) })
	}
	<-f.llm.started
	time.Sleep(200 * time.Millisecond)
	close(f.llm.gate)
	wg.Wait()
	busy := 0
	for _, err := range errs {
		if errors.Is(err, ErrPitchBusy) {
			busy++
		} else if err != nil {
			t.Errorf("unexpected error %v", err)
		}
	}
	if len(f.llm.calls()) != 1 || busy != 2 {
		t.Fatalf("calls %d, busy %d; want one call and two refused", len(f.llm.calls()), busy)
	}
}

func TestPitchNeverChargesAboveItsHold(t *testing.T) {
	f, url := newPitchFixture(t)
	f.llm.replies[AnalystModel] = fakeReply{text: pitchJSON, cost: 5}
	got, err := f.pitch(t, url, "k1")
	if err != nil {
		t.Fatal(err)
	}
	if f.meter.settled[0].customer != f.meter.holds[0] || got.Credits > PitchMaxCredits {
		t.Fatalf("charged %v of hold %v (%d credits); want the hold at most", f.meter.settled[0].customer, f.meter.holds[0], got.Credits)
	}
}
