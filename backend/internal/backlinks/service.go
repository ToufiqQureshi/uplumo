package backlinks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/dataforseo"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
	"golang.org/x/sync/singleflight"
)

const (
	// The provider's backlink index changes slowly, and the screen shows when
	// a report was fetched, so a day of reuse saves a paid repeat per lookup.
	cacheTTL            = 24 * time.Hour
	partialCacheTTL     = 10 * time.Minute
	cacheWriteTimeout   = 2 * time.Second
	maxFilterConditions = 8
	// flightTimeout bounds shared paid work that no longer follows its first
	// caller's context. Each provider request also has its own 60s limit.
	flightTimeout = 3 * time.Minute
)

type cache interface {
	Get(context.Context, string) ([]byte, error)
	Set(context.Context, string, []byte, time.Duration) error
}
type redisCache struct{ client *redis.Client }

func (c redisCache) Get(ctx context.Context, key string) ([]byte, error) {
	return c.client.Get(ctx, key).Bytes()
}
func (c redisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

// Service retrieves and caches backlinks reports for organizations.
type Service struct {
	provider provider
	cache    cache
	logger   *slog.Logger
	now      func() time.Time
	flights  singleflight.Group
}

// NewService constructs a backlinks service using the shared DataForSEO client.
func NewService(client *dataforseo.Client, rdb *redis.Client, logger *slog.Logger) *Service {
	return newService(client, redisCache{rdb}, logger, time.Now)
}
func newService(client *dataforseo.Client, c cache, logger *slog.Logger, now func() time.Time) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{provider: provider{client: client}, cache: c, logger: logger, now: now}
}

type pageInput struct {
	Page     int
	PageSize int
	Sort     string
	Mode     string
}
type lookupInput struct {
	Target string `json:"target"`
	Scope  string `json:"scope,omitempty"`
}
type rowsFilters struct {
	Include          string   `json:"include,omitempty"`
	Exclude          string   `json:"exclude,omitempty"`
	MinDomainRank    *float64 `json:"minDomainRank,omitempty"`
	MaxDomainRank    *float64 `json:"maxDomainRank,omitempty"`
	MinLinkAuthority *float64 `json:"minLinkAuthority,omitempty"`
	MaxLinkAuthority *float64 `json:"maxLinkAuthority,omitempty"`
	MinSpamScore     *float64 `json:"minSpamScore,omitempty"`
	MaxSpamScore     *float64 `json:"maxSpamScore,omitempty"`
	LinkType         string   `json:"linkType,omitempty"`
	HideLost         *bool    `json:"hideLost,omitempty"`
	HideBroken       *bool    `json:"hideBroken,omitempty"`
	DomainFrom       string   `json:"domainFrom,omitempty"`
}
type domainsFilters struct {
	Include      string   `json:"include,omitempty"`
	Exclude      string   `json:"exclude,omitempty"`
	MinBacklinks *float64 `json:"minBacklinks,omitempty"`
	MaxBacklinks *float64 `json:"maxBacklinks,omitempty"`
	MinRank      *float64 `json:"minRank,omitempty"`
	MaxRank      *float64 `json:"maxRank,omitempty"`
	MinSpamScore *float64 `json:"minSpamScore,omitempty"`
	MaxSpamScore *float64 `json:"maxSpamScore,omitempty"`
}
type pagesFilters struct {
	Include             string   `json:"include,omitempty"`
	Exclude             string   `json:"exclude,omitempty"`
	MinBacklinks        *float64 `json:"minBacklinks,omitempty"`
	MaxBacklinks        *float64 `json:"maxBacklinks,omitempty"`
	MinReferringDomains *float64 `json:"minReferringDomains,omitempty"`
	MaxReferringDomains *float64 `json:"maxReferringDomains,omitempty"`
	MinRank             *float64 `json:"minRank,omitempty"`
	MaxRank             *float64 `json:"maxRank,omitempty"`
}

// These identities are shared by speculative whole-list buys and paged
// requests, so an undercounted list can reuse the same unfiltered block.
type rowsQuery struct {
	Target   Target
	Filters  rowsFilters
	HideSpam bool
}
type domainsQuery struct {
	Target  Target
	Filters domainsFilters
}

