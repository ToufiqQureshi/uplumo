package gsc

import (
	"context"
	"fmt"
	"math"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

func qp(query, page string, clicks, impressions, position float64) SearchRow {
	return SearchRow{Keys: []string{query, page}, Clicks: clicks, Impressions: impressions, Position: position}
}

func one(key string, clicks, impressions float64) SearchRow {
	return SearchRow{Keys: []string{key}, Clicks: clicks, Impressions: impressions}
}

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestAggregateQueriesSumsPagesAndSkipsBadRows(t *testing.T) {
	rows := []SearchRow{
		qp("red shoes", "/a", 5, 100, 4),
		qp("red shoes", "/b", 3, 100, 8),
		qp("जूते", "/c", 1, 10, 2), // unicode query
		{Keys: []string{"only-one-key"}, Clicks: 9, Impressions: 9},
		qp("", "/x", 1, 1, 1),
		qp("q", "", 1, 1, 1),
	}
	got := aggregateQueries(rows)
	if len(got) != 2 {
		t.Fatalf("queries = %d, want 2: %+v", len(got), got)
	}
	red := got[0]
	if red.query != "red shoes" || red.clicks != 8 || red.impressions != 200 || len(red.pages) != 2 {
		t.Fatalf("red shoes = %+v", red)
	}
	if !near(red.position(), 6) { // (4*100 + 8*100) / 200
		t.Fatalf("position = %v, want impression-weighted 6", red.position())
	}
	if got[1].query != "जूते" {
		t.Fatalf("unicode query lost: %+v", got[1])
	}
	if len(aggregateQueries(nil)) != 0 {
		t.Fatal("nil rows must give no queries")
	}
}

func TestBrandTermsMatchLoosely(t *testing.T) {
	terms := normalizeBrandTerms([]string{" My Brand ", "my-brand", "x", "", "Café"})
	if !slices.Equal(terms, []string{"mybrand", "café"}) {
		t.Fatalf("terms = %v (duplicates and one-letter terms must go)", terms)
	}
	tests := map[string]bool{
		"my brand login":      true,
		"MYBRAND pricing":     true,
		"my-brand reviews":    true,
		"best CAFÉ near me":   true,
		"red shoes":           false,
		"unrelated brand new": false,
	}
	for query, want := range tests {
		if got := isBranded(query, terms); got != want {
			t.Errorf("isBranded(%q) = %v, want %v", query, got, want)
		}
	}
	if isBranded("anything", nil) {
		t.Fatal("no brand terms means nothing is branded")
	}
}

func TestBuildSegments(t *testing.T) {
	rules := DefaultInsightRules()
	stats := aggregateQueries([]SearchRow{
		qp("acme login", "/", 50, 100, 1),
		qp("buy red running shoes online", "/shoes", 2, 100, 7),
		qp("shoes", "/shoes", 3, 300, 5),
	})
	t.Run("without brand names there is no branded split", func(t *testing.T) {
		segments := buildSegments(stats, nil, rules)
		kinds := []string{}
		for _, s := range segments {
			kinds = append(kinds, s.Kind)
		}
		if !slices.Equal(kinds, []string{SegmentAll, SegmentLongTail}) {
			t.Fatalf("kinds = %v", kinds)
		}
		if segments[0].Queries != 3 || segments[0].Clicks != 55 || segments[0].Impressions != 500 {
			t.Fatalf("all = %+v", segments[0])
		}
		if segments[1].Queries != 1 || segments[1].Clicks != 2 {
			t.Fatalf("long tail = %+v", segments[1])
		}
	})
	t.Run("with brand names it splits", func(t *testing.T) {
		segments := buildSegments(stats, normalizeBrandTerms([]string{"acme"}), rules)
		byKind := map[string]Segment{}
		for _, s := range segments {
			byKind[s.Kind] = s
		}
		if byKind[SegmentBranded].Clicks != 50 || byKind[SegmentNonBrand].Clicks != 5 || byKind[SegmentNonBrand].Queries != 2 {
			t.Fatalf("segments = %+v", segments)
		}
		if !near(byKind[SegmentNonBrand].CTR, 5.0/400) {
			t.Fatalf("non-brand ctr = %v", byKind[SegmentNonBrand].CTR)
		}
	})
	t.Run("empty input does not divide by zero", func(t *testing.T) {
		for _, s := range buildSegments(nil, nil, rules) {
			if s.CTR != 0 || s.Position != 0 || s.Queries != 0 {
				t.Fatalf("empty segment = %+v", s)
			}
		}
	})
}

func TestFindCannibalization(t *testing.T) {
	rules := DefaultInsightRules()
	stats := aggregateQueries([]SearchRow{
		qp("red shoes", "/a", 5, 100, 4),
		qp("red shoes", "/b", 3, 80, 6),
		qp("blue shoes", "/a", 9, 95, 2), // /b holds only 5%: not competing
		qp("blue shoes", "/b", 0, 5, 30),
		qp("tiny", "/a", 1, 4, 3), // below the impressions floor
		qp("tiny", "/b", 0, 4, 4),
		qp("acme store", "/a", 5, 50, 1), // branded: home vs about is normal
		qp("acme store", "/b", 5, 50, 1),
	})
	t.Run("flags only searches with two real competitors", func(t *testing.T) {
		got := findCannibalization(stats, nil, rules)
		queries := []string{}
		for _, c := range got {
			queries = append(queries, c.Query)
		}
		// "acme store" is flagged here because no brand names were given.
		if !slices.Equal(queries, []string{"acme store", "red shoes"}) {
			t.Fatalf("queries = %v", queries)
		}
	})
	t.Run("leader has the most clicks and branded searches are skipped", func(t *testing.T) {
		got := findCannibalization(stats, normalizeBrandTerms([]string{"acme"}), rules)
		if len(got) != 1 || got[0].Query != "red shoes" || got[0].Leader != "/a" {
			t.Fatalf("got = %+v", got)
		}
		if len(got[0].Pages) != 2 || got[0].Pages[0].Page != "/a" || !near(got[0].Pages[0].Share, 100.0/180) {
			t.Fatalf("pages = %+v", got[0].Pages)
		}
	})
	t.Run("a single page never cannibalizes", func(t *testing.T) {
		if got := findCannibalization(aggregateQueries([]SearchRow{qp("x y", "/a", 1, 500, 1)}), nil, rules); len(got) != 0 {
			t.Fatalf("got = %+v", got)
		}
	})
}

func gapRows(count int, ctr float64) []SearchRow {
	rows := make([]SearchRow, 0, count)
	for i := range count {
		rows = append(rows, qp(fmt.Sprintf("query %d", i), "/p", ctr*100, 100, 3))
	}
	return rows
}

func TestFindCTRGapsUsesTheSitesOwnCTR(t *testing.T) {
	rules := DefaultInsightRules()
	rows := append(gapRows(10, 0.10), qp("weak query", "/weak", 4, 200, 3))
	got := findCTRGaps(aggregateQueries(rows), nil, rules)
	if len(got) != 1 || got[0].Query != "weak query" || got[0].Page != "/weak" {
		t.Fatalf("got = %+v", got)
	}
	if !near(got[0].ExpectedCTR, 0.10) || !near(got[0].PossibleClicks, 200*0.10-4) {
		t.Fatalf("expected ctr / possible clicks = %v / %v", got[0].ExpectedCTR, got[0].PossibleClicks)
	}

	t.Run("too few searches at a position gives no verdict", func(t *testing.T) {
		few := append(gapRows(4, 0.10), qp("weak query", "/weak", 4, 200, 3))
		if got := findCTRGaps(aggregateQueries(few), nil, rules); len(got) != 0 {
			t.Fatalf("got = %+v, a baseline of 5 searches is not trusted", got)
		}
	})
	t.Run("branded searches are neither judged nor part of the baseline", func(t *testing.T) {
		withBrand := append(rows, qp("acme login", "/", 1, 300, 3))
		got := findCTRGaps(aggregateQueries(withBrand), normalizeBrandTerms([]string{"acme"}), rules)
		if len(got) != 1 || got[0].Query != "weak query" {
			t.Fatalf("got = %+v", got)
		}
	})
	t.Run("page two and beyond is not compared", func(t *testing.T) {
		far := []SearchRow{}
		for i := range 12 {
			far = append(far, qp(fmt.Sprintf("far %d", i), "/p", 5, 100, 14))
		}
		far = append(far, qp("farther", "/p", 0, 500, 14))
		if got := findCTRGaps(aggregateQueries(far), nil, rules); len(got) != 0 {
			t.Fatalf("got = %+v", got)
		}
	})
	t.Run("a search already above the site's typical CTR is not a gap", func(t *testing.T) {
		if got := findCTRGaps(aggregateQueries(gapRows(12, 0.10)), nil, rules); len(got) != 0 {
			t.Fatalf("got = %+v", got)
		}
	})
}

func TestFindDecay(t *testing.T) {
	rules := DefaultInsightRules()
	oldest := []SearchRow{one("/falling", 100, 1000), one("/steady", 100, 1000), one("/gone", 50, 500), one("/small", 5, 50), one("/recovered", 100, 1000), one("", 99, 1)}
	middle := []SearchRow{one("/falling", 70, 900), one("/steady", 95, 900), one("/gone", 20, 200), one("/small", 1, 10), one("/recovered", 50, 500)}
	newest := []SearchRow{one("/falling", 40, 800), one("/steady", 90, 900), one("/recovered", 90, 900)}
	got := findDecay(oldest, middle, newest, rules, windowCuts{})
	pages := []string{}
	for _, d := range got {
		pages = append(pages, d.Page)
	}
	// /gone is missing from the newest window, so it lost every click.
	if !slices.Equal(pages, []string{"/falling", "/gone"}) {
		t.Fatalf("pages = %v", pages)
	}
	if got[0].LostClicks != 60 || got[0].Clicks != [3]float64{100, 70, 40} || got[1].LostClicks != 50 {
		t.Fatalf("got = %+v", got)
	}
	if len(findDecay(nil, nil, nil, rules, windowCuts{})) != 0 {
		t.Fatal("no rows, no decay")
	}
}

func TestPickActionsOrdersByBasisCapsPerKindAndSkipsZero(t *testing.T) {
	losing := []PageChange{
		{Page: "/urgent", ChangeMetrics: ChangeMetrics{Clicks: 10, PrevClicks: 50, Severity: SeverityUrgent}},
		{Page: "/minor", ChangeMetrics: ChangeMetrics{Clicks: 9, PrevClicks: 10, Severity: SeverityMinor}},
	}
	decaying := []Decay{{Page: "/d1", LostClicks: 90}, {Page: "/d2", LostClicks: 80}, {Page: "/d3", LostClicks: 70}, {Page: "/zero", LostClicks: 0}}
	gaps := []CTRGap{{Query: "gap", Page: "/g", PossibleClicks: 500}}
	cannibal := []Cannibalized{{Query: "shared", Leader: "/c", Clicks: 900}}

	got := pickActions(losing, decaying, gaps, cannibal)
	subjects := []string{}
	for _, a := range got {
		subjects = append(subjects, a.Subject)
	}
	// Measured losses first (90, 80 from decay; then the urgent page at 40),
	// even though an estimate (500) and a shared search (900) look bigger.
	if !slices.Equal(subjects, []string{"/d1", "/d2", "/urgent"}) {
		t.Fatalf("subjects = %v", subjects)
	}
	if got[0].Basis != BasisMeasured || got[2].Kind != ActionLosingPage {
		t.Fatalf("got = %+v", got)
	}
	if got := pickActions(nil, nil, nil, nil); len(got) != 0 {
		t.Fatalf("got = %+v", got)
	}
	onlyEstimates := pickActions(nil, nil, gaps, cannibal)
	if len(onlyEstimates) != 2 || onlyEstimates[0].Basis != BasisEstimated || onlyEstimates[1].Basis != BasisContested {
		t.Fatalf("onlyEstimates = %+v", onlyEstimates)
	}
}

type insightsStub struct {
	mu       sync.Mutex
	requests []SearchRequest
	rows     func(SearchRequest) []SearchRow
}

func (s *insightsStub) QuerySearchAnalytics(_ context.Context, _ string, request SearchRequest) ([]SearchRow, error) {
	s.mu.Lock()
	s.requests = append(s.requests, request)
	s.mu.Unlock()
	if err := ValidateSearchRequest(request); err != nil {
		return nil, err // what the real client would do
	}
	return s.rows(request), nil
}

type missingConnection struct{}

func (missingConnection) GetByProjectID(context.Context, string, string) (Connection, error) {
	return Connection{}, ErrConnectionNotFound
}

func insightsService(stub *insightsStub) *Service {
	return &Service{
		Connections: performanceConnectionStub{connection: Connection{SiteURL: "https://example.test/"}},
		NewClient:   func(string, string) SearchClient { return stub },
		Now:         func() time.Time { return time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC) },
	}
}

