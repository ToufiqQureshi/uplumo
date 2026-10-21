package backlinks

import (
	"cmp"
	"context"
	"slices"
	"strings"
)

// A target whose whole list fits in one provider task (blockSize items) is
// bought once, unfiltered, and every view of it — another sort, Best links,
// one link per domain, any filter, any page — is cut from that copy here.
// Asking the provider for each view would buy the same links again.
//
// The overview (bought first) says how many links and domains there are, so
// the decision is made before buying. Top pages have no such count: their
// first unfiltered block becomes the whole list when it holds everything.

type wholeList[T any] struct {
	Rows      []T    `json:"rows"`
	FetchedAt string `json:"fetchedAt"`
}

// view describes how to filter and sort one kind of row locally, matching
// what the provider would answer for the same request.
type view[T any] struct {
	keep  func(T) bool
	value func(T, string) sortValue
	// onePer, when set, keeps the first row (in sort order) per key.
	onePer func(T) string
}

type sortValue struct {
	num  *float64
	text *string
}

func (v sortValue) missing() bool { return v.num == nil && v.text == nil }

// readWhole returns the cached whole list for kind, if there is one.
func readWhole[T any](ctx context.Context, s *Service, kind, org string, target Target) ([]T, string, bool) {
	key, err := cacheKey(kind+"-whole", org, target)
	if err != nil {
		return nil, "", false
	}
	var got wholeList[T]
	if !s.readCache(ctx, key, &got) {
		return nil, "", false
	}
	if got.Rows == nil {
		got.Rows = []T{}
	}
	return got.Rows, got.FetchedAt, true
}

func writeWhole[T any](ctx context.Context, s *Service, kind, org string, target Target, rows []T, fetchedAt string) {
	key, err := cacheKey(kind+"-whole", org, target)
	if err != nil {
		return
	}
	s.writeCache(ctx, key, wholeList[T]{Rows: rows, FetchedAt: fetchedAt})
}

// overviewCount reads a count from the cached overview of the same target.
func (s *Service) overviewCount(ctx context.Context, org string, target Target, pick func(Summary) *float64) (float64, bool) {
	key, err := cacheKey("overview", org, target)
	if err != nil {
		return 0, false
	}
	var ov Overview
	if !s.readCache(ctx, key, &ov) {
		return 0, false
	}
	n := pick(ov.Summary)
	if n == nil {
		return 0, false
	}
	return *n, true
}

// ensureWhole returns the whole list for kind: cached, or bought once when the
// overview says it fits in one task. ok is false when the list is longer, the
// count is unknown, or the target is a subfolder (filtered on the provider).
func ensureWhole[T any](ctx context.Context, s *Service, kind, org string, target Target, pick func(Summary) *float64, sort string, buy func(pageInput) ([]T, *int, error)) ([]T, string, bool, error) {
	if target.Scope == ScopeSubfolder {
		return nil, "", false, nil
	}
	if rows, fetchedAt, ok := readWhole[T](ctx, s, kind, org, target); ok {
		return rows, fetchedAt, true, nil
	}
	if pick == nil {
		return nil, "", false, nil
	}
	count, known := s.overviewCount(ctx, org, target, pick)
	if !known || count > blockSize {
		return nil, "", false, nil
	}
	rows, total, err := buy(pageInput{Page: 1, PageSize: blockSize, Sort: sort, Mode: "as_is"})
	if err != nil {
		return nil, "", false, err
	}
	fetchedAt := timestamp(s.now())
	if total == nil || *total > len(rows) {
		// The overview undercounted. Keep what was bought as the first block
		// of the unfiltered view so it is not bought again, and fall back.
		s.logger.WarnContext(ctx, "backlinks whole list larger than the overview said", "kind", kind, "overview", count, "total", total)
		return nil, "", false, nil
	}
	writeWhole(ctx, s, kind, org, target, rows, fetchedAt)
	return rows, fetchedAt, true, nil
}

// cut answers one page of a view from the whole list.
func cut[T any](all []T, v view[T], sort string, page pageInput, fetchedAt string) Page[T] {
	field, dir, _ := strings.Cut(sort, ",")
	desc := dir == "desc"
	rows := make([]T, 0, len(all))
	for _, row := range all {
		if v.keep(row) {
			rows = append(rows, row)
		}
	}
	slices.SortStableFunc(rows, func(a, b T) int {
		return compareValues(v.value(a, field), v.value(b, field), desc)
	})
	if v.onePer != nil {
		seen := map[string]bool{}
		unique := rows[:0]
		for _, row := range rows {
			if key := v.onePer(row); !seen[key] {
				seen[key] = true
				unique = append(unique, row)
			}
		}
		rows = unique
	}
	total := len(rows)
	offset := min((page.Page-1)*page.PageSize, total)
	end := min(offset+page.PageSize, total)
	return Page[T]{Rows: rows[offset:end], TotalCount: &total, HasMore: end < total, Page: page.Page, PageSize: page.PageSize, FetchedAt: fetchedAt}
}

