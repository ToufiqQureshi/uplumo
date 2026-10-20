package indexnow

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

const testKey = "0123456789abcdef0123456789abcdef"

type memoryStore struct {
	mu     sync.Mutex
	domain *string
	keys   map[string]string
	err    error
}

func (m *memoryStore) Domain(_ context.Context, organizationID, projectID string) (*string, error) {
	if organizationID != "org" || projectID != "p" {
		return nil, ErrProjectNotFound
	}
	return m.domain, m.err
}

func (m *memoryStore) Key(_ context.Context, projectID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.keys[projectID], m.err
}

func (m *memoryStore) CreateKey(_ context.Context, projectID, key string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.keys == nil {
		m.keys = map[string]string{}
	}
	if existing := m.keys[projectID]; existing != "" {
		return existing, nil
	}
	m.keys[projectID] = key
	return key, nil
}

type planStub struct {
	included bool
	err      error
}

func (p planStub) HasPaidPlan(context.Context, string) (bool, error) { return p.included, p.err }

type limiterStub struct {
	mu    sync.Mutex
	left  int
	calls int
	err   error
}

func (l *limiterStub) Allow(context.Context, string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls++
	if l.err != nil {
		return false, l.err
	}
	if l.left == 0 {
		return false, nil
	}
	l.left--
	return true, nil
}

// roundTrip sends every request to handler, whatever its host.
type roundTrip struct{ handler http.Handler }

func (rt roundTrip) RoundTrip(r *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	rt.handler.ServeHTTP(recorder, r)
	response := recorder.Result()
	response.Request = r
	return response, nil
}

// fakeWeb serves key files per host and records engine submissions.
type fakeWeb struct {
	mu sync.Mutex
	// files maps a host to its key-file handler; missing hosts answer 404.
	files map[string]http.HandlerFunc
	// engines maps an endpoint host to the status it answers.
	engines     map[string]int
	submissions []submission
	fileChecks  int
}

type submission struct {
	Engine  string
	Host    string   `json:"host"`
	Key     string   `json:"key"`
	URLList []string `json:"urlList"`
}

func (f *fakeWeb) sites(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.fileChecks++
	handler := f.files[r.URL.Hostname()]
	f.mu.Unlock()
	if handler == nil || r.URL.Path != "/"+testKey+".txt" {
		http.NotFound(w, r)
		return
	}
	handler(w, r)
}

func (f *fakeWeb) engine(w http.ResponseWriter, r *http.Request) {
	var got submission
	body, _ := io.ReadAll(r.Body)
	_ = json.Unmarshal(body, &got)
	got.Engine = r.URL.Hostname()
	f.mu.Lock()
	f.submissions = append(f.submissions, got)
	status := f.engines[got.Engine]
	f.mu.Unlock()
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
}

func serveKey(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, testKey+"\n") }

