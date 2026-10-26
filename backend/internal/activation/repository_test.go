package activation

import (
	"context"
	"crypto/rand"
	"os"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/database"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/pgdb"
)

// The "Audit your site" step reads the table the audit feature writes. The
// legacy audits table is no longer written, so reading it kept the step open
// after every finished crawl.
func TestDashboardActivationSeesAuditsFromTheAuditFeature(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("TEST_DATABASE_URL required")
		}
		t.Skip("TEST_DATABASE_URL required")
	}
	ctx := t.Context()
	db, err := pgdb.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	if err = database.Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	org, audited, fresh := "org-"+rand.Text(), "project-"+rand.Text(), "project-"+rand.Text()
	t.Cleanup(func() {
		if _, err := db.Exec(context.Background(), `DELETE FROM organization WHERE id=$1`, org); err != nil {
			t.Error(err)
		}
	})
	for _, sql := range []struct {
		query string
		args  []any
	}{
		{`INSERT INTO organization(id,name,slug,created_at) VALUES($1,'Activation',$1,now())`, []any{org}},
		{`INSERT INTO projects(id,organization_id,name) VALUES($1,$2,'Audited'),($3,$2,'Fresh')`, []any{audited, org, fresh}},
		{`INSERT INTO go_audits(id,project_id,started_by_user_id,start_url,status) VALUES($1,$2,'user-1','https://example.com','completed')`, []any{"audit-" + rand.Text(), audited}},
	} {
		if _, err = db.Exec(ctx, sql.query, sql.args...); err != nil {
			t.Fatal(err)
		}
	}
	repo := Repository{DB: db}
	got, err := repo.GetDashboardActivation(ctx, "user-1", audited, org)
	if err != nil || !got.HasAudit {
		t.Fatalf("audited project hasAudit=%v err=%v, want true", got.HasAudit, err)
	}
	got, err = repo.GetDashboardActivation(ctx, "user-1", fresh, org)
	if err != nil || got.HasAudit {
		t.Fatalf("project without an audit hasAudit=%v err=%v, want false", got.HasAudit, err)
	}
}
