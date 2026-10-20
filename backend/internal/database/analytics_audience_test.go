package database

import (
	"context"
	"crypto/rand"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestAnalyticsRecognitionStorageDeniesPublicRole(t *testing.T) {
	ctx := t.Context()
	pool := openFresh(ctx, t)
	if err := Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	for _, sql := range []string{
		`INSERT INTO organization(id,name,slug,created_at) VALUES ('audience-org','Test','audience-org',now())`,
		`INSERT INTO projects(id,organization_id,name) VALUES ('audience-project','audience-org','Test')`,
		`INSERT INTO go_analytics_sites(project_id,site_key) VALUES ('audience-project','test-site-key')`,
		`INSERT INTO go_analytics_browsers(site_id,browser_hash,first_seen) SELECT id,decode(repeat('aa',32),'hex'),now() FROM go_analytics_sites`,
		`INSERT INTO go_analytics_events(site_id,occurred_at,path,channel,device,visitor_hash) SELECT id,now(),'/','direct','mobile',decode(repeat('aa',32),'hex') FROM go_analytics_sites`,
		`INSERT INTO go_analytics_custom_events(site_id,occurred_at,name,path,visitor_hash) SELECT id,now(),'Click','/',decode(repeat('aa',32),'hex') FROM go_analytics_sites`,
	} {
		if _, err := pool.Exec(ctx, sql); err != nil {
			t.Fatal(err)
		}
	}
	for _, table := range []string{"go_analytics_browsers", "go_analytics_events", "go_analytics_custom_events"} {
		t.Run(table, func(t *testing.T) {
			tx, err := pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
			role := "audience_public_" + strings.ToLower(rand.Text())
			for _, sql := range []string{"CREATE ROLE " + role + " NOLOGIN", "GRANT USAGE ON SCHEMA public TO " + role, "GRANT SELECT,INSERT,UPDATE,DELETE ON " + table + " TO " + role} {
				if _, err := tx.Exec(ctx, sql); err != nil {
					t.Fatal(err)
				}
			}
			var count int
			if err := tx.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil || count != 1 {
				t.Fatalf("owner access=%d %v", count, err)
			}
			if _, err := tx.Exec(ctx, "SET LOCAL ROLE "+role); err != nil {
				t.Fatal(err)
			}
			if err := tx.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil || count != 0 {
				t.Fatalf("exposed rows=%d %v", count, err)
			}
			for _, sql := range []string{"UPDATE " + table + " SET site_id=site_id", "DELETE FROM " + table} {
				tag, err := tx.Exec(ctx, sql)
				if err != nil || tag.RowsAffected() != 0 {
					t.Fatalf("public write=%v %v", tag, err)
				}
			}
			_, err = tx.Exec(ctx, "INSERT INTO "+table+"(site_id) VALUES (1)")
			if pgErr, ok := errors.AsType[*pgconn.PgError](err); !ok || pgErr.Code != "42501" {
				t.Fatalf("insert not blocked by RLS: %v", err)
			}
		})
	}
	if _, err := pool.Exec(ctx, `DELETE FROM projects WHERE id='audience-project'`); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"go_analytics_browsers", "go_analytics_events", "go_analytics_custom_events"} {
		var count int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("project deletion left %s rows: %d %v", table, count, err)
		}
	}
}
