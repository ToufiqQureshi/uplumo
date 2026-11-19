// Package freeaudit serves the public, sign-up-free SEO and AEO audit of one
// page. It never calls a paid provider. It keeps the load on our server and on
// the checked site small: a cached result per page, a per-visitor and a
// global limit on fresh checks, a few checks at a time, and one check per page
// however many visitors ask at once.
package freeaudit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"

	"github.com/toufiqqureshi/uplumo/backend/internal/audit"
)

// Limits. A fresh check costs no money, only a few requests to the checked
// site and a little CPU here.
const (
	// cacheTTL is how long a page's result is reused for every visitor.
	cacheTTL = 24 * time.Hour
	// failedCacheTTL is how long an unreachable page's result is reused, short
	// so a fixed site can be checked again soon.
	failedCacheTTL = 10 * time.Minute
	// lookupsPerHour bounds every request of one visitor, cached ones too;
	// each starts with a DNS lookup and a redirect probe.
	lookupsPerHour = 30
	// freshPerDay bounds the fresh checks of one visitor.
	freshPerDay = 10
	// freshPerHourGlobal bounds fresh checks for everyone together.
	freshPerHourGlobal = 600
	// concurrentChecks is how many fresh checks run at once.
	concurrentChecks = 4
	// slotWait is how long a check waits for a free slot.
	slotWait = 15 * time.Second
	// checkTimeout bounds one fresh check.
	checkTimeout = 30 * time.Second
)

// Errors the handler turns into messages.
var (
	ErrInvalidURL = errors.New("free audit: not a website address")
	ErrBlocked    = errors.New("free audit: private or blocked address")
	// ErrTooMany carries no retry time; the message names the window.
	ErrTooMany = errors.New("free audit: too many checks")
	ErrBusy    = errors.New("free audit: all check slots busy")
)

// Checker reads and scores one page (audit.PageChecker).
type Checker interface {
	NormalizeURL(ctx context.Context, input string) (string, error)
	Check(ctx context.Context, pageURL string) audit.PageReport
}

// Limiter counts requests per key in a fixed window (emailauth.RedisLimiter).
type Limiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (ok bool, retryAfter time.Duration, err error)
}

// Service runs free audits.
type Service struct {
	checker Checker
	redis   *redis.Client
	limiter Limiter
	logger  *slog.Logger
	slots   chan struct{}
	flights singleflight.Group
}

// New builds the service.
func New(checker Checker, rdb *redis.Client, limiter Limiter, logger *slog.Logger) *Service {
	return &Service{checker: checker, redis: rdb, limiter: limiter, logger: logger, slots: make(chan struct{}, concurrentChecks)}
}

// Run returns the report for input, the address a visitor typed, checked on
// behalf of the visitor at ip. A page checked in the last day is served from
// the cache and does not count toward the visitor's fresh checks.
func (s *Service) Run(ctx context.Context, input, ip string) (audit.PageReport, error) {
	if err := s.allow(ctx, "lookup:"+ip, lookupsPerHour, time.Hour); err != nil {
		return audit.PageReport{}, err
	}
	pageURL, err := s.checker.NormalizeURL(ctx, input)
	switch {
	case errors.Is(err, audit.ErrCrawlTargetBlocked):
		return audit.PageReport{}, ErrBlocked
	case err != nil:
		return audit.PageReport{}, fmt.Errorf("%w: %w", ErrInvalidURL, err)
	}
	if report, ok := s.cached(ctx, pageURL); ok {
		return report, nil
	}
	result, err, _ := s.flights.Do(pageURL, func() (any, error) {
		// Another request may have filled the cache while this one waited.
		if report, ok := s.cached(ctx, pageURL); ok {
			return report, nil
		}
		if err := s.allow(ctx, "fresh:"+ip, freshPerDay, 24*time.Hour); err != nil {
			return nil, err
		}
		if err := s.allow(ctx, "fresh-all", freshPerHourGlobal, time.Hour); err != nil {
			return nil, err
		}
		return s.check(ctx, pageURL)
	})
	if err != nil {
		return audit.PageReport{}, err
	}
	report, _ := result.(audit.PageReport)
	return report, nil
}

// check runs one fresh check in a free slot and caches its result.
func (s *Service) check(ctx context.Context, pageURL string) (audit.PageReport, error) {
	wait, cancelWait := context.WithTimeout(ctx, slotWait)
	defer cancelWait()
	select {
	case s.slots <- struct{}{}:
	case <-wait.Done():
		return audit.PageReport{}, ErrBusy
	}
	defer func() { <-s.slots }()

	// The check outlives a visitor who closes the tab, so the result is
	// cached for the next one; it is still bounded by checkTimeout.
	checkCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), checkTimeout)
	defer cancel()
	report := s.checker.Check(checkCtx, pageURL)
	ttl := cacheTTL
	if !report.Reachable {
		ttl = failedCacheTTL
	}
	if body, err := json.Marshal(report); err == nil {
		if err := s.redis.Set(checkCtx, cacheKey(pageURL), body, ttl).Err(); err != nil {
			s.logger.WarnContext(ctx, "cache free audit", "err", err)
		}
	}
	return report, nil
}

func (s *Service) cached(ctx context.Context, pageURL string) (audit.PageReport, bool) {
	body, err := s.redis.Get(ctx, cacheKey(pageURL)).Bytes()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			s.logger.WarnContext(ctx, "read free audit cache", "err", err)
		}
		return audit.PageReport{}, false
	}
	var report audit.PageReport
	if err := json.Unmarshal(body, &report); err != nil {
		return audit.PageReport{}, false
	}
	return report, true
}

// allow applies one limit. When Redis is down the check goes ahead: the slot
// cap still bounds the load, and a free check costs no money.
func (s *Service) allow(ctx context.Context, key string, limit int, window time.Duration) error {
	ok, _, err := s.limiter.Allow(ctx, "free-audit:"+key, limit, window)
	if err != nil {
		s.logger.WarnContext(ctx, "free audit rate limit unavailable", "err", err)
		return nil
	}
	if !ok {
		return ErrTooMany
	}
	return nil
}

func cacheKey(pageURL string) string {
	sum := sha256.Sum256([]byte(pageURL))
	return "free-audit:v1:" + hex.EncodeToString(sum[:16])
}
