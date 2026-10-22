# AI Visibility and Prompt Explorer: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API/trigger:** POST /api/v1/projects/{projectId}/ai-search/brand-lookup; POST /api/v1/projects/{projectId}/ai-search/prompt-explorer
- **Source:** `handler.go`, `service.go`, `provider.go`, `shape.go`, `prompt.go`

## Request or job flow

Session/project middleware identifies the owning organization; handler validates and checks plan; service applies cache and bounded provider fan-out; provider uses DataForSEO; pure shaping code produces the React contract.

## Dependencies and cost boundary

Redis cache; DataForSEO AI endpoints; plans/billing.

## Monthly Brand Lookup history

`POST /api/v1/projects/{projectId}/ai-search/brand-lookup/history` uses the same session, project and paid-plan guards as Brand Lookup. It validates the same lookup inputs, derives the organization/project/target/competitor/market/scope Redis key and returns stored aggregates only; it never calls DataForSEO. `BrandLookup` saves a single `YYYY-MM` snapshot only after every platform/cross-brand call completes and the result contains data. Redis reads/writes are best effort: read failure logs and returns empty history without failing the current lookup. The hash keeps at most 13 months, removes older fields and expires after 400 days.

## Failure, security and data invariants

- Plan and request validation must happen before provider spend.
- Keep all billed calls attributed to the project organization; never blindly retry paid POSTs.
- Cache only successful complete results; preserve per-platform partial failures and safe citation URLs.
- On one server, an organization-scoped in-flight key coalesces concurrent
  identical Brand Lookups before their paid tasks (1 to 7) are reserved.
- Identical Prompt Explorer requests share a bounded three-minute flight keyed
  by organization, project, normalized prompt, sorted models, web search and
  country. The winner rechecks each model cache before reserving up to one
  answer per missing model (two when web search permits the existing bounded
  retry). Each caller retains its model order and highlight brand. Separate
  processes are not coalesced.
- A cached prompt may contain user-entered text; retain expiry and privacy-erasure limitations in the feature README.

## Saved prompt tracking

`tracking_handler.go` reads saved state/evidence or validates explicit save/estimate/run requests. `tracking_store.go` snapshots approved configuration and writes an `ai_visibility` job in the same transaction. Two supervised workers in `tracking_worker.go` reserve one workspace budget, checkpoint accepted tasks, and poll free task reads; retries never replay uncertain paid submissions. The five-minute scheduler claims due projects, reconciles exhausted jobs and expires 13-month evidence.

Preparation boundary: `BrandLookup` rechecks cancellation after `cacheGet` and before `flights.DoChan`. A cancelled request returns its context error
without starting a new provider operation. Existing shared work is unaffected.

Citation shaping: `responseAnnotation.DirectURL` preserves Gemini’s original
publisher. `extractCitations` validates it before falling back to the redirect,
then deduplicates and computes publisher/brand matching from the selected URL.
It never follows or fetches citation URLs. Only Gemini’s cache namespace advances
to `prompt-response-v2`; the next explicit uncached Gemini run buys a fresh answer.
