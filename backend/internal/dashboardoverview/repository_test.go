package dashboardoverview

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toufiqqureshi/uplumo/backend/internal/backlinks"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/spend"
)

func dashboardDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("TEST_DATABASE_URL must be set in CI")
		}
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("dashboard_overview_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); admin.Close() })
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	db, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	for _, sql := range []string{`CREATE TABLE projects(id text,organization_id text,domain text,archived_at text)`, `CREATE TABLE go_audits(id text PRIMARY KEY,project_id text NOT NULL,status text NOT NULL,pages_crawled integer NOT NULL DEFAULT 0,started_at timestamptz NOT NULL DEFAULT now())`, `CREATE TABLE go_audit_issues(id text PRIMARY KEY,audit_id text NOT NULL REFERENCES go_audits(id),page_url text NOT NULL,issue_type text NOT NULL,severity text NOT NULL)`, `CREATE TABLE audits(id text,project_id text,status text,pages_crawled int,started_at text)`, `CREATE TABLE backlink_snapshots(id serial,project_id text,domain text,rank int,backlinks bigint,referring_domains bigint,new_backlinks bigint,lost_backlinks bigint,new_referring_domains bigint,lost_referring_domains bigint,captured_at text)`, `INSERT INTO projects VALUES('p','o','example.com',NULL)`, `INSERT INTO go_audits VALUES('old','p','completed',3,'2026-10-09T10:00:00Z'),('a','p','completed',12,'2026-10-10T10:00:00Z'),('other','p2','completed',99,'2026-10-11T10:00:00Z')`, `INSERT INTO go_audit_issues VALUES('1','a','https://example.com/a','title','critical'),('2','a','https://example.com/a','title','critical'),('3','a','https://example.com/b','links','warning'),('4','a','https://example.com/c','meta','info'),('5','old','https://example.com/x','old_issue','critical')`, `INSERT INTO audits VALUES('legacy','p','running',1,'2026-10-12T10:00:00.000Z')`, `INSERT INTO backlink_snapshots(project_id,domain,rank,backlinks,referring_domains,captured_at) VALUES('p','example.com',12,500,40,'2026-10-10T11:00:00.000Z'),('p','old.example',9,20,3,'2026-10-10T11:30:00.000Z')`} {
		if _, err = db.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	return db
}
func TestRepositoryReadsProjectScopedDashboardCards(t *testing.T) {
	db := dashboardDB(t)
	repo := Repository{DB: db}
	domain, err := repo.ProjectDomain(context.Background(), "p", "o")
	if err != nil || domain == nil || *domain != "example.com" {
		t.Fatalf("project domain=%v err=%v", domain, err)
	}
	audit, err := repo.LatestAudit(context.Background(), "p")
	// The newest go_audits row of this project wins: not the older crawl, not
	// another project's newer crawl, and not the unwritten legacy audits table.
	if err != nil || audit == nil || audit.Status != "completed" || audit.PagesCrawled != 12 || audit.StartedAt != "2026-10-10T10:00:00.000Z" || audit.TotalIssueTypes != 3 || len(audit.TopIssues) != 3 || audit.TopIssues[0].Count != 1 {
		t.Fatalf("audit=%+v err=%v", audit, err)
	}
	if audit, err = repo.LatestAudit(context.Background(), "no-audits"); err != nil || audit != nil {
		t.Fatalf("project without audits: audit=%+v err=%v", audit, err)
	}
	snapshot, found, err := repo.LatestBacklinkSnapshot(context.Background(), "p", "example.com")
	if err != nil || !found || snapshot.Domain != "example.com" {
		t.Fatalf("snapshot=%+v found=%v err=%v", snapshot, found, err)
	}
	if _, err = repo.ProjectDomain(context.Background(), "p", "other-org"); !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("unauthorized organization err=%v", err)
	}
}

type snapshotProvider struct {
	calls atomic.Int32
	delay time.Duration
	err   error
}

func (p *snapshotProvider) DashboardSummary(ctx context.Context, _, _ string) (backlinks.Summary, error) {
	p.calls.Add(1)
	timer := time.NewTimer(p.delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return backlinks.Summary{}, ctx.Err()
	case <-timer.C:
	}
	count := 42.0
	return backlinks.Summary{Backlinks: &count}, p.err
}

