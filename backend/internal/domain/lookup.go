package domain

import (
	"context"
	"time"
)

// cachedDomainLookup shares equivalent report purchases within this server.
// Every caller has already validated its input and supplied the full tenant,
// project, market, scope and result-changing input cache identity.
func cachedDomainLookup[T any](ctx context.Context, s *Service, key string, usable func(T) bool, load func(context.Context) (T, error)) (T, error) {
	var zero T
	valid := func(value T) bool { return usable == nil || usable(value) }
	var cached T
	if s.readCache(ctx, key, &cached) && valid(cached) {
		return cached, nil
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	result := s.lookups.DoChan(key, func() (any, error) {
		shared, cancel := context.WithTimeout(context.WithoutCancel(ctx), 90*time.Second)
		defer cancel()
		var recent T
		if s.readCache(shared, key, &recent) && valid(recent) {
			return recent, nil
		}
		value, err := load(shared)
		if err == nil && valid(value) {
			s.writeCache(shared, key, value)
		}
		return value, err
	})
	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case finished := <-result:
		if finished.Err != nil {
			return zero, finished.Err
		}
		return finished.Val.(T), nil
	}
}
