# Local Visibility

A project's Google Business Profile compared with the businesses that beat it
on Google Maps, from DataForSEO's Maps SERP and Business Data APIs. Pro and
Business plans (`plans.LocalVisibility`). Five tabs: profile vs top 3, map
rank grid, reviews (themes and reply drafts), weekly competitor watch, AI
answers (a hand-off to AI Visibility, no request of its own).

## Code map

| File | Holds |
| --- | --- |
| `provider.go` | DataForSEO task bodies, `post` / `collect`, the live business search, response parsers |
| `analysis.go` | Result types and the rules: fixes vs the top 3, grid layout and summary, review stats, watch diffs |
| `service.go` | Start a run (cache, one running run per kind, post), the collect job (`Process`), results, weekly `Tick`, `PriceList` |
| `llm.go` | Review themes and reply drafts (OpenRouter, metered, fixed ceilings) |
| `overview.go` | Page load (`Overview`), business search (`Find`), `SaveBusiness`, `SetWatch` |
| `store.go` | SQL for the five tables |
| `handler.go` | Routes, plan gate, error mapping |
| `frontend/src/client/features/local-visibility/` | The page, one file per tab, `localVisibilityApi.ts` |

Wiring: `cmd/server/main.go` `buildLocalVisibility` (queue worker x2, 15-minute
ticker); routes in `internal/httpapi`. Pricing of Maps and Business Data tasks:
`internal/platform/dataforseo/pricing.go`.

## Routes

All `POST /api/v1/projects/{projectId}/local/...`, plan-gated:
`overview`, `find`, `business`, `runs/start`, `runs/get`, `watch`, `reply`.

## Tables (migration 00057)

`go_local_businesses` (one per project), `go_local_runs` (each paid check and
its result; `find` searches too), `go_local_tasks` (every posted DataForSEO
task), `go_local_watch`, `go_local_reply_drafts`.
