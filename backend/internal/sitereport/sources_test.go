package sitereport

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/gsc"
)

type fakeConnections struct{ err error }

func (f fakeConnections) GetByProjectID(context.Context, string, string) (gsc.Connection, error) {
	return gsc.Connection{SiteURL: "sc-domain:example.com", ConnectedByUserID: "u"}, f.err
}

// fakeGoogle answers Search Console by dimension and counts every read.
type fakeGoogle struct {
	mu       sync.Mutex
	requests []gsc.SearchRequest
	// emptyBefore answers nothing for ranges that end before this day, like
	// a site that was not in Search yet.
	emptyBefore string
}

func (f *fakeGoogle) QuerySearchAnalytics(_ context.Context, _ string, r gsc.SearchRequest) ([]gsc.SearchRow, error) {
	f.mu.Lock()
	f.requests = append(f.requests, r)
	f.mu.Unlock()
	if r.EndDate < f.emptyBefore {
		return nil, nil
	}
	switch {
	case slices.Equal(r.Dimensions, []string{"date"}):
		return []gsc.SearchRow{{Keys: []string{r.EndDate}, Clicks: 10, Impressions: 100, Position: 8}, {Keys: []string{r.StartDate}, Clicks: 5, Impressions: 50, Position: 9}}, nil
	case slices.Equal(r.Dimensions, []string{"query"}):
		return []gsc.SearchRow{{Keys: []string{"small"}, Clicks: 1, Impressions: 200}, {Keys: []string{"big"}, Clicks: 9, Impressions: 90}, {Keys: []string{""}, Clicks: 99}}, nil
	case slices.Equal(r.Dimensions, []string{"page"}):
		return []gsc.SearchRow{{Keys: []string{"https://example.com/"}, Clicks: 15, Impressions: 150}}, nil
	case slices.Equal(r.Dimensions, []string{"query", "page"}):
		return []gsc.SearchRow{{Keys: []string{"near", "https://example.com/n"}, Impressions: 400, Position: 7}, {Keys: []string{"top", "https://example.com/"}, Impressions: 900, Position: 1.2}, {Keys: []string{"far", "https://example.com/f"}, Impressions: 50, Position: 45}}, nil
	}
	return nil, nil
}

func TestGSCSourceReadsAtMostTenFreeRequests(t *testing.T) {
	google := &fakeGoogle{}
	svc := &gsc.Service{Connections: fakeConnections{}, NewClient: func(string, string) gsc.SearchClient { return google }, Now: func() time.Time { return fixedNow }}
	p := periodFor(fixedNow, 28)
	got, err := GSC{Service: svc}.Search(t.Context(), "o", "p", p)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(google.requests); n == 0 || n > 10 {
		t.Fatalf("one report made %d Search Console reads", n)
	}
	for _, r := range google.requests {
		if r.EndDate > p.End.Format(day) {
			t.Errorf("read past the final-data day: %+v", r)
		}
	}
	if got.Current.Clicks != 15 || !got.HasPrevious || got.Daily[0].Date != p.Start.Format(day) {
		t.Errorf("totals/daily = %+v %+v", got.Current, got.Daily)
	}
	if len(got.TopQueries) != 2 || got.TopQueries[0].Key != "big" || len(got.TopPages) != 1 {
		t.Errorf("top rows = %+v %+v", got.TopQueries, got.TopPages)
	}
	for _, g := range got.Gains {
		if g.Key == "" {
			t.Errorf("a rising search with no text: %+v", got.Gains)
		}
	}
	if len(got.Opportunities) != 1 || got.Opportunities[0].Query != "near" {
		t.Errorf("opportunities are the striking-distance rows: %+v", got.Opportunities)
	}
}

func TestGSCSourceReportsNotConnected(t *testing.T) {
	svc := &gsc.Service{Connections: fakeConnections{err: gsc.ErrConnectionNotFound}, NewClient: func(string, string) gsc.SearchClient { return &fakeGoogle{} }, Now: func() time.Time { return fixedNow }}
	if _, err := (GSC{Service: svc}).Search(t.Context(), "o", "p", periodFor(fixedNow, 7)); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("err = %v", err)
	}
	if _, err := (GSC{}).Search(t.Context(), "o", "p", periodFor(fixedNow, 7)); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("nil service err = %v", err)
	}
	if _, err := (GA4{}).Traffic(t.Context(), "p", periodFor(fixedNow, 7)); !errors.Is(err, ErrNotConnected) {
		t.Fatalf("nil analytics err = %v", err)
	}
	if got, err := (RankTracking{}).Rankings(t.Context(), "p", "example.com", periodFor(fixedNow, 7)); got != nil || err != nil {
		t.Fatalf("nil rank tracking = %v, %v", got, err)
	}
}

func TestFillDaysDrawsEveryDayOfThePeriod(t *testing.T) {
	p := periodFor(fixedNow, 7)
	third := p.Start.AddDate(0, 0, 2).Format(day)
	got := fillDays(p, []TrafficDay{{Date: p.End.Format(day), Sessions: 3}, {Date: third, Sessions: 1}},
		func(d TrafficDay) string { return d.Date }, func(date string) TrafficDay { return TrafficDay{Date: date} })
	if len(got) != 7 || got[0].Date != p.Start.Format(day) || got[6].Date != p.End.Format(day) {
		t.Fatalf("days = %+v", got)
	}
	if got[2].Sessions != 1 || got[6].Sessions != 3 || got[1].Sessions != 0 {
		t.Errorf("values moved or missing days were not zero: %+v", got)
	}
	if empty := fillDays(p, nil, func(d TrafficDay) string { return d.Date }, func(date string) TrafficDay { return TrafficDay{Date: date} }); len(empty) != 7 {
		t.Errorf("no rows gives %d days", len(empty))
	}
}

func TestGSCSourceFillsMissingDays(t *testing.T) {
	google := &fakeGoogle{}
	svc := &gsc.Service{Connections: fakeConnections{}, NewClient: func(string, string) gsc.SearchClient { return google }, Now: func() time.Time { return fixedNow }}
	got, err := GSC{Service: svc}.Search(t.Context(), "o", "p", periodFor(fixedNow, 28))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Daily) != 28 || got.Daily[1].Clicks != 0 || got.Daily[27].Clicks != 10 {
		t.Errorf("daily = %d rows %+v", len(got.Daily), got.Daily)
	}
}

func TestGSCSourceHasNoComparisonWhenThePreviousPeriodIsEmpty(t *testing.T) {
	p := periodFor(fixedNow, 28)
	google := &fakeGoogle{emptyBefore: p.Start.Format(day)}
	svc := &gsc.Service{Connections: fakeConnections{}, NewClient: func(string, string) gsc.SearchClient { return google }, Now: func() time.Time { return fixedNow }}
	got, err := GSC{Service: svc}.Search(t.Context(), "o", "p", p)
	if err != nil {
		t.Fatal(err)
	}
	if got.HasPrevious || got.Current.Clicks != 15 || len(got.Gains) != 0 {
		t.Errorf("a previous period with no data must not show \"was 0\": %+v", got)
	}
}