// Overview returns summary and history reports for a target.
func (s *Service) Overview(ctx context.Context, org string, in lookupInput) (Overview, error) {
	target, err := normalizeTarget(in.Target, in.Scope)
	if err != nil {
		return Overview{}, err
	}
	key, err := cacheKey("overview", org, target)
	if err != nil {
		return Overview{}, err
	}
	var cached Overview
	if s.readCache(ctx, key, &cached) {
		return cached, nil
	}
	if err := ctx.Err(); err != nil {
		return Overview{}, err
	}
	// A cold cache must not let simultaneous requests purchase the same
	// summary and history twice. The key includes organization and scope.
	result := s.flights.DoChan(key, func() (any, error) {
		// The shared purchase outlives any one waiter: another caller may still
		// need it, and a cancelled paid call may already have been billed.
		flightCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), flightTimeout)
		defer cancel()
		var recent Overview
		if s.readCache(flightCtx, key, &recent) {
			return recent, nil
		}
		return s.loadOverview(flightCtx, org, target, key)
	})
	select {
	case <-ctx.Done():
		return Overview{}, ctx.Err()
	case finished := <-result:
		if finished.Err != nil {
			return Overview{}, finished.Err
		}
		return finished.Val.(Overview), nil
	}
}

func (s *Service) loadOverview(ctx context.Context, org string, target Target, key string) (Overview, error) {
	now := s.now().UTC()
	result := Overview{Target: target.APITarget, DisplayTarget: target.Display, Scope: target.Scope, Trends: []Trend{}, NewLostTrends: []NewLostTrend{}, FetchedAt: timestamp(now)}
	if target.Scope == ScopeSubfolder {
		filters := scopeFilters("url_to", target)
		_, allTotal, err := s.provider.rows(ctx, org, target, pageInput{Page: 1, PageSize: 1, Sort: "rank,desc", Mode: "as_is"}, filters, false)
		if err != nil {
			return Overview{}, err
		}
		_, domainTotal, err := s.provider.rows(ctx, org, target, pageInput{Page: 1, PageSize: 1, Sort: "rank,desc", Mode: "one_per_domain"}, filters, false)
		if err != nil {
			return Overview{}, err
		}
		result.Summary.Backlinks = floatInt(allTotal)
		result.Summary.ReferringDomains = floatInt(domainTotal)
		s.writeCache(ctx, key, result)
		return result, nil
	}
	// Sequential: a billing or provider error on the first task must stop the
	// second one, because failed provider tasks are billed.
	summary, err := s.provider.summary(ctx, org, target)
	if err != nil {
		return Overview{}, err
	}
	var history []providerHistory
	historyLost := false
	if target.Scope != ScopeExactURL {
		dateTo := now.Truncate(24 * time.Hour).Add(-24 * time.Hour)
		dateFrom := dateTo.AddDate(-1, 0, 0)
		history, err = s.provider.history(ctx, org, target.APITarget, dateFrom.Format("2006-01-02"), dateTo.Format("2006-01-02"))
		if err != nil {
			// The summary is already billed. Losing only the trend chart is no
			// reason to throw it away and bill it again on the retry. A credit
			// refusal, a provider billing problem or a customer who left still
			// stop here, as they would for the first task.
			if spend.Refused(err) || errors.Is(err, dataforseo.ErrBillingIssue) || ctx.Err() != nil {
				return Overview{}, err
			}
			s.logger.WarnContext(ctx, "backlinks history failed; answering with the summary only", "err", err)
			history, historyLost = nil, true
		}
	}
	result.Summary = mapSummary(summary)
	if target.Scope != ScopeExactURL {
		for _, item := range history {
			if item.Date == nil || len(*item.Date) < 10 {
				continue
			}
			date := (*item.Date)[:10]
			result.Trends = append(result.Trends, Trend{Date: date, Backlinks: item.Backlinks, ReferringDomains: item.ReferringDomains, Rank: item.Rank})
			result.NewLostTrends = append(result.NewLostTrends, NewLostTrend{Date: date, NewBacklinks: item.NewBacklinks, LostBacklinks: item.LostBacklinks, NewReferringDomains: first(item.NewReferringDomains, item.NewReferringDomainsLegacy), LostReferringDomains: first(item.LostReferringDomains, item.LostReferringDomainsLegacy)})
		}
	}
	ttl := cacheTTL
	if historyLost {
		// Kept just long enough that a retry costs nothing; the full report is
		// bought again once the provider has recovered.
		ttl = partialCacheTTL
	}
	s.writeCacheFor(ctx, key, result, ttl)
	return result, nil
}

