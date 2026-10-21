package backlinks

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend/spendtest"
)

func TestOverviewCancelledDuringCacheReadDoesNotBuyTasks(t *testing.T) {
	svc, _, calls := newOverviewService(t, 90, http.StatusOK, &spendtest.Guard{})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	cache := &spendtest.CancelOnReadCache{Cancel: cancel}
	svc.cache = cache
	_, err := svc.Overview(ctx, "org", lookupInput{Target: "example.com", Scope: string(ScopeDomain)})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled lookup=%v", err)
	}
	// Drain any detached work registered before Overview returned; no sleeps.
	_, _, _ = svc.flights.Do(cache.Key, func() (any, error) { return nil, nil })
	if calls.Load() != 0 {
		t.Fatalf("cache cancellation still bought %d tasks", calls.Load())
	}
}
