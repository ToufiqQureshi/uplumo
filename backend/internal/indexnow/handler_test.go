package indexnow

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/toufiqqureshi/uplumo/backend/internal/auth"
)

func testMux(service *Service) *http.ServeMux {
	mux := http.NewServeMux()
	Mount(mux, Deps{
		Service: service,
		WithSession: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				user := auth.User{ID: "user-1", OrganizationID: "org", Role: "member"}
				next.ServeHTTP(w, r.WithContext(auth.WithUser(r.Context(), user)))
			})
		},
		WithProjectAccess: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r.WithContext(auth.WithProjectOrganization(r.Context(), "org")))
			})
		},
	})
	return mux
}

func post(t *testing.T, mux *http.ServeMux, path, body string) (int, map[string]any) {
	t.Helper()
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, strings.NewReader(body)))
	var decoded map[string]any
	_ = json.Unmarshal(recorder.Body.Bytes(), &decoded)
	return recorder.Code, decoded
}

func errorCode(body map[string]any) string {
	if inner, ok := body["error"].(map[string]any); ok {
		code, _ := inner["code"].(string)
		return code
	}
	code, _ := body["code"].(string)
	return code
}

func TestHandlerSetUpCheckAndSubmit(t *testing.T) {
	web := &fakeWeb{files: map[string]http.HandlerFunc{"example.com": func(w http.ResponseWriter, r *http.Request) { serveKey(w, r) }}}
	store := &memoryStore{domain: domain("example.com")}
	mux := testMux(newTestService(t, web, store, planStub{included: true}, &limiterStub{left: 10}))

	if status, body := post(t, mux, "/api/v1/projects/p/indexnow/get", "{}"); status != http.StatusOK || body["included"] != true || body["key"] != "" || body["domain"] != "example.com" {
		t.Fatalf("get before set up = %d %v", status, body)
	}
	if status, body := post(t, mux, "/api/v1/projects/p/indexnow/submit", `{"urls":["https://example.com/"]}`); status != http.StatusConflict || errorCode(body) != "indexnow_not_set_up" {
		t.Fatalf("submit before set up = %d %v", status, body)
	}
	status, body := post(t, mux, "/api/v1/projects/p/indexnow/setup", "{}")
	key, _ := body["key"].(string)
	if status != http.StatusOK || len(key) != 32 {
		t.Fatalf("set up = %d %v", status, body)
	}
	store.keys["p"] = testKey // the fake site serves testKey
	if status, body := post(t, mux, "/api/v1/projects/p/indexnow/check", "{}"); status != http.StatusOK || len(body["files"].([]any)) != 2 {
		t.Fatalf("check = %d %v", status, body)
	}
	status, body = post(t, mux, "/api/v1/projects/p/indexnow/submit", `{"urls":["https://example.com/a","https://other.com/b"]}`)
	if status != http.StatusOK || body["skippedCount"] != float64(1) || len(body["hosts"].([]any)) != 1 {
		t.Fatalf("submit = %d %v", status, body)
	}
	if len(web.submissions) != 1 {
		t.Errorf("engine requests = %d, want 1", len(web.submissions))
	}
}

func TestHandlerRefusals(t *testing.T) {
	ready := func(plans Plans, limiter *limiterStub) *http.ServeMux {
		return testMux(newTestService(t, &fakeWeb{}, readyStore(), plans, limiter))
	}
	tooMany := `{"urls":[` + strings.Repeat(`"https://example.com/",`, MaxURLs) + `"https://example.com/"]}`
	// 1,100 maximum-length URLs: under MaxURLs, over the 2 MB body cap.
	oversized := `{"urls":[` + strings.Repeat(`"https://example.com/`+strings.Repeat("a", 2000)+`",`, 1100) + `"https://example.com/"]}`
	longURL := `{"urls":["https://example.com/` + strings.Repeat("a", maxURLLength) + `"]}`
	for _, test := range []struct {
		name       string
		mux        *http.ServeMux
		path, body string
		wantStatus int
		wantCode   string
	}{
		{"free plan set up", ready(planStub{}, &limiterStub{left: 1}), "setup", "{}", http.StatusPaymentRequired, "plan_required"},
		{"free plan submit", ready(planStub{}, &limiterStub{left: 1}), "submit", `{"urls":["https://example.com/"]}`, http.StatusPaymentRequired, "plan_required"},
		{"free plan check", ready(planStub{}, &limiterStub{left: 1}), "check", "{}", http.StatusPaymentRequired, "plan_required"},
		{"hourly limit", ready(planStub{included: true}, &limiterStub{}), "submit", `{"urls":["https://example.com/"]}`, http.StatusTooManyRequests, "rate_limited"},
		{"no urls", ready(planStub{included: true}, &limiterStub{left: 1}), "submit", `{"urls":[]}`, http.StatusBadRequest, "invalid_request"},
		{"too many urls", ready(planStub{included: true}, &limiterStub{left: 1}), "submit", tooMany, http.StatusBadRequest, "invalid_request"},
		{"body over 2 MB", ready(planStub{included: true}, &limiterStub{left: 1}), "submit", oversized, http.StatusRequestEntityTooLarge, "too_large"},
		{"url too long", ready(planStub{included: true}, &limiterStub{left: 1}), "submit", longURL, http.StatusBadRequest, "invalid_request"},
		{"unknown field", ready(planStub{included: true}, &limiterStub{left: 1}), "submit", `{"urls":["https://example.com/"],"key":"x"}`, http.StatusBadRequest, "invalid_request"},
		{"fields on get", ready(planStub{included: true}, &limiterStub{left: 1}), "get", `{"x":1}`, http.StatusBadRequest, "invalid_request"},
		{"no website", testMux(newTestService(t, &fakeWeb{}, &memoryStore{}, planStub{included: true}, &limiterStub{left: 1})), "setup", "{}", http.StatusBadRequest, "no_website"},
		{"not configured", testMux(nil), "get", "{}", http.StatusServiceUnavailable, "indexnow_unavailable"},
	} {
		t.Run(test.name, func(t *testing.T) {
			status, body := post(t, test.mux, "/api/v1/projects/p/indexnow/"+test.path, test.body)
			if status != test.wantStatus || errorCode(body) != test.wantCode {
				t.Fatalf("= %d %v, want %d %s", status, body, test.wantStatus, test.wantCode)
			}
		})
	}
}

func TestHandlerGetTellsAFreePlanWithoutRefusing(t *testing.T) {
	mux := testMux(newTestService(t, &fakeWeb{}, readyStore(), planStub{}, &limiterStub{}))
	if status, body := post(t, mux, "/api/v1/projects/p/indexnow/get", "{}"); status != http.StatusOK || body["included"] != false {
		t.Fatalf("get on free = %d %v", status, body)
	}
}