// routeRows answers by what was asked: the dimensions and the window start of
// the default 28-day range ending 2026-05-25 (now, middle, oldest windows).
func routeRows(request SearchRequest) []SearchRow {
	const now, middle, oldest = "2026-04-27", "2026-03-29", "2026-02-28"
	switch strings.Join(request.Dimensions, ",") {
	case "query,page":
		return []SearchRow{qp("red shoes", "/a", 5, 100, 4), qp("red shoes", "/b", 3, 80, 6), qp("a b c d", "/a", 1, 50, 9)}
	case "page":
		switch request.StartDate {
		case now:
			return []SearchRow{one("/a", 10, 100)}
		case middle:
			return []SearchRow{one("/a", 50, 300)}
		case oldest:
			return []SearchRow{one("/a", 100, 600)}
		}
	case "query":
		if request.StartDate == now {
			return []SearchRow{one("red shoes", 8, 180)}
		}
		return []SearchRow{one("red shoes", 30, 400)}
	}
	return nil
}

func TestGetInsightsFindsWhatIsWorthDoing(t *testing.T) {
	stub := &insightsStub{rows: routeRows}
	got, err := insightsService(stub).GetInsights(context.Background(), "org-1", "project-1", InsightsInput{BrandTerms: []string{"acme"}})
	if err != nil {
		t.Fatal(err)
	}
	// At most six free reads for one analysis, however many rows come back.
	if n := len(stub.requests); n != 6 {
		t.Fatalf("provider reads = %d, want 6", n)
	}
	if !got.Connected || !got.Coverage.DecayAvailable || !got.Coverage.ComparisonAvailable || !got.Coverage.BrandTermsUsed || got.Coverage.Truncated {
		t.Fatalf("coverage = %+v", got.Coverage)
	}
	if len(got.Cannibalization) != 1 || got.Cannibalization[0].Leader != "/a" {
		t.Fatalf("cannibalization = %+v", got.Cannibalization)
	}
	if len(got.Decaying) != 1 || got.Decaying[0].LostClicks != 90 {
		t.Fatalf("decaying = %+v", got.Decaying)
	}
	if len(got.LosingPages) != 0 {
		// /a fell 50 -> 10, which is a real loss, so it must be listed.
		if got.LosingPages[0].Page != "/a" || got.LosingPages[0].Severity != SeverityUrgent {
			t.Fatalf("losing = %+v", got.LosingPages)
		}
	} else {
		t.Fatal("the page that fell from 50 to 10 clicks must be listed")
	}
	if len(got.QueryLosses) != 1 || got.QueryLosses[0].Query != "red shoes" || len(got.QueryGains) != 0 {
		t.Fatalf("query changes = %+v / %+v", got.QueryLosses, got.QueryGains)
	}
	if len(got.Actions) == 0 || got.Actions[0].Kind != ActionDecayingPage {
		t.Fatalf("actions = %+v", got.Actions)
	}
	for _, r := range stub.requests {
		if r.RowLimit > maxReportRowLimit {
			t.Fatalf("request over the provider bound: %+v", r)
		}
	}
}

