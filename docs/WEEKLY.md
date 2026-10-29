# Weekly summary

One short section per week (Saturday to Friday), newest first, last 8 weeks.
Read the latest week first; rules are in `CLAUDE.md` section 7.

## 24–30 Oct 2026 (updated 29 Oct; week closes Friday 30 Oct)

**Production:** settings fixed on 27 Oct (proxy range, pool cap, password
rotation; #237). #243, #245 and #246 are merged but not deployed: merging to
`main` does not deploy (`docs/DEPLOY.md`).

**Merged this week:**

- AI visibility and Site Audit: #219–#223, #227 (why AI does not cite you,
  local customer questions, AI-crawler/`llms.txt` checks, AI readiness score).
- Search Performance and Analytics: #225, #226, #234–#236, #239, #240 (lost
  clicks with why, GA4 channels and real enquiries, GA4 provider fix).
- Dashboard and reports: #197, #198, #209, #214, #216 (customizable cards,
  never rebuy a saved report unasked, AI Overview answers that were paid for
  are now shown).
- Money: #210 shows the maximum credits before Brand Lookup and Prompt
  Explorer; #238 three plans, credit packs, site add-on, no double payments.
- Cite Fix: #243 (score, paid Quick/Deep page agent, AI Visibility proof,
  JS rendering, Google AI Overview, rivals; includes Codex #244) and #246
  (free checks, topic clusters, citation history).
- Keywords: #245 turns the page into a chat agent; one answer charges at most
  1,200 credits, data and AI together, enforced by the server.
- Insights PR 1: #241. Product guides and themes: #230, #233. CI: #212, #215.
- Reverted: #205 and part of #203.

**Owner decisions:** Basic ₹999 / Pro ₹2,499 / Business ₹4,999, Agency
removed (27 Oct); Cite Fix scope and caps (Quick ≤25, Deep ≤60, JS render
≤39, AI Overview ≤16 credits); Keywords agent with no Genkit and no
DataForSEO MCP in the product; "credits must be cut exactly as shown". See
`DECISIONS.md`.

**Provider cost:** new paid paths are Cite Fix (OpenRouter DeepSeek,
DataForSEO On-Page and SERP) and the Keywords agent; each holds credits
first and is capped. Strix found and Claude fixed under-booked Keywords
reservations (local seeds, `site:` queries) before merge.

**Open owner asks:** deploy latest `main`; real-provider acceptance runs for
Cite Fix and Keywords; confirm Keywords limits; Razorpay webhook and Google
OAuth verification (section 2 of `DECISIONS.md`).

**Docs:** `docs/audits/2026-10-28-cite-fix-costs-and-references.md`;
Cite Fix, Keywords and SAM `decision.md`/`flow.md`; ROADMAP Cite Fix and
Keywords rows.

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
