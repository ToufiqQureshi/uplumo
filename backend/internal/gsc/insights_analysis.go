package gsc

import (
	"cmp"
	"math"
	"slices"
	"strings"
	"unicode"
)

// InsightRules are the cut-offs the free analysis uses. They are the only
// place those numbers live. Every one is relative (a share, a ratio, or a
// fraction of this site's own traffic), never a figure borrowed from another
// site or an industry benchmark.
type InsightRules struct {
	// MinCompetingShare is the share of a search's impressions a page needs
	// to count as competing for that search.
	MinCompetingShare float64
	// MinImpressionsFloor and MinImpressionsShare set how many impressions a
	// search needs before it is judged: the larger of the floor and this share
	// of all the site's search impressions.
	MinImpressionsFloor float64
	MinImpressionsShare float64
	// CTRGapRatio flags a search whose CTR is below this fraction of the
	// site's own typical CTR at the same position.
	CTRGapRatio float64
	// MinBucketQueries is how many searches a position needs before the
	// site's own typical CTR there can be trusted.
	MinBucketQueries int
	// DecayDrop is the fall, window over window, that counts as a decline.
	DecayDrop float64
	// DecayMinClicks is the clicks a page needed in the oldest window to be
	// worth reporting.
	DecayMinClicks float64
	// LongTailWords is the word count from which a search is long-tail.
	LongTailWords int
	// ListLimit bounds each returned list.
	ListLimit int
}

// DefaultInsightRules returns the rules used in production.
func DefaultInsightRules() InsightRules {
	return InsightRules{
		MinCompetingShare:   0.2,
		MinImpressionsFloor: 20,
		MinImpressionsShare: 0.0005,
		CTRGapRatio:         0.6,
		MinBucketQueries:    8,
		DecayDrop:           0.2,
		DecayMinClicks:      10,
		LongTailWords:       4,
		ListLimit:           500,
	}
}

// Segment kinds.
const (
	SegmentAll       = "all"
	SegmentBranded   = "branded"
	SegmentNonBrand  = "non_branded"
	SegmentLongTail  = "long_tail"
	maxBrandTerms    = 20
	maxBrandTermRune = 100
)

// Segment is one slice of the site's named searches.
type Segment struct {
	Kind        string  `json:"kind"`
	Queries     int     `json:"queries"`
	Clicks      float64 `json:"clicks"`
	Impressions float64 `json:"impressions"`
	CTR         float64 `json:"ctr"`
	Position    float64 `json:"position"`
}

// CompetingPage is one page that competes for a search.
type CompetingPage struct {
	Page        string  `json:"page"`
	Clicks      float64 `json:"clicks"`
	Impressions float64 `json:"impressions"`
	Position    float64 `json:"position"`
	Share       float64 `json:"share"`
}

// Cannibalized is a search that two or more of the site's pages compete for.
type Cannibalized struct {
	Query       string          `json:"query"`
	Clicks      float64         `json:"clicks"`
	Impressions float64         `json:"impressions"`
	Leader      string          `json:"leader"`
	Pages       []CompetingPage `json:"pages"`
}

// CTRGap is a search clicked far less than this site's own pages are at the
// same position. PossibleClicks is an estimate from the site's own numbers.
type CTRGap struct {
	Query          string  `json:"query"`
	Page           string  `json:"page"`
	Clicks         float64 `json:"clicks"`
	Impressions    float64 `json:"impressions"`
	CTR            float64 `json:"ctr"`
	Position       float64 `json:"position"`
	ExpectedCTR    float64 `json:"expectedCtr"`
	PossibleClicks float64 `json:"possibleClicks"`
}

// Decay is a page that lost clicks in two windows in a row.
type Decay struct {
	Page       string     `json:"page"`
	Clicks     [3]float64 `json:"clicks"` // oldest window first
	LostClicks float64    `json:"lostClicks"`
}