// DashboardSummary fetches the metered summary only, without the history task
// used by the full backlinks report. The dashboard stores this result as a
// daily snapshot so repeated page visits do not repeat provider spend.
func (s *Service) DashboardSummary(ctx context.Context, org, domain string) (Summary, error) {
	target, err := normalizeTarget(domain, string(ScopeSubdomains))
	if err != nil {
		return Summary{}, err
	}
	got, err := s.provider.summary(ctx, org, target)
	if err != nil {
		return Summary{}, err
	}
	return mapSummary(got), nil
}

// Rows returns paginated backlink rows matching the request filters.
func (s *Service) Rows(ctx context.Context, org string, in lookupInput, page pageInput, filters rowsFilters, hideSpam bool) (Page[BacklinkRow], error) {
	target, err := normalizeTarget(in.Target, in.Scope)
	if err != nil {
		return Page[BacklinkRow]{}, err
	}
	expr := buildRowsFilters(filters)
	scope := scopeFilters("url_to", target)
	if countFilters(expr)+countFilters(scope)+boolInt(hideSpam)*2 > maxFilterConditions {
		return Page[BacklinkRow]{}, inputError("Too many filter conditions (maximum 8).")
	}
	buy := func(ctx context.Context, block pageInput, expr []any, hideSpam bool) ([]BacklinkRow, *int, error) {
		items, total, err := s.provider.rows(ctx, org, target, block, expr, hideSpam)
		if err != nil {
			return nil, nil, err
		}
		rows := make([]BacklinkRow, 0, len(items))
		for _, v := range items {
			rows = append(rows, mapBacklink(v))
		}
		return rows, total, nil
	}
	unfilteredQuery := rowsQuery{Target: target}
	all, fetchedAt, ok, err := ensureWhole(ctx, s, "rows", org, target, func(v Summary) *float64 { return v.Backlinks }, "rank,desc",
		func(block pageInput) ([]BacklinkRow, *int, error) {
			// Store the speculative whole-list purchase as a normal block too.
			// A stale overview can undercount, but the paid rows remain reusable.
			got, err := loadPage(ctx, s, "rows", org, unfilteredQuery, block, func(ctx context.Context, block pageInput) ([]BacklinkRow, *int, error) {
				return buy(ctx, block, nil, false)
			})
			return got.Rows, got.TotalCount, err
		})
	if err != nil {
		return Page[BacklinkRow]{}, err
	}
	if ok {
		return cut(all, rowsView(filters, hideSpam, page.Mode), page.Sort, page, fetchedAt), nil
	}
	expr = joinWithAnd(scope, expr)
	unfiltered := len(expr) == 0 && !hideSpam && page.Mode == "as_is"
	query := rowsQuery{target, filters, hideSpam}
	return loadPage(ctx, s, "rows", org, query, page, func(ctx context.Context, block pageInput) ([]BacklinkRow, *int, error) {
		rows, total, err := buy(ctx, block, expr, hideSpam)
		if err == nil && unfiltered {
			keepIfWhole(ctx, s, "rows", org, target, block, rows, total)
		}
		return rows, total, err
	})
}

