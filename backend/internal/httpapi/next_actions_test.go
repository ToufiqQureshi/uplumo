package httpapi

import (
	"context"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/dashboard"
	"github.com/toufiqqureshi/uplumo/backend/internal/database"
	"github.com/toufiqqureshi/uplumo/backend/internal/site"
)

func TestNextActionsAreSharedByTheTeamAndScopedToTheProjectOrganization(t *testing.T) {
	ctx := context.Background()
	pool := openTestDB(ctx, t)
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	id := rand.Text()
	asha, ravi, stranger := "na-asha-"+id, "na-ravi-"+id, "na-stranger-"+id
	ownOrg, otherOrg := "na-own-"+id, "na-other-"+id
	ownProject, otherProject := "na-own-p-"+id, "na-other-p-"+id
	ashaToken, raviToken := "na-asha-t-"+id, "na-ravi-t-"+id
	t.Cleanup(func() {
		cleanupCtx := context.WithoutCancel(ctx)
		exec(cleanupCtx, t, pool, `DELETE FROM "user" WHERE id IN ($1,$2,$3)`, asha, ravi, stranger)
		exec(cleanupCtx, t, pool, `DELETE FROM organization WHERE id IN ($1,$2)`, ownOrg, otherOrg)
	})
	exec(ctx, t, pool, `INSERT INTO "user" (id,name,email) VALUES ($1,'Asha',$1 || '@example.com'),($2,'Ravi',$2 || '@example.com'),($3,'Stranger',$3 || '@example.com')`, asha, ravi, stranger)
	exec(ctx, t, pool, `INSERT INTO organization (id,name,slug,created_at) VALUES ($1,'Own',$1,now()),($2,'Other',$2,now())`, ownOrg, otherOrg)
	exec(ctx, t, pool, `INSERT INTO member (id,organization_id,user_id,created_at) VALUES ($1,$2,$3,now()),($4,$2,$5,now()),($6,$7,$8,now())`, "na-m1-"+id, ownOrg, asha, "na-m2-"+id, ravi, "na-m3-"+id, otherOrg, stranger)
	exec(ctx, t, pool, `INSERT INTO projects (id,organization_id,name) VALUES ($1,$2,'Own'),($3,$4,'Other')`, ownProject, ownOrg, otherProject, otherOrg)
	exec(ctx, t, pool, `INSERT INTO session (id,token,user_id,expires_at,updated_at) VALUES ($1,$2,$3,now()+interval '1 hour',now()),($4,$5,$6,now()+interval '1 hour',now())`, "na-s1-"+id, ashaToken, asha, "na-s2-"+id, raviToken, ravi)

	pages, err := site.New(&url.URL{Scheme: "https", Host: "uplumo.com"})
	if err != nil {
		t.Fatal(err)
	}
	store := dashboard.Store{DB: pool}
	h := NewHandler(Deps{
		Logger: discardLogger, DB: healthy, Redis: healthy, Auth: auth.NewService(pool, testSecret),
		DashboardLayouts: store, DashboardActions: store, Site: pages,
	})
	call := func(token, project, action, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/v1/projects/"+project+"/next-actions/"+action, strings.NewReader(body))
		if token != "" {
			req.AddCookie(&http.Cookie{Name: "__Secure-better-auth.session_token", Value: sign(token), HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	states := func(token string) string {
		t.Helper()
		rec := call(token, ownProject, "get", `{}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("get: status=%d body=%s", rec.Code, rec.Body)
		}
		return rec.Body.String()
	}

	assertError(t, call("", ownProject, "get", `{}`), http.StatusUnauthorized, "unauthenticated")
	assertError(t, call(ashaToken, otherProject, "set", `{"key":"audit:title","status":"done"}`), http.StatusNotFound, "project_not_found")
	assertError(t, call(ashaToken, ownProject, "set", `{"key":"Audit Title","status":"done"}`), http.StatusBadRequest, "invalid_request")
	assertError(t, call(ashaToken, ownProject, "set", `{"key":"audit:title","status":"archived"}`), http.StatusBadRequest, "invalid_request")
	assertError(t, call(ashaToken, ownProject, "set", `{"key":"audit:title","status":"done","extra":1}`), http.StatusBadRequest, "invalid_request")

	if rec := call(ashaToken, ownProject, "set", `{"key":"audit:title","status":"done"}`); rec.Code != http.StatusOK {
		t.Fatalf("set: status=%d body=%s", rec.Code, rec.Body)
	}
	// Ravi is on the same team: he sees Asha's decision and can change it.
	if got := states(raviToken); !strings.Contains(got, `"key":"audit:title","status":"done"`) {
		t.Fatalf("teammate states = %s, want Asha's decision", got)
	}
	if rec := call(raviToken, ownProject, "set", `{"key":"audit:title","status":"ignored"}`); rec.Code != http.StatusOK {
		t.Fatalf("change: status=%d body=%s", rec.Code, rec.Body)
	}
	if got := states(ashaToken); !strings.Contains(got, `"status":"ignored"`) || strings.Contains(got, `"status":"done"`) {
		t.Fatalf("states after change = %s, want one ignored decision", got)
	}
	// Back to "to do" removes the row.
	if rec := call(ashaToken, ownProject, "set", `{"key":"audit:title","status":"todo"}`); rec.Code != http.StatusOK {
		t.Fatalf("reset: status=%d body=%s", rec.Code, rec.Body)
	}
	if got := strings.TrimSpace(states(ashaToken)); got != `{"states":[]}` {
		t.Fatalf("states after reset = %s, want none", got)
	}

	// The cap refuses a new key but still lets an existing one change.
	exec(ctx, t, pool, `INSERT INTO go_next_action_states (project_id,organization_id,action_key,status,updated_by)
		SELECT $1,$2,'bulk:'||g,'done',$3 FROM generate_series(1,$4::int) g`, ownProject, ownOrg, asha, dashboard.MaxActionStates)
	assertError(t, call(ashaToken, ownProject, "set", `{"key":"audit:new","status":"done"}`), http.StatusConflict, "too_many_actions")
	if rec := call(ashaToken, ownProject, "set", `{"key":"bulk:1","status":"ignored"}`); rec.Code != http.StatusOK {
		t.Fatalf("change at cap: status=%d body=%s", rec.Code, rec.Body)
	}
	var otherRows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM go_next_action_states WHERE project_id=$1`, otherProject).Scan(&otherRows); err != nil || otherRows != 0 {
		t.Fatalf("the other workspace's project holds %d rows (err %v)", otherRows, err)
	}
}
