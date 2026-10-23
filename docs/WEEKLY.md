# Weekly summary

One short section per week (Saturday to Friday), newest first, last 8 weeks.
Read the latest week first; rules are in `CLAUDE.md` section 7.

## 17–23 Oct 2026 (updated 23 Oct; week closes Friday 23 Oct)

**Production:** Owner confirmed Razorpay live keys, so working subscriptions
charge real money. All four plan variable names are present; live terms and renewal/cancel
acceptance remain unconfirmed. The owner confirms DataForSEO works and keeps
an intentional $2/day account cap; paid provider calls incur real cost.
Emergency RLS/grant hardening closed direct client access to app tables.
PR #196 contains the code counterpart, financial fixes, cleanup and launch
checks; see `docs/audits/2026-10-23-launch-readiness.md`.

**Merged this week:**

- Paid-work safety and value: #159–176; cost hints, provider usage accounting,
  no automatic dashboard purchases, safe retries/cancellation and no repeated
  report purchases.
- AI/keyword/analytics/editor: #157, #158, #162, #163, #166, #168, #170,
  #173, #178, #179; citations, search intent, attribution and enquiry tracking
  now connect to existing actions and evidence.
- Navigation and security: #167, #169, #171, #177; Local SEO was removed by
  owner decision.
- Reliability and customer billing: #180–190 preserve unknown provider data,
  stabilize idempotency, coalesce paid refreshes, recover holds safely, disable
  paid query retries and validate cited URLs.
- Provider spend: #188, #191, #192 reuse paid SERP/domain/backlink snapshots
  after Redis expiry; tenant scope and age limits remain, incomplete backlinks
  results do not replace a complete snapshot.
- Rank and provider correctness: #193 removes a false weekly setup blocker and
  sets MCP depth 10; #194 bills valid empty SERP 40102 and refunds invalid-field
  40501.

**Owner decisions:** Solo ₹999 / Growth ₹2,499 / Business ₹4,999 / Agency
₹9,999; no per-workspace daily credit cap; Local SEO removed; IndexNow is paid;
Razorpay keys are live. See `DECISIONS.md`.

**Open owner asks:** complete live billing acceptance, finish Google OAuth
verification, and reconnect Railway GitHub auto-deploy. See section 2 of `DECISIONS.md`.

**Docs:** agent guidance now requires code-checked feature docs and a weekly
summary; external schedule status is outside this repository.