// Domains returns paginated referring domains matching the request filters.
func (s *Service) Domains(ctx context.Context, org string, in lookupInput, page pageInput, filters domainsFilters) (Page[ReferringDomainRow], error) {
	target, err := normalizeTarget(in.Target, in.Scope)
	if err != nil {
		return Page[ReferringDomainRow]{}, err
	}
	if target.Scope == ScopeSubfolder {
		return Page[ReferringDomainRow]{}, inputError("Referring domains can't be broken down for a subfolder — use the Backlinks tab, or switch to Domain or Subdomains scope.")
	}
	expr := buildDomainsFilters(filters)
	if countFilters(expr) > maxFilterConditions {
		return Page[ReferringDomainRow]{}, inputError("Too many filter conditions (maximum 8).")
	}
	buy := func(ctx context.Context, block pageInput, expr []any) ([]ReferringDomainRow, *int, error) {
		items, total, err := s.provider.domains(ctx, org, target, block, expr)
		if err != nil {
			return nil, nil, err
		}
		rows := make([]ReferringDomainRow, 0, len(items))
		for _, v := range items {
			rows = append(rows, ReferringDomainRow(v))
		}
		return rows, total, nil
	}
	unfilteredQuery := domainsQuery{Target: target}
	all, fetchedAt, ok, err := ensureWhole(ctx, s, "domains", org, target, func(v Summary) *float64 { return v.ReferringDomains }, "backlinks,desc",
		func(block pageInput) ([]ReferringDomainRow, *int, error) {
			got, err := loadPage(ctx, s, "domains", org, unfilteredQuery, block, func(ctx context.Context, block pageInput) ([]ReferringDomainRow, *int, error) {
				return buy(ctx, block, nil)
			})
			return got.Rows, got.TotalCount, err
		})
	if err != nil {
		return Page[ReferringDomainRow]{}, err
	}
	if ok {
		return cut(all, domainsView(filters), page.Sort, page, fetchedAt), nil
	}
	query := domainsQuery{target, filters}
	return loadPage(ctx, s, "domains", org, query, page, func(ctx context.Context, block pageInput) ([]ReferringDomainRow, *int, error) {
		rows, total, err := buy(ctx, block, expr)
		if err == nil && len(expr) == 0 {
			keepIfWhole(ctx, s, "domains", org, target, block, rows, total)
		}
		return rows, total, err
	})
}

// Pages returns top pages and their backlink metrics for a target.
func (s *Service) Pages(ctx context.Context, org string, in lookupInput, page pageInput, filters pagesFilters) (Page[TopPageRow], error) {
	target, err := normalizeTarget(in.Target, in.Scope)
	if err != nil {
		return Page[TopPageRow]{}, err
	}
	expr := buildPagesFilters(filters)
	scope := scopeFilters("url", target)
	if countFilters(expr)+countFilters(scope) > maxFilterConditions {
		return Page[TopPageRow]{}, inputError("Too many filter conditions (maximum 8).")
	}
	// The overview has no page count, so the whole list only comes from an
	// unfiltered first block that turned out to hold everything.
	if all, fetchedAt, ok := readWhole[TopPageRow](ctx, s, "pages", org, target); ok && target.Scope != ScopeSubfolder {
		return cut(all, pagesView(filters), page.Sort, page, fetchedAt), nil
	}
	expr = joinWithAnd(scope, expr)
	query := struct {
		Target  Target
		Filters pagesFilters
	}{target, filters}
	return loadPage(ctx, s, "pages", org, query, page, func(ctx context.Context, block pageInput) ([]TopPageRow, *int, error) {
		items, total, err := s.provider.pages(ctx, org, target, block, expr)
		if err != nil {
			return nil, nil, err
		}
		rows := make([]TopPageRow, 0, len(items))
		for _, v := range items {
			pageURL := v.Page
			if pageURL == nil {
				pageURL = v.URL
			}
			rows = append(rows, TopPageRow{Page: pageURL, Backlinks: v.Backlinks, ReferringDomains: v.ReferringDomains, Rank: v.Rank, BrokenBacklinks: v.BrokenBacklinks})
		}
		if len(expr) == 0 {
			keepIfWhole(ctx, s, "pages", org, target, block, rows, total)
		}
		return rows, total, nil
	})
}

// keepIfWhole stores an unfiltered first block that holds the whole list, so
// every later sort, filter or page of that target is cut from it for free.
func keepIfWhole[T any](ctx context.Context, s *Service, kind, org string, target Target, block pageInput, rows []T, total *int) {
	if target.Scope != ScopeSubfolder && block.Page == 1 && total != nil && *total <= len(rows) {
		writeWhole(ctx, s, kind, org, target, rows, timestamp(s.now()))
	}
}

