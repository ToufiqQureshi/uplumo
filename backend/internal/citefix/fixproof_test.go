package citefix

import (
	"testing"
	"time"
)

func TestFixProofsCompareAIVisitsAndLeadsAroundTheFix(t *testing.T) {
	f := newAgentFixture(t)
	ctx := t.Context()
	fixedAt := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	now := fixedAt.Add(10 * 24 * time.Hour)
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := f.pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	exec(`INSERT INTO go_citefix_agent_runs (id, organization_id, project_id, url, mode, idempotency_key, status, finished_at)
		VALUES ('r1', $1, $2, $3, 'quick', 'k1', 'completed', $4),
		       ('r2', $1, $2, $3, 'quick', 'k2', 'completed', $4 - interval '1 day'),
		       ('r3', $1, $2, 'https://x.example/cached', 'quick', 'k3', 'completed', $4)`,
		f.organizationID, f.projectID, f.pageURL, fixedAt)
	exec(`UPDATE go_citefix_agent_runs SET cached = true WHERE id = 'r3'`)
	var siteID int64
	if err := f.pool.QueryRow(ctx, `INSERT INTO go_analytics_sites (project_id, site_key) VALUES ($1, 'k-fixproof') RETURNING id`, f.projectID).Scan(&siteID); err != nil {
		t.Fatal(err)
	}
	visit := func(at time.Time, visitor, session, aiSource string) {
		t.Helper()
		exec(`INSERT INTO go_analytics_events (site_id, occurred_at, path, channel, ai_source, device, visitor_hash, session_hash)
			VALUES ($1, $2, '/guide', $3, nullif($4, ''), 'desktop', $5, $6)`,
			siteID, at, map[bool]string{true: "ai", false: "direct"}[aiSource != ""], aiSource, hash32(visitor), hash32(session))
	}
	lead := func(at time.Time, visitor, session string) {
		t.Helper()
		exec(`INSERT INTO go_analytics_custom_events (site_id, occurred_at, name, path, visitor_hash, session_hash)
			VALUES ($1, $2, 'Book', '/guide', $3, $4)`, siteID, at, hash32(visitor), hash32(session))
	}
	visit(fixedAt.Add(-48*time.Hour), "v1", "s1", "chatgpt")
	visit(fixedAt.Add(24*time.Hour), "v2", "s2", "perplexity")
	visit(fixedAt.Add(25*time.Hour), "v3", "s3", "chatgpt")
	lead(fixedAt.Add(25*time.Hour+time.Minute), "v3", "s3")
	visit(fixedAt.Add(26*time.Hour), "v4", "s4", "") // not from AI
	lead(fixedAt.Add(26*time.Hour+time.Minute), "v4", "s4")
	visit(fixedAt.Add(-20*24*time.Hour), "v5", "s5", "chatgpt") // outside the 10-day window

	proof := &Proof{YourPages: []URLCount{{URL: f.pageURL + "?utm_source=chatgpt.com", Count: 1}}}
	got, err := f.agent.Store.fixProofs(ctx, f.organizationID, f.projectID, proof, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("fixes = %+v; want one page (cached runs are not fixes)", got)
	}
	p := got[0]
	if !p.FixedAt.Equal(fixedAt) || p.Days != 10 || p.AIVisitorsPrior != 1 || p.AIVisitorsAfter != 2 || !p.CitedNow {
		t.Fatalf("proof = %+v", p)
	}
	if p.LeadsTracked || p.LeadsAfter != 0 {
		t.Fatalf("leads before choosing = %+v; want none counted", p)
	}
	exec(`INSERT INTO go_analytics_lead_events (site_id, name) VALUES ($1, 'Book')`, siteID)
	got, err = f.agent.Store.fixProofs(ctx, f.organizationID, f.projectID, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if p := got[0]; !p.LeadsTracked || p.LeadsAfter != 1 || p.LeadsPrior != 0 || p.CitedNow {
		t.Fatalf("with leads = %+v; want 1 AI lead after (the direct visit's lead does not count)", p)
	}
}

// hash32 pads an id to the 32-byte hashes the analytics tables store.
func hash32(id string) []byte {
	b := make([]byte, 32)
	copy(b, id)
	return b
}

func TestSaidQuotesAnswersThatNameTheCustomer(t *testing.T) {
	answers := []proofAnswer{
		{prompt: "q1", engine: "claude", answer: "Top picks: Shop Co is known for fast delivery."},
		{prompt: "q1", engine: "gemini", answer: "Try rival.example instead."},
		{prompt: "q2", engine: "gemini", answer: "See shop.example for prices."},
	}
	got := said("Shop Co", "shop.example", answers)
	if len(got) != 2 || got[0].Engine != "claude" || got[1].Prompt != "q2" {
		t.Fatalf("said = %+v", got)
	}
	if len(said("Shop Co", "shop.example", nil)) != 0 || said("", "", answers) == nil {
		t.Fatal("empty input must give an empty, non-nil list")
	}
}
