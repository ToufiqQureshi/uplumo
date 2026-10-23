package mcp

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"log/slog"
	"os"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/database"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/pgdb"
)

func TestOnlySuccessfulExternalToolsCompleteActivation(t *testing.T) {
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
	org := "mcp-activation-" + rand.Text()
	if _, err := pool.Exec(t.Context(), `INSERT INTO organization(id,name,slug,created_at) VALUES($1,'Test',$1,now())`, org); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := pool.Exec(context.Background(), "DELETE FROM organization WHERE id=$1", org); err != nil {
			t.Error(err)
		}
	}()
	h := newHandler(Deps{DB: pool, Logger: slog.New(slog.DiscardHandler)})
	candidate := &tool{Name: "test"}
	for _, tc := range []struct {
		external, fails bool
		want            int
	}{
		{false, false, 0}, {true, true, 0}, {true, false, 1}, {true, false, 1},
	} {
		identity := Auth{OrganizationID: org}
		if tc.external {
			identity.ClientID = "api_key"
		}
		candidate.Handler = func(context.Context, json.RawMessage, *callEnv) (*callResult, error) {
			if tc.fails {
				return errorResult("refused"), nil
			}
			return textResult("ok"), nil
		}
		h.callTool(t.Context(), candidate, json.RawMessage(`{}`), identity)
		var count int
		if err := pool.QueryRow(t.Context(), `SELECT count(*) FROM organization_activation_state WHERE organization_id=$1 AND first_mcp_tool_call_at IS NOT NULL`, org).Scan(&count); err != nil || count != tc.want {
			t.Fatalf("external=%v failed=%v: milestones=%d, want %d; err=%v", tc.external, tc.fails, count, tc.want, err)
		}
	}
}
