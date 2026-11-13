package aisearch

import (
	"context"
	"crypto/rand"
	"testing"
	"time"
)

// A saved answer marks the customer's own pages, and only those, so the
// screen can offer to open them in the editor.
func TestTrackingMarksOnlyTheCustomersOwnCitations(t *testing.T) {
	db := trackingDB(t)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	org, project := "org-"+rand.Text(), "project-"+rand.Text()
	if _, err := db.Exec(ctx, `INSERT INTO organization(id,name,slug,created_at) VALUES($1,'Tracking',$1,now())`, org); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = db.Exec(context.Background(), `DELETE FROM organization WHERE id=$1`, org) })
	if _, err := db.Exec(ctx, `INSERT INTO projects(id,organization_id,name) VALUES($1,$2,'Tracking')`, project, org); err != nil {
		t.Fatal(err)
	}
	svc := &Tracking{DB: db, Plans: trackingPaidPlans{}}
	c := validTracking()
	if err := svc.Save(ctx, org, project, c); err != nil {
		t.Fatal(err)
	}
	run, err := svc.Start(ctx, org, project, "own-citation", false, 10)
	if err != nil {
		t.Fatal(err)
	}
	var answerID string
	if err = db.QueryRow(ctx, `UPDATE go_ai_tracking_answers SET status='submitted' WHERE id=(SELECT id FROM go_ai_tracking_answers WHERE run_id=$1 LIMIT 1) RETURNING id`, run).Scan(&answerID); err != nil {
		t.Fatal(err)
	}
	domain := func(s string) *string { return &s }
	citations := []Citation{
		{URL: "https://www.acme.com/pricing", Domain: domain("www.acme.com")},
		{URL: "https://other.com/", Domain: domain("other.com")},
		{URL: "https://notacme.com/", Domain: domain("notacme.com")},
	}
	if err = svc.finishAnswer(ctx, answerID, c, "Acme and Other are good.", citations, nil, ""); err != nil {
		t.Fatal(err)
	}
	answers, err := svc.Answers(ctx, org, project, run)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range answers {
		if a.ID != answerID {
			continue
		}
		got := []bool{}
		for _, cite := range a.Citations {
			got = append(got, cite.MatchedBrand)
		}
		if len(got) != 3 || !got[0] || got[1] || got[2] {
			t.Fatalf("matchedBrand = %v, want [true false false] for own, competitor and look-alike", got)
		}
		return
	}
	t.Fatalf("answer %s not returned", answerID)
}
