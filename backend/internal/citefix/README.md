# Cite Fix

Tells a customer why AI assistants (ChatGPT, Perplexity, Claude, Gemini) do not
quote their site, scores it 0-100, and lists what to fix first. Read this before
changing the score, the checks or the report route.

It scores the project's **latest completed site audit**. The crawler already
stores each page's citation signals (`citation_signals_json`, see
`backend/internal/audit/README.md`), so a report makes **no provider call and
costs nothing**. Every Site Audit run refreshes Cite Fix too.

Internal guide for developers and coding agents. Read [flow.md](flow.md) for
the request path and [decision.md](decision.md) for why the score is built this
way.

## Source map

- `checks.go`: the six categories (weights sum to 100), the 17 checks with the
  copy the customer reads, and the thresholds.
- `citefix.go`: `Build`, the pure scoring of one scan; report types.
- `store.go`: `Store.Report`, the Postgres reads, scoped to the workspace.
- `handler.go`: `Mount` and the route.

## API

`POST /api/v1/projects/{projectId}/cite-fix/report`, body `{}` (anything else
is 400). Session and project membership required; any member may call it.

| `state`        | Meaning                                                                                    |
| -------------- | ------------------------------------------------------------------------------------------ |
| `ready`        | Latest completed audit scored: `score`, `categories`, `fixes`, `pages`.                    |
| `no_scan`      | No completed audit. `running` says whether one is under way.                               |
| `needs_rescan` | Latest completed audit was crawled before signals were stored (migration 47). Never zeros. |

`running` is the newest audit with status `running`, or null, in every state.

## The score

- 6 categories: AI can read the site 20, AI knows who you are 15, author and
  freshness 15, direct answers 20, facts and sources 15, content depth 15.
- A **site check** (robots.txt search bots, llms.txt, business schema, one
  phone number) passes or fails once. A **page check** scores the share of
  content pages that pass. A category is the weighted mean of its checks; a
  check with nothing to judge (no dated page for "fresh") is left out, and a
  category with no checks left is left out of the site score.
- Content page: status 2xx, indexable, at least one word, not an app shell. App
  shells fail only "pages-readable".
- A fix's `gain` is the points the site score rises if that check passed
  everywhere; `impact` is high from 5 points, medium from 2, else low. Fixes
  sort by gain.
- Thresholds: lead paragraph 10-80 words, thin under 300 words, fresh within
  365 days of the scan day. Only search bots count as blocked
  (`OAI-SearchBot`, `ChatGPT-User`, `PerplexityBot`, `Claude-SearchBot`);
  blocking training bots is the owner's choice and costs no points.

## Rules that must never break

- No paid call. A change that adds one is a new PR with cost approval
  (`CLAUDE.md` section 3).
- Every read joins `projects` on the caller's workspace: another workspace's
  project reads as `no_scan`, never as data.
- A pre-signal audit is `needs_rescan`, never a low score.
- The copy says "easier to quote", never "you will be cited".

## Limits

Reads at most `audit.PaidMaxAuditPages` (10,000) pages; lists 20 pages per fix
and the 50 weakest pages. Store reads time out after 10 seconds.

## Tests

`citefix_test.go` (scoring, worked by hand), `handler_test.go` (route),
`store_test.go` (Postgres; needs `TEST_DATABASE_URL`, runs in CI).
