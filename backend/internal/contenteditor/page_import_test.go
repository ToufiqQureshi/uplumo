package contenteditor

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
)

const importArticle = `<!doctype html><html><head><title>Old tab title</title>
<meta name="description" content="How to pick a CRM."></head><body>
<header><nav><a href="/">Home</a> Pricing Login</nav></header>
<main><h1>Best CRM for small teams</h1>
<img src="/hero.png" alt="hero">
<p>Picking a CRM is <strong>hard</strong>. Read our <a href="/pricing">pricing page</a>.</p>
<h2>What to look for</h2>
<ul><li>Email tracking</li><li>Simple pipeline</li></ul>
<h3>मूल्य और योजना</h3><p>हिंदी में सामग्री।</p>
<script>alert(1)</script></main>
<footer>© Example 2026</footer></body></html>`

// importSite serves pages for every host through one test server, so drafts
// can be imported from "http://example.test" without real DNS.
type importSite struct {
	server *httptest.Server
	hits   atomic.Int32
	pages  map[string]func(http.ResponseWriter, *http.Request)
}

func newImportSite(t *testing.T) *importSite {
	s := &importSite{pages: map[string]func(http.ResponseWriter, *http.Request){}}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.hits.Add(1)
		if page, ok := s.pages[strings.TrimPrefix(r.Host, "www.")+r.URL.Path]; ok {
			page(w, r)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(s.server.Close)
	return s
}

func (s *importSite) client(timeout time.Duration) *http.Client {
	addr := s.server.Listener.Addr().String()
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		}},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func html200(body string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(body))
	}
}

type importFixture struct {
	mux       *http.ServeMux
	site      *importSite
	repo      Repository
	projectID string
	noDomain  string
}

