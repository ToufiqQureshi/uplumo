# Cite Fix

Shows why AI assistants (ChatGPT, Claude, Gemini, Perplexity, Google AI
Overviews) do or do not name the customer's site, and what to change. It
reads the latest site scan and the saved AI Visibility checks; it never runs
an AI Visibility check itself.

## Code map

| Part | File |
| --- | --- |
| Routes | `handler.go` |
| Report assembly (scan state, score, fixes, topics, evidence) | `store.go`, `citefix.go`, `checks.go`, `topics.go`, `seo_issues.go` |
| AI citation proof from saved answers and history | `proof.go` (+ `aisearch.LoadVisibilityMetrics`) |
| Rival pages: free fetch, signals, list-page detection, 7-day cache | `rivals.go` |
| Saved Brand Lookup mentions | `mentions.go` |
| Google AI Overview check (paid, DataForSEO) | `overview.go` |
| Page-fix AI agent (paid, OpenRouter) | `agent.go`, `agent_prompts.go`, `agent_store.go` |
| Screen | `frontend/src/client/features/cite-fix/` |

## Routes

| Route | Cost |
| --- | --- |
| `POST /api/v1/projects/{id}/cite-fix/report` | Free |
| `POST /api/v1/projects/{id}/cite-fix/google-ai-overview/refresh` | Paid, shown before the click |
| `POST /api/v1/projects/{id}/cite-fix/google-ai-overview/results` | Free (collects a paid task's result) |
| `POST /api/v1/projects/{id}/cite-fix/agent/run` | Paid, at most the shown credits per page |
| `POST /api/v1/projects/{id}/cite-fix/agent/results` | Free |
| `POST /api/v1/projects/{id}/cite-fix/pitch` | Paid, at most 25 credits, shown before the click |

## Tables

`go_citefix_rival_pages` (rival page signals and outbound link hosts, per
project), `go_citefix_agent_runs`, `go_citefix_ai_mentions`,
`go_citefix_ai_overviews`, `go_citefix_pitches`; it reads `go_audits`, `go_ai_tracking_runs` and
`go_ai_tracking_answers`.
