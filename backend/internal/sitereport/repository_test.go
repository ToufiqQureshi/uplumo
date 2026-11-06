package sitereport

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func reportDB(t *testing.T) *pgxpool.Pool {
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
	schema := fmt.Sprintf("site_report_test_%d", time.Now().UnixNano())
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
	for _, sql := range []string{
		`CREATE TABLE projects(id text, organization_id text, name text, domain text, archived_at timestamptz)`,
		`CREATE TABLE go_audits(id text PRIMARY KEY, project_id text, status text, pages_crawled int, started_at timestamptz, completed_at timestamptz)`,
		`CREATE TABLE go_audit_issues(id text PRIMARY KEY, audit_id text, page_url text, issue_type text, severity text)`,
		`CREATE TABLE go_audit_lighthouse_results(id text PRIMARY KEY, audit_id text, strategy text, performance_score int, accessibility_score int, best_practices_score int, seo_score int, lcp_ms double precision, cls double precision, inp_ms double precision, error_message text)`,
		`CREATE TABLE backlink_snapshots(id serial, project_id text, domain text, rank int, backlinks int, referring_domains int, new_backlinks int, lost_backlinks int, new_referring_domains int, lost_referring_domains int, captured_at text)`,
		`INSERT INTO projects VALUES ('p','o','Acme','example.com',NULL), ('gone','o','Old','old.com',now()), ('nodomain','o','No domain',NULL,NULL)`,
		`INSERT INTO go_audits VALUES
			('done','p','completed',40,'2026-11-01T10:00:00Z','2026-11-01T10:20:00Z'),
			('running','p','running',3,'2026-11-05T10:00:00Z',NULL),
			('other','p2','completed',9,'2026-11-06T10:00:00Z',NULL)`,
		`INSERT INTO go_audit_issues VALUES
			('1','done','https://example.com/f','broken-page','critical'),
			('2','done','https://example.com/a','broken-page','critical'),
			('3','done','https://example.com/a','broken-page','critical'),
			('4','done','https://example.com/b','missing-h1','warning'),
			('5','done','https://example.com/c','images-missing-alt','info'),
			('6','done','https://example.com/d','images-missing-alt','info'),
			('7','done','https://example.com/e','images-missing-alt','info'),
			('8','done','https://example.com/g','images-missing-alt','info'),
			('9','done','https://example.com/h','images-missing-alt','info'),
			('10','done','https://example.com/i','images-missing-alt','info'),
			('11','running','https://example.com/x','server-error','critical')`,
		`INSERT INTO go_audit_lighthouse_results VALUES
			('l1','done','mobile',70,90,100,80,2000,0.1,200,NULL),
			('l2','done','mobile',81,95,100,91,3000,0.0,100,NULL),
			('l3','done','mobile',NULL,NULL,NULL,NULL,NULL,NULL,NULL,'timeout'),
			('l4','done','desktop',10,10,10,10,1,1,1,NULL)`,
		`INSERT INTO backlink_snapshots(project_id,domain,rank,backlinks,referring_domains,new_referring_domains,lost_referring_domains,captured_at) VALUES
			('p','example.com',100,500,40,2,1,'2026-11-01T00:00:00.000Z'),
			('p','example.com',120,600,45,6,2,'2026-11-04T00:00:00.000Z'),
			('p','other.com',999,9,9,9,9,'2026-11-05T00:00:00.000Z')`,
	} {
		if _, err := db.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	return db
}

func TestRepositoryReadsOnlyThisProjectsFinishedData(t *testing.T) {
	repo := Repository{DB: reportDB(t)}
	ctx := t.Context()

	project, err := repo.Project(ctx, "o", "p")
	if err != nil || project != (Project{Name: "Acme", Domain: "example.com"}) {
		t.Fatalf("project = %+v, %v", project, err)
	}
	for _, tc := range []struct{ org, id string }{{"other-org", "p"}, {"o", "gone"}, {"o", "missing"}} {
		if _, err := repo.Project(ctx, tc.org, tc.id); !errors.Is(err, ErrProjectNotFound) {
			t.Errorf("project %s/%s err = %v", tc.org, tc.id, err)
		}
	}
	if p, err := repo.Project(ctx, "o", "nodomain"); err != nil || p.Domain != "" {
		t.Errorf("no-domain project = %+v, %v", p, err)
	}

	audit, err := repo.LatestAudit(ctx, "p")
	if err != nil || audit == nil {
		t.Fatalf("audit = %+v, %v", audit, err)
	}
	if audit.PagesCrawled != 40 || audit.CompletedAt.Format(time.RFC3339) != "2026-11-01T10:20:00Z" {
		t.Errorf("a running audit must not replace the finished one: %+v", audit)
	}
	if audit.Critical != 2 || audit.Warning != 1 || audit.Info != 6 || len(audit.Issues) != 3 {
		t.Fatalf("counts = %d/%d/%d issues=%d", audit.Critical, audit.Warning, audit.Info, len(audit.Issues))
	}
	first := audit.Issues[0]
	if first.Type != "broken-page" || first.Pages != 2 || fmt.Sprint(first.Examples) != "[https://example.com/a https://example.com/f]" {
		t.Errorf("first issue = %+v", first)
	}
	if last := audit.Issues[2]; last.Pages != 6 || len(last.Examples) != examplesPerIssue {
		t.Errorf("examples must cap at %d: %+v", examplesPerIssue, last)
	}
	lh := audit.Lighthouse
	if lh == nil || lh.Pages != 2 || *lh.Performance != 76 || *lh.SEO != 86 || *lh.LCPMillis != 2500 {
		t.Errorf("lighthouse = %+v", lh)
	}
	if none, err := repo.LatestAudit(ctx, "nobody"); err != nil || none != nil {
		t.Errorf("no audit = %+v, %v", none, err)
	}

	links, err := repo.LatestBacklinks(ctx, "p", "example.com")
	if err != nil || links == nil || *links.ReferringDomains != 45 || *links.NewReferringDomains != 6 || links.CapturedAt.IsZero() {
		t.Fatalf("backlinks = %+v, %v", links, err)
	}
	if none, err := repo.LatestBacklinks(ctx, "p", "missing.com"); err != nil || none != nil {
		t.Errorf("missing domain = %+v, %v", none, err)
	}
}