func newImportFixture(ctx context.Context, t *testing.T, timeout time.Duration) importFixture {
	pool := contentTestPool(ctx, t)
	suffix := strings.ReplaceAll(time.Now().UTC().Format("150405.000000000"), ".", "")
	org := "import-org-" + suffix
	if _, err := pool.Exec(ctx, `INSERT INTO organization (id, name, slug, created_at) VALUES ($1,$1,$1,now())`, org); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM organization WHERE id=$1`, org) })
	f := importFixture{site: newImportSite(t), repo: Repository{DB: pool}, projectID: "import-p-" + suffix, noDomain: "import-nd-" + suffix}
	if _, err := pool.Exec(ctx, `INSERT INTO projects (id, organization_id, name, domain) VALUES ($1,$2,$1,'example.test'),($3,$2,$3,NULL)`, f.projectID, org, f.noDomain); err != nil {
		t.Fatal(err)
	}
	pass := func(next http.Handler) http.Handler { return next }
	withOrg := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(auth.WithProjectOrganization(r.Context(), org)))
		})
	}
	f.mux = http.NewServeMux()
	MountImport(f.mux, &PageImporter{Drafts: f.repo, Client: f.site.client(timeout)}, pass, withOrg)
	return f
}

func (f importFixture) post(t *testing.T, project, pageURL string) (int, Draft, string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"url": pageURL})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+project+"/content/drafts/import", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	f.mux.ServeHTTP(rec, req)
	var draft Draft
	var failure struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &draft)
	_ = json.Unmarshal(rec.Body.Bytes(), &failure)
	return rec.Code, draft, failure.Error.Code + rec.Body.String()
}

func TestImportPageBuildsAnEditableDraft(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	f := newImportFixture(ctx, t, 2*time.Second)
	f.site.pages["example.test/blog/best-crm"] = html200(importArticle)

	code, draft, raw := f.post(t, f.projectID, "http://www.example.test/blog/best-crm#top")
	if code != http.StatusOK {
		t.Fatalf("status %d: %s", code, raw)
	}
	if draft.Title != "Best CRM for small teams" || draft.MetaDescription != "How to pick a CRM." || draft.Slug != "blog/best-crm" {
		t.Fatalf("fields: %+v", draft)
	}
	for _, want := range []string{
		"Picking a CRM is **hard**. Read our [pricing page](http://www.example.test/pricing).",
		"## What to look for",
		"- Email tracking\n- Simple pipeline",
		"### मूल्य और योजना",
		"हिंदी में सामग्री।",
	} {
		if !strings.Contains(draft.Content, want) {
			t.Errorf("content lacks %q:\n%s", want, draft.Content)
		}
	}
	for _, unwanted := range []string{"# Best CRM", "Pricing Login", "alert(1)", "© Example", "hero", "Old tab title"} {
		if strings.Contains(draft.Content, unwanted) {
			t.Errorf("content keeps %q:\n%s", unwanted, draft.Content)
		}
	}

	// Opening the same page again returns the draft with its edits, unfetched.
	edited := Input{ID: draft.ID, Title: draft.Title, Content: "My edits"}
	if _, err := f.repo.Save(ctx, f.projectID, edited); err != nil {
		t.Fatal(err)
	}
	hits := f.site.hits.Load()
	code, again, raw := f.post(t, f.projectID, "http://www.example.test/blog/best-crm")
	if code != http.StatusOK || again.ID != draft.ID || again.Content != "My edits" || f.site.hits.Load() != hits {
		t.Fatalf("second import: %d %+v %s (hits %d→%d)", code, again, raw, hits, f.site.hits.Load())
	}
}

func TestImportPageRefusesAndExplains(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	f := newImportFixture(ctx, t, 300*time.Millisecond)
	f.site.pages["example.test/moved"] = func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://evil.test/page", http.StatusFound)
	}
	f.site.pages["evil.test/page"] = html200(importArticle)
	f.site.pages["example.test/file.pdf"] = func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("%PDF"))
	}
	f.site.pages["example.test/empty"] = html200("<html><body><nav>Only navigation</nav></body></html>")
	f.site.pages["example.test/slow"] = func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second):
		case <-r.Context().Done():
		}
	}
	for _, tc := range []struct {
		name, project, url string
		status             int
		code               string
	}{
		{"another site", f.projectID, "http://evil.test/page", 400, "other_site"},
		{"look-alike domain", f.projectID, "http://notexample.test/page", 400, "other_site"},
		{"not a web link", f.projectID, "javascript:alert(1)", 400, "other_site"},
		{"credentials in url", f.projectID, "http://user:pass@example.test/", 400, "other_site"},
		{"too long", f.projectID, "http://example.test/" + strings.Repeat("a", 2100), 400, "other_site"},
		{"empty url", f.projectID, "", 400, "invalid_request"},
		{"no website", f.noDomain, "http://example.test/a", 400, "no_domain"},
		{"redirect off the site", f.projectID, "http://example.test/moved", 422, "page_unreadable"},
		{"not html", f.projectID, "http://example.test/file.pdf", 422, "page_unreadable"},
		{"no text", f.projectID, "http://example.test/empty", 422, "page_unreadable"},
		{"missing", f.projectID, "http://example.test/nope", 422, "page_unreadable"},
		{"timeout", f.projectID, "http://example.test/slow", 422, "page_unreadable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code, _, raw := f.post(t, tc.project, tc.url)
			if code != tc.status || !strings.HasPrefix(raw, tc.code) {
				t.Fatalf("got %d %s, want %d %s", code, raw, tc.status, tc.code)
			}
		})
	}
	if n, _ := f.repo.List(ctx, f.projectID); len(n) != 0 {
		t.Fatalf("refused imports created drafts: %d", len(n))
	}
}

func TestImportPageKeepsHugePagesWithinTheDraftLimit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	f := newImportFixture(ctx, t, 5*time.Second)
	big := "<html><body><main><h1>Long</h1>" + strings.Repeat("<p>"+strings.Repeat("word ", 200)+"</p>", 400) + "</main></body></html>"
	f.site.pages["example.test/long"] = html200(big)
	code, draft, raw := f.post(t, f.projectID, "http://example.test/long")
	if code != http.StatusOK || len(draft.Content) > MaxContentLength || len(draft.Content) < MaxContentLength/2 {
		t.Fatalf("%d len=%d %s", code, len(draft.Content), raw[:min(len(raw), 200)])
	}
}

func TestConcurrentImportsOfOnePageMakeOneDraft(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	f := newImportFixture(ctx, t, 5*time.Second)
	f.site.pages["example.test/a"] = func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		html200(importArticle)(w, r)
	}
	var wg sync.WaitGroup
	ids := make([]string, 5)
	for i := range ids {
		wg.Go(func() {
			code, draft, raw := f.post(t, f.projectID, "http://example.test/a")
			if code != http.StatusOK {
				t.Errorf("%d %s", code, raw)
			}
			ids[i] = draft.ID
		})
	}
	wg.Wait()
	drafts, err := f.repo.List(ctx, f.projectID)
	if err != nil || len(drafts) != 1 || f.site.hits.Load() != 1 {
		t.Fatalf("drafts=%d fetches=%d err=%v", len(drafts), f.site.hits.Load(), err)
	}
	for _, id := range ids {
		if id != drafts[0].ID {
			t.Fatalf("callers got different drafts: %v", ids)
		}
	}
}