// blockSize is how many rows one provider call buys: the provider's maximum.
// Every page size the screen offers (50, 100, 200) divides it, so a page never
// spans two blocks, and changing the page size or paging through the first
// 1,000 rows is served from the cache. A task is $0.024 plus $0.000036 a row
// (measured live), so a block is about $0.06; most targets have fewer than
// 1,000 links, which makes the first table the only one the customer buys.
const blockSize = 1000

type rowBlock[T any] struct {
	Rows      []T    `json:"rows"`
	Total     *int   `json:"total"`
	FetchedAt string `json:"fetchedAt"`
}

// loadPage answers one page from the cached block that holds it, buying that
// block once when it is not cached. query holds everything but the paging.
func loadPage[T any](ctx context.Context, s *Service, kind, org string, query any, page pageInput, buy func(context.Context, pageInput) ([]T, *int, error)) (Page[T], error) {
	offset := (page.Page - 1) * page.PageSize
	block := page
	block.Page, block.PageSize = offset/blockSize+1, blockSize
	key, err := cacheKey(kind, org, struct {
		Query any
		Block pageInput
	}{query, block})
	if err != nil {
		return Page[T]{}, err
	}
	var got rowBlock[T]
	if !s.readCache(ctx, key, &got) {
		if err := ctx.Err(); err != nil {
			return Page[T]{}, err
		}
		flight := s.flights.DoChan(key, func() (any, error) {
			sharedCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), flightTimeout)
			defer cancel()
			var recent rowBlock[T]
			if s.readCache(sharedCtx, key, &recent) {
				return recent, nil
			}
			rows, total, err := buy(sharedCtx, block)
			if err != nil {
				return rowBlock[T]{}, err
			}
			recent = rowBlock[T]{Rows: rows, Total: total, FetchedAt: timestamp(s.now())}
			s.writeCache(sharedCtx, key, recent)
			return recent, nil
		})
		select {
		case <-ctx.Done():
			return Page[T]{}, ctx.Err()
		case finished := <-flight:
			if finished.Err != nil {
				return Page[T]{}, finished.Err
			}
			got = finished.Val.(rowBlock[T])
		}
	}
	if got.Rows == nil {
		got.Rows = []T{}
	}
	start := min(offset-(block.Page-1)*blockSize, len(got.Rows))
	end := min(start+page.PageSize, len(got.Rows))
	rows := got.Rows[start:end]
	hasMore := end < len(got.Rows) || len(got.Rows) == blockSize
	if got.Total != nil {
		hasMore = offset+len(rows) < *got.Total
	}
	return Page[T]{Rows: rows, TotalCount: got.Total, HasMore: hasMore, Page: page.Page, PageSize: page.PageSize, FetchedAt: got.FetchedAt}, nil
}

func (s *Service) readCache(ctx context.Context, key string, dst any) bool {
	if s.cache == nil {
		return false
	}
	raw, err := s.cache.Get(ctx, key)
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			s.logger.WarnContext(ctx, "read backlinks cache", "err", err)
		}
		return false
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		s.logger.WarnContext(ctx, "decode backlinks cache", "err", err)
		return false
	}
	return true
}
func (s *Service) writeCache(ctx context.Context, key string, value any) {
	s.writeCacheFor(ctx, key, value, cacheTTL)
}

