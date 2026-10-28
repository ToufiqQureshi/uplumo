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

	_, err = handleCiteFixReport(t.Context(), args, &callEnv{h: h, deps: h.deps, auth: Auth{OrganizationID: other}})
	if appErr, ok := errors.AsType[*appError](err); !ok || appErr.code != "FORBIDDEN" {
		t.Fatalf("another workspace got err %v, want FORBIDDEN", err)
	}

	if _, err := handleCiteFixReport(t.Context(), json.RawMessage(`{"projectId":" "}`), &callEnv{h: h, deps: h.deps, auth: Auth{OrganizationID: owner}}); err == nil {
		t.Fatal("a blank projectId must be rejected")
	}
}