func TestConcurrentRefreshPersistsOnePurchaseAndReusesDailySnapshot(t *testing.T) {
	db := dashboardDB(t)
	provider := &snapshotProvider{delay: 100 * time.Millisecond}
	svc := &Service{Store: Repository{DB: db}, Backlinks: provider}
	start := make(chan struct{})
	var group sync.WaitGroup
	for range 8 {
		group.Go(func() {
			<-start
			if err := svc.RefreshBacklinkSnapshot(t.Context(), "o", "p"); err != nil {
				t.Error(err)
			}
		})
	}
	close(start)
	group.Wait()
	if got := provider.calls.Load(); got != 1 {
		t.Fatalf("provider purchases=%d, want one for eight concurrent refreshes", got)
	}
	if err := svc.RefreshBacklinkSnapshot(t.Context(), "o", "p"); err != nil {
		t.Fatal(err)
	}
	overview, err := svc.Get(t.Context(), "o", "p")
	if err != nil || overview.Backlinks == nil || overview.Backlinks.Stale || overview.Backlinks.Backlinks == nil || *overview.Backlinks.Backlinks != 42 || !overview.BacklinkRefreshAvailable {
		t.Fatalf("persisted result=%+v, err=%v", overview, err)
	}
	var rows int
	if err = db.QueryRow(t.Context(), "SELECT count(*) FROM backlink_snapshots WHERE project_id='p'").Scan(&rows); err != nil || rows != 3 {
		t.Fatalf("rows=%d, want two fixtures plus one refresh: %v", rows, err)
	}
	if got := provider.calls.Load(); got != 1 {
		t.Fatalf("fresh snapshot caused another purchase: %d", got)
	}
	if err = svc.RefreshBacklinkSnapshot(t.Context(), "other-org", "p"); !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("cross-tenant refresh=%v", err)
	}
	if _, err = db.Exec(t.Context(), "INSERT INTO projects VALUES('p2','o2','example.com',NULL)"); err != nil {
		t.Fatal(err)
	}
	if err = svc.RefreshBacklinkSnapshot(t.Context(), "o2", "p2"); err != nil {
		t.Fatal(err)
	}
	if got := provider.calls.Load(); got != 2 {
		t.Fatalf("independent organization purchases=%d, want two total", got)
	}
}

func TestFailedRefreshKeepsStaleDataAndReturnsTheRefusal(t *testing.T) {
	db := dashboardDB(t)
	if _, err := db.Exec(t.Context(), "UPDATE backlink_snapshots SET domain='example.com' WHERE project_id='p'"); err != nil {
		t.Fatal(err)
	}
	provider := &snapshotProvider{err: spend.ErrNoCredits}
	svc := &Service{Store: Repository{DB: db}, Backlinks: provider}
	if err := svc.RefreshBacklinkSnapshot(t.Context(), "o", "p"); !errors.Is(err, spend.ErrNoCredits) {
		t.Fatalf("refusal hidden as refresh success: %v", err)
	}
	overview, err := svc.Get(t.Context(), "o", "p")
	if err != nil || overview.Backlinks == nil || !overview.Backlinks.Stale {
		t.Fatalf("lost stale snapshot: %+v, %v", overview, err)
	}
	var rows int
	if err = db.QueryRow(t.Context(), "SELECT count(*) FROM backlink_snapshots").Scan(&rows); err != nil || rows != 2 {
		t.Fatalf("failed refresh changed stored rows=%d: %v", rows, err)
	}
	if provider.calls.Load() != 1 {
		t.Fatalf("failure was automatically retried: %d", provider.calls.Load())
	}
}

type domainSnapshotProvider func(context.Context, string, string) (backlinks.Summary, error)

func (f domainSnapshotProvider) DashboardSummary(ctx context.Context, org, domain string) (backlinks.Summary, error) {
	return f(ctx, org, domain)
}

func TestDomainChangeDoesNotJoinOldPurchaseOrHideNewSnapshot(t *testing.T) {
	db := dashboardDB(t)
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	oldStarted, releaseOld := make(chan struct{}), make(chan struct{})
	defer close(releaseOld)
	var calls atomic.Int32
	provider := domainSnapshotProvider(func(ctx context.Context, _, domain string) (backlinks.Summary, error) {
		calls.Add(1)
		if domain == "example.com" {
			close(oldStarted)
			select {
			case <-releaseOld:
			case <-ctx.Done():
				return backlinks.Summary{}, ctx.Err()
			}
		}
		return backlinks.Summary{}, nil
	})
	svc := &Service{Store: Repository{DB: db}, Backlinks: provider}
	oldDone := make(chan error, 1)
	go func() { oldDone <- svc.RefreshBacklinkSnapshot(ctx, "o", "p") }()
	select {
	case <-oldStarted:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if _, err := db.Exec(ctx, "UPDATE projects SET domain='new.example' WHERE id='p'"); err != nil {
		t.Fatal(err)
	}
	if err := svc.RefreshBacklinkSnapshot(ctx, "o", "p"); err != nil {
		t.Fatalf("new domain joined the blocked old-domain purchase: %v", err)
	}
	releaseOld <- struct{}{}
	if err := <-oldDone; err != nil {
		t.Fatal(err)
	}
	overview, err := svc.Get(ctx, "o", "p")
	if err != nil || overview.Backlinks == nil || overview.Backlinks.Domain != "new.example" || calls.Load() != 2 {
		t.Fatalf("current snapshot=%+v err=%v purchases=%d", overview.Backlinks, err, calls.Load())
	}
	if err := svc.RefreshBacklinkSnapshot(ctx, "o", "p"); err != nil || calls.Load() != 2 {
		t.Fatalf("current domain cache missed: %v purchases=%d", err, calls.Load())
	}
}
