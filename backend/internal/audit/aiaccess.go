package audit

import (
	"context"
	"strings"
	"time"
)

// aiCrawler is a user agent an AI assistant uses to read a site. Blocking one
// in robots.txt keeps that assistant from quoting or linking the site.
type aiCrawler struct {
	token string // robots.txt user-agent token, lower case
	name  string // what the customer sees
}

// aiCrawlers that decide whether an assistant can read and cite a page.
// Google AI Overviews use Googlebot, which the normal audit already covers.
var aiCrawlers = []aiCrawler{
	{"oai-searchbot", "OAI-SearchBot (ChatGPT search)"},
	{"chatgpt-user", "ChatGPT-User (ChatGPT browsing)"},
	{"gptbot", "GPTBot (OpenAI)"},
	{"perplexitybot", "PerplexityBot (Perplexity)"},
	{"claude-searchbot", "Claude-SearchBot (Claude search)"},
	{"claudebot", "ClaudeBot (Anthropic)"},
	{"google-extended", "Google-Extended (Gemini)"},
}

// maxLLMsTxtBytes is enough to tell a text file from an HTML error page.
const maxLLMsTxtBytes = 64 * 1024

// blockedAICrawlers lists the AI crawlers robots.txt keeps off the home page.
// A missing robots.txt (nil) blocks nothing.
func blockedAICrawlers(origin string, robotsText *string) []string {
	if robotsText == nil {
		return nil
	}
	groups, _ := parseRobotsGroups(*robotsText, origin)
	var blocked []string
	for _, crawler := range aiCrawlers {
		if !robotsAllowed(origin+"/", selectRobotsRules(groups, crawler.token)) {
			blocked = append(blocked, crawler.name)
		}
	}
	return blocked
}

// hasLLMsTxt reports whether origin serves a real /llms.txt. Many sites answer
// every path with an HTML page and status 200, so an HTML body does not count.
func (d discoverer) hasLLMsTxt(ctx context.Context, origin string) bool {
	fetched, err := d.fetchFollowingRedirects(ctx, origin+"/llms.txt", 10*time.Second)
	if err != nil || fetched == nil {
		return false
	}
	defer func() { _ = fetched.resp.Body.Close() }()
	if fetched.resp.StatusCode < 200 || fetched.resp.StatusCode >= 300 {
		return false
	}
	if strings.Contains(strings.ToLower(fetched.resp.Header.Get("Content-Type")), "html") {
		return false
	}
	body, err := ReadTextUpTo(fetched.resp, maxLLMsTxtBytes)
	if err != nil {
		return false
	}
	body = strings.TrimSpace(body)
	return body != "" && !strings.HasPrefix(body, "<")
}

// aiAccessIssues reports site-wide problems that stop AI assistants from
// reading the site. They cost nothing: robots.txt is already fetched and
// llms.txt is one extra request to the customer's own site.
func aiAccessIssues(origin string, robotsText *string, llmsTxt bool) []DetectedIssue {
	var issues []DetectedIssue
	if blocked := blockedAICrawlers(origin, robotsText); len(blocked) > 0 {
		issues = append(issues, DetectedIssue{
			IssueType: IssueAICrawlersBlocked,
			PageURL:   origin + "/robots.txt",
			Details:   map[string]any{"crawlers": blocked},
		})
	}
	if !llmsTxt {
		issues = append(issues, DetectedIssue{IssueType: IssueMissingLLMsTxt, PageURL: origin + "/llms.txt"})
	}
	return issues
}