func newTestService(t *testing.T, web *fakeWeb, store *memoryStore, plans Plans, limiter *limiterStub) *Service {
	t.Helper()
	return &Service{
		Store: store, Plans: plans, Limiter: limiter,
		Sites:     &http.Client{Transport: roundTrip{http.HandlerFunc(web.sites)}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		Engines:   &http.Client{Transport: roundTrip{http.HandlerFunc(web.engine)}},
		Endpoints: Engines,
	}
}

func domain(d string) *string { return &d }

func readyStore() *memoryStore {
	return &memoryStore{domain: domain("example.com"), keys: map[string]string{"p": testKey}}
}

func TestSubmitGroupsByHostAndSendsOneRequestPerHost(t *testing.T) {
	web := &fakeWeb{files: map[string]http.HandlerFunc{"example.com": serveKey, "www.example.com": serveKey, "blog.example.com": serveKey}}
	limiter := &limiterStub{left: 10}
	service := newTestService(t, web, readyStore(), planStub{included: true}, limiter)

	result, err := service.Submit(t.Context(), "org", "p", []string{
		"https://example.com/a",
		"https://example.com/a#section", // same page
		"https://EXAMPLE.com/b",
		"https://www.example.com/c",
		"https://blog.example.com/d",
		"https://shop.example.com/e", // fourth host: skipped
		"https://other.com/x",
		"https://notexample.com/x",
		"ftp://example.com/x",
		"https://user@example.com/x",
		"https://example.com:8443/x",
		"not a url",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Hosts) != 3 {
		t.Fatalf("hosts = %+v, want 3", result.Hosts)
	}
	if result.SkippedCount != 7 {
		t.Errorf("skipped = %d %v, want 7", result.SkippedCount, result.Skipped)
	}
	if len(web.submissions) != 3 {
		t.Fatalf("engine requests = %d, want one per host", len(web.submissions))
	}
	first := web.submissions[0]
	if first.Engine != "www.bing.com" || first.Host != "example.com" || first.Key != testKey ||
		strings.Join(first.URLList, " ") != "https://example.com/a https://example.com/b" {
		t.Errorf("first submission = %+v", first)
	}
	for _, host := range result.Hosts {
		if host.Status != "submitted" {
			t.Errorf("%s = %+v", host.Host, host)
		}
	}
	if limiter.calls != 1 {
		t.Errorf("limiter calls = %d, want 1 per submission", limiter.calls)
	}
}

func TestSubmitKeepsOnlyTwentySkippedURLs(t *testing.T) {
	web := &fakeWeb{}
	service := newTestService(t, web, readyStore(), planStub{included: true}, &limiterStub{left: 1})
	urls := make([]string, 30)
	for i := range urls {
		urls[i] = "https://other.com/page"
	}
	result, err := service.Submit(t.Context(), "org", "p", urls)
	if err != nil {
		t.Fatal(err)
	}
	if result.SkippedCount != 30 || len(result.Skipped) != maxSkippedShown || len(web.submissions) != 0 || web.fileChecks != 0 {
		t.Fatalf("result = %+v, submissions %d, checks %d", result, len(web.submissions), web.fileChecks)
	}
}

func TestSubmitTriesTheNextEngineOnlyWhenOneIsUnavailable(t *testing.T) {
	for _, test := range []struct {
		name       string
		engines    map[string]int
		wantCalls  int
		wantStatus string
		wantText   string
	}{
		{"bing accepts", nil, 1, "submitted", "Submitted 1 URL."},
		{"unverified key", map[string]int{"www.bing.com": 202}, 1, "submitted", "key file"},
		{"rate limited then accepted", map[string]int{"www.bing.com": 429}, 2, "submitted", "Submitted"},
		{"server errors everywhere", map[string]int{"www.bing.com": 500, "yandex.com": 503, "indexnow.yep.com": 502}, 3, "failed", "HTTP 502"},
		{"key rejected stops", map[string]int{"www.bing.com": 403}, 1, "failed", "key was not accepted"},
		{"bad request stops", map[string]int{"www.bing.com": 422}, 1, "failed", "refused"},
	} {
		t.Run(test.name, func(t *testing.T) {
			web := &fakeWeb{files: map[string]http.HandlerFunc{"example.com": serveKey}, engines: test.engines}
			service := newTestService(t, web, readyStore(), planStub{included: true}, &limiterStub{left: 1})
			result, err := service.Submit(t.Context(), "org", "p", []string{"https://example.com/"})
			if err != nil {
				t.Fatal(err)
			}
			if len(web.submissions) != test.wantCalls {
				t.Errorf("engine requests = %d, want %d", len(web.submissions), test.wantCalls)
			}
			host := result.Hosts[0]
			if host.Status != test.wantStatus || !strings.Contains(host.Message, test.wantText) {
				t.Errorf("result = %+v", host)
			}
		})
	}
}

func TestSubmitStopsBeforeTheEnginesWhenTheKeyFileIsWrong(t *testing.T) {
	for _, test := range []struct {
		name       string
		file       http.HandlerFunc
		wantText   string
		wantEngine bool
	}{
		{"missing", nil, "No key file", false},
		{"wrong content", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "<html>home</html>") }, "only the key", false},
		{"redirect", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/", http.StatusFound) }, "redirects", false},
		{"moved to www", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://www.example.com/"+testKey+".txt", http.StatusMovedPermanently)
		}, "redirects", false},
		// A bot wall or server error is inconclusive: the engines decide.
		{"bot wall", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) }, "Submitted", true},
		{"server error", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) }, "Submitted", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			web := &fakeWeb{files: map[string]http.HandlerFunc{}}
			if test.file != nil {
				web.files["example.com"] = test.file
			}
			service := newTestService(t, web, readyStore(), planStub{included: true}, &limiterStub{left: 1})
			result, err := service.Submit(t.Context(), "org", "p", []string{"https://example.com/"})
			if err != nil {
				t.Fatal(err)
			}
			if got := len(web.submissions) > 0; got != test.wantEngine {
				t.Errorf("reached an engine = %v, want %v", got, test.wantEngine)
			}
			if !strings.Contains(result.Hosts[0].Message, test.wantText) {
				t.Errorf("message = %q, want %q", result.Hosts[0].Message, test.wantText)
			}
		})
	}
}

