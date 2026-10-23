package domain

import (
	"context"
	"time"
)

// cachedDomainLookup shares equivalent report purchases within this server.
// Every caller has already validated its input and supplied the full tenant,
// project, market, scope and result-changing input cache identity.
func cachedDomainLookup[T any](ctx context.Context, s *Service, key string, lifetime func(T) time.Duration, load func(context.Context) (T, error)) (T, error) {
	var zero T
	ttl := func(value T) time.Duration {
		if lifetime != nil {
			return lifetime(value)
		}
		return cacheTTL
	}
	var cached T
	if s.readCache(ctx, key, &cached) && ttl(cached) > 0 {
		return cached, nil
	}
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	result := s.lookups.DoChan(key, func() (any, error) {
		shared, cancel := context.WithTimeout(context.WithoutCancel(ctx), 90*time.Second)
		defer cancel()
		var recent T
		if s.readCache(shared, key, &recent) && ttl(recent) > 0 {
			return recent, nil
		}
		value, err := load(shared)
		if err == nil && ttl(value) > 0 {
			s.writeCacheFor(shared, key, value, ttl(value))
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
