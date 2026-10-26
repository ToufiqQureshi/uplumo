# aisearch

AI search lookups for a project: **Brand Lookup** (how often ChatGPT and Google
AI Overviews mention a brand or domain, which pages they cite, how it compares
with competitors) and **Prompt Explorer** (one prompt asked of ChatGPT, Claude,
Gemini and Perplexity, answers side by side). Both lookups store no rows of their own, only
Redis cache; saved prompt tracking (below) keeps its own tables. Successful, user-triggered Brand Lookups also save one
aggregate visibility snapshot per month in Redis for a 13-month trend; history
reads use saved snapshots and do not call DataForSEO.
The Brand Lookup page can also show first-party AI referral totals from Uplumo
Analytics. Those site-wide visits are kept separate from citation counts and
are not matched to specific cited URLs.
Each top query keeps what the mentions call returns with it: the AI model, its
full answer (6,000 characters), the pages it retrieved (20, with description)
and the searches it ran (`TopQuery`, shown under "See the AI answer"), plus
`branded` (the question names the brand; `brand_terms.go`).
Brand Lookup buys, in order: one `target_metrics` task (mentions per platform,
`sourceSites`, `retrievedSites`, `brandsAlongside`); stop there when no
platform has a mention; then per platform with mentions a prompt sample of up
to 100 answers, most searched first; `top_pages` only when the sample cannot
hold every answer (otherwise cited pages are counted from the sample). ChatGPT's
data is US English only, so its sample and comparison are bought only for that
market. Measured 22 Oct 2026 (India): dellaresorts.com 2 tasks, $0.283;
revmerito.com (no mentions) 1 task, $0.101. Old flow: six tasks, about $0.71.
The hold is one AI-mentions ceiling ($0.25) per possible task.
A Prompt Explorer run holds $0.10 per model answer (twice that with web
search, which may retry once); a real ChatGPT answer with web search bills $0.03.

## Flow

```
POST /api/v1/projects/{projectId}/ai-search/brand-lookup
POST /api/v1/projects/{projectId}/ai-search/brand-lookup/history
POST /api/v1/projects/{projectId}/ai-search/prompt-explorer
  httpapi: requireSession -> requireProjectAccess (puts the project's organization in the context)
  handler.go   decode, validate, plan check, map errors to HTTP
  service.go   cache, fan-out to the provider, partial-failure rules
  provider.go  DataForSEO AI optimization endpoints (through platform/dataforseo)
  shape.go     provider rows -> the result the page renders (pure functions)
  prompt.go    answer text, citations, brand highlight (pure functions)
  models.go    which model name to ask for (live catalog, pins, fallbacks)
  target.go    query -> domain or keyword; research scope; URL safety
```

Request and response JSON is byte-compatible with the legacy server functions
(`frontend/src/types/schemas/ai-search.ts` validates it on the client), so the pages did
not change shape.

## Rules that must not break

- **Only the project's organization pays.** Every provider call carries the
  organization from `auth.ProjectOrganizationFromContext`, never the session's
  active organization, and `platform/dataforseo` writes one `go_dataforseo_usage`
  row per billed task. A handler without it fails closed (500).
- **Paid plan only, when billing exists.** `Deps.Plans` (the billing service)
  gates both endpoints with 402 before any provider spend. With no Razorpay
  configured there is no plan to check and the endpoints are open.
- **Never send the provider a request it rejects.** DataForSEO bills failed
  tasks. So: scopes and countries are validated first, `model_name` is checked
  against the provider's free catalog, `web_search_country_iso_code` is dropped
  when web search is off, `force_web_search` goes to Claude only, and domains
  must end in a real public suffix.
- Brand Lookup reserves its largest possible fan-out before calling a
  provider: 1 + 2 per bought platform, plus 1 per platform for competitor
  comparisons (3-7 tasks). Prompt Explorer
  reserves every selected model and its bounded web-search retry up front.
  Successful paid results settle by the tasks' reported costs; failed tasks
  return their credits, and cache reads settle at zero.
- **No automatic retries of paid calls** (`retrySafe=false`). The one deliberate
  retry: a model that was allowed to search but answered from memory is asked
  once more; the retry replaces the answer only if it actually searched.
- **Partial failure is data.** A failed platform shows as `status: "error"` in
  its row; a failed sub-call falls back to empty data. Only
  `dataforseo.ErrBillingIssue` (and a canceled request) fails the whole call,
  and it stops at the first failing call.
- **Never cache a degraded result.** Brand lookups are cached 24 h only when
  every call succeeded and there is data; prompt answers 7 days, successes only.
- Concurrent identical Brand Lookup cache misses share one fan-out within a
  server process. The key includes
  organization and project; each caller keeps its own display query. This
  does not coordinate multiple server instances. Prompt Explorer similarly
  shares identical model-set requests before reserving paid answers; caller
  model order and highlight brand are applied separately to the shared result.
- Gemini citation links prefer the safe original `direct_url` over the Vertex
  redirect; an absent/unsafe original falls back to a safe `url` without
  fetching either. Gemini answer cache keys use `prompt-response-v2` so old
  redirect-only results expire without being reused. Other model caches remain.
- **Untrusted URLs.** Citations come from LLM answers. Only `http(s)` URLs without
  credentials are returned (`safeHTTPURL`), because the page renders them as links.
- **ChatGPT is US/English only.** It is always queried with 2840/`en` and left out
  of totals, trends and share of voice for any other locale.
- **URL scopes are post-filtered.** The provider cannot target a URL, so
  `exact_url` and `subfolder` filter page rows afterwards; totals, trends and
  share of voice stay domain-wide and the result says so
  (`aggregatesAreDomainLevel`).