func TestGetInsightsRowCapIsReportedAsTruncated(t *testing.T) {
	stub := &insightsStub{rows: func(r SearchRequest) []SearchRow {
		if len(r.Dimensions) == 2 {
			rows := make([]SearchRow, r.RowLimit)
			for i := range rows {
				rows[i] = qp(fmt.Sprintf("q%d", i), "/p", 1, 10, 3)
			}
			return rows
		}
		return nil
	}}
	got, err := insightsService(stub).GetInsights(context.Background(), "o", "p", InsightsInput{})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Coverage.Truncated || got.Coverage.QueryPageRows != insightQueryPageLimit {
		t.Fatalf("coverage = %+v", got.Coverage)
	}
}

func TestGetInsightsSkipsWindowsOutsideSixteenMonths(t *testing.T) {
	tests := []struct {
		name              string
		start, end        string
		reads             int
		comparison, decay bool
	}{
		{"all three windows fit", "2026-04-27", "2026-05-25", 6, true, true},
		{"third window falls off", "2025-12-01", "2026-05-25", 5, true, false},
		{"previous window falls off", "2025-06-01", "2026-05-25", 1, false, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			stub := &insightsStub{rows: func(SearchRequest) []SearchRow { return nil }}
			got, err := insightsService(stub).GetInsights(context.Background(), "o", "p", InsightsInput{PerformanceInput: PerformanceInput{StartDate: test.start, EndDate: test.end}})
			if err != nil {
				t.Fatal(err)
			}
			if len(stub.requests) != test.reads || got.Coverage.ComparisonAvailable != test.comparison || got.Coverage.DecayAvailable != test.decay {
				t.Fatalf("reads=%d coverage=%+v", len(stub.requests), got.Coverage)
			}
			if got.LosingPages == nil || got.Decaying == nil || got.QueryLosses == nil || got.Segments == nil || got.Actions == nil {
				t.Fatalf("lists must be empty arrays, not null: %+v", got)
			}
		})
	}
}