// Action kinds and the basis of their impact figure.
const (
	ActionLosingPage      = "losing_page"
	ActionDecayingPage    = "decaying_page"
	ActionCTRGap          = "ctr_gap"
	ActionCannibalization = "cannibalization"

	BasisMeasured  = "measured"  // clicks the site really lost
	BasisEstimated = "estimated" // clicks possible at the site's own CTR
	BasisContested = "contested" // clicks on a search several pages share
)

// Action is one of the few things worth doing first.
type Action struct {
	Kind    string  `json:"kind"`
	Subject string  `json:"subject"`
	Page    string  `json:"page,omitempty"`
	Impact  float64 `json:"impactClicks"`
	Basis   string  `json:"basis"`
}

type pageStat struct {
	page                       string
	clicks, impressions, wtPos float64
}

func (p pageStat) position() float64 {
	if p.impressions <= 0 {
		return 0
	}
	return p.wtPos / p.impressions
}

// queryStat is one search with every page it was seen on, summed.
type queryStat struct {
	query                      string
	clicks, impressions, wtPos float64
	pages                      []pageStat
}

func (q queryStat) position() float64 {
	if q.impressions <= 0 {
		return 0
	}
	return q.wtPos / q.impressions
}

func (q queryStat) ctr() float64 {
	if q.impressions <= 0 {
		return 0
	}
	return q.clicks / q.impressions
}

// aggregateQueries sums query+page rows into one entry per search. Rows that
// are not a query and a page, or have an empty key, are skipped. Order follows
// first appearance, so the result is stable for the same input.
func aggregateQueries(rows []SearchRow) []queryStat {
	index := map[string]int{}
	var out []queryStat
	for _, row := range rows {
		if len(row.Keys) != 2 || row.Keys[0] == "" || row.Keys[1] == "" {
			continue
		}
		at, seen := index[row.Keys[0]]
		if !seen {
			at = len(out)
			index[row.Keys[0]] = at
			out = append(out, queryStat{query: row.Keys[0]})
		}
		q := &out[at]
		q.clicks += row.Clicks
		q.impressions += row.Impressions
		q.wtPos += row.Position * row.Impressions
		q.pages = append(q.pages, pageStat{page: row.Keys[1], clicks: row.Clicks, impressions: row.Impressions, wtPos: row.Position * row.Impressions})
	}
	return out
}

// normalizeBrandTerms trims, lower-cases, drops empty or one-letter terms and
// removes duplicates. A term of one letter would match nearly every search.
func normalizeBrandTerms(terms []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, term := range terms {
		term = compact(term)
		if len([]rune(term)) < 2 || seen[term] {
			continue
		}
		seen[term] = true
		out = append(out, term)
	}
	return out
}

