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
	svc := &Tracking{DB: db}
	c := validTracking()
	run, answerID := "run-"+rand.Text(), "answer-"+rand.Text()
	if _, err := db.Exec(ctx, `INSERT INTO go_ai_trackers(project_id,organization_id,configuration) VALUES($1,$2,'{"brand":"Acme","domain":"acme.com"}')`, project, org); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO go_ai_tracking_runs(id,project_id,organization_id,configuration,scope_key,trigger,status,request_key) VALUES($1,$2,$3,'{"brand":"Acme","domain":"acme.com"}','s','manual','running',$1)`, run, project, org); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO go_ai_tracking_answers(id,run_id,prompt,topic,engine,status) VALUES($1,$2,'Which service?','General','chat_gpt','submitted')`, answerID, run); err != nil {
		t.Fatal(err)
	}
	domain := func(s string) *string { return &s }
	citations := []Citation{
		{URL: "https://www.acme.com/pricing", Domain: domain("www.acme.com")},
		{URL: "https://other.com/", Domain: domain("other.com")},
		{URL: "https://notacme.com/", Domain: domain("notacme.com")},
	}
	if err := svc.finishAnswer(ctx, answerID, c, "Acme and Other are good.", citations, nil, ""); err != nil {
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