## Cache

Redis, keys `ai-search:<kind>:<organizationId>:<digest>`. The digest covers the
organization, project, target, competitors (sorted, lowercased), locale, scope
and path (path only under URL scopes); for prompts the model, resolved model
name, whitespace-collapsed prompt (case kept), web search and country. The
highlight brand is deliberately **not** in the key: it is re-applied on every
read. The organization is in the clear so everything it cached can be deleted by
prefix. **Privacy erasure:** cached prompts can hold personal data, and
`cmd/erase-user` works on Postgres only, so it does not clear this Redis
namespace. Entries expire within 7 days. To erase sooner, `SCAN`/`UNLINK`
`ai-search:*:<organizationId>:*`.

## Configuration

| Variable             | Meaning                                                                                                        |
| -------------------- | -------------------------------------------------------------------------------------------------------------- |
| `DATAFORSEO_API_KEY` | Base64 `login:password` of the DataForSEO account. Unset: both endpoints answer 503 and the server still runs. |

Provider spend is in `go_dataforseo_usage` (migration 00006), per organization.

## Gotchas

- Limits are UTF-16 units, like the web app's schemas, so `truncate` and
  `utf16Len` count units and never split a character.
- `mentionsBrand` reimplements the legacy regex without lookaround (Go's RE2 has
  none): word boundaries are required only on sides where the brand ends in a
  word character, otherwise the side must not repeat the brand's edge character
  (`C++` does not match `C+++`).
- Model names: the catalog is cached for an hour; an outage falls back to a
  snapshot list and is not cached. ChatGPT is pinned to `gpt-5.6-luna` while the
  catalog lists it.
- A country that a model does not support is reported per model
  (`UNSUPPORTED_COUNTRY`) without calling it.

## Tests

`go test ./internal/aisearch/` needs `TEST_REDIS_URL` (and uses an in-process
fake DataForSEO). The end-to-end test in `internal/httpapi` also needs
`TEST_DATABASE_URL` and proves authorization and billing attribution.

## Saved prompt tracking

`Tracking` (`tracking*.go`, API contract `backend/api/ai-tracking.yaml`) adds a
project-scoped tracker beside Brand Lookup and Prompt Explorer. It saves up to
30 distinct questions, topics, pause flags, five competitor identities, country,
language and engines. Removing/editing a question changes future checks only.
`go_ai_trackers`, `go_ai_tracking_runs` and `go_ai_tracking_answers` retain immutable
configuration and evidence for 13 months. Organization/project deletion cascades.

- `GET .../ai-search/tracking` and `GET .../tracking/runs/{runId}` read saved data
  without provider calls or credits, including after a downgrade.
- `POST .../tracking/estimate` is free. `save` and `run` require the AI Visibility
  entitlement. Enabling a schedule approves `maxCreditsPerRun`; manual checks
  approve `maxCredits`. Both are checked against the actual saved scope.
- ChatGPT, Gemini and Google AI Overviews use DataForSEO's standard consumer-site
  collection queue even for Run now. Answers arrive asynchronously. Claude and
  Perplexity use their model APIs and can differ from consumer apps. Their
  country is passed to web search; their language follows the question itself.
- Each collected answer keeps what the scraper returns with it: the answer's
  markdown (rendered, with AI-vendor images shown as links, not loaded), its
  cited sources with publisher name and date, and the businesses the AI named
  (`brand_entities`, in order, up to 20). Those are stored in the `brands`
  column after the tracked brands, flagged `recommended`, so no migration was
  needed; `splitStoredBrands` separates them on read. Tracked brands stay
  first because rate queries read `brands->0`.
- A full run reserves its ceiling before any paid call. Consumer collection is
  budgeted at $0.0012 per answer; model API collection holds $0.10
  per answer. Actual successful usage settles after submission. No automatic
  retry purchases a fresh answer. Successful queued submissions are billed even
  if later collection fails; polling is free. A finished task that fails or
  cannot be read is marked failed (`provider_failed`, `unreadable_result`) at
  once and is not polled again; only queued tasks keep the run waiting.
- Run creation, immutable questions and the `go_jobs` outbox insert are one
  transaction. A tracker row lock and partial unique index allow only one active
  check. The worker checkpoints task IDs, recovers free collection, and marks
  uncertain paid submissions interrupted instead of posting them again.
- A five-minute ticker claims due schedules transactionally and advances the
  previous UTC anchor. Changing frequency schedules a new baseline. Pausing
  cancels queued scheduled checks; an already-running check may finish.
- Failed/empty answers never enter mention/citation rates. Mentions use answer
  prose; a source URL alone is only a citation. Generic names can still collide.
  The UI compares complete runs only with identical scope and at least ten
  answered observations; a 20-point decline is an in-app investigation warning.
- Billing forecasts these schedules alongside rank checks. Forecasts do not
  reserve credits and cannot promise that shared credits will remain available.

AI tracking jobs allow two hours: the permitted 60 sequential live model calls
at the transport's 60-second deadline, DataForSEO's standard queue of up to
45 minutes, and 15 minutes for submission/collection. This is a bounded job
window, not a promised completion time. Accepted task reads remain resumable
after process shutdown without buying another task.

Cancellation during cache/project preparation is rechecked before detaching
paid work. Active callers retain cache-outage fallback; existing shared work
can finish for other waiters. See the cancellation decision and regressions.

AI tracking workers use the shared `jobs.Worker.RunSupervised` restart loop. Temporary storage errors resume after five seconds; existing accepted-task checkpoints and retry limits still prevent replaying uncertain purchases.
