package citefix

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/toufiqqureshi/uplumo/backend/internal/audit"
	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/database"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/pgdb"
)

// testPool opens the test database with migrations applied. It skips when
// TEST_DATABASE_URL is unset outside CI.
func testPool(ctx context.Context, t *testing.T) *pgxpool.Pool {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("TEST_DATABASE_URL must be set in CI")
		}
		t.Skip("TEST_DATABASE_URL not set; skipping PostgreSQL integration test")
	}
	pool, err := pgdb.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS projects (
		id text PRIMARY KEY, organization_id text NOT NULL, name text, archived_at timestamptz)`); err != nil {
		t.Fatalf("ensure projects table: %v", err)
	}
	return pool
}

type countedOverview struct{ calls int }

func (o *countedOverview) Refresh(context.Context, string, string, string) (GoogleAIOverviewState, error) {
	o.calls++
	return GoogleAIOverviewState{}, nil
}
func (o *countedOverview) Collect(context.Context, string, string) (GoogleAIOverviewState, error) {
	o.calls++
	return GoogleAIOverviewState{}, nil
}

func TestReportAndTopicsNeverStartPaidProviderWork(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool := testPool(ctx, t)
	org, project := seedProject(ctx, t, pool)
	page := crawledPage("https://shop.example/solar", 500, audit.AIPageSignals{TopicHeadings: []string{"Solar panels"}})
	page.Title = "Solar panels guide"
	crawl(ctx, t, audit.NewRepository(pool), project, true, []audit.CrawledPageResult{page}, nil)
	agent := &fakeRunner{}
	overview := &countedOverview{}
	h := handler{Deps{Reporter: Store{DB: pool}, Agent: agent, GoogleAIOverview: overview}}
	for range 3 {
		request := httptest.NewRequestWithContext(auth.WithProjectOrganization(ctx, org), http.MethodPost, "/report", strings.NewReader(`{}`))
		request.SetPathValue("projectId", project)
		response := httptest.NewRecorder()
		h.report(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("report status %d: %s", response.Code, response.Body)
		}
		var report Report
		if err := json.Unmarshal(response.Body.Bytes(), &report); err != nil || report.Topics == nil || len(report.Topics.Clusters) == 0 {
			t.Fatalf("topics = %+v, err = %v", report.Topics, err)
		}
	}
	if agent.runs != 0 || overview.calls != 0 {
		t.Fatalf("report caused paid work: agent %d, overview %d", agent.runs, overview.calls)
	}
}

func randomID(t *testing.T, prefix string) string {
	t.Helper()
	var value [6]byte
	if _, err := rand.Read(value[:]); err != nil {
		t.Fatal(err)
	}
	return prefix + hex.EncodeToString(value[:])
}

// seedProject inserts a workspace and a project in it.
func seedProject(ctx context.Context, t *testing.T, pool *pgxpool.Pool) (organizationID, projectID string) {
	t.Helper()
	organizationID, projectID = randomID(t, "org-"), randomID(t, "proj-")
	if _, err := pool.Exec(ctx, `INSERT INTO organization (id, name, slug, created_at) VALUES ($1, 'cite fix test', $1, now())`, organizationID); err != nil {
		t.Fatalf("insert organization: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO projects (id, organization_id, name) VALUES ($1, $2, 'test')`, projectID, organizationID); err != nil {
		t.Fatalf("insert project: %v", err)
	}
	t.Cleanup(func() {
		cleanup := context.WithoutCancel(ctx)
		_, _ = pool.Exec(cleanup, `DELETE FROM go_audits WHERE project_id = $1`, projectID)
		_, _ = pool.Exec(cleanup, `DELETE FROM projects WHERE id = $1`, projectID)
		_, _ = pool.Exec(cleanup, `DELETE FROM organization WHERE id = $1`, organizationID)
	})
	return organizationID, projectID
}

// crawl stores an audit through the audit repository, the same path the
// crawler takes, and completes it unless complete is false.
func crawl(ctx context.Context, t *testing.T, repo *audit.Repository, projectID string, complete bool, pages []audit.CrawledPageResult, issues []audit.DetectedIssue) string {
	t.Helper()
	auditID := randomID(t, "audit-")
	config, err := audit.MarshalAuditConfig(audit.Config{MaxPages: 50, LighthouseStrategy: audit.LighthouseNone})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateAudit(ctx, audit.CreateAuditInput{
		ID: auditID, ProjectID: projectID, StartedByUserID: "user-1", StartURL: "https://shop.example/",
		WorkflowInstanceID: auditID, Config: config, PagesTotal: len(pages),
	}); err != nil {
		t.Fatalf("create audit: %v", err)
	}
	for i := range pages {
		pages[i].ID = randomID(t, "page-")
	}
	if err := repo.InsertCrawledBatch(ctx, auditID, pages, issues); err != nil {
		t.Fatalf("insert pages: %v", err)
	}
	if complete {
		if err := repo.CompleteAudit(ctx, auditID, auditID, len(pages), len(pages)); err != nil {
			t.Fatalf("complete audit: %v", err)
		}
	}
	// Audits are ordered by start time; keep each one strictly later.
	time.Sleep(5 * time.Millisecond)
	return auditID
}

func crawledPage(url string, words int, signals audit.AIPageSignals) audit.CrawledPageResult {
	return audit.CrawledPageResult{
		URL: url, StatusCode: 200, FetchClass: audit.FetchOK, IsHTML: true, IsIndexable: true,
		Title: "Page " + url, WordCount: words, AI: signals,
	}
}

