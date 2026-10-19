package analytics

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestEnquirySignalsDoNotInventLeadsOrBrokenForms(t *testing.T) {
	metric := func(visitors, goals int64) EnquiryMetric {
		m := EnquiryMetric{Visitors: visitors, GoalVisitors: goals}
		m.setRate()
		return m
	}
	for _, tc := range []struct {
		name          string
		before, after EnquiryMetric
		observed      bool
		want          string
	}{
		{"unverified goal", metric(100, 0), metric(100, 0), false, "verify_tracking"},
		{"small sample", metric(29, 8), metric(29, 0), true, "low_data"},
		{"traffic fell too", metric(100, 10), metric(40, 0), true, "traffic_changed"},
		{"rates fell on stable traffic", metric(100, 10), metric(110, 4), true, "enquiry_drop"},
		{"tiny baseline goals", metric(100, 2), metric(100, 0), true, "no_goal_events"},
		{"normal variation", metric(100, 10), metric(100, 9), true, "stable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := enquirySignal(tc.before, tc.after, tc.observed); got != tc.want {
				t.Fatalf("signal=%s want=%s", got, tc.want)
			}
		})
	}
	for _, path := range []string{"https://evil.com/a", "//evil.com", "/a?email=private", "/a#fragment", "/a\n", ""} {
		if _, err := enquiryPath(path); err == nil {
			t.Fatal("unsafe page accepted", path)
		}
	}
	for _, name := range []string{"", "bad\nname", string(bytes.Repeat([]byte("a"), 65))} {
		if validGoal(name) {
			t.Fatal("invalid event name accepted", name)
		}
	}
}