func (s *Service) writeCacheFor(ctx context.Context, key string, value any, ttl time.Duration) {
	if s.cache == nil {
		return
	}
	raw, err := json.Marshal(value)
	if err != nil {
		s.logger.ErrorContext(ctx, "encode backlinks cache", "err", err)
		return
	}
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cacheWriteTimeout)
	defer cancel()
	if err := s.cache.Set(writeCtx, key, raw, ttl); err != nil {
		s.logger.WarnContext(ctx, "write backlinks cache", "err", err)
	}
}
func cacheKey(kind, org string, value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode backlinks cache key: %w", err)
	}
	sum := sha256.Sum256(raw)
	// v2: ranks on the 0-100 scale and the paid profile fields; entries cached
	// before that must not be served.
	return "backlinks:v2:" + kind + ":" + org + ":" + hex.EncodeToString(sum[:]), nil
}
func floatInt(v *int) *float64 {
	if v == nil {
		return nil
	}
	n := float64(*v)
	return &n
}
func first(a, b *float64) *float64 {
	if a != nil {
		return a
	}
	return b
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func mapSummary(v providerSummary) Summary {
	return Summary{Rank: v.Rank, Backlinks: v.Backlinks, ReferringPages: v.ReferringPages, ReferringDomains: v.ReferringDomains, BrokenBacklinks: v.BrokenBacklinks, BrokenPages: v.BrokenPages, BacklinksSpamScore: v.BacklinksSpamScore, TargetSpamScore: v.Info.TargetSpamScore, NewBacklinks: v.NewBacklinks, LostBacklinks: v.LostBacklinks, NewReferringDomains: first(v.NewReferringDomains, v.NewReferringDomainsLegacy), LostReferringDomains: first(v.LostReferringDomains, v.LostReferringDomainsLegacy),
		Profile: Profile{FirstSeen: v.FirstSeen, CrawledPages: v.CrawledPages, ReferringDomainsNofollow: v.ReferringDomainsNofollow, ReferringIPs: v.ReferringIPs, ReferringSubnets: v.ReferringSubnets,
			TLDs: counts(v.TLDs), LinkTypes: counts(v.LinkTypes), LinkAttributes: counts(v.LinkAttributes), PlatformTypes: counts(v.PlatformTypes), Placements: counts(v.Placements), Countries: counts(v.Countries)}}
}

// counts drops the provider's blank key (links it could not classify) and
// never returns nil, so the screen always gets an object.
func counts(m map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(m))
	for k, v := range m {
		if k != "" {
			out[k] = v
		}
	}
	return out
}
func mapBacklink(v providerBacklink) BacklinkRow {
	spam := first(v.SpamScore, v.SpamScoreLegacy)
	attrs := v.RelAttributes
	if attrs == nil {
		attrs = v.Attributes
	}
	if attrs == nil {
		attrs = []string{}
	}
	lost := false
	if v.IsLost != nil {
		lost = *v.IsLost
	} else if v.LostDate != nil {
		lost = *v.LostDate != ""
	}
	broken := v.IsBroken != nil && *v.IsBroken
	last := v.LostDate
	if last == nil {
		last = v.LastSeen
	}
	return BacklinkRow{DomainFrom: v.DomainFrom, URLFrom: v.URLFrom, URLTo: v.URLTo, Anchor: v.Anchor, ItemType: v.ItemType, IsDofollow: v.Dofollow, RelAttributes: attrs, Rank: v.Rank, DomainFromRank: v.DomainFromRank, PageFromRank: v.PageFromRank, SpamScore: spam, FirstSeen: v.FirstSeen, LastSeen: last, IsLost: lost, IsBroken: broken, IsNew: v.IsNew != nil && *v.IsNew, LinksCount: v.LinksCount,
		TextBefore: v.TextPre, TextAfter: v.TextPost, SourceTitle: v.PageFromTitle, Placement: v.Placement, TargetStatus: v.TargetStatus, SourceCountry: v.Country}
}

