package keywords

import (
	"context"
	"errors"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend/spendtest"
)

func TestResearchCancelledDuringCacheReadDoesNotBuyTasks(t *testing.T) {
	f := newResearchFixture(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	cache := &spendtest.CancelOnReadCache{Cancel: cancel}
	f.svc.Cache = cache
	_, err := f.svc.Research(ctx, ResearchInput{OrganizationID: "org", ProjectID: f.project, Keywords: []string{"seed"}, LocationCode: 2840, LanguageCode: "en", ResultLimit: 150, Mode: "suggestions"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled research=%v", err)
	}
	_, _, _ = f.svc.flights.Do(cache.Key, func() (any, error) { return nil, nil })
	f.data.mu.Lock()
	defer f.data.mu.Unlock()
	if len(f.data.labs) != 0 || len(f.data.overviews) != 0 {
		t.Fatalf("cache cancellation bought labs=%d overview=%d", len(f.data.labs), len(f.data.overviews))
	}
}