func TestGetInsightsRegexFilterReachesEveryRead(t *testing.T) {
	stub := &insightsStub{rows: func(SearchRequest) []SearchRow { return nil }}
	input := InsightsInput{PerformanceInput: PerformanceInput{QueryFilter: &TextFilter{Operator: "includingRegex", Expression: `^(how|what)\b`}}}
	if _, err := insightsService(stub).GetInsights(context.Background(), "o", "p", input); err != nil {
		t.Fatal(err)
	}
	for _, r := range stub.requests {
		filters := r.DimensionFilterGroups[0].Filters
		if len(filters) != 1 || filters[0].Operator != "includingRegex" || filters[0].Dimension != "query" {
			t.Fatalf("filter missing on %+v", r)
		}
	}
}

func TestGetInsightsNotConnected(t *testing.T) {
	s := &Service{Connections: missingConnection{}, NewClient: func(string, string) SearchClient { return &insightsStub{} }}
	got, err := s.GetInsights(context.Background(), "o", "p", InsightsInput{})
	if err != nil || got.Connected {
		t.Fatalf("got = %+v, err = %v", got, err)
	}
}

func TestGetInsightsRejectsBadBrandNamesBeforeAnyRead(t *testing.T) {
	tests := map[string][]string{
		"too many":        make([]string, maxBrandTerms+1),
		"control char":    {"bad\x00name"},
		"too long":        {strings.Repeat("ब", maxBrandTermRune+1)},
		"newline in term": {"a\nb"},
	}
	for name, terms := range tests {
		t.Run(name, func(t *testing.T) {
			stub := &insightsStub{rows: func(SearchRequest) []SearchRow { return nil }}
			_, err := insightsService(stub).GetInsights(context.Background(), "o", "p", InsightsInput{BrandTerms: terms})
			if !IsValidationError(err) {
				t.Fatalf("err = %v, want a validation error", err)
			}
			if len(stub.requests) != 0 {
				t.Fatal("a rejected request must not read from Google")
			}
		})
	}
	stub := &insightsStub{rows: func(SearchRequest) []SearchRow { return nil }}
	if _, err := insightsService(stub).GetInsights(context.Background(), "o", "p", InsightsInput{BrandTerms: []string{strings.Repeat("ब", maxBrandTermRune)}}); err != nil {
		t.Fatalf("100 unicode characters are allowed: %v", err)
	}
}

