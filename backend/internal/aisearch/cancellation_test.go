package aisearch

import (
	"context"
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend/spendtest"
)

type cancelReadHook struct{ cache *spendtest.CancelOnReadCache }

func (cancelReadHook) DialHook(next redis.DialHook) redis.DialHook { return next }
func (cancelReadHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return next
}
func (h cancelReadHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if cmd.Name() == "get" {
			_, err := h.cache.Get(ctx, cmd.Args()[1].(string))
			return err
		}
		return next(ctx, cmd)
	}
}

func TestBrandLookupCancelledDuringCacheReadDoesNotBuyTasks(t *testing.T) {
	api := newFakeAPI(t)
	svc := newTestService(t, api)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	cache := &spendtest.CancelOnReadCache{Cancel: cancel}
	svc.redis.AddHook(cancelReadHook{cache: cache})
	_, err := svc.BrandLookup(ctx, newOrg(), "project", lookupInput())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled brand lookup=%v", err)
	}
	_, _, _ = svc.flights.Do(cache.Key, func() (any, error) { return nil, nil })
	if api.total() != 0 {
		t.Fatalf("cache cancellation bought %d AI tasks", api.total())
	}
}
