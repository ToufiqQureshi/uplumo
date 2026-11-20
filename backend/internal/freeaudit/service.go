// Package freeaudit serves the public, sign-up-free SEO and AEO audit of one
// page. It never calls a paid provider. One fresh check fetches the site once
// and scores it with both engines; the cache keeps both reports, so switching
// between them is free. The limits keep the load on our server and on the
// checked site small: a per-visitor and a global limit on fresh checks, a few
// checks at a time, and one check per page however many visitors ask at once.
package freeaudit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
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

// Checker fetches one page and its site's files (audit.PageChecker).
type Checker interface {
	// ValidateURL returns the start address of input without any request to
	// the site; NormalizeURL also follows the site's redirects (one request).
	ValidateURL(ctx context.Context, input string) (string, error)
	NormalizeURL(ctx context.Context, input string) (string, error)
	Check(ctx context.Context, pageURL string) audit.PageSnapshot
}

// audits is what the cache holds for one page: both engines' reports from
// one fetch.
type audits struct {
	SEO audit.PageReport `json:"seo"`
	AEO audit.PageReport `json:"aeo"`
}

// ParseKind reads a report type from a request: "seo" or "aeo", where ""
// means the SEO report. ok is false for any other value.
func ParseKind(raw string) (kind string, ok bool) {
	switch raw {
	case "", audit.KindSEO:
		return audit.KindSEO, true
	case audit.KindAEO:
		return audit.KindAEO, true
	}
	return "", false
}

// pick returns the report of kind from both engines.
func (a audits) pick(kind string) audit.PageReport {
	if kind == audit.KindAEO {
		return a.AEO
	}
	return a.SEO
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

// Run returns the report of kind (KindSEO or KindAEO) for input, the address a
// visitor typed, checked on behalf of the visitor at ip. A page checked in the
// last day is served from the cache and does not count toward the visitor's
// fresh checks. A fresh check fetches the page once and fills the cache with
// both reports, so the other kind is then served without a check.
func (s *Service) Run(ctx context.Context, input, ip, kind string) (audit.PageReport, error) {
	if err := s.allow(ctx, "lookup:"+ip, lookupsPerHour, time.Hour); err != nil {
		return audit.PageReport{}, err
	}
	start, err := s.validate(ctx, input)
	if err != nil {
		return audit.PageReport{}, err
	}
	if both, ok := s.cached(ctx, start); ok {
		return both.pick(kind), nil
	}
	result, err, _ := s.flights.Do(start, func() (any, error) {
		// Another request may have filled the cache while this one waited.
		if both, ok := s.cached(ctx, start); ok {
			return both, nil
		}
		if err := s.allow(ctx, "fresh:"+ip, freshPerDay, 24*time.Hour); err != nil {
			return nil, err
		}
		if err := s.allow(ctx, "fresh-all", freshPerHourGlobal, time.Hour); err != nil {
			return nil, err
		}
		pageURL, err := s.checker.NormalizeURL(ctx, input)
		if err != nil {
			return nil, normalizeError(err)
		}
		return s.check(ctx, start, pageURL)
	})
	if err != nil {
		return audit.PageReport{}, err
	}
	both, _ := result.(audits)
	return both.pick(kind), nil
}

// validate maps a visitor's address to its start address, with no request to
// the site, and the errors the handler turns into messages.
func (s *Service) validate(ctx context.Context, input string) (string, error) {
	start, err := s.checker.ValidateURL(ctx, input)
	if err != nil {
		return "", normalizeError(err)
	}
	return start, nil
}

// normalizeError maps a checker's address error to the free audit's errors.
func normalizeError(err error) error {
	if errors.Is(err, audit.ErrCrawlTargetBlocked) {
		return ErrBlocked
	}
	return fmt.Errorf("%w: %w", ErrInvalidURL, err)
}

// check runs one fresh check of pageURL (the page after the site's redirects)
// in a free slot, scores it with both engines and caches both reports under
// the start address the visitor typed and under the page's addresses.
func (s *Service) check(ctx context.Context, start, pageURL string) (audits, error) {
	wait, cancelWait := context.WithTimeout(ctx, slotWait)
	defer cancelWait()
	select {
	case s.slots <- struct{}{}:
	case <-wait.Done():
		return audits{}, ErrBusy
	}
	defer func() { <-s.slots }()

	// The check outlives a visitor who closes the tab, so the result is
	// cached for the next one; it is still bounded by checkTimeout.
	checkCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), checkTimeout)
	defer cancel()
	snap := s.checker.Check(checkCtx, pageURL)
	both := audits{SEO: audit.ScoreSEO(snap), AEO: audit.ScoreAEO(snap)}
	ttl := cacheTTL
	if !snap.Reachable {
		ttl = failedCacheTTL
	}
	body, err := json.Marshal(both)
	if err != nil {
		return both, nil
	}
	// The reports are cached under the address the visitor typed (what a
	// toggle or a Word download asks for, with no site request), under the
	// page address after the redirect probe, and under the page's final
	// address; the Word link is built from the final one.
	keys := []string{cacheKey(start)}
	for _, address := range []string{pageURL, snap.URL} {
		if address != "" && !slices.Contains(keys, cacheKey(address)) {
			keys = append(keys, cacheKey(address))
		}
	}
	for _, key := range keys {
		if err := s.redis.Set(checkCtx, key, body, ttl).Err(); err != nil {
			s.logger.WarnContext(ctx, "cache free audit", "err", err)
		}
	}
	return both, nil
}

// Cached returns the report of kind for input's page when a check of it ran
// in the last day. It never runs a fresh check and does not count toward fresh
// checks; it reads the same per-visitor lookup limit as Run. ok is false when
// the page has no cached report.
func (s *Service) Cached(ctx context.Context, input, ip, kind string) (report audit.PageReport, ok bool, err error) {
	if err := s.allow(ctx, "lookup:"+ip, lookupsPerHour, time.Hour); err != nil {
		return audit.PageReport{}, false, err
	}
	start, err := s.validate(ctx, input)
	if err != nil {
		return audit.PageReport{}, false, err
	}
	both, ok := s.cached(ctx, start)
	return both.pick(kind), ok, nil
}

func (s *Service) cached(ctx context.Context, pageURL string) (audits, bool) {
	body, err := s.redis.Get(ctx, cacheKey(pageURL)).Bytes()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			s.logger.WarnContext(ctx, "read free audit cache", "err", err)
		}
		return audits{}, false
	}
	var both audits
	if err := json.Unmarshal(body, &both); err != nil {
		return audits{}, false
	}
	return both, true
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
	return "free-audit:v3:" + hex.EncodeToString(sum[:16])
}