func TestRegexFiltersAreValidatedBeforeTheyReachGoogle(t *testing.T) {
	good := PerformanceInput{QueryFilter: &TextFilter{Operator: "includingRegex", Expression: `^(buy|get)\s`}}
	if err := validatePerformanceInput(&good); err != nil {
		t.Fatal(err)
	}
	for _, filter := range []TextFilter{
		{Operator: "includingRegex", Expression: `(`},
		{Operator: "excludingRegex", Expression: `[`},
		{Operator: "startsWith", Expression: "x"},
		{Operator: "contains", Expression: strings.Repeat("x", 1025)},
	} {
		input := PerformanceInput{PageFilter: &filter}
		if err := validatePerformanceInput(&input); err == nil {
			t.Errorf("%+v must be rejected", filter)
		}
	}
	for _, op := range []string{"includingRegex", "excludingRegex"} {
		request, err := BuildSearchAnalyticsRequest(PerformanceInput{QueryFilter: &TextFilter{Operator: op, Expression: "^a"}}, []string{"query"}, 10, 0, time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC))
		if err != nil || ValidateSearchRequest(request) != nil {
			t.Fatalf("%s request refused: %v", op, err)
		}
	}
}

func TestInsightsInputDecoding(t *testing.T) {
	decode := func(body string) error {
		req := httptest.NewRequestWithContext(context.Background(), "POST", "/", strings.NewReader(body))
		var input InsightsInput
		return decodePerformance(httptest.NewRecorder(), req, &input)
	}
	for _, body := range []string{`{}`, `{"brandTerms":[]}`, `{"brandTerms":["Acme","आइबी"],"dateRange":"last_28_days"}`} {
		if err := decode(body); err != nil {
			t.Errorf("%s rejected: %v", body, err)
		}
	}
	for _, body := range []string{`{"brandTerms":null}`, `{"brandTerm":["x"]}`, `{"brandTerms":"acme"}`} {
		if err := decode(body); err == nil {
			t.Errorf("%s must be rejected", body)
		}
	}
}