func buildRowsFilters(f rowsFilters) []any {
	out := []any{}
	appendTerms(&out, "url_from", f.Exclude, "not_ilike")
	appendRange(&out, "domain_from_rank", f.MinDomainRank, f.MaxDomainRank)
	appendRange(&out, "rank", f.MinLinkAuthority, f.MaxLinkAuthority)
	appendRange(&out, "backlink_spam_score", f.MinSpamScore, f.MaxSpamScore)
	if f.LinkType != "" {
		out = append(out, []any{"dofollow", "=", f.LinkType == "dofollow"})
	}
	if f.HideLost != nil && *f.HideLost {
		out = append(out, []any{"is_lost", "=", false})
	}
	if f.HideBroken != nil && *f.HideBroken {
		out = append(out, []any{"is_broken", "=", false})
	}
	if f.DomainFrom != "" {
		out = append(out, []any{"domain_from", "=", f.DomainFrom})
	}
	return prependInclude(out, "url_from", f.Include)
}
func buildDomainsFilters(f domainsFilters) []any {
	out := []any{}
	appendTerms(&out, "domain", f.Exclude, "not_ilike")
	appendRange(&out, "backlinks", f.MinBacklinks, f.MaxBacklinks)
	appendRange(&out, "rank", f.MinRank, f.MaxRank)
	appendRange(&out, "backlinks_spam_score", f.MinSpamScore, f.MaxSpamScore)
	return prependInclude(out, "domain", f.Include)
}
func buildPagesFilters(f pagesFilters) []any {
	out := []any{}
	appendTerms(&out, "url", f.Exclude, "not_ilike")
	appendRange(&out, "backlinks", f.MinBacklinks, f.MaxBacklinks)
	appendRange(&out, "referring_domains", f.MinReferringDomains, f.MaxReferringDomains)
	appendRange(&out, "rank", f.MinRank, f.MaxRank)
	return prependInclude(out, "url", f.Include)
}
func appendRange(out *[]any, field string, minimum, maximum *float64) {
	if minimum != nil {
		*out = append(*out, []any{field, ">=", *minimum})
	}
	if maximum != nil {
		*out = append(*out, []any{field, "<=", *maximum})
	}
}
func prependInclude(out []any, field, value string) []any {
	terms := filterTerms(value)
	conditions := make([]any, 0, len(terms))
	for _, term := range terms {
		conditions = append(conditions, []any{field, "ilike", "%" + escapeLike(term) + "%"})
	}
	clauses := make([]any, 0, len(out)+1)
	if len(conditions) == 1 {
		clauses = append(clauses, conditions[0])
	} else if len(conditions) > 1 {
		group := []any{}
		for i, c := range conditions {
			if i > 0 {
				group = append(group, "or")
			}
			group = append(group, c)
		}
		clauses = append(clauses, group)
	}
	clauses = append(clauses, out...)
	if len(clauses) == 0 {
		return nil
	}
	joined := make([]any, 0, len(clauses)*2-1)
	for i, clause := range clauses {
		if i > 0 {
			joined = append(joined, "and")
		}
		joined = append(joined, clause)
	}
	return joined
}
func appendTerms(out *[]any, field, value, operator string) {
	for _, term := range filterTerms(value) {
		*out = append(*out, []any{field, operator, "%" + escapeLike(term) + "%"})
	}
}
func filterTerms(value string) []string {
	terms := []string{}
	for _, term := range strings.FieldsFunc(strings.ToLower(value), func(r rune) bool { return r == ',' || r == '+' }) {
		if term = strings.TrimSpace(term); term != "" {
			terms = append(terms, term)
		}
	}
	return terms
}
func escapeLike(value string) string {
	return strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(value)
}
func countFilters(expr []any) int {
	n := 0
	for _, item := range expr {
		switch x := item.(type) {
		case []any:
			if len(x) > 0 {
				if _, ok := x[0].([]any); ok {
					n += countFilters(x)
				} else {
					n++
				}
			}
		}
	}
	return n
}
func scopeFilters(field string, target Target) []any {
	if target.Scope != ScopeSubfolder {
		return nil
	}
	hosts := []string{target.APITarget, "www." + target.APITarget}
	group := []any{}
	for _, host := range hosts {
		for _, suffix := range []string{target.Path, target.Path + "/%"} {
			if len(group) > 0 {
				group = append(group, "or")
			}
			group = append(group, []any{field, "like", "%://" + escapeLike(host) + suffix})
		}
	}
	return []any{group}
}
func joinWithAnd(a, b []any) []any {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	out := append([]any{}, a...)
	out = append(out, "and")
	return append(out, b...)
}

// MCPOverview reuses the Go report service for an MCP request.
func (s *Service) MCPOverview(ctx context.Context, organizationID, target, scope string) (Overview, error) {
	return s.Overview(ctx, organizationID, lookupInput{Target: target, Scope: scope})
}

// MCPReferringDomains returns the top referring domains with the MCP default
// spam filter. Subfolder breakdowns are unavailable because the provider does
// not support URL filtering for this report.
func (s *Service) MCPReferringDomains(ctx context.Context, organizationID, target, scope string, hideSpam bool) (Page[ReferringDomainRow], error) {
	filters := domainsFilters{}
	if hideSpam {
		maxSpam := 40.0
		filters.MaxSpamScore = &maxSpam
	}
	return s.Domains(ctx, organizationID, lookupInput{Target: target, Scope: scope},
		pageInput{Page: 1, PageSize: 100, Sort: "backlinks,desc"}, filters)
}
