package sitereport

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/ga4"
	"github.com/toufiqqureshi/uplumo/backend/internal/google"
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

type ga4TestConnection struct{}

func (ga4TestConnection) GetByProjectID(context.Context, string) (ga4.Connection, error) {
	return ga4.Connection{PropertyID: "properties/123", PropertyTimeZone: "UTC", ConnectedByUserID: "user"}, nil
}

type ga4TestProvider struct {
	period   Period
	requests []ga4.APIRequest
}

func (f *ga4TestProvider) DoJSON(_ context.Context, request google.APIRequest) error {
	body := request.Body.(ga4.APIRequest)
	f.requests = append(f.requests, body)
	row := ga4.ProviderRow{}
	for _, dimension := range body.Dimensions {
		value := map[string]string{"date": "20261103", "sessionDefaultChannelGroup": "Paid Search", "hostName": "example.com",
			"landingPage": "/pricing?internal=secret", "eventName": "purchase"}[dimension.Name]
		row.DimensionValues = append(row.DimensionValues, ga4.Value{Value: value})
	}
	for _, metric := range body.Metrics {
		value := 48
		if body.DateRanges[0].StartDate == f.period.PrevStart.Format(day) {
			value = 31
		}
		if metric.Name == "engagementRate" || metric.Name == "sessionKeyEventRate" {
			row.MetricValues = append(row.MetricValues, ga4.Value{Value: "0.75"})
		} else {
			row.MetricValues = append(row.MetricValues, ga4.Value{Value: strconv.Itoa(value)})
		}
	}
	*request.Response.(*ga4.ProviderResponse) = ga4.ProviderResponse{
		DimensionHeaders: body.Dimensions, MetricHeaders: body.Metrics, RowCount: 1, Rows: []ga4.ProviderRow{row},
	}
	return nil
}

func TestGA4SourceReadsSevenFreeReportsAndKeepsDashboardEvidence(t *testing.T) {
	period := periodFor(fixedNow, 28)
	provider := &ga4TestProvider{period: period}
	source := GA4{Service: &ga4.Service{Connections: ga4TestConnection{}, Google: provider, Now: func() time.Time { return fixedNow }}}
	data, err := source.Traffic(t.Context(), "project", period)
	if err != nil {
		t.Fatal(err)
	}
	if data.breakdownError != nil || len(provider.requests) != 7 || len(data.Channels) != 1 || len(data.LandingPages) != 1 || len(data.Events) != 1 {
		t.Fatalf("requests=%d data=%+v", len(provider.requests), data)
	}
	event := data.Events[0]
	if data.Channels[0].Name != "Paid Search" || data.LandingPages[0].Name != "/pricing" || event.Name != "purchase" ||
		event.KeyEvents == nil || *event.KeyEvents != 48 || event.PreviousKeyEvents == nil || *event.PreviousKeyEvents != 31 {
		t.Fatalf("unexpected evidence: %+v", data)
	}
	if provider.requests[3].DimensionFilter != nil || provider.requests[4].DimensionFilter != nil || provider.requests[5].DimensionFilter == nil {
		t.Fatal("channel and landing page breakdowns should span all channels, and key events should be scoped to organic search")
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