// compareValues orders present values by direction and puts missing ones last.
func compareValues(a, b sortValue, desc bool) int {
	switch {
	case a.missing() && b.missing():
		return 0
	case a.missing():
		return 1
	case b.missing():
		return -1
	}
	var c int
	if a.num != nil && b.num != nil {
		c = cmp.Compare(*a.num, *b.num)
	} else if a.text != nil && b.text != nil {
		c = strings.Compare(*a.text, *b.text)
	}
	if desc {
		return -c
	}
	return c
}

func num(v *float64) sortValue { return sortValue{num: v} }
func text(v *string) sortValue { return sortValue{text: v} }

// matchTerms mirrors the provider's include (any term, ilike) and exclude
// (no term, not_ilike) filters on one text field.
func matchTerms(value *string, include, exclude string) bool {
	s := ""
	if value != nil {
		s = strings.ToLower(*value)
	}
	if terms := filterTerms(include); len(terms) > 0 {
		if !slices.ContainsFunc(terms, func(term string) bool { return strings.Contains(s, term) }) {
			return false
		}
	}
	for _, term := range filterTerms(exclude) {
		if strings.Contains(s, term) {
			return false
		}
	}
	return true
}

// inRange mirrors provider >= / <= filters: a missing value never matches.
func inRange(value, minimum, maximum *float64) bool {
	if minimum == nil && maximum == nil {
		return true
	}
	if value == nil {
		return false
	}
	return (minimum == nil || *value >= *minimum) && (maximum == nil || *value <= *maximum)
}

func rowsView(f rowsFilters, hideSpam bool, mode string) view[BacklinkRow] {
	v := view[BacklinkRow]{
		keep: func(r BacklinkRow) bool {
			if hideSpam && r.SpamScore != nil && *r.SpamScore >= 40 {
				return false
			}
			if f.LinkType != "" && (r.IsDofollow == nil || *r.IsDofollow != (f.LinkType == "dofollow")) {
				return false
			}
			if f.HideLost != nil && *f.HideLost && r.IsLost {
				return false
			}
			if f.HideBroken != nil && *f.HideBroken && r.IsBroken {
				return false
			}
			if f.DomainFrom != "" && (r.DomainFrom == nil || *r.DomainFrom != f.DomainFrom) {
				return false
			}
			return matchTerms(r.URLFrom, f.Include, f.Exclude) &&
				inRange(r.DomainFromRank, f.MinDomainRank, f.MaxDomainRank) &&
				inRange(r.Rank, f.MinLinkAuthority, f.MaxLinkAuthority) &&
				inRange(r.SpamScore, f.MinSpamScore, f.MaxSpamScore)
		},
		value: func(r BacklinkRow, field string) sortValue {
			switch field {
			case "rank":
				return num(r.Rank)
			case "domain_from_rank":
				return num(r.DomainFromRank)
			case "backlink_spam_score":
				return num(r.SpamScore)
			default:
				return text(r.FirstSeen)
			}
		},
	}
	if mode == "one_per_domain" {
		v.onePer = func(r BacklinkRow) string {
			if r.DomainFrom == nil {
				return ""
			}
			return *r.DomainFrom
		}
	}
	return v
}

func domainsView(f domainsFilters) view[ReferringDomainRow] {
	return view[ReferringDomainRow]{
		keep: func(r ReferringDomainRow) bool {
			return matchTerms(r.Domain, f.Include, f.Exclude) &&
				inRange(r.Backlinks, f.MinBacklinks, f.MaxBacklinks) &&
				inRange(r.Rank, f.MinRank, f.MaxRank) &&
				inRange(r.SpamScore, f.MinSpamScore, f.MaxSpamScore)
		},
		value: func(r ReferringDomainRow, field string) sortValue {
			switch field {
			case "domain":
				return text(r.Domain)
			case "referring_pages":
				return num(r.ReferringPages)
			case "rank":
				return num(r.Rank)
			case "backlinks_spam_score":
				return num(r.SpamScore)
			case "first_seen":
				return text(r.FirstSeen)
			case "broken_backlinks":
				return num(r.BrokenBacklinks)
			default:
				return num(r.Backlinks)
			}
		},
	}
}

func pagesView(f pagesFilters) view[TopPageRow] {
	return view[TopPageRow]{
		keep: func(r TopPageRow) bool {
			return matchTerms(r.Page, f.Include, f.Exclude) &&
				inRange(r.Backlinks, f.MinBacklinks, f.MaxBacklinks) &&
				inRange(r.ReferringDomains, f.MinReferringDomains, f.MaxReferringDomains) &&
				inRange(r.Rank, f.MinRank, f.MaxRank)
		},
		value: func(r TopPageRow, field string) sortValue {
			switch field {
			case "referring_domains":
				return num(r.ReferringDomains)
			case "rank":
				return num(r.Rank)
			case "broken_backlinks":
				return num(r.BrokenBacklinks)
			default:
				return num(r.Backlinks)
			}
		},
	}
}