func TestStoreReportStates(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool := testPool(ctx, t)
	repo := audit.NewRepository(pool)
	store := Store{DB: pool}
	organizationID, projectID := seedProject(ctx, t, pool)

	report, err := store.Report(ctx, organizationID, projectID)
	if err != nil || report.State != StateNoScan || report.Score != nil || report.Running != nil {
		t.Fatalf("no audits: %+v, %v", report, err)
	}

	// A running audit is reported, but there is still nothing to score.
	runningID := crawl(ctx, t, repo, projectID, false, nil, nil)
	report, err = store.Report(ctx, organizationID, projectID)
	if err != nil || report.State != StateNoScan || report.Running == nil || report.Running.AuditID != runningID {
		t.Fatalf("running audit: %+v, %v", report, err)
	}

	// An audit crawled before the column existed: its pages have NULL signals.
	legacyID := crawl(ctx, t, repo, projectID, true, []audit.CrawledPageResult{crawledPage("https://shop.example/", 500, audit.AIPageSignals{})}, nil)
	if _, err := pool.Exec(ctx, `UPDATE go_audit_pages SET citation_signals_json = NULL WHERE audit_id = $1`, legacyID); err != nil {
		t.Fatal(err)
	}
	report, err = store.Report(ctx, organizationID, projectID)
	if err != nil || report.State != StateNeedsRescan || report.Score != nil || report.Scan == nil || report.Scan.AuditID != legacyID {
		t.Fatalf("legacy audit: %+v, %v", report, err)
	}
	// A PR 1/2 scan has JSON but not the new versioned fields: it is also
	// unknown, not a collection of failed checks.
	if _, err := pool.Exec(ctx, `UPDATE go_audit_pages SET citation_signals_json = '{"schemaTypes":[],"author":true}' WHERE audit_id = $1`, legacyID); err != nil {
		t.Fatal(err)
	}
	report, err = store.Report(ctx, organizationID, projectID)
	if err != nil || report.State != StateNeedsRescan || report.Score != nil || report.Topics != nil {
		t.Fatalf("pre-version signals: %+v, %v", report, err)
	}

	// A new crawl writes signals with each page, and the report reads them.
	signals := audit.AIPageSignals{
		SchemaTypes: []string{"Article"}, QuestionHeadings: 1, Author: true, Published: "2026-09-01",
		LeadWords: 30, Lists: 1, Statistics: 2, Quotes: 1, AnswerCapsule: true,
		DefinitionAfterHeading: true, Sections: 1, ChunkReadySections: 1, AuthorityLinks: 1,
	}
	page := crawledPage("https://shop.example/guide", 900, signals)
	page.Links = []audit.PageLink{{TargetURL: "https://gov.example/rule", IsInternal: false}}
	issues := []audit.DetectedIssue{
		{IssueType: audit.IssueAICrawlersBlocked, PageURL: "https://shop.example/robots.txt", Details: map[string]any{"crawlers": []string{"PerplexityBot (Perplexity)"}}},
		{IssueType: audit.IssueMissingTitle, PageURL: "https://shop.example/guide"},
	}
	readyID := crawl(ctx, t, repo, projectID, true, []audit.CrawledPageResult{page}, issues)
	report, err = store.Report(ctx, organizationID, projectID)
	if err != nil {
		t.Fatal(err)
	}
	if report.State != StateReady || report.Scan.AuditID != readyID || report.PagesChecked != 1 || report.Running == nil {
		t.Fatalf("ready report: %+v", report)
	}
	// The page passes every page check; only the blocked search bot fails, so
	// the stored signals, link counts and issue details all round-tripped.
	if got := fixIDs(report); len(got) != 1 || got[0] != "ai-search-bots-allowed" {
		t.Fatalf("fixes = %v, want only the blocked search bot", got)
	}
	if *report.Score != 93 {
		t.Fatalf("score = %d, want 93", *report.Score)
	}
}

func TestStoreReportIsScopedToTheWorkspace(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool := testPool(ctx, t)
	repo := audit.NewRepository(pool)
	store := Store{DB: pool}
	_, projectID := seedProject(ctx, t, pool)
	otherOrganization, _ := seedProject(ctx, t, pool)
	crawl(ctx, t, repo, projectID, true, []audit.CrawledPageResult{crawledPage("https://shop.example/", 500, audit.AIPageSignals{})}, nil)
	crawl(ctx, t, repo, projectID, false, nil, nil)

	// Another workspace asking for this project sees no scan and no running
	// audit: nothing about the project leaks.
	report, err := store.Report(ctx, otherOrganization, projectID)
	if err != nil || report.State != StateNoScan || report.Scan != nil || report.Running != nil {
		t.Fatalf("foreign workspace: %+v, %v", report, err)
	}
}

func TestStoreReportSkipsFailedAudits(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool := testPool(ctx, t)
	repo := audit.NewRepository(pool)
	organizationID, projectID := seedProject(ctx, t, pool)
	completedID := crawl(ctx, t, repo, projectID, true, []audit.CrawledPageResult{crawledPage("https://shop.example/", 500, audit.AIPageSignals{})}, nil)
	failedID := crawl(ctx, t, repo, projectID, false, nil, nil)
	if err := repo.FailAudit(ctx, failedID, failedID, audit.ErrorInfo{ErrorCode: audit.ErrorCode("internal_error")}, "crawling"); err != nil {
		t.Fatal(err)
	}
	report, err := (Store{DB: pool}).Report(ctx, organizationID, projectID)
	if err != nil || report.State != StateReady || report.Scan.AuditID != completedID || report.Running != nil {
		t.Fatalf("report after a failed audit: %+v, %v", report, err)
	}
}