func TestSubmitRefusesBeforeAnyOutboundCall(t *testing.T) {
	for _, test := range []struct {
		name        string
		store       *memoryStore
		plans       Plans
		limiter     *limiterStub
		project     string
		want        error
		wantLimiter int
	}{
		// Each case passes every earlier check, so it fails for its own reason.
		{"plan without indexnow", readyStore(), planStub{included: false}, &limiterStub{left: 1}, "p", ErrNotIncluded, 0},
		{"another workspace's project", readyStore(), planStub{included: true}, &limiterStub{left: 1}, "q", ErrProjectNotFound, 0},
		{"no website", &memoryStore{keys: map[string]string{"p": testKey}}, planStub{included: true}, &limiterStub{left: 1}, "p", ErrNoDomain, 0},
		{"not set up", &memoryStore{domain: domain("example.com")}, planStub{included: true}, &limiterStub{left: 1}, "p", ErrNotSetUp, 0},
		{"hourly limit used", readyStore(), planStub{included: true}, &limiterStub{left: 0}, "p", ErrRateLimited, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			web := &fakeWeb{files: map[string]http.HandlerFunc{"example.com": serveKey}}
			service := newTestService(t, web, test.store, test.plans, test.limiter)
			_, err := service.Submit(t.Context(), "org", test.project, []string{"https://example.com/"})
			if !errors.Is(err, test.want) {
				t.Fatalf("err = %v, want %v", err, test.want)
			}
			if web.fileChecks != 0 || len(web.submissions) != 0 {
				t.Errorf("outbound calls: %d checks, %d submissions", web.fileChecks, len(web.submissions))
			}
			if test.limiter.calls != test.wantLimiter {
				t.Errorf("limiter calls = %d, want %d", test.limiter.calls, test.wantLimiter)
			}
		})
	}
}

func TestLimiterAndPlanFailuresFailClosed(t *testing.T) {
	web := &fakeWeb{files: map[string]http.HandlerFunc{"example.com": serveKey}}
	down := errors.New("redis down")
	service := newTestService(t, web, readyStore(), planStub{included: true}, &limiterStub{err: down})
	if _, err := service.Submit(t.Context(), "org", "p", []string{"https://example.com/"}); !errors.Is(err, down) {
		t.Fatalf("limiter down: err = %v", err)
	}
	service = newTestService(t, web, readyStore(), planStub{err: down}, &limiterStub{left: 1})
	if _, err := service.Submit(t.Context(), "org", "p", []string{"https://example.com/"}); !errors.Is(err, down) {
		t.Fatalf("billing down: err = %v", err)
	}
	if web.fileChecks != 0 || len(web.submissions) != 0 {
		t.Errorf("outbound calls while a check was down: %d, %d", web.fileChecks, len(web.submissions))
	}
}

