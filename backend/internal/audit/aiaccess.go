package audit

import (
	"context"
	"strings"
	"time"
)

// aiCrawler is a user agent an AI assistant uses to read a site. Blocking one
// in robots.txt keeps that assistant from quoting or linking the site.
type aiCrawler struct {
	token  string // robots.txt user-agent token, lower case
	name   string // what the customer sees
	search bool   // this bot can retrieve pages for an answer
}

// aiCrawlers that decide whether an assistant can read and cite a page.
// Google AI Overviews use Googlebot, which the normal audit already covers.
var aiCrawlers = []aiCrawler{
	{"oai-searchbot", "OAI-SearchBot (ChatGPT search)", true},
	{"chatgpt-user", "ChatGPT-User (ChatGPT browsing)", true},
	{"perplexitybot", "PerplexityBot (Perplexity)", true},
	{"claude-searchbot", "Claude-SearchBot (Claude search)", true},
	{"bingbot", "Bingbot (Copilot)", true},
	{"duckassistbot", "DuckAssistBot (DuckDuckGo)", true},
	{"amazonbot", "Amazonbot (Amazon)", true},
	{"youbot", "YouBot (You.com)", true},
	{"xai-bot", "xAI-Bot (xAI)", true},
	{"applebot", "Applebot (Apple)", true},
	{"gptbot", "GPTBot (OpenAI training)", false},
	{"claudebot", "ClaudeBot (Anthropic training)", false},
	{"google-extended", "Google-Extended (Google training)", false},
	{"ccbot", "CCBot (Common Crawl training)", false},
	{"applebot-extended", "Applebot-Extended (Apple training)", false},
}

// IsAISearchBot identifies a name emitted by blockedAICrawlers. Training-only
// bots remain visible in the audit but never lower the citation score.
func IsAISearchBot(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, crawler := range aiCrawlers {
		if crawler.search && (name == crawler.token || strings.HasPrefix(name, crawler.token+" ")) {
			return true
		}
	}
	return false
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
	var search, training []string
	for _, crawler := range blockedAICrawlers(origin, robotsText) {
		if IsAISearchBot(crawler) {
			search = append(search, crawler)
		} else {
			training = append(training, crawler)
		}
	}
	if len(search) > 0 {
		issues = append(issues, DetectedIssue{
			IssueType: IssueAICrawlersBlocked,
			PageURL:   origin + "/robots.txt",
			Details:   map[string]any{"crawlers": search},
		})
	}
	if len(training) > 0 {
		issues = append(issues, DetectedIssue{
			IssueType: IssueAITrainingBotsBlocked,
			PageURL:   origin + "/robots.txt",
			Details:   map[string]any{"crawlers": training},
		})
	}
	if !llmsTxt {
		issues = append(issues, DetectedIssue{IssueType: IssueMissingLLMsTxt, PageURL: origin + "/llms.txt"})
	}
	return issues
}
