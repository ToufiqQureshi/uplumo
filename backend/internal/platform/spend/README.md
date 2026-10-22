# spend

The contract between code that costs Uplumo money at a provider and the
credit ledger that pays for it. `billing.SpendGuard` implements it; see
`internal/billing/README.md` for the rules (holds, settling, the daily
limit, AI).

- A provider transport asks `Guard.Hold` before it sends billed work, with
  the most the work can cost, and calls `Hold.Settle` exactly once with the
  real cost, also when the work failed.
- `Feature` labels the spend so the Billing screen can show where credits
  went. `Reference` is free lookup data (location lists, model lists).
- A refusal is `ErrNoCredits` or `ErrPaused`. Handlers call
  `WriteHTTPError` (402 `insufficient_credits`, 503 `usage_paused`) before their
  own mapping; MCP and rank tracking use `Message`. The DataForSEO client also
  wraps refusals in its `ErrBillingIssue`, so a handler that forgets the
  mapping still stops and answers 503 instead of 500.
- `spendtest.Guard` allows everything and records holds, for tests only.
- **One idempotency key is one user action, never one hold.** `requireSession`
  puts the browser's `Idempotency-Key` on the context
  (`spend.WithIdempotencyKey`). One action can buy several results (a backlink
  summary and its history, two keyword legs in parallel, the steps of a chat
  turn), so every hold derives its own key with
  `spend.ScopedKey(ctx, parts...)`: DataForSEO uses method, path and body;
  `DoBudgeted` and `AIMeter` use feature, endpoint and ceiling. Identical
  repeats in one action are numbered. A replay of the action therefore gets the
  same keys and is refused call by call (`ErrIdempotentReplay`, 409
  `duplicate_request`) with nothing sent or charged again, and a new action
  buys again. Work with no action key (background jobs) is not deduplicated.
