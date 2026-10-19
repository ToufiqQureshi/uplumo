# Regression testing

Run focused tests for a changed feature, then repository checks. Frontend: `cd frontend && pnpm ci:check && pnpm test:ci`. Backend: set disposable `TEST_DATABASE_URL` and `TEST_REDIS_URL`, then `cd backend && go test -p 1 -race -count=1 ./...`. CI also runs static analysis and vulnerability checks in `.github/workflows/ci.yml`. `frontend/scripts/docs-map.test.ts` verifies that cross-cutting Markdown appears in `docs/INDEX.md`, local index links resolve, and feature folders contain README/flow/decision. It does not check prose accuracy.

Use a fake provider for paid-call tests. Backlinks mounted query regression expects two operations (overview and current rows) on a 503, without automatic repeats. Its Go concurrency regression expects eight equivalent organization requests to share one summary and one history call. Domain mounted query regression expects one failed paid attempt. Existing Go handler and transport suites exercise validation, plan gating, cache isolation, errors and cancellation. For changes to costs, add specific call-count and different-tenant cases, then test failure and ambiguous timeout paths. Do not run live paid APIs or production databases.

Additional Go regressions: `aisearch/service_test.go` checks that simultaneous
Brand Lookups share one six-call fan-out and distinct organizations stay
separate; `keywords/research_test.go` checks equivalent paid research is
coalesced while different organizations are independent;
`ranktracking/queued_test.go` checks an ambiguous task-post error never buys
the same keyword/device pair through live SERP fallback.
`frontend/src/client/features/billing/credits/CreditsCard.test.ts` verifies
UTC midnight billing dates render consistently across host timezones.

`backend/internal/billing/spendguard_test.go` uses a fake DataForSEO response
and real test Postgres to assert a $1 successful budgeted task held at $0.50
charges the customer only the $0.50 hold while recording $1 of provider
spend. The provider guard suite
checks billed failed tasks remain visible in spend telemetry but refund the
customer. Use actual provider usage exports to reconcile production costs;
these deterministic fixtures cannot establish plan profitability.