// compact lower-cases text and removes spaces and punctuation, so "My Brand"
// matches "mybrand" and "my-brand".
func compact(text string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isBranded(query string, compactTerms []string) bool {
	if len(compactTerms) == 0 {
		return false
	}
	q := compact(query)
	for _, term := range compactTerms {
		if strings.Contains(q, term) {
			return true
		}
	}
	return false
}

type segmentSum struct {
	queries                    int
	clicks, impressions, wtPos float64
}

func (s *segmentSum) add(q queryStat) {
	s.queries++
	s.clicks += q.clicks
	s.impressions += q.impressions
	s.wtPos += q.wtPos
}

func (s segmentSum) segment(kind string) Segment {
	out := Segment{Kind: kind, Queries: s.queries, Clicks: s.clicks, Impressions: s.impressions}
	if s.impressions > 0 {
		out.CTR = s.clicks / s.impressions
		out.Position = s.wtPos / s.impressions
	}
	return out
}

// buildSegments splits the named searches. Branded and non-branded appear only
// when the owner gave brand names: guessing a brand from the data would be
// wrong for most sites. Long-tail counts words, so a language written without
// spaces never reaches it; the page says so.
func buildSegments(stats []queryStat, compactTerms []string, rules InsightRules) []Segment {
	var all, branded, other, longTail segmentSum
	for _, q := range stats {
		all.add(q)
		if len(compactTerms) > 0 {
			if isBranded(q.query, compactTerms) {
				branded.add(q)
			} else {
				other.add(q)
			}
		}
		if len(strings.Fields(q.query)) >= rules.LongTailWords {
			longTail.add(q)
		}
	}
	out := []Segment{all.segment(SegmentAll)}
	if len(compactTerms) > 0 {
		out = append(out, branded.segment(SegmentBranded), other.segment(SegmentNonBrand))
	}
	return append(out, longTail.segment(SegmentLongTail))
}

// minImpressions is the impressions a search needs before it is judged.
func minImpressions(stats []queryStat, rules InsightRules) float64 {
	total := 0.0
	for _, q := range stats {
		total += q.impressions
	}
	return math.Max(rules.MinImpressionsFloor, total*rules.MinImpressionsShare)
}

// findCannibalization lists searches that two or more pages each hold a real
// share of. Branded searches are left out when brand names are known: a home
// page and an about page both showing for the brand is normal.
func findCannibalization(stats []queryStat, compactTerms []string, rules InsightRules) []Cannibalized {
	floor := minImpressions(stats, rules)
	out := []Cannibalized{}
	for _, q := range stats {
		if len(q.pages) < 2 || q.impressions < floor || isBranded(q.query, compactTerms) {
			continue
		}
		var competing []CompetingPage
		for _, p := range q.pages {
			if share := p.impressions / q.impressions; share >= rules.MinCompetingShare {
				competing = append(competing, CompetingPage{Page: p.page, Clicks: p.clicks, Impressions: p.impressions, Position: p.position(), Share: share})
			}
		}
		if len(competing) < 2 {
			continue
		}
		slices.SortFunc(competing, func(a, b CompetingPage) int {
			return cmp.Or(cmp.Compare(b.Impressions, a.Impressions), cmp.Compare(a.Page, b.Page))
		})
		leader := slices.MaxFunc(competing, func(a, b CompetingPage) int {
			return cmp.Or(cmp.Compare(a.Clicks, b.Clicks), cmp.Compare(a.Impressions, b.Impressions), cmp.Compare(b.Page, a.Page))
		})
		out = append(out, Cannibalized{Query: q.query, Clicks: q.clicks, Impressions: q.impressions, Leader: leader.Page, Pages: competing})
	}
	slices.SortFunc(out, func(a, b Cannibalized) int {
		return cmp.Or(cmp.Compare(b.Clicks, a.Clicks), cmp.Compare(b.Impressions, a.Impressions), cmp.Compare(a.Query, b.Query))
	})
	return out
}

func median(values []float64) float64 {
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

// positionBucket rounds an average position to the whole place it ranks at.
// Beyond page one the CTR is too small to compare, so those get 0.
func positionBucket(position float64) int {
	bucket := int(math.Round(position))
	if bucket < 1 || bucket > 10 {
		return 0
	}
	return bucket
}

// findCTRGaps compares each search with the site's own typical CTR at the
// same position (the median of the site's other searches there), never with a
// published average. Branded searches are left out of both sides when brand
// names are known, because they click far above the rest.
func findCTRGaps(stats []queryStat, compactTerms []string, rules InsightRules) []CTRGap {
	floor := minImpressions(stats, rules)
	var judged []queryStat
	buckets := map[int][]float64{}
	for _, q := range stats {
		if q.impressions < floor || isBranded(q.query, compactTerms) {
			continue
		}
		bucket := positionBucket(q.position())
		if bucket == 0 {
			continue
		}
		judged = append(judged, q)
		buckets[bucket] = append(buckets[bucket], q.ctr())
	}
	expected := map[int]float64{}
	for bucket, ctrs := range buckets {
		if len(ctrs) >= rules.MinBucketQueries {
			expected[bucket] = median(ctrs)
		}
	}
	out := []CTRGap{}
	for _, q := range judged {
		want, ok := expected[positionBucket(q.position())]
		if !ok || q.ctr() >= want*rules.CTRGapRatio {
			continue
		}
		possible := q.impressions*want - q.clicks
		if possible <= 0 {
			continue
		}
		top := slices.MaxFunc(q.pages, func(a, b pageStat) int {
			return cmp.Or(cmp.Compare(a.impressions, b.impressions), cmp.Compare(b.page, a.page))
		})
		out = append(out, CTRGap{Query: q.query, Page: top.page, Clicks: q.clicks, Impressions: q.impressions, CTR: q.ctr(), Position: q.position(), ExpectedCTR: want, PossibleClicks: possible})
	}
	slices.SortFunc(out, func(a, b CTRGap) int {
		return cmp.Or(cmp.Compare(b.PossibleClicks, a.PossibleClicks), cmp.Compare(a.Query, b.Query))
	})
	return out
}

// findDecay lists pages whose clicks fell by at least DecayDrop in each of
// two consecutive windows. oldest, middle and newest hold single-key page rows.
func findDecay(oldest, middle, newest []SearchRow, rules InsightRules) []Decay {
	clicksByPage := func(rows []SearchRow) map[string]float64 {
		out := make(map[string]float64, len(rows))
		for _, row := range rows {
			if len(row.Keys) == 1 && row.Keys[0] != "" {
				out[row.Keys[0]] = row.Clicks
			}
		}
		return out
	}
	mid, now := clicksByPage(middle), clicksByPage(newest)
	keep := 1 - rules.DecayDrop
	out := []Decay{}
	for _, row := range oldest {
		if len(row.Keys) != 1 || row.Keys[0] == "" || row.Clicks < rules.DecayMinClicks {
			continue
		}
		page, a := row.Keys[0], row.Clicks
		b, c := mid[page], now[page]
		if b > a*keep || c > b*keep {
			continue
		}
		out = append(out, Decay{Page: page, Clicks: [3]float64{a, b, c}, LostClicks: a - c})
	}
	slices.SortFunc(out, func(x, y Decay) int {
		return cmp.Or(cmp.Compare(y.LostClicks, x.LostClicks), cmp.Compare(x.Page, y.Page))
	})
	return out
}

// basisRank puts clicks really lost before estimates, and estimates before
// searches that are only shared.
func basisRank(basis string) int {
	switch basis {
	case BasisMeasured:
		return 0
	case BasisEstimated:
		return 1
	default:
		return 2
	}
}

const (
	topActionCount   = 3
	topActionPerKind = 2
)

// pickActions chooses the few things worth doing first: real losses before
// estimates, biggest first, and no more than two of one kind so the list is
// not one problem repeated.
func pickActions(losing []PageChange, decaying []Decay, gaps []CTRGap, cannibal []Cannibalized) []Action {
	var all []Action
	for _, c := range losing {
		if c.Severity == SeverityUrgent || c.Severity == SeverityWatch {
			all = append(all, Action{Kind: ActionLosingPage, Subject: c.Page, Page: c.Page, Impact: c.PrevClicks - c.Clicks, Basis: BasisMeasured})
		}
	}
	for _, d := range decaying {
		all = append(all, Action{Kind: ActionDecayingPage, Subject: d.Page, Page: d.Page, Impact: d.LostClicks, Basis: BasisMeasured})
	}
	for _, g := range gaps {
		all = append(all, Action{Kind: ActionCTRGap, Subject: g.Query, Page: g.Page, Impact: g.PossibleClicks, Basis: BasisEstimated})
	}
	for _, c := range cannibal {
		all = append(all, Action{Kind: ActionCannibalization, Subject: c.Query, Page: c.Leader, Impact: c.Clicks, Basis: BasisContested})
	}
	all = slices.DeleteFunc(all, func(a Action) bool { return a.Impact <= 0 })
	slices.SortStableFunc(all, func(a, b Action) int {
		return cmp.Or(cmp.Compare(basisRank(a.Basis), basisRank(b.Basis)), cmp.Compare(b.Impact, a.Impact), cmp.Compare(a.Subject, b.Subject))
	})
	perKind := map[string]int{}
	out := []Action{}
	for _, a := range all {
		if perKind[a.Kind] >= topActionPerKind {
			continue
		}
		perKind[a.Kind]++
		out = append(out, a)
		if len(out) == topActionCount {
			break
		}
	}
	return out
}
