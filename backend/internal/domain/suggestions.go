package domain

import (
	"context"
	"time"
)

// KeywordSuggestions returns the top 100 ranked keywords for a domain.
func (s *Service) KeywordSuggestions(ctx context.Context, organizationID, projectID, rawDomain string, scope Scope, locationCode int, languageCode string) ([]KeywordSuggestion, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	target, err := parseResearchTarget(rawDomain, scope)
	if err != nil {
		return nil, badRequest(err.Error())
	}
	locationCode = normalizeLocationCode(locationCode)
	languageCode = normalizeLanguageCode(languageCode)
	key, err := makeCacheKey("keyword-suggestions", organizationID, projectID, struct {
		Domain       string
		Scope        Scope
		Path         string
		LocationCode int
		LanguageCode string
	}{target.Hostname, target.Scope, target.Path, locationCode, languageCode})
	if err != nil {
		return nil, err
	}
	var cached []KeywordSuggestion
	if s.readCache(ctx, key, &cached) && cached != nil {
		return cached, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result := s.suggestions.DoChan(key, func() (any, error) {
		shared, cancel := context.WithTimeout(context.WithoutCancel(ctx), 90*time.Second)
		defer cancel()
		var recent []KeywordSuggestion
		if s.readCache(shared, key, &recent) && recent != nil {
			return recent, nil
		}

		filters := buildRankedKeywordsScopeFilter(target)
		request := map[string]any{
			"target": target.Hostname, "location_code": locationCode,
			"language_code": languageCode, "limit": 100,
			"order_by": []string{"ranked_serp_element.serp_item.etv,desc"},
		}
		if len(filters.clauses) != 0 {
			request["filters"] = join(filters.clauses, "and")
		}
		response, err := s.provider.rankedKeywords(shared, organizationID, request)
		if err != nil {
			return nil, err
		}
		out := make([]KeywordSuggestion, 0, len(response.Items))
		for _, item := range response.Items {
			mapped, ok := mapKeyword(item)
			if !ok {
				continue
			}
			out = append(out, mapped.KeywordSuggestion)
		}
		ttl := cacheTTL
		if len(out) == 0 {
			ttl = emptyResultCacheTTL
		}
		s.writeCacheFor(shared, key, out, ttl)
		return out, nil
	})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case finished := <-result:
		if finished.Err != nil {
			return nil, finished.Err
		}
		return finished.Val.([]KeywordSuggestion), nil
	}
}
