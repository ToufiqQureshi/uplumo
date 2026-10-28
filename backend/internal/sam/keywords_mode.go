package sam

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/toufiqqureshi/uplumo/backend/internal/keywords"
	"github.com/toufiqqureshi/uplumo/backend/internal/mcp"
	"github.com/toufiqqureshi/uplumo/backend/internal/platform/openrouter"
)

// Keywords mode is the agent behind the Keywords page. One reply may buy at
// most these many DataForSEO purchases; the page shows the resulting ceiling
// before the customer sends (see flow.md for the dollar maths).
const (
	maxKeywordSeedsPerTurn   = 3  // research_keywords seeds, each one auto research
	maxSerpQueriesPerTurn    = 2  // get_serp_results queries
	maxSerpDepthPerTurn      = 20 // results per SERP query
	maxMetricCallsPerTurn    = 1  // get_keyword_metrics calls
	maxMetricKeywordsPerCall = 50

	maxPresentedKeywords = 20
)

const keywordsModePrompt = `
This turn is Keywords mode: the customer is on the Keywords page and wants keyword research that tells them what to do.
Work like a senior SEO, in this order:
1. Read what they asked: market, intent, how many keywords, what "high volume" or "easy" means for them. Keep the seed short: research the topic ("hotel revenue management"), not filler words like "best", "top" or "company".
2. Research with research_keywords (at most 3 seeds this reply) and, when ranking strength matters, get_serp_results for the 1–2 best keywords (depth 10).
3. If Search Console is connected, call get_search_console_performance to find queries and pages the site already ranks for; those are quick wins (improve the existing page) before new content.
4. Pick keywords: one per coreKeyword group (variants share one Google volume), drop jobs, courses, salary and other companies' brand names unless asked, and respect the number they asked for.
5. Call present_keywords exactly once with your picks, the quick wins and the content plan. Only present keywords that research_keywords returned in this reply; their numbers are filled in from that data, so never type metrics yourself.
6. Then reply in 3–5 short sentences: the honest takeaway (a B2B niche has hundreds of searches, not thousands), and the one thing to do first.
Rankings are never guaranteed: say "possible", never "will rank". Paid calls over the per-reply limit are refused; say what you skipped instead of retrying. Write the reply in English.`

// keywordsModeTool lists what Keywords mode may run: keyword data, the free
// Google and site-reading tools that find quick wins, and saving.
func keywordsModeTool(name string) bool {
	switch name {
	case "research_keywords", "get_keyword_metrics", "get_serp_results", "search_serp_locations",
		"list_saved_keywords", "save_keywords", "get_search_console_performance",
		"get_product_info", "map_links", "read_pages", "present_keywords":
		return true
	default:
		return false
	}
}

// keywordTurn is one Keywords-mode reply's purchase counters and the research
// rows it bought, which present_keywords reads so numbers come from data.
type keywordTurn struct {
	seeds, serpQueries, metricCalls int
	rows                            map[string]map[string]any
}

func (ts *toolset) restrictToKeywords() {
	ts.keywords = &keywordTurn{rows: map[string]map[string]any{}}
	present := ts.presentKeywordsTool()
	ts.local[present.def.Name] = present
	ts.defs = append(ts.defs, present.def)
	ts.defs = slices.DeleteFunc(ts.defs, func(def openrouter.Tool) bool { return !keywordsModeTool(def.Name) })
}

