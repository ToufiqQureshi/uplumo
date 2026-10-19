# DataForSEO billing and retry boundary

`backend/internal/platform/dataforseo/client.go` is the shared authenticated transport. Feature providers build endpoint-specific request bodies. The client validates paths, requests a spend hold, sends the request, records task-reported costs in `go_dataforseo_usage`, and settles holds. See its colocated README and `backend/internal/platform/spend/README.md` for ceilings and limit semantics.

Paid POSTs use `Client.Do(..., false)` and must not be automatically retried on network or 5xx errors: a timeout does not prove the provider did not bill. Query hooks for Backlinks, Domain and the rank-tracking keyword suggestion explicitly set `retry: false`, `refetchOnWindowFocus: false` and `refetchOnReconnect: false`, because the global query policy retries some errors and refetches stale queries on focus and reconnect. Explicit user Retry remains available. Overview cache misses for concurrent equivalent Backlinks requests on one server share an in-flight operation, keyed by organization, target and scope. A shared in-flight operation (Backlinks overview, AI brand lookup, Keyword Research) runs on a context detached from its first caller, capped at three minutes, so one caller leaving never cancels paid work another caller is still waiting for. Redis holds successful responses for six hours; concurrent requests on different server instances can still duplicate paid work. Other Backlinks tabs and Domain have cache hits but no in-flight coalescing; investigate distributed locking or provider idempotency before promising exactly-once spend.

A domain overview may use one Labs operation; a Backlinks domain overview legitimately calls both summary and history. The initial Backlinks screen also loads a selected table. Assert provider operations separately in tests, rather than assuming one click equals one request. Validate tenant, scope, market, filters and pagination before reaching the provider. Recheck authorization on every HTTP request, even on cache hits. No test should consume live provider credits.

Brand Lookup can issue six provider tasks (eight with competitor comparisons).
Equivalent cold misses within one Go process now share that fan-out under an
organization/project scoped key. Separate instances and Prompt Explorer can
still buy identical work concurrently. Scheduled Rank Tracking task-post uses
a stable run/chunk spend key; if its response is ambiguous, the worker records
an error without live fallback for that chunk. Earlier accepted tasks are
still collected. Explicitly rejected/failed/timed-out known tasks retain their
documented fallback behavior.

Keyword Research also shares equivalent cold misses per server instance; its
key includes organization, project, market, location, mode, limits and options.
Different tenants remain separate. SERP snapshots and Domain Research are
still only cache-backed and can duplicate work on concurrent cold misses.

Reported provider spend and customer credits are distinct. To validate plan economics, compare actual usage exports with `backend/internal/plans/catalog.go`, spend ceilings and billing policy. Do not infer customer margin from the credit face value alone; live plan profitability requires the provider cost CSV, representative usage, currency conversion and time window.

## Provider cost versus customer credits

The transport records every task's DataForSEO `cost`, including provider-billed
failures. Successful task costs settle at **1,280 credits per USD**, rounded up
per operation (`billing.CreditsForUSD`); a $1 successful task should consume
1,280 credits. A multi-call (`DoBudgeted`) hold is also the most the customer
pays: if the provider reports a higher successful cost, the customer is
charged the held amount, Uplumo absorbs the rest, and the overage is logged so
the estimate in `pricing.go` can be fixed.

Provider-billed failed tasks still refund customer credits. Those are real
Uplumo losses; the spend guard logs the unrecovered amount as an error so the
endpoint, status and input validation can be investigated. Do not charge a
customer for a provider failure without an explicit product policy. Ambiguous
timeouts may have cost the provider but cannot be inferred from a missing
response; never automatically retry the paid POST.

To reconcile a paid workspace, compare `go_dataforseo_usage` task costs with
`go_billing_credit_ledger.provider_cost_usd` and negative `amount_credits`
per organization, feature, endpoint and time window. Include pending holds,
refunded tasks, self-hosted/staff workspaces, credit expiry and rounding when
explaining a difference. The user's provider CSV is needed to confirm any
specific $1 discrepancy and Basic/Pro/Agency margins.

## One action, several paid calls

The browser sends one `Idempotency-Key` per user action. Each paid call in that
action holds credits under a key derived from it (`spend.ScopedKey`: method,
path and body), so a backlink overview can buy its summary and its history, and
a 3×3 map grid can buy nine searches. Sending the same action again within a
minute is refused call by call with `409 duplicate_request` and buys nothing.

When a later call of an action fails after an earlier one was billed, the
answer keeps what was paid for instead of failing: Keyword Research answers with
the leg that worked, Backlinks with the summary and no trend, Local SEO with
the points already searched (`summary.stoppedReason` says why it stopped). Such
partial answers are cached for ten minutes only, so a retry costs nothing and
the full report is bought again once the provider has recovered. A credit
refusal, a replay or a customer who left still fails the request.

The HTTP server's write timeout (150s) outlasts two provider calls (60s each),
and a budgeted operation stops starting calls after 120s.