func TestEnquiryBaselineUsesOneClockAcrossUTCMidnight(t *testing.T) {
	ctx := t.Context()
	f := newFixture(ctx, t)
	site := f.addSite(ctx, t, "midnight.example")
	if err := f.svc.SaveEnquiryGoal(ctx, site.project, "contact"); err != nil {
		t.Fatal(err)
	}
	beforeMidnight := time.Date(2026, 10, 18, 23, 59, 59, 0, time.UTC)
	calls := 0
	f.svc.now = func() time.Time {
		calls++
		if calls == 1 {
			return beforeMidnight
		}
		return beforeMidnight.Add(2 * time.Second)
	}
	id, err := f.svc.SaveEnquiryChange(ctx, site.project, "/contact", "Improve form copy", "midnight-key")
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("clock called %d times, want one", calls)
	}
	var created, end time.Time
	if err := f.svc.repo.db.QueryRow(ctx, `SELECT created_at,baseline_end FROM go_analytics_enquiry_changes WHERE id=$1`, id).Scan(&created, &end); err != nil {
		t.Fatal(err)
	}
	if !created.Equal(beforeMidnight) || !end.Equal(beforeMidnight.Truncate(24*time.Hour)) {
		t.Fatalf("mismatched baseline clock: created=%v end=%v", created, end)
	}
}
func TestEnquiryEvidenceBaselinesAndProjectIsolation(t *testing.T) {
	ctx := t.Context()
	f := newFixture(ctx, t)
	own := f.addSite(ctx, t, "example.com")
	foreign := f.addSite(ctx, t, "other.com")
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	f.svc.now = func() time.Time { return now }
	if err := f.svc.SaveEnquiryGoal(ctx, own.project, "WhatsApp"); err != nil {
		t.Fatal(err)
	}
	seed := func(project string, at time.Time, offset, goals int) {
		t.Helper()
		for i := range 40 {
			hashSum := sha256.Sum256(fmt.Appendf(nil, "fixture:%d", i+offset))
			hash := hashSum[:]
			f.exec(ctx, t, `INSERT INTO go_analytics_events(site_id,occurred_at,path,channel,device,visitor_hash) SELECT id,$2,'/services','search','mobile',$3 FROM go_analytics_sites WHERE project_id=$1`, project, at, hash)
			if i < goals {
				for range 3 {
					f.exec(ctx, t, `INSERT INTO go_analytics_custom_events(site_id,occurred_at,name,path,visitor_hash) SELECT id,$2,'WhatsApp','/services',$3 FROM go_analytics_sites WHERE project_id=$1`, project, at, hash)
				}
			}
		}
	}
	seed(own.project, now.AddDate(0, 0, -10), 1, 10)
	seed(own.project, now.AddDate(0, 0, -3), 50, 2)
	seed(foreign.project, now.AddDate(0, 0, -3), 100, 40)
	// An event without a pageview must not create another converted visitor.
	f.exec(ctx, t, `INSERT INTO go_analytics_custom_events(site_id,occurred_at,name,path,visitor_hash) SELECT id,$2,'WhatsApp','/services',$3 FROM go_analytics_sites WHERE project_id=$1`, own.project, now.AddDate(0, 0, -3), bytes.Repeat([]byte{200}, 32))
	report, err := f.svc.Enquiries(ctx, own.project)
	if err != nil || len(report.Pages) != 1 {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	p := report.Pages[0]
	if p.Before.Visitors != 40 || p.Before.GoalVisitors != 10 || p.After.Visitors != 40 || p.After.GoalVisitors != 2 || p.Signal != "enquiry_drop" || !report.GoalObserved {
		t.Fatalf("incorrect evidence: %+v", report)
	}
	var wg sync.WaitGroup
	ids := make([]string, 5)
	for i := range ids {
		wg.Go(func() {
			var err error
			ids[i], err = f.svc.SaveEnquiryChange(ctx, own.project, "/services", "Make enquiry button visible on mobile", "same-change-key")
			if err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	for _, id := range ids {
		if id == "" || id != ids[0] {
			t.Fatal("duplicate change request", ids)
		}
	}
	comparison, err := f.svc.CompareEnquiryChange(ctx, own.project, ids[0])
	if err != nil || comparison.Ready || comparison.Change.Baseline.GoalVisitors != 2 || comparison.Change.Baseline.Visitors != 40 {
		t.Fatalf("baseline=%+v err=%v", comparison, err)
	}
	if _, err = f.svc.CompareEnquiryChange(ctx, foreign.project, ids[0]); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("foreign baseline read", err)
	}
	if err = f.svc.DeleteEnquiryChange(ctx, foreign.project, ids[0]); err != nil {
		t.Fatal(err)
	}
	// A new selected goal cannot rewrite what this change was measuring.
	if err = f.svc.SaveEnquiryGoal(ctx, own.project, "Booking"); err != nil {
		t.Fatal(err)
	}
	report, err = f.svc.Enquiries(ctx, own.project)
	if err != nil || report.GoalObserved || report.Pages[0].Signal != "verify_tracking" {
		t.Fatal("unconfigured event became lost enquiries", report, err)
	}
	seed(own.project, now.AddDate(0, 0, 3), 150, 8)
	now = now.AddDate(0, 0, 9)
	comparison, err = f.svc.CompareEnquiryChange(ctx, own.project, ids[0])
	if err != nil || !comparison.Ready || comparison.After.GoalVisitors != 8 || comparison.Change.EventName != "WhatsApp" || comparison.Change.Baseline.GoalVisitors != 2 {
		t.Fatalf("post-change evidence=%+v err=%v", comparison, err)
	}
	if err = f.svc.DeleteEnquiryChange(ctx, own.project, ids[0]); err != nil {
		t.Fatal(err)
	}
	if _, err = f.svc.CompareEnquiryChange(ctx, own.project, ids[0]); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("baseline not deleted", err)
	}
}

func TestEnquiryGoalRejectsInvalidDataBeforeDatabase(t *testing.T) {
	svc := &Service{}
	if err := svc.SaveEnquiryGoal(context.Background(), "project", "bad\nevent"); !errors.Is(err, errEnquiryInput) {
		t.Fatal(err)
	}
	if _, err := svc.SaveEnquiryChange(context.Background(), "project", "//foreign.com", "note", "valid-key"); !errors.Is(err, errEnquiryInput) {
		t.Fatal(err)
	}
}
