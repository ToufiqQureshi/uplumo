package ranktracking

import (
	"context"
	"testing"
	"time"
)

func TestScheduledCreditsUsesOnlyActiveWorkspaceSchedules(t *testing.T) {
	f := newCheckFixture(t, "plain topic", "site:example.com")
	store := f.s.Repo.(Store)
	ctx := t.Context()
	next := time.Now().UTC().Add(time.Hour)
	org := "forecast-" + f.project
	if _, err := store.DB.Exec(ctx, `INSERT INTO organization(id,name,slug,created_at) VALUES($1,'Forecast',$1,now())`, org); err != nil {
		t.Fatal(err)
	}
	if _, err := store.DB.Exec(ctx, `INSERT INTO projects(id,organization_id,name) VALUES($1,$2,'Forecast')`, f.project, org); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := store.DB.Exec(context.Background(), `DELETE FROM organization WHERE id=$1`, org); err != nil {
			t.Error(err)
		}
	})
	if _, err := store.DB.Exec(ctx, `UPDATE go_rank_tracking_configs SET schedule_interval='weekly',next_check_at=$2,devices='both',serp_depth=30 WHERE id=$1`, f.cfg.ID, next); err != nil {
		t.Fatal(err)
	}
	want := int64(EstimateCheck([]string{"plain topic", "site:example.com"}, "both", 30, Queued).CostCredits) * 2
	got, err := store.ScheduledCredits(ctx, org, next.AddDate(0, 0, 8))
	if err != nil || got != want {
		t.Fatalf("forecast=%d want=%d err=%v", got, want, err)
	}
	monthEnd := time.Date(time.Now().Year()+1, time.January, 31, 10, 0, 0, 0, time.UTC)
	if _, err := store.DB.Exec(ctx, `UPDATE go_rank_tracking_configs SET schedule_interval='monthly',next_check_at=$2 WHERE id=$1`, f.cfg.ID, monthEnd); err != nil {
		t.Fatal(err)
	}
	got, err = store.ScheduledCredits(ctx, org, time.Date(monthEnd.Year(), time.March, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || got != want {
		t.Fatalf("month-end forecast=%d want two checks=%d err=%v", got, want, err)
	}
	if _, err := store.DB.Exec(ctx, `UPDATE go_rank_tracking_configs SET schedule_interval='weekly',next_check_at=$2 WHERE id=$1`, f.cfg.ID, next); err != nil {
		t.Fatal(err)
	}
	got, err = store.ScheduledCredits(ctx, "foreign-org", next.AddDate(0, 0, 8))
	if err != nil || got != 0 {
		t.Fatal("foreign workspace counted", got, err)
	}
	if _, err := store.DB.Exec(ctx, `UPDATE go_rank_tracking_configs SET is_active=false WHERE id=$1`, f.cfg.ID); err != nil {
		t.Fatal(err)
	}
	got, err = store.ScheduledCredits(ctx, org, next.AddDate(0, 0, 8))
	if err != nil || got != 0 {
		t.Fatal("paused schedule counted", got, err)
	}
}