func TestSetUpKeepsTheFirstKey(t *testing.T) {
	store := &memoryStore{domain: domain("Example.com")}
	service := newTestService(t, &fakeWeb{}, store, planStub{included: true}, &limiterStub{})
	var wg sync.WaitGroup
	keys := make([]string, 8)
	for i := range keys {
		wg.Go(func() {
			setup, err := service.SetUp(t.Context(), "org", "p")
			if err != nil {
				t.Error(err)
			}
			keys[i] = setup.Key
		})
	}
	wg.Wait()
	for _, key := range keys {
		if key != keys[0] || len(key) != 32 {
			t.Fatalf("keys = %v, want one 32-character key", keys)
		}
	}
	setup, err := service.Get(t.Context(), "org", "p")
	if err != nil || setup.Key != keys[0] || setup.Domain != "example.com" {
		t.Fatalf("get = %+v, %v", setup, err)
	}

	if _, err := newTestService(t, &fakeWeb{}, &memoryStore{domain: domain("example.com")}, planStub{}, &limiterStub{}).SetUp(t.Context(), "org", "p"); !errors.Is(err, ErrNotIncluded) {
		t.Errorf("free plan set up: err = %v", err)
	}
	if _, err := newTestService(t, &fakeWeb{}, &memoryStore{}, planStub{included: true}, &limiterStub{}).SetUp(t.Context(), "org", "p"); !errors.Is(err, ErrNoDomain) {
		t.Errorf("no website set up: err = %v", err)
	}
}

func TestCheckKeyFilesChecksBareAndWWWHosts(t *testing.T) {
	web := &fakeWeb{files: map[string]http.HandlerFunc{"example.com": func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://www.example.com/"+testKey+".txt", http.StatusMovedPermanently)
	}, "www.example.com": serveKey}}
	service := newTestService(t, web, readyStore(), planStub{included: true}, &limiterStub{left: 1})
	files, err := service.CheckKeyFiles(t.Context(), "org", "p")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 || files[0] != (KeyFile{"example.com", "moved"}) || files[1] != (KeyFile{"www.example.com", "found"}) {
		t.Fatalf("files = %+v", files)
	}

	www := &memoryStore{domain: domain("www.example.com"), keys: map[string]string{"p": testKey}}
	files, err = newTestService(t, web, www, planStub{included: true}, &limiterStub{left: 1}).CheckKeyFiles(t.Context(), "org", "p")
	if err != nil || len(files) != 1 {
		t.Fatalf("www domain files = %+v, %v", files, err)
	}
}

func TestSubmitPublished(t *testing.T) {
	for _, test := range []struct {
		name        string
		store       *memoryStore
		plans       Plans
		limiter     *limiterStub
		page        string
		wantStatus  string
		wantText    string
		wantEngines int
	}{
		{"set up", readyStore(), planStub{included: true}, &limiterStub{left: 1}, "https://example.com/post", "submitted", "Submitted 1 URL.", 1},
		// Setting up the key is how a customer turns this on.
		{"not set up", &memoryStore{domain: domain("example.com")}, planStub{included: true}, &limiterStub{left: 1}, "https://example.com/post", "", "", 0},
		{"plan without indexnow", readyStore(), planStub{}, &limiterStub{left: 1}, "https://example.com/post", "", "", 0},
		{"no website", &memoryStore{keys: map[string]string{"p": testKey}}, planStub{included: true}, &limiterStub{left: 1}, "https://example.com/post", "", "", 0},
		{"hourly limit used", readyStore(), planStub{included: true}, &limiterStub{}, "https://example.com/post", "failed", "this hour", 0},
		{"limiter down", readyStore(), planStub{included: true}, &limiterStub{err: errors.New("redis down")}, "https://example.com/post", "failed", "error on our side", 0},
		{"wordpress on another domain", readyStore(), planStub{included: true}, &limiterStub{left: 1}, "https://blog.other.com/post", "failed", "not on this project's website", 0},
		{"lookalike domain", readyStore(), planStub{included: true}, &limiterStub{left: 1}, "https://notexample.com/post", "failed", "not on this project's website", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			web := &fakeWeb{files: map[string]http.HandlerFunc{"example.com": serveKey}}
			service := newTestService(t, web, test.store, test.plans, test.limiter)
			status, message := service.SubmitPublished(t.Context(), "org", "p", test.page)
			if status != test.wantStatus || !strings.Contains(message, test.wantText) {
				t.Errorf("SubmitPublished = %q %q", status, message)
			}
			if len(web.submissions) != test.wantEngines {
				t.Errorf("engine requests = %d, want %d", len(web.submissions), test.wantEngines)
			}
		})
	}
}