// reserveKeywordPurchase counts a paid call against the reply's limits before
// it runs. A refused call is never sent, so it costs nothing; a counted call
// stays counted even if it fails, because a failed provider task can still be
// billed and must not be retried in a loop.
func (kt *keywordTurn) reserve(name string, arguments json.RawMessage) error {
	if name != "research_keywords" && name != "get_serp_results" && name != "get_keyword_metrics" {
		return nil
	}
	var args struct {
		Seeds    []json.RawMessage `json:"seeds"`
		Queries  []json.RawMessage `json:"queries"`
		Depth    int               `json:"depth"`
		Keywords []string          `json:"keywords"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return fmt.Errorf("invalid tool arguments: %w", err)
	}
	switch name {
	case "research_keywords":
		if kt.seeds+len(args.Seeds) > maxKeywordSeedsPerTurn {
			return fmt.Errorf("this reply may research at most %d seeds and %d are used; present what you have", maxKeywordSeedsPerTurn, kt.seeds)
		}
		kt.seeds += len(args.Seeds)
	case "get_serp_results":
		if args.Depth > maxSerpDepthPerTurn {
			return fmt.Errorf("depth may be at most %d in Keywords mode", maxSerpDepthPerTurn)
		}
		if kt.serpQueries+len(args.Queries) > maxSerpQueriesPerTurn {
			return fmt.Errorf("this reply may check at most %d Google results pages and %d are used", maxSerpQueriesPerTurn, kt.serpQueries)
		}
		kt.serpQueries += len(args.Queries)
	case "get_keyword_metrics":
		if len(args.Keywords) > maxMetricKeywordsPerCall {
			return fmt.Errorf("ask for at most %d keywords", maxMetricKeywordsPerCall)
		}
		if kt.metricCalls >= maxMetricCallsPerTurn {
			return fmt.Errorf("this reply may look up keyword metrics only %d time", maxMetricCallsPerTurn)
		}
		kt.metricCalls++
	}
	return nil
}

// record keeps the full rows a research call bought, before the model's copy
// is trimmed to fit its context.
func (kt *keywordTurn) record(result mcp.ToolResult) {
	raw, err := json.Marshal(result.Structured)
	if err != nil {
		return
	}
	var data struct {
		Results []struct {
			Rows []map[string]any `json:"rows"`
		} `json:"results"`
	}
	if json.Unmarshal(raw, &data) != nil {
		return
	}
	for _, result := range data.Results {
		for _, row := range result.Rows {
			if keyword, ok := row["keyword"].(string); ok {
				kt.rows[keywords.NormalizeKeyword(keyword)] = row
			}
		}
	}
}

type presentArgs struct {
	Understood []struct {
		Label string `json:"label"`
		Value string `json:"value"`
	} `json:"understood"`
	Summary  string `json:"summary"`
	Keywords []struct {
		Keyword string `json:"keyword"`
		Chance  string `json:"chance"`
		Note    string `json:"note"`
	} `json:"keywords"`
	QuickWins []struct {
		Page     string   `json:"page"`
		Keyword  string   `json:"keyword"`
		Position *float64 `json:"position"`
		Target   string   `json:"target"`
		Actions  []string `json:"actions"`
	} `json:"quickWins"`
	ContentIdeas []struct {
		Type     string   `json:"type"`
		Title    string   `json:"title"`
		Keywords []string `json:"keywords"`
		Outline  []string `json:"outline"`
		Why      string   `json:"why"`
	} `json:"contentIdeas"`
	FollowUps []string `json:"followUps"`
}

const presentKeywordsSchema = `{"type":"object","additionalProperties":false,"required":["understood","summary","keywords"],"properties":{
"understood":{"type":"array","maxItems":6,"description":"How you read the ask, e.g. Market: India, English; Intent: wants to hire","items":{"type":"object","additionalProperties":false,"required":["label","value"],"properties":{"label":{"type":"string","maxLength":30},"value":{"type":"string","maxLength":80}}}},
"summary":{"type":"string","maxLength":600,"description":"The honest takeaway in 1-3 sentences"},
"keywords":{"type":"array","minItems":1,"maxItems":20,"items":{"type":"object","additionalProperties":false,"required":["keyword","chance"],"properties":{"keyword":{"type":"string","maxLength":120,"description":"Exactly as research_keywords returned it"},"chance":{"type":"string","enum":["high","medium","low"],"description":"Chance to reach page one for this site, from difficulty, page-one domain strength and current rank"},"note":{"type":"string","maxLength":160}}}},
"quickWins":{"type":"array","maxItems":3,"description":"Pages that already rank (from Search Console) and what to change","items":{"type":"object","additionalProperties":false,"required":["page","keyword","actions"],"properties":{"page":{"type":"string","maxLength":300},"keyword":{"type":"string","maxLength":120},"position":{"type":["number","null"],"description":"Average position from Search Console"},"target":{"type":"string","maxLength":30,"description":"e.g. Top 5"},"actions":{"type":"array","minItems":1,"maxItems":5,"items":{"type":"string","maxLength":220}}}}},
"contentIdeas":{"type":"array","maxItems":3,"items":{"type":"object","additionalProperties":false,"required":["type","title","keywords","outline"],"properties":{"type":{"type":"string","maxLength":40,"description":"Service page, Guide, Comparison..."},"title":{"type":"string","maxLength":140},"keywords":{"type":"array","maxItems":5,"items":{"type":"string","maxLength":120}},"outline":{"type":"array","minItems":1,"maxItems":6,"items":{"type":"string","maxLength":160}},"why":{"type":"string","maxLength":240}}}},
"followUps":{"type":"array","maxItems":4,"items":{"type":"string","maxLength":80}}}}`

// presentKeywordsTool is free: it only shapes the answer for the Keywords
// page. Metrics are joined from this reply's research rows, so a number on the
// screen is always one DataForSEO returned, never one the model typed.
func (ts *toolset) presentKeywordsTool() localTool {
	return localTool{
		def: openrouter.Tool{
			Name:        "present_keywords",
			Description: "Show your keyword picks, quick wins and content plan on the Keywords page as a table, charts and cards. Call once, after research. Metrics are filled in from this reply's research. Uses no credits.",
			Parameters:  json.RawMessage(presentKeywordsSchema),
		},
		run: func(_ context.Context, raw json.RawMessage) (any, error) {
			var args presentArgs
			if err := decodeArgs(raw, &args); err != nil {
				return nil, err
			}
			if len(args.Keywords) == 0 || len(args.Keywords) > maxPresentedKeywords {
				return nil, fmt.Errorf("present 1 to %d keywords", maxPresentedKeywords)
			}
			rows := make([]map[string]any, 0, len(args.Keywords))
			missing := []string{}
			for _, k := range args.Keywords {
				row := map[string]any{"keyword": strings.TrimSpace(k.Keyword), "chance": k.Chance, "note": k.Note, "metrics": nil}
				if metrics, ok := ts.keywords.rows[keywords.NormalizeKeyword(k.Keyword)]; ok {
					row["metrics"] = metrics
				} else {
					missing = append(missing, k.Keyword)
				}
				rows = append(rows, row)
			}
			out := map[string]any{
				"understood": args.Understood, "summary": args.Summary, "keywords": rows,
				"quickWins": args.QuickWins, "contentIdeas": args.ContentIdeas, "followUps": args.FollowUps,
				"budget": map[string]int{"seedsUsed": ts.keywords.seeds, "seedsMax": maxKeywordSeedsPerTurn, "serpUsed": ts.keywords.serpQueries, "serpMax": maxSerpQueriesPerTurn},
			}
			if len(missing) > 0 {
				out["missingMetrics"] = missing
			}
			return out, nil
		},
	}
}
