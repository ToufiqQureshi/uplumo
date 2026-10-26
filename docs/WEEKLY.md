# Weekly summary

One short section per week (Saturday to Friday), newest first, last 8 weeks.
Read the latest week first; rules are in `CLAUDE.md` section 7.

## 24–30 Oct 2026 (in progress, updated 26 Oct)

**Merged:**

- Vision and audits: #208 (`FOUNDER.md`, `imp_sessions_talked.md`, credits
  audit), #209 (DataForSEO audit; AI Overview answers paid and never shown,
  fixed), #211 (wiring/tests/size audit, one dead route removed), #217
  (owner's 26 Oct decisions).
- Why not cited and AI readiness: #219, #221, #222, #220, #223, #227, #224.
- Value on existing pages: #197, #198, #214 (dashboard), #202–#205, #225
  (Search Performance: what changed, why, what to fix), #204, #226
  (Analytics).
- Money safety: #210 (max credits shown), #216 (saved reports never rebought
  without a click). CI: #212 (four Go test shards), #215 (paid-call browser
  tests). Content: #218 (Ubersuggest review), #230 (10 guides, theme menu).

**Decisions:** customer = Indian businesses, small to big; goal = first 10
paying customers; show max credits; hold only the typical cost and charge
the actual (not built); test workspace tracker stays daily.

**Money:** production provider spend $4.24 (DataForSEO $4.17), all from the
owner's 3 test workspaces, fully charged back.

**Docs changed:** `FOUNDER.md`, `imp_sessions_talked.md`, `docs/PRODUCT.md`,
`ROADMAP.md`, `README.md`, `CLAUDE.md`, `AGENTS.md`, `DECISIONS.md`, three
audits.

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
  sets MCP depth 10; #194 bills valid empty SERP 40102 and refunds invalid-field 40501.

**Owner decisions:** Solo ₹999 / Growth ₹2,499 / Business ₹4,999 / Agency
₹9,999; no per-workspace daily credit cap; Local SEO removed; IndexNow is paid;
Razorpay keys are live. See `DECISIONS.md`.

**Open owner asks:** complete live billing acceptance, finish Google OAuth
verification, and reconnect Railway GitHub auto-deploy. See section 2 of `DECISIONS.md`.

**Docs:** agent guidance now requires code-checked feature docs and a weekly
summary; external schedule status is outside this repository.
