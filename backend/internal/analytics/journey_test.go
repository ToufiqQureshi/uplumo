package analytics

import (
	"fmt"
	"slices"
	"testing"
	"time"
)

func TestDailyAudienceDeduplicatesAndSeparatesUnknown(t *testing.T) {
	ctx := t.Context()
	f := newFixture(ctx, t)
	s := f.addSite(ctx, t, "acme.com")
	other := f.addSite(ctx, t, "other.test")
	d1 := time.Date(2026, 10, 19, 23, 59, 0, 0, time.UTC)
	send := func(site testSite, at time.Time, browser, page string) {
		t.Helper()
		f.svc.now = func() time.Time { return at }
		h := hit(t, site.key, "https://"+map[bool]string{true: "acme.com", false: "other.test"}[site == s]+"/", "", "203.0.113.7", desktopUA)
		h.BrowserID, h.PageviewID, h.SessionID = browser, page, "session-"+page
		if err := f.svc.Collect(ctx, h); err != nil {
			t.Fatal(err)
		}
	}
	for i := range 10 {
		send(s, d1, "browser-a", fmt.Sprint(i))
	}
	// A separate tab/IP is still the same browser; a separate browser on the
	// same network is new. The daily anonymous visitor count is not reused.
	send(s, d1, "browser-b", "b")
	send(s, d1, "", "unknown")
	send(s, d1.Add(2*time.Minute), "browser-a", "next")
	send(other, d1.Add(2*time.Minute), "browser-a", "other")
	got, err := f.svc.Summarize(ctx, s.project, d1, d1.AddDate(0, 0, 2))
	if err != nil {
		t.Fatal(err)
	}
	want := []AudienceDay{{"2026-10-19", 2, 0, 1}, {"2026-10-20", 0, 1, 0}, {"2026-10-21", 0, 0, 0}}
	if !slices.Equal(got.Audience, want) {
		t.Fatalf("daily audience=%+v want %+v", got.Audience, want)
	}
	foreign, err := f.svc.Summarize(ctx, other.project, d1.AddDate(0, 0, 1), d1.AddDate(0, 0, 1))
	if err != nil || foreign.Audience[0].NewBrowsers != 1 {
		t.Fatalf("cross-site recognition: %+v %v", foreign.Audience, err)
	}
	var raw int
	f.queryRow(ctx, t, `SELECT count(*) FROM go_analytics_browsers WHERE browser_hash=$1`, []any{[]byte("browser-a")}, &raw)
	if raw != 0 {
		t.Fatal("raw browser ID persisted")
	}
}

func TestClickDestinationsAndObservedArrivals(t *testing.T) {
	ctx := t.Context()
	f := newFixture(ctx, t)
	s := f.addSite(ctx, t, "acme.com")
	d := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	send := func(at time.Time, h Hit) {
		t.Helper()
		f.svc.now = func() time.Time { return at }
		if err := f.svc.Collect(ctx, h); err != nil {
			t.Fatal(err)
		}
	}
	view := hit(t, s.key, "https://acme.com/services", "", "203.0.113.8", desktopUA)
	view.SessionID, view.PageviewID = "session-a", "origin"
	send(d, view)
	click := view
	click.IsClick, click.EventName, click.EventID, click.Destination = true, "Book", "click-a", "https://ACME.com/book?email=private@example.test#secret"
	send(d.Add(time.Second), click)
	send(d.Add(time.Second), click) // Delivery retry must not double count.
	arrival := view
	arrival.Page, _ = view.Page.Parse("/book")
	arrival.PageviewID = "arrival"
	send(d.Add(2*time.Second), arrival)
	// Wrong tab, host, old origin and delayed pageviews cannot confirm another click.
	miss := click
	miss.EventID, miss.EventName, miss.Destination = "miss", "Other", "https://acme.com/unvisited"
	send(d.Add(3*time.Second), miss)
	wrong := arrival
	wrong.Page, _ = view.Page.Parse("/unvisited")
	wrong.SessionID, wrong.PageviewID = "another-tab", "wrong"
	send(d.Add(4*time.Second), wrong)
	wrongHost := wrong
	wrongHost.Page, _ = view.Page.Parse("https://blog.acme.com/unvisited")
	wrongHost.SessionID, wrongHost.PageviewID = "session-a", "wrong-host"
	send(d.Add(5*time.Second), wrongHost)
	late := wrong
	late.SessionID = "session-a"
	late.PageviewID = "late"
	send(d.Add(31*time.Minute), late)
	external := click
	external.EventID, external.EventName, external.Destination = "external", "Partner", "https://partner.test/book?token=secret#private"
	send(d.Add(32*time.Minute), external)
	unknown := click
	unknown.EventID, unknown.EventName, unknown.Destination = "unknown", "Modal", ""
	send(d.Add(33*time.Minute), unknown)
	r, err := f.svc.Clicks(ctx, s.project, d, d, 2, 0)
	if err != nil || len(r.Rows) != 2 || !r.HasMore {
		t.Fatalf("page one=%+v %v", r, err)
	}
	second, err := f.svc.Clicks(ctx, s.project, d, d, 2, 2)
	if err != nil || len(second.Rows) != 2 || second.HasMore {
		t.Fatalf("page two=%+v %v", second, err)
	}
	all := append(r.Rows, second.Rows...)
	for _, row := range all {
		if row.Clicks != 1 || row.VisitorDays != 1 {
			t.Errorf("duplicate click=%+v", row)
		}
		switch row.Name {
		case "Book":
			if row.Host != "acme.com" || row.Path != "/book" || row.Type != "internal" || row.Arrivals != 1 {
				t.Errorf("internal=%+v", row)
			}
		case "Other":
			if row.Arrivals != 0 {
				t.Errorf("false arrival=%+v", row)
			}
		case "Partner":
			if row.Host != "partner.test" || row.Path != "/book" || row.Type != "external" || row.Arrivals != 0 {
				t.Errorf("external=%+v", row)
			}
		case "Modal":
			if row.Type != "unknown" || row.Host != "" || row.Path != "" {
				t.Errorf("unknown=%+v", row)
			}
		default:
			t.Fatalf("unexpected click=%+v", row)
		}
	}
	other := f.addSite(ctx, t, "acme.com")
	empty, err := f.svc.Clicks(ctx, other.project, d, d, 20, 0)
	if err != nil || len(empty.Rows) != 0 {
		t.Fatalf("foreign report=%+v %v", empty, err)
	}
	var leaked bool
	f.queryRow(ctx, t, `SELECT EXISTS(SELECT 1 FROM go_analytics_custom_events WHERE destination_path LIKE '%secret%' OR destination_path LIKE '%email=%')`, nil, &leaked)
	if leaked {
		t.Fatal("sensitive destination persisted")
	}
}
