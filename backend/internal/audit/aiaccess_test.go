package audit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"
)

func TestBlockedAICrawlers(t *testing.T) {
	const origin = "https://shop.example"
	text := func(s string) *string { return &s }
	tests := []struct {
		name   string
		robots *string
		want   []string
	}{
		{name: "no robots.txt blocks nothing", robots: nil},
		{name: "empty robots.txt", robots: text("")},
		{name: "only a private folder", robots: text("User-agent: *\nDisallow: /admin\n")},
		{
			name:   "everyone blocked",
			robots: text("User-agent: *\nDisallow: /\n"),
			want:   []string{"OAI-SearchBot (ChatGPT search)", "ChatGPT-User (ChatGPT browsing)", "GPTBot (OpenAI)", "PerplexityBot (Perplexity)", "Claude-SearchBot (Claude search)", "ClaudeBot (Anthropic)", "Google-Extended (Gemini)"},
		},
		{
			name:   "training bots only, search bots still allowed",
			robots: text("User-agent: GPTBot\nUser-agent: ClaudeBot\nDisallow: /\n\nUser-agent: *\nAllow: /\n"),
			want:   []string{"GPTBot (OpenAI)", "ClaudeBot (Anthropic)"},
		},
		{
			name:   "a specific allow beats a blanket block",
			robots: text("User-agent: *\nDisallow: /\n\nUser-agent: OAI-SearchBot\nAllow: /\n"),
			want:   []string{"ChatGPT-User (ChatGPT browsing)", "GPTBot (OpenAI)", "PerplexityBot (Perplexity)", "Claude-SearchBot (Claude search)", "ClaudeBot (Anthropic)", "Google-Extended (Gemini)"},
		},
		{
			name:   "Claude-SearchBot is not caught by a ClaudeBot rule",
			robots: text("User-agent: claudebot\nDisallow: /\n"),
			want:   []string{"ClaudeBot (Anthropic)"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := blockedAICrawlers(origin, tt.robots); !slices.Equal(got, tt.want) {
				t.Fatalf("blocked = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestHasLLMsTxt(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    bool
	}{
		{"plain text file", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("# Shop\n> Hand-made shoes in Pune\n"))
		}, true},
		{"missing", http.NotFound, false},
		{"soft 404 served as HTML", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte("<!doctype html><title>Home</title>"))
		}, false},
		{"text body sent as an HTML page", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("Page not found"))
		}, false},
		{"HTML with a text content type", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("  <html><body>Not found</body></html>"))
		}, false},
		{"empty file", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
		}, false},
		{"server error", func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "boom", http.StatusInternalServerError)
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/llms.txt" {
					t.Errorf("fetched %s, want only /llms.txt", r.URL.Path)
				}
				tt.handler(w, r)
			}))
			defer server.Close()
			guard := &Guard{Resolver: fakeResolver{addrs: map[string][]string{"audit.test": {"93.184.216.34"}}}}
			d := discoverer{guard: guard, client: dialAllTo(server), now: time.Now}
			if got := d.hasLLMsTxt(context.Background(), "http://audit.test"); got != tt.want {
				t.Fatalf("hasLLMsTxt = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAIAccessIssues(t *testing.T) {
	blockAll := "User-agent: *\nDisallow: /\n"
	issues := aiAccessIssues("https://shop.example", &blockAll, false)
	if len(issues) != 2 || issues[0].IssueType != IssueAICrawlersBlocked || issues[1].IssueType != IssueMissingLLMsTxt {
		t.Fatalf("issues = %+v", issues)
	}
	if issues[0].PageID != nil || issues[0].PageURL != "https://shop.example/robots.txt" {
		t.Fatalf("site-wide issue tied to a page: %+v", issues[0])
	}
	if got := aiAccessIssues("https://shop.example", nil, true); len(got) != 0 {
		t.Fatalf("an open site with llms.txt has issues: %+v", got)
	}
	for _, issueType := range []IssueType{IssueAICrawlersBlocked, IssueMissingLLMsTxt} {
		descriptor, ok := DescribeIssueType(string(issueType))
		if !ok || descriptor.HowToFix == "" {
			t.Fatalf("%s has no title or fix: %+v", issueType, descriptor)
		}
	}
}
