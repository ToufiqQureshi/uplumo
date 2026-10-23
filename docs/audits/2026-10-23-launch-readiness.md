# Launch readiness review — 23 October 2026

Scope: Uplumo launch changes in PR #196. Evidence comes from code, disposable PostgreSQL/Redis, browser tests and bounded production reads. The live security remediation below is the only production database change made during the audit.

## Confirmed business defects addressed

| Boundary | Defect and correction |
| --- | --- |
| Direct database access | 83 public tables lacked RLS; client roles had auth/billing read, write and TRUNCATE grants. Apply server-only RLS and revoke grants; migration 42 records the same idempotent hardening. |
| Credit settlement | A failed ledger transaction could later refund already-paid work. Persist known outcomes, recover once and commit credits/spend together; unknown outcomes stay reserved for evidence-based resolution. Migration 43 also persists zero-estimate postpaid outcomes before any charging failure; PR review regression is covered. |
| Provider transport | Missing/invalid responses and terminal 5xx were treated as zero-cost releases. Keep uncertain dispatched purchases pending and stop remaining budget legs; explicit rejections still release. |
| Payments | Checkout did not compare provider terms with catalog. Verify plan identity, monthly frequency, INR and exact amount before offering a payable subscription. Startup logs read-only verification results. |
| Webhooks | Different signed bodies for the same payment could mint another grant. Retain body fingerprints and add signed payment-ID identity when supplied. Legacy identities cannot be reconstructed retrospectively. |
| DataForSEO repeat spend | Briefly reuse successful empty Domain/suggestion/Keyword results; return/cache a billed Backlinks summary when history is refused. Retain browser replay keys after malformed success and prevent failed-query remount retries. |
| AI expense | Preserve actual provider cost on unusable writing output and capped customer charges. Existing customer refund/15-credit ceiling remains. |
| Background work | Audit/rank workers stopped permanently on database failure. Share supervised recovery with AI tracking, retaining paid-task checkpoints/attempt limits. |
| Tenant and MCP safety | Bind crawler manager roles to the owning workspace; contain MCP panic/error/nil results; keep raw internal errors out of client responses. Persist the successful external MCP onboarding milestone. |
| Customer promises | Disclose that schedules/research/AI share credits; a five-run month can exhaust the allowance. No price, grant or keyword-cap change. |

## Monthly contribution scenarios

Assumptions: all newly granted credits consumed, ₹100/USD provider-cost stress rate and 3% payment fee allowance. These are scenario inputs, not a fetched exchange rate or a Razorpay contract quote. Figures exclude hosting, database, email, support, staff use, provider taxes/FX fees, refunds, disputes and other fixed costs. For normally charged work, the new-grant provider-cost ceiling is grant / 1,280 USD. Provider-billed failures, unusable AI output and customer charge caps can leave additional unrecovered expense; these figures are not a hard ceiling on total provider spend. Existing rollover is a liability from earlier paid months; spending it can make a single month's cash costs exceed that month's new-grant cost.

| Plan | Revenue INR | Credits | New-grant provider ceiling USD | Contribution INR | Contribution % |
| --- | ---: | ---: | ---: | ---: | ---: |
| Solo | 999 | 4,000 | 3.12500 | 656.53 | 65.7% |
| Growth | 2,499 | 10,000 | 7.81250 | 1,642.78 | 65.7% |
| Business | 4,999 | 15,000 | 11.71875 | 3,677.16 | 73.6% |
| Agency | 9,999 | 25,000 | 19.53125 | 7,745.91 | 77.5% |

At an illustrative ₹5,000/month fixed cost, those contributions cover it with 8 Solo, 4 Growth, 2 Business or 1 Agency customer, before the exclusions above. Actual launch profitability requires real invoices, retention, refund data and customer mix.

Rank schedule ceilings: five weekly runs at maximum plain-keyword capacity/depth, one device, 100-keyword batches and per-batch credit rounding require approximately Solo 1,155; Growth 7,700; Business 15,400; Agency 23,100 credits. Business exceeds its 15,000 grant; other plans also share remaining credits with research/AI. Advanced search operators/multiple devices can increase this further. Billing forecasts and the spend guard remain mandatory.

Sources: [DataForSEO SERP pricing](https://dataforseo.com/serp-api-calc), [Razorpay pricing](https://razorpay.com/pricing/), [plan API contract](https://razorpay.com/docs/api/payments/subscriptions/fetch-a-plan/). The repository catalog and task-reported costs remain authoritative for application behavior; published prices alone are not an invoice reconciliation.

## Cleanup and verification boundaries

- Remove duplicate Go declarations, unused cache helper and role/config wrappers. Use standard `crypto/rand.Text` and `strings.SplitSeq` where applicable.
- Remove unwired MCP timing/quota telemetry and duplicate worker supervision. Actual provider accounting stays in the shared ledger.
- Remove runtime imports/flags for canned research data. Playwright intercepts HTTP and installs its own session/project fixtures, exercising the actual schema/replay transport without paid calls.
- Correct stale browser selectors and assert writing during delayed saves is preserved. Run throttled performance checks against the built production SPA; interaction thresholds remain unchanged.
- Validation: full Go suite with race detector, coverage and disposable PostgreSQL/Redis; frontend 124 files / 633 tests; 32 production-build browser checks. Formatting, module consistency, vet, staticcheck, golangci-lint (0 issues), dead-code check (empty output), Prettier, Knip, TypeScript and Oxlint pass. Govulncheck finds no affected symbols/imported packages; one required-module advisory is outside called code. Real-database failure injection, payment mismatch/replay and unknown-outcome regressions reproduce faults before their fixes; removing 40203 billing classification makes the account-cap regression fail. Browser mocks do not exercise real provider/payment processing.

## Production observations and remaining acceptance

- Emergency RLS SQL applied successfully. Recheck: 0 unprotected public tables, 0 client grants; trusted-owner reads work; `/readyz` and signed-out session endpoint return 200. All 89 current application tables intentionally have no Data API policies. [Advisor remediation reference](https://supabase.com/docs/guides/database/database-linter?lint=0013_rls_disabled_in_public).
- At review time the application deployment was 21 October and Goose history was 36. New code/migrations in this PR require a release; an open PR alone does not update production.
- Railway and Supabase access work. All four Razorpay plan variable names are present. On deployment, inspect each `billing plan verified` result; a variable's presence alone does not verify its value or provider terms.
- Owner confirms DataForSEO currently works and deliberately keeps its account-side $2/day ceiling. Documented [40203 account-cap refusals](https://dataforseo.com/help-center/limit-my-payments) release customer credits and stop subsequent budget legs even when task-cost metadata is absent. Preserve it and the existing global emergency brake; a shared provider ceiling may pause paid features despite customer credits.
- No real charge, refund, renewal, WordPress publication, external email or new paid provider task was initiated by these checks. Owner-controlled live payment/cancel and connected-account acceptance remain necessary before advertising those flows as production-verified.
- Unknown provider holds require evidence-based resolution; tests do not reconcile production receipts. No open holds existed in the initial production aggregate. Dependency/index advisor information and Google verification status are outside a claim of exhaustive security or commercial acceptance.
