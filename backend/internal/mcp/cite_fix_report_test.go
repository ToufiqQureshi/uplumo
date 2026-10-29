package mcp

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/citefix"
	"github.com/toufiqqureshi/uplumo/backend/internal/database"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/pgdb"
)

func TestCiteFixReportToolReadsOnlyTheCallersProject(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		if os.Getenv("CI") != "" {
			t.Fatal("TEST_DATABASE_URL is required")
		}
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgdb.Open(t.Context(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := database.Migrate(t.Context(), pool); err != nil {
		t.Fatal(err)
	}
	owner, other := "mcp-citefix-"+rand.Text(), "mcp-citefix-"+rand.Text()
	projectID := "mcp-citefix-project-" + rand.Text()
	for _, org := range []string{owner, other} {
		if _, err := pool.Exec(t.Context(), `INSERT INTO organization(id,name,slug,created_at) VALUES($1,'Test',$1,now())`, org); err != nil {
			t.Fatal(err)
		}
	}
	defer func() {
		if _, err := pool.Exec(context.Background(), "DELETE FROM organization WHERE id = ANY($1)", []string{owner, other}); err != nil {
			t.Error(err)
		}
	}()
	if _, err := pool.Exec(t.Context(), `INSERT INTO projects (id, organization_id, name) VALUES ($1, $2, 'test')`, projectID, owner); err != nil {
		t.Fatal(err)
	}
	h := newHandler(Deps{DB: pool, Logger: slog.New(slog.DiscardHandler)})
	args := json.RawMessage(`{"projectId":"` + projectID + `"}`)

	result, err := handleCiteFixReport(t.Context(), args, &callEnv{h: h, deps: h.deps, auth: Auth{OrganizationID: owner}})
	if err != nil {
		t.Fatal(err)
	}
	structured, ok := result.StructuredContent.(map[string]any)
	if !ok || structured["state"] != "no_scan" || !strings.Contains(result.Content[0].Text, "run_site_audit") {
		t.Fatalf("owner got %+v, want no_scan pointing at run_site_audit", result)
	}
	if _, leaked := structured["agent"]; leaked {
		t.Fatal("the tool must not report an agent offer it did not price")
	}
	for _, freeField := range []string{"mentions", "rivals", "seoIssues", "topics", "proof"} {
		if _, exists := structured[freeField]; !exists {
			t.Fatalf("free report output missing %q", freeField)
		}
	}
	// The same authorized tool exposes topic estimates from saved audit pages.
	auditID := "mcp-topic-audit-" + rand.Text()
	if _, err := pool.Exec(t.Context(), `INSERT INTO go_audits(id,project_id,started_by_user_id,start_url,status,completed_at) VALUES($1,$2,'tester','https://shop.example/','completed',now())`, auditID, projectID); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM go_audits WHERE id=$1`, auditID); err != nil {
			t.Error(err)
		}
	}()
	if _, err := pool.Exec(t.Context(), `INSERT INTO go_audit_pages(id,audit_id,url,status_code,title,word_count,is_indexable,citation_signals_json) VALUES($1,$2,'https://shop.example/solar',200,'Solar panels guide',500,true,'{"version":2,"topicHeadings":["Solar panels"]}')`, auditID+"-page", auditID); err != nil {
		t.Fatal(err)
	}
	result, err = handleCiteFixReport(t.Context(), args, &callEnv{h: h, deps: h.deps, auth: Auth{OrganizationID: owner}})
	if err != nil {
		t.Fatal(err)
	}
	structured = result.StructuredContent.(map[string]any)
	topics, ok := structured["topics"].(*citefix.TopicReport)
	if !ok || topics == nil || len(topics.Clusters) == 0 || structured["state"] != "ready" {
		t.Fatalf("saved topic report = %+v", structured)
	}

	_, err = handleCiteFixReport(t.Context(), args, &callEnv{h: h, deps: h.deps, auth: Auth{OrganizationID: other}})
	if appErr, ok := errors.AsType[*appError](err); !ok || appErr.code != "FORBIDDEN" {
		t.Fatalf("another workspace got err %v, want FORBIDDEN", err)
	}

	if _, err := handleCiteFixReport(t.Context(), json.RawMessage(`{"projectId":" "}`), &callEnv{h: h, deps: h.deps, auth: Auth{OrganizationID: owner}}); err == nil {
		t.Fatal("a blank projectId must be rejected")
	}
}
