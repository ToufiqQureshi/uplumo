package citefix

import (
	"context"
	"slices"
	"testing"
	"time"
)

func cite(urls ...string) []storedCitation {
	out := make([]storedCitation, 0, len(urls))
	for _, u := range urls {
		out = append(out, storedCitation{URL: u})
	}
	return out
}

func TestBuildProofCountsCitationsPerAnswer(t *testing.T) {
	domain := "rival.example"
	answers := []proofAnswer{
		// Cites the customer (www. and a subdomain both count) and a rival.
		{prompt: "best goa resort", engine: "chatgpt", citations: append(cite("https://www.shop.example/goa", "https://blog.shop.example/tips"), storedCitation{URL: "https://r.example/x", Domain: &domain})},
		// Two pages of one rival: the rival counts once for this answer.
		{prompt: "goa hotel price", engine: "perplexity", citations: cite("https://rival.example/a", "https://rival.example/b", "https://other.example/")},
		// No citations: not an answer the proof can use.
		{prompt: "goa weather", engine: "gemini"},
		// A malformed URL is skipped, not counted.
		{prompt: "goa forts", engine: "chatgpt", citations: cite("::not a url", "https://rival.example/forts")},
	}
	proof := buildProof(time.Unix(0, 0), "WWW.Shop.Example", answers)
	if proof.Domain != "shop.example" || proof.Answers != 3 || proof.CitingYou != 1 {
		t.Fatalf("proof = %+v, want 3 answers, 1 citing shop.example", proof)
	}
	if want := []DomainCount{{"rival.example", 3}, {"other.example", 1}}; !slices.Equal(proof.Rivals, want) {
		t.Fatalf("rivals = %v, want %v", proof.Rivals, want)
	}
	if len(proof.YourPages) != 2 || proof.YourPages[0].Count != 1 {
		t.Fatalf("your pages = %v", proof.YourPages)
	}
	if len(proof.Misses) != 2 || proof.Misses[0].Prompt != "goa hotel price" || !slices.Equal(proof.Misses[0].Cited, []string{"rival.example", "other.example"}) {
		t.Fatalf("misses = %+v", proof.Misses)
	}
	if got := proof.Misses[0].CitedPages; len(got) != 3 || got[0].URL != "https://rival.example/a" || got[1].URL != "https://rival.example/b" {
		t.Fatalf("miss cited pages = %+v, want the three saved rival URLs", got)
	}
	// A look-alike domain is not the customer's.
	if isOwnHost("notshop.example", "shop.example") || !isOwnHost("shop.example", "shop.example") {
		t.Fatal("own host matching is wrong")
	}
}

func TestBuildProofCapsItsLists(t *testing.T) {
	answers := []proofAnswer{}
	for i := range 12 {
		answers = append(answers, proofAnswer{prompt: string(rune('a' + i)), engine: "chatgpt", citations: cite("https://r"+string(rune('a'+i))+".example/", "https://x1.example/", "https://x2.example/", "https://x3.example/")})
	}
	proof := buildProof(time.Unix(0, 0), "shop.example", answers)
	if len(proof.Rivals) != maxProofRivals || len(proof.Misses) != maxProofMisses || len(proof.Misses[0].Cited) != maxMissCited || len(proof.Misses[0].CitedPages) != maxMissCited {
		t.Fatalf("rivals %d misses %d cited %d citedPages %d", len(proof.Rivals), len(proof.Misses), len(proof.Misses[0].Cited), len(proof.Misses[0].CitedPages))
	}
}

func TestStoreProofReadsTheLatestFinishedRunOfTheWorkspace(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool := testPool(ctx, t)
	store := Store{DB: pool}
	organizationID, projectID := seedProject(ctx, t, pool)
	otherOrganization, _ := seedProject(ctx, t, pool)
	if proof, err := store.proof(ctx, organizationID, projectID); err != nil || proof != nil {
		t.Fatalf("no tracker: %+v, %v", proof, err)
	}
	exec := func(sql string, args ...any) {
		t.Helper()
		if _, err := pool.Exec(ctx, sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	config := `{"brand":"Shop","domain":"shop.example"}`
	exec(`INSERT INTO go_ai_trackers (project_id, organization_id, configuration) VALUES ($1, $2, $3)`, projectID, organizationID, config)
	run := func(id, status string, at time.Time, citations string) {
		exec(`INSERT INTO go_ai_tracking_runs (id, project_id, organization_id, configuration, scope_key, trigger, status, created_at, request_key)
			VALUES ($1, $2, $3, $4, 's', 'manual', $5, $6, $1)`, id, projectID, organizationID, config, status, at)
		exec(`INSERT INTO go_ai_tracking_answers (id, run_id, prompt, topic, engine, status, citations)
			VALUES ($1 || '-a', $1, 'best shop', 't', 'chatgpt', 'completed', $2::jsonb)`, id, citations)
	}
	run(projectID+"-old", "completed", time.Now().Add(-48*time.Hour), `[{"url":"https://shop.example/"}]`)
	run(projectID+"-new", "partial", time.Now().Add(-time.Hour), `[{"url":"https://rival.example/","domain":"rival.example"}]`)
	run(projectID+"-failed", "failed", time.Now(), `[{"url":"https://shop.example/"}]`)
	proof, err := store.proof(ctx, organizationID, projectID)
	if err != nil || proof == nil || proof.Answers != 1 || proof.CitingYou != 0 || proof.Rivals[0].Domain != "rival.example" {
		t.Fatalf("proof = %+v, %v; want the newest finished run (partial counts, failed does not)", proof, err)
	}
	if proof, err := store.proof(ctx, otherOrganization, projectID); err != nil || proof != nil {
		t.Fatalf("foreign workspace: %+v, %v", proof, err)
	}
}