func TestAPageMissingFromACutReadIsUnknownNotZero(t *testing.T) {
	rules := DefaultInsightRules()
	oldest := []SearchRow{one("/steady", 100, 1000)}
	// /steady is missing from the newer windows only because they were read up
	// to their row limit; it never lost anything.
	if got := findDecay(oldest, nil, nil, rules, windowCuts{middle: true, newest: true}); len(got) != 0 {
		t.Fatalf("decay = %+v; an unseen page must not be reported as lost", got)
	}
	if got := findDecay(oldest, nil, nil, rules, windowCuts{middle: true}); len(got) != 0 {
		t.Fatalf("decay = %+v; one cut window is enough to make it unknown", got)
	}
	// Read in full, the same absence is real: no clicks at all.
	if got := findDecay(oldest, nil, nil, rules, windowCuts{}); len(got) != 1 || got[0].LostClicks != 100 {
		t.Fatalf("decay = %+v; a page missing from a full read lost every click", got)
	}
}

func TestInsightsNeverCallACutReadsAbsenceALossOrANewSearch(t *testing.T) {
	// Every page and search read comes back full (at the row limit), and the
	// pages and searches that matter are in the oldest and previous windows
	// only. Nothing here is a real loss, gain or decay.
	filler := func(prefix string) []SearchRow {
		rows := make([]SearchRow, 0, insightPageLimit)
		for i := range insightPageLimit {
			rows = append(rows, one(fmt.Sprintf("%s-%d", prefix, i), 1, 10))
		}
		return rows
	}
	stub := &insightsStub{rows: func(r SearchRequest) []SearchRow {
		switch strings.Join(r.Dimensions, ",") {
		case "query,page":
			return []SearchRow{qp("red shoes", "/a", 5, 100, 4)}
		case "page":
			rows := filler("page")
			switch r.StartDate {
			case "2026-02-28", "2026-03-29": // oldest and previous: /steady is there
				return append([]SearchRow{one("/steady", 100, 1000)}, rows[:insightPageLimit-1]...)
			}
			return rows // current: cut, /steady is below the cut-off
		case "query":
			rows := filler("q")
			if r.StartDate == "2026-04-27" { // current has a search the previous read cut off
				return append([]SearchRow{one("steady search", 50, 500)}, rows[:insightPageLimit-1]...)
			}
			return rows
		}
		return nil
	}}
	got, err := insightsService(stub).GetInsights(context.Background(), "o", "p", InsightsInput{})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Coverage.PageReadsCut {
		t.Fatal("coverage must say a read stopped at its row limit")
	}
	for _, c := range got.LosingPages {
		if c.Page == "/steady" {
			t.Fatalf("/steady reported as lost: %+v", c)
		}
	}
	for _, d := range got.Decaying {
		if d.Page == "/steady" {
			t.Fatalf("/steady reported as decaying: %+v", d)
		}
	}
	for _, c := range got.QueryGains {
		if c.Query == "steady search" {
			t.Fatalf("a search missing from a cut previous read reported as new: %+v", c)
		}
	}
	for _, a := range got.Actions {
		if a.Subject == "/steady" {
			t.Fatalf("/steady became a top action: %+v", a)
		}
	}
}
