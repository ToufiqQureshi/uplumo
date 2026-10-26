package httpapi

import (
	"context"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
	"github.com/toufiqqureshi/uplumo/backend/internal/dashboard"
	"github.com/toufiqqureshi/uplumo/backend/internal/database"
	"github.com/toufiqqureshi/uplumo/backend/internal/site"
)

func TestDashboardLayoutIsPerMemberAndScopedToTheProjectOrganization(t *testing.T) {
	ctx := context.Background()
	pool := openTestDB(ctx, t)
	if err := database.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	id := rand.Text()
	asha, ravi, stranger := "dl-asha-"+id, "dl-ravi-"+id, "dl-stranger-"+id
	ownOrg, otherOrg := "dl-own-"+id, "dl-other-"+id
	ownProject, otherProject := "dl-own-p-"+id, "dl-other-p-"+id
	ashaToken, raviToken := "dl-asha-t-"+id, "dl-ravi-t-"+id
	t.Cleanup(func() {
		cleanupCtx := context.WithoutCancel(ctx)
		exec(cleanupCtx, t, pool, `DELETE FROM "user" WHERE id IN ($1,$2,$3)`, asha, ravi, stranger)
		exec(cleanupCtx, t, pool, `DELETE FROM organization WHERE id IN ($1,$2)`, ownOrg, otherOrg)
	})
	exec(ctx, t, pool, `INSERT INTO "user" (id,name,email) VALUES ($1,'Asha',$1 || '@example.com'),($2,'Ravi',$2 || '@example.com'),($3,'Stranger',$3 || '@example.com')`, asha, ravi, stranger)
	exec(ctx, t, pool, `INSERT INTO organization (id,name,slug,created_at) VALUES ($1,'Own',$1,now()),($2,'Other',$2,now())`, ownOrg, otherOrg)
	exec(ctx, t, pool, `INSERT INTO member (id,organization_id,user_id,created_at) VALUES ($1,$2,$3,now()),($4,$2,$5,now()),($6,$7,$8,now())`, "dl-m1-"+id, ownOrg, asha, "dl-m2-"+id, ravi, "dl-m3-"+id, otherOrg, stranger)
	exec(ctx, t, pool, `INSERT INTO projects (id,organization_id,name) VALUES ($1,$2,'Own'),($3,$4,'Other')`, ownProject, ownOrg, otherProject, otherOrg)
	exec(ctx, t, pool, `INSERT INTO session (id,token,user_id,expires_at,updated_at) VALUES ($1,$2,$3,now()+interval '1 hour',now()),($4,$5,$6,now()+interval '1 hour',now())`, "dl-s1-"+id, ashaToken, asha, "dl-s2-"+id, raviToken, ravi)

	pages, err := site.New(&url.URL{Scheme: "https", Host: "uplumo.com"})
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(Deps{
		Logger: discardLogger, DB: healthy, Redis: healthy, Auth: auth.NewService(pool, testSecret),
		DashboardLayouts: dashboard.Store{DB: pool}, Site: pages,
	})
	call := func(token, project, action, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/v1/projects/"+project+"/dashboard/layout/"+action, strings.NewReader(body))
		if token != "" {
			req.AddCookie(&http.Cookie{Name: "__Secure-better-auth.session_token", Value: sign(token), HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}
	expectBody := func(rec *httptest.ResponseRecorder, want string) {
		t.Helper()
		if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != want {
			t.Fatalf("status=%d body=%s, want 200 %s", rec.Code, rec.Body, want)
		}
	}
	layout := `{"cards":[{"id":"ai_visibility","hidden":false},{"id":"gsc","hidden":true}]}`
	var tooMany strings.Builder
	tooMany.WriteString(`{"cards":[`)
	for i := range dashboard.MaxCards + 1 {
		if i > 0 {
			tooMany.WriteString(",")
		}
		tooMany.WriteString(`{"id":"c` + strconv.Itoa(i) + `","hidden":false}`)
	}
	tooMany.WriteString(`]}`)

	assertError(t, call("", ownProject, "get", `{}`), http.StatusUnauthorized, "unauthenticated")
	assertError(t, call(ashaToken, otherProject, "save", layout), http.StatusNotFound, "project_not_found")
	var rows int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM go_dashboard_layouts WHERE project_id=$1`, otherProject).Scan(&rows); err != nil || rows != 0 {
		t.Fatalf("a refused save wrote %d rows (err %v)", rows, err)
	}

	expectBody(call(ashaToken, ownProject, "get", `{}`), `{"cards":null}`)
	expectBody(call(ashaToken, ownProject, "save", layout), layout)
	expectBody(call(ashaToken, ownProject, "get", `{}`), layout)
	// A teammate on the same project keeps the default until they change it.
	expectBody(call(raviToken, ownProject, "get", `{}`), `{"cards":null}`)

	for name, body := range map[string]string{
		"missing cards": `{}`,
		"duplicate":     `{"cards":[{"id":"gsc","hidden":false},{"id":"gsc","hidden":true}]}`,
		"bad id":        `{"cards":[{"id":"GSC; DROP","hidden":false}]}`,
		"unknown field": `{"cards":[],"owner":"x"}`,
		"too many":      tooMany.String(),
	} {
		t.Run(name, func(t *testing.T) {
			assertError(t, call(ashaToken, ownProject, "save", body), http.StatusBadRequest, "invalid_request")
		})
	}
	expectBody(call(ashaToken, ownProject, "get", `{}`), layout)

	// Reset deletes the saved layout so cards added later appear by default.
	expectBody(call(ashaToken, ownProject, "save", `{"cards":[]}`), `{"cards":[]}`)
	expectBody(call(ashaToken, ownProject, "get", `{}`), `{"cards":null}`)
}
