# Uplumo: owner decisions and what has been done

Owner: Toufiq. This file records the owner's decisions and what still waits
on the owner. Feature status lives in `ROADMAP.md`, history in git.
**No secret values here, only variable names.** Agents: keep this file and
update it when the owner decides something or a Railway variable changes.

Last updated: 20 Oct 2026

---

## 1. Decisions taken by the owner

| Date | Decision | Why |
|---|---|---|
| 21 Oct | Four paid plans: **Solo ₹999 (1 site, 4,000 credits) / Growth ₹2,499 (2 sites, 10,000) / Business ₹4,999 (5 sites, 15,000, 1,000 keywords, 5 members, no white-label) / Agency ₹9,999 (10 sites, 25,000)**. Internal ids `basic`/`pro`/`business`/`agency`. Every card lists everything the plan includes. | Credits, not sites, cap provider cost; Agency at ₹5,999 for 10 sites was cheaper per site than Growth. |
| 14 Oct | WordPress publishing is on the **Basic, Pro and Agency** plans. Free gets a link to the plans. | It is a paid feature with no provider cost. |
| 14 Oct | Only an **owner or admin** can connect a website. Any member can publish. | The site password is sensitive. |
| 14 Oct | Onboarding has a **6th step**, "connect website", and it can be skipped. | Customers can connect early without being blocked. |
| 14 Oct | We **never** ask for cPanel or hPanel logins. We use only the WordPress application password. | Full server access would be too risky for us. |
| 14 Oct | Shopify, Wix, Webflow, SFTP and a hosted blog come **later**. WordPress is first. | Most target sites run on WordPress. |
| 14 Oct | Content editor: a **real scoring engine** with SERP briefs and AI help, built by Codex in 3 PRs. | The old score gave 26 words of spam 65/100. |
| 14 Oct | A brief = Google top 10 + "People also ask" only, **no keyword-volume call**. Cost about $0.0065, **9 credits** per brief. | Volume and difficulty do not change the score, and they were 65% of the brief's cost. |
| 14 Oct | Volume and difficulty come from a **"See keyword data →"** link to the Keyword Research page. | Data stays available, and is charged only when the user asks. |
| 14 Oct | The brief cache is **shared by all workspaces** for 30 days. Only a cache miss is charged. The same keyword at the same time makes one provider call. | Lower DataForSEO bill. |
| 14 Oct | **Free plan:** no brief and no AI, only the local checks. | Cost control. |
| 14 Oct | Scoring weights and caps follow the Codex proposal: stuffing → max 40 and "Publish now" blocked; copied text → max 30; under 100 words → max 30. | Writers cannot game the score with spam. |
| 14 Oct | AI model `openai/gpt-4.1-mini`, 10 actions, 11–15 credits each. Credits are reserved before a call and fully released if the provider fails. | Fixed cost; the customer never pays for our failure. |
| 14 Oct | The keyword-stuffing block applies on the **server** too, not only on screen. "Send as draft" still works. | An API or agent cannot publish spam to a customer's site. |
| 15 Oct | The PageSpeed API key has **API restrictions only** (PageSpeed and CrUX) and application restriction **None**. | The server sends no referrer, so a "Websites" restriction would block it. Railway has no fixed IP. |
| 15 Oct | `GOOGLE_TOKEN_ENCRYPTION_KEY` and `AUDIT_BROWSER_RENDERING` are **not set for now**. | The first would break existing Google connections; the second needs Chrome on the server and a lot of memory. |
| 15 Oct | A search brief reads 8 competitor pages (worst case $0.0072, about 9.2 credits) instead of 10, with one content-parsing call per page. | DataForSEO's live parsing endpoint takes one page per call. 8 pages keep the brief near the approved 9 credits. |
| 15 Oct | The AI writer always uses `openai/gpt-4.1-mini`, whatever `OPENROUTER_MODEL` says. Free OpenRouter models are not used. | The per-action credit ceilings are priced for that model. Free models have tight rate limits and may train on customer content. |
| 15 Oct | "Track this article" opens Rank Tracking and Prompt Explorer after publishing; no separate scheduled tracking yet. | Reuses the existing tools and their credit checks instead of a second scheduler. |
| 15 Oct | Migration `00023` is the brief cache. The next migration (Buffy's author tables) must be `00024`. | Two PRs used `00023`; goose refuses a gap or a duplicate. |
| 19 Oct | **A paid action that loses one call keeps what was already paid for.** Keyword Research answers with the leg that worked, Backlinks with the summary and no trend, Local SEO with the points already searched; each is cached ten minutes only. Credit refusals, replays and a departed customer still fail. | Before, the customer paid for the first call and saw an error, and the retry paid again. |
| 19 Oct | Each paid call in one user action holds credits under its own key (`spend.ScopedKey`); the browser's `Idempotency-Key` identifies the action, not the hold. | The 17 Oct change reused one key for every call in a request, so Backlinks overview, Local SEO grids, SAM turns and Keyword Research failed after the first call was charged. |
| 19 Oct | HTTP `writeTimeout` is 150s (was 30s) and a budgeted operation stops starting calls after 120s. | A provider call can take 60s; at 30s the connection closed and the customer saw a generic error after paying. |
| 19 Oct | The "about N credits" hints are the live measured cost of one click (keyword search 61, domain tab 40, backlink report 64, 3×3 map grid 27). AI Visibility has two hints: prompt explorer 128 and brand lookup about 910 (a real lookup billed $0.706 and 964 credits). | Customers saw 26 and paid 40. |
| 19 Oct | **No per-workspace daily credit limit.** `WorkspaceDailyLimitUSD` returns 0; the credits a customer bought are the limit. The global `PROVIDER_DAILY_LIMIT_USD` brake stays until the owner removes it. | Owner: a customer who paid must not be stopped by a day cap. It showed "used today's share" to paying customers. |
| 19 Oct | AI call hold ceilings are per endpoint: $0.10 per model answer, $0.25 per mentions task (was a flat $0.50). | A flat $0.50 made 4 models with web search hold $4.00 and Brand Lookup $6, so neither could run; live calls bill $0.03 and $0.10-0.15. |
| 20 Oct | **IndexNow** is on **Basic, Pro and Agency**, not Free. Pages published live to WordPress from Uplumo are submitted automatically once the project has set up IndexNow. | Owner decision. Each workspace may make 60 IndexNow requests an hour, so one cannot get Uplumo's servers rate-limited for everyone. Google does not use IndexNow. |

## 2. Waiting for the owner

- [ ] **DataForSEO:** verify the account and top up (at least $50). Until then these return 40104 "verify your account": keyword research, domain, backlinks, brand lookup, prompt explorer, rank tracking and local SEO. No money is charged meanwhile.
- [ ] **Google OAuth data-access verification:** add the scope justification, upload the demo video (YouTube, unlisted), then submit in the Verification Center. Branding is already verified.
- [ ] **GA4 connect:** use "Advanced → Go to uplumo" until Google verifies the app. The user cap is 100, and 1 is used.
- [ ] **WordPress (revmerito.com):** add the 2 `.htaccess` lines on Hostinger (the host drops the Authorization header), then connect with a real application password (24 characters).
- [ ] **Railway GitHub auto-deploy** is broken: Railway shows "Could not load branches". Reconnect GitHub in Railway Account Settings → Connections. Until then, deploy with `railway up` (see `docs/DEPLOY.md`).
- [ ] Mark the 2 GitGuardian incidents from PR #122 as false positives.
- [ ] Codex PR 2 (SERP briefs and credits) and PR 3 (AI actions) still have to be built.

## 3. Keys and settings in Railway (names only)

| Variable | Status | What it is for |
|---|---|---|
| `DATABASE_URL` | set | Supabase Postgres |
| `REDIS_URL` | set | Railway Redis |
| `BETTER_AUTH_SECRET`, `BETTER_AUTH_URL`, `PUBLIC_URL` | set | Login and links |
| `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET` | set | Google login, Search Console, GA4 |
| `RAZORPAY_KEY_ID`, `RAZORPAY_KEY_SECRET`, `RAZORPAY_WEBHOOK_SECRET` | set (**test mode**) | Billing |
| `RAZORPAY_PLAN_ID_BASIC`, `_PRO`, `_AGENCY` | set (test) | Monthly plans. Test Agency plan is still ₹5,999: recreate at ₹9,999 |
| `RAZORPAY_PLAN_ID_BUSINESS` | **not set** | Business ₹4,999 monthly plan; without it Business cannot be bought |
| `LOOPS_API_KEY` + 3 `LOOPS_TRANSACTIONAL_*_ID` | set | Email |
| `SUPABASE_URL`, `SUPABASE_SERVICE_ROLE_KEY`, `SUPABASE_CONTENT_IMAGE_BUCKET` | set (14 Oct) | Content editor images (`content-images` bucket) |
| `DATAFORSEO_API_KEY` | **set 14 Oct** | SEO data; the account is not verified yet |
| `OPENROUTER_API_KEY`, `OPENROUTER_MODEL` | set | AI writing and assistant |
| `PAGESPEED_API_KEY` | **set 15 Oct** | Lighthouse in Site Audit (free, 200 checks per workspace per day) |
| `CRUX_API_KEY` | **set 15 Oct** (same key) | Real-visitor Core Web Vitals |
| `SENTRY_DSN`, `STAFF_EMAILS`, `PORT` | set | Errors, staff access |
| `PROVIDER_DAILY_LIMIT_USD` | set; **keep until the paid-call bugs are fixed**, then the owner removes it | Global emergency spending brake |
| `TURNSTILE_SECRET_KEY` | not set | Bot check on sign-up |
| `GOOGLE_TOKEN_ENCRYPTION_KEY` | not set (decision above) | — |

Railway project `jubilant-kindness`, service `uplumo`. Service settings: healthcheck `/readyz`, region Singapore, 1 replica.

Google Cloud project `agentproject-473109`:

- OAuth app is "In production"; branding is verified.
- Scopes: `openid`, email, profile, `webmasters.readonly` and `analytics.readonly`. The last one needs verification.
- APIs enabled: PageSpeed Insights and Chrome UX Report.

## 4. What was built and shipped (PRs)

| PR | What | Live |
|---|---|---|
| #120 (Codex) | New content editor, PNG/JPEG image upload to Supabase (WebP), Markdown preview | yes |
| #121 | Billing page waits for the plan to change after a payment | yes |
| #122 | WordPress connect plus "Send as draft / Publish now" | yes |
| #123 | The dashboard toast "Something went wrong" (backlinks panic without DataForSEO) | yes |
| #124 | WordPress connect: a clear message when a firewall blocks us | yes |
| #125 (Codex) | Scoring engine plus new editor UI: stuffing and copy caps, SEO/AEO checks, brief, outline | yes |
| #126 | Server refuses "Publish now" on a stuffed draft; `DEPLOY.md` corrected | yes |
| #127 | Live-test fixes: preview formatting, "Your drafts" shown twice, provider outage message, audit polling halved, 0-word read time, Hostinger dropping the login header | yes (12:46 UTC, 14 Oct) |

## 5. Live test results (14 Oct, owner's Chrome)

- All 15 pages load with no console errors. Every API call answers 200, and no paid call is made when a page opens.
- Content editor: a 265-word article scored SEO 82 and AEO 85. Typing makes 0 API calls. Save, reload and reopen work. The image upload reaches Supabase and shows in the preview.
- Site audit on revmerito.com, 10 pages: 25 issues (3 warnings, 22 info).
- Security: another workspace's project answers 404.
- Blocked by DataForSEO verification: keyword research and prompt explorer.
- Blocked by Hostinger: WordPress connect (the `.htaccess` fix is pending).

## Content publishing implementation choices (15 October 2026)

- Author profiles belong to projects, not login accounts. A draft may select
  one; deleting the profile clears its reference without deleting the draft.
  The review date is set by the writer and feeds the existing AEO freshness
  check without changing its weight.
- WordPress JSON-LD is sent as script tags only after the connected account's
  `unfiltered_html` capability is checked. WordPress can strip scripts for
  other roles; in that case publication succeeds without schema and the
  writer sees the reason. The API does not assume Rank Math or Yoast REST meta
  fields are exposed on every customer site.
- Internal links are suggested from the project's Search Console pages, then
  its latest audit as fallback. The writer starts the free lookup explicitly;
  typing and page loads do not make provider calls.

## 6. Cost notes

- A content brief costs about $0.0065 (9 credits). Pro (10,000 credits) ≈ 1,111 briefs a month. The daily limit per workspace is a third of its monthly credits.
- An AI action costs at most 11–15 credits.
- PageSpeed and CrUX are free (Google quota).
- 1 credit = $1/1,280 of provider cost.

## Approved implementation scope (18 October 2026)

- Owner requested two PRs: (1) AI question tracking, scheduled usage previews and self-service credit packs; (2) content editor improvements, published-page tracking and enquiry opportunities.
- Reuse Go/Postgres jobs and the existing credit ledger. Paid features share workspace credits and retain daily limits. No subscription prices or grants are changed.
- Credit-pack commercial terms require an explicit `BILLING_TOPUP_PACKS` configuration; no default pack price or new production configuration is activated by these PRs.

- Owner requested two improvement PRs including the content editor and a paid
  product differentiator. The second implements evidence → editorial fix →
  enquiry measurement, reusing existing analytics and writing services. It adds
  no plan-price change, new paid provider, automatic publishing or externally
  delivered alerts. Competitor feature overlap is acknowledged in the PR.

## Search Growth scope and maintenance decisions

- Owner requested GA4 and Uplumo tags for uplumo.com, including subscription
  intent clicks and source attribution, with direct push to `main` and no PR.
  The shared loader and measurement limits are recorded in
  `backend/internal/site/decision.md`; billing clicks remain separate from
  server-confirmed paid subscriptions. This is a task-specific workflow exception.

- **Decision:** consolidate Search Performance (#165) and Search Growth (#166)
  into a single PR #166 against `main`, superseding #165 without merging or
  deploying it. **Why:** the owner wants one reviewable change; the growth branch
  already contains the original Search Performance commits. Keep the developer
  handover in `PR_166_DEVELOPER_HANDOVER.txt` aligned with that combined scope.
- **Decision:** deliver business-goal-led actions and evidence rather than a
  claim to ingest every GA4/GSC screen. **Why:** useful decisions require correct
  denominators, technical context and honest coverage. UI, measurement, storage
  and AI boundaries are maintained in the Search Performance, GA4, GSC,
  database, SAM and MCP `decision.md` files.
- **Decision:** every repository-changing task must update its relevant
  `decision.md` in the same commit; repository-wide decisions use this file.
  **Why:** an AI or developer must understand the choice, evidence, trade-offs
  and constraints without reading chat history or guessing from code. This
  includes UI fixes, refactors, tests and documentation changes. The mandatory
  procedure is defined once in `CLAUDE.md` section 7; a PR handover does not
  replace feature decisions. No automated enforcement is claimed.

## Analytics daily audience definition

Owner-approved scope: one focused PR adds optional 180-day browser recognition and click destinations. Daily UTC new browsers were first observed that day; returning browsers were first observed on an earlier day. A browser counts once daily, even across ten visits. Unknown coverage stays separate. Revenue/refunds/ROI, AI, alerts and GA4 redesign are excluded. Implementation, privacy and verification invariants live in `backend/internal/analytics/decision.md`.

## Autonomous feature-wiring review

- **Decision:** The owner authorized a repository-wide frontend/backend/data wiring review and necessary fixes, with Go built-ins and standard libraries preferred over redundant helpers. Review findings live in `docs/audits/2026-10-20-feature-wiring-review.md`.
- **Why:** Customers need working, truthful features; frontend presence alone does not prove backend persistence or production availability.
- **Preserve:** Normal PR review and separate deployment, tenant isolation, paid-provider guards and the mandatory feature decision records. This authorization does not claim every external integration has been reconciled live.
- **Trace:** Feature decisions in audit, GSC, MCP, AI search and billing; the dated review distinguishes source inspection from executed tests. No agent-rule changes.

## Owner authorization to complete and ship

- **Decision:** The owner explicitly authorized autonomous implementation, tests, PR creation/merge, Railway deployment and Supabase work to make Uplumo usable end to end. Routine reversible implementation and release steps do not wait for the owner while they sleep.
- **Why:** There is no separate development team; source changes alone do not establish that customers can use the result.
- **Preserve:** Validate real data paths and failure/cost behavior, report remaining limits honestly, protect existing customer data and credentials, and never turn test-mode billing into live charges without validated payment readiness. Every change still updates the affected decision record.
- **Trace:** Root agent guide, feature decision records, PR checks and deployment/live verification evidence. This records owner authorization; it does not alter agent rules or assert production readiness without evidence.

## Patched Go certificate verification in release builds

- **Decision:** Require Go 1.27.3 in `backend/go.mod` and pin the Docker Go builder to the same patched release.
- **Why:** The release `govulncheck` found reachable standard-library advisory [GO-2026-6809](https://pkg.go.dev/vuln/GO-2026-6809), concerning compromised intermediate certificate distrust, with the local Go 1.27.2 toolchain. The official advisory identifies 1.27.3 as fixed. A floating minor Docker tag does not document the minimum safe compiler for local builds or CI.
- **Preserve:** Keep certificate verification enabled and retain the existing application, provider billing and migration behavior. Future toolchain updates must check the official advisory database and build both server executables. No vulnerability exclusion or CI bypass was added.
- **Trace:** `backend/go.mod`, root `Dockerfile`; full Go checks against disposable Postgres/Redis and `govulncheck` on the patched compiler. This maintenance correction accompanies the dashboard release review; no product pricing or agent/CI rules changed.

- **Release readiness:** `railway.json` uses the existing `/readyz` endpoint, which checks Postgres and Redis with deadlines, instead of liveness-only `/api/health`. A listening process must not qualify as a healthy replacement while required stores are unavailable. `docs/DEPLOY.md` documents both meanings; existing mounted readiness tests cover dependency failures. Live readiness is checked before and after rollout.
