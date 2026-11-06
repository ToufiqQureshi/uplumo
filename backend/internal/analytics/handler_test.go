package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
)

func TestCollectRejectsInvalidEvents(t *testing.T) {
	long := "https://acme.com/" + strings.Repeat("a", maxURL)
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{name: "not JSON", body: "k=abc", wantStatus: http.StatusBadRequest, wantCode: "invalid_event"},
		{name: "invalid browser", body: `{"k":"abc","u":"https://acme.com/","b":"person@example.test"}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_event"},
		{name: "click without event identity", body: `{"k":"abc","u":"https://acme.com/","c":true,"n":"Click"}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_event"},
		{name: "destination on nonclick", body: `{"k":"abc","u":"https://acme.com/","x":"https://partner.test/"}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_event"},
		{name: "destination credentials", body: `{"k":"abc","u":"https://acme.com/","c":true,"n":"Click","s":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","p":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","i":"cccccccccccccccccccccccccccccccc","x":"https://userinfo@partner.test/"}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_event"},
		{name: "empty body", wantStatus: http.StatusBadRequest, wantCode: "invalid_event"},
		{name: "wrong type", body: `{"k":"abc","u":"https://acme.com/","w":"wide"}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_event"},
		{name: "missing site key", body: `{"u":"https://acme.com/"}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_event"},
		{name: "site key too long", body: fmt.Sprintf(`{"k":%q,"u":"https://acme.com/"}`, strings.Repeat("k", maxSiteKey+1)), wantStatus: http.StatusBadRequest, wantCode: "invalid_event"},
		{name: "relative URL", body: `{"k":"abc","u":"/pricing"}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_event"},
		{name: "non-http URL", body: `{"k":"abc","u":"file:///etc/passwd"}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_event"},
		{name: "URL without host", body: `{"k":"abc","u":"https:///x"}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_event"},
		{name: "URL too long", body: fmt.Sprintf(`{"k":"abc","u":%q}`, long[:maxURL+1]), wantStatus: http.StatusBadRequest, wantCode: "invalid_event"},
		{name: "referrer too long", body: fmt.Sprintf(`{"k":"abc","u":"https://acme.com/","r":%q}`, long[:maxURL+1]), wantStatus: http.StatusBadRequest, wantCode: "invalid_event"},
		{name: "negative width", body: `{"k":"abc","u":"https://acme.com/","w":-1}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_event"},
		{name: "absurd width", body: `{"k":"abc","u":"https://acme.com/","w":100001}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_event"},
		{name: "event name too long", body: fmt.Sprintf(`{"k":"abc","u":"https://acme.com/","n":%q}`, strings.Repeat("é", maxEventName+1)), wantStatus: http.StatusBadRequest, wantCode: "invalid_event"},
		{name: "event name only spaces", body: `{"k":"abc","u":"https://acme.com/","n":"   "}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_event"},
		{name: "event name with a control character", body: `{"k":"abc","u":"https://acme.com/","n":"Sign\nup"}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_event"},
		{name: "body over the limit", body: fmt.Sprintf(`{"k":"abc","u":"https://acme.com/","x":%q}`, strings.Repeat("x", maxCollectBody)), wantStatus: http.StatusRequestEntityTooLarge, wantCode: "payload_too_large"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			collect(nil, nil, nil)(rec, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/collect", strings.NewReader(tt.body)))
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body)
			}
			var got struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			if got.Error.Code != tt.wantCode {
				t.Fatalf("error code = %q, want %q", got.Error.Code, tt.wantCode)
			}
			if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
				t.Errorf("CORS origin = %q, want *", rec.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

func TestCollectPreflight(t *testing.T) {
	rec := httptest.NewRecorder()
	collectPreflight(rec, httptest.NewRequestWithContext(context.Background(), http.MethodOptions, "/collect", nil))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	for name, want := range map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "POST",
		"Access-Control-Allow-Headers": "Content-Type",
	} {
		if got := rec.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
}

func TestBehindUntrustedProxy(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		xff  string
		want bool
	}{
		{name: "railway edge without trust", ip: "100.64.0.3", xff: "203.0.113.9", want: true},
		{name: "private proxy", ip: "10.1.2.3", xff: "203.0.113.9", want: true},
		{name: "ipv4-mapped shared address", ip: "::ffff:100.100.1.1", xff: "203.0.113.9", want: true},
		{name: "public visitor address", ip: "203.0.113.9", xff: "203.0.113.9", want: false},
		{name: "private address without a proxy header", ip: "10.1.2.3", want: false},
		{name: "just outside the shared range", ip: "100.128.0.1", xff: "203.0.113.9", want: false},
		{name: "unparseable address", ip: "unix", xff: "203.0.113.9", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/collect", nil)
			if tt.xff != "" {
				r.Header.Set("X-Forwarded-For", tt.xff)
			}
			if got := behindUntrustedProxy(tt.ip, r); got != tt.want {
				t.Errorf("behindUntrustedProxy(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestCollectWarnsOnceAboutAnUntrustedProxy(t *testing.T) {
	var logs strings.Builder
	h := collect(slog.New(slog.NewTextHandler(&logs, nil)), nil, nil)
	for range 3 {
		r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/collect", strings.NewReader("not json"))
		r.RemoteAddr = "100.64.0.2:5000"
		r.Header.Set("X-Forwarded-For", "203.0.113.9")
		h(httptest.NewRecorder(), r)
	}
	if got := strings.Count(logs.String(), "TRUSTED_PROXY_CIDRS"); got != 1 {
		t.Errorf("proxy warnings = %d, want exactly 1:\n%s", got, logs.String())
	}

	logs.Reset()
	trusted := collect(slog.New(slog.NewTextHandler(&logs, nil)), nil, []netip.Prefix{netip.MustParsePrefix("100.64.0.0/10")})
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/collect", strings.NewReader("not json"))
	r.RemoteAddr = "100.64.0.2:5000"
	r.Header.Set("X-Forwarded-For", "203.0.113.9")
	trusted(httptest.NewRecorder(), r)
	if logs.Len() != 0 {
		t.Errorf("a trusted proxy must not warn:\n%s", logs.String())
	}
}

func TestLeadsHandlers(t *testing.T) {
	ctx := context.Background()
	f := newFixture(ctx, t)
	s := f.addSite(ctx, t, "acme.com")
	logger := slog.New(slog.DiscardHandler)
	call := func(h http.HandlerFunc, method, project, body string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequestWithContext(ctx, method, "/api/v1/projects/"+project+"/analytics/leads", strings.NewReader(body))
		req.SetPathValue("projectId", project)
		rec := httptest.NewRecorder()
		h(rec, req)
		return rec
	}
	save := saveLeadsHandler(logger, f.svc)
	tests := []struct {
		name, project, body string
		wantStatus          int
		wantCode            string
	}{
		{"not JSON", s.project, "names=a", http.StatusBadRequest, "invalid_leads"},
		{"names missing", s.project, `{}`, http.StatusBadRequest, "invalid_leads"},
		{"blank name", s.project, `{"names":["  "]}`, http.StatusBadRequest, "invalid_leads"},
		{"body over the limit", s.project, `{"names":["` + strings.Repeat("a", maxLeadsBody) + `"]}`, http.StatusRequestEntityTooLarge, "payload_too_large"},
		{"no tracking site", "no-such-project", `{"names":["Sign up"]}`, http.StatusConflict, "no_site"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := call(save, http.MethodPost, tt.project, tt.body)
			var got struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			_ = json.Unmarshal(rec.Body.Bytes(), &got)
			if rec.Code != tt.wantStatus || got.Error.Code != tt.wantCode {
				t.Fatalf("status, code = %d, %q; want %d, %q (body %s)", rec.Code, got.Error.Code, tt.wantStatus, tt.wantCode, rec.Body)
			}
		})
	}

	if rec := call(save, http.MethodPost, s.project, `{"names":["Sign up","Book a call"]}`); rec.Code != http.StatusOK {
		t.Fatalf("save status = %d (body %s)", rec.Code, rec.Body)
	}
	rec := call(leadsHandler(logger, f.svc), http.MethodGet, s.project, "")
	var got leadsBody
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || rec.Code != http.StatusOK {
		t.Fatalf("get = %d %s, %v", rec.Code, rec.Body, err)
	}
	if strings.Join(got.Names, "|") != "Sign up|Book a call" {
		t.Errorf("names = %v, want the saved order", got.Names)
	}
}
