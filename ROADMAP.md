# Uplumo roadmap

Status: ✅ done · 🔨 in progress · ⏳ not started.
Product reasoning lives in `docs/PRODUCT.md`; rules for agents in `CLAUDE.md`.

Feature wiring and Railway configuration reviewed on 20 October 2026.
See `docs/audits/2026-10-20-feature-wiring-review.md` for scope and validation limits.

## Where we are

Uplumo is live at `https://uplumo.com` on Railway. The whole backend is Go and
the React app is a static SPA the Go server serves. People can sign up, verify
their email and use the free features. The owner confirmed Razorpay uses
**live** keys, so a working subscription takes real money. All four plan variable names are present (23 October); their live terms and
real purchase/cancel acceptance still require release verification. DataForSEO
and OpenRouter variables are configured in production; accepted provider calls
incur real costs. The owner confirms DataForSEO works and intentionally keeps
its $2/day account cap. See `docs/audits/2026-10-23-launch-readiness.md`.

## Content editor

- Built: visual editor, on-page SEO/AEO scoring, search briefs (Brief tab,
  `POST /api/v1/projects/{projectId}/content/brief`, shared 30-day cache),
  13 AI writing actions with per-action credit ceilings, publish to
  WordPress, and "Track this article" links to Rank Tracking and Prompt
  Explorer after publishing. The free editor pack also adds project author
  profiles, review dates, WordPress Article/FAQ schema where permitted, and
  internal links from Search Console or the latest audit.
- Not built: scheduled tracking of a published URL's position and AI
  mentions inside the editor (today the writer opens Rank Tracking and
  Prompt Explorer from the publish card).

## Launch checklist

| Item                                                                                                   | Status                                                   |
| ------------------------------------------------------------------------------------------------------ | -------------------------------------------------------- |
| Railway deploy, domain, `www` redirect                                                                 | ✅                                                       |
| Sign-up, email verification, password reset, Google sign-in                                            | ✅                                                       |
| `support@uplumo.com` (Cloudflare forwards it)                                                          | ✅                                                       |
| Loops email: verify and reset working; invitation template made, not tested                            | 🔨                                                       |
| Public pages: landing, pricing, about, contact, terms, privacy, refunds, shipping                      | ✅                                                       |
| Razorpay account approved (KYC)                                                                        | ✅                                                       |
| Razorpay live keys in Railway                                                                          | ✅                                                       |
| Live Razorpay plan IDs match the catalog (Business included)                                           | 🔨 all four variables present; live terms pending        |
| Razorpay live renewal and one-click cancel confirmed on a real subscription                            | ⏳                                                       |
| One-click cancel (`POST /api/v1/billing/cancel`, "Cancel <plan>" on Billing)                           | ✅ built, not yet run against Razorpay                   |
| DataForSEO key in production (keywords, rank tracking, backlinks, AI visibility answer 503 without it) | ✅ owner confirms working 23 Oct; intentional $2/day cap |
| Google OAuth app verification (Search Console and Analytics scopes)                                    | ⏳                                                       |
| Live usage shown to the customer (credits on Billing, cost next to paid actions)                       | ✅ built                                                 |
| Spend guard: credits held before every paid lookup and AI call; provider-wide emergency brake          | ✅ built                                                 |
| Lawyer review of the legal pages                                                                       | ⏳                                                       |

## What is built

Built means the Go API and the page exist and are tested in CI. It does not
mean each one has been tried on production with real provider keys.

| Area                 | What exists                                                                                                                                                                                            |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Accounts             | Email and password, verification, reset, Google sign-in, workspaces, invitations, roles, API keys                                                                                                      |
| Uplumo Analytics     | Tracking script, ingest, dashboard (visitors, pages, sources, devices, countries), button clicks via `data-uplumo-event`                                                                               |
| AI traffic detection | ChatGPT, Perplexity, Gemini, Claude, Copilot referrers, plus AI traffic hiding as "Direct"                                                                                                             |
| AI Visibility        | Brand Lookup (mentions, citations, share of voice) and Prompt Explorer                                                                                                                                 |
| Keywords             | Research, SERP analysis, saved keywords with tags                                                                                                                                                      |
| Rank tracking        | Configs, manual and scheduled checks, city-level locations                                                                                                                                             |
| Site audit           | Crawl, issues, Lighthouse through Google PageSpeed Insights (free; homepage only on Free) with real-visitor Core Web Vitals                                                                            |
| Backlinks, Domain    | Overview, rows, referring domains, top pages; domain keywords and pages                                                                                                                                |
| Google               | Search Console performance opportunities and URL inspection, GA4 reports and measurement health                                                                                                        |
| Reports              | White-label reports (logo, name, colour, website), public share links                                                                                                                                  |
| Content editor       | Project drafts, live SEO/AEO checks, AI writing actions                                                                                                                                                |
| Website publishing   | Connect a WordPress site (onboarding or Settings → Integrations), publish or update drafts with their images (Basic and above)                                                                         |
| IndexNow             | Key file setup and check, submit changed URLs to Bing and other IndexNow engines; live WordPress publishes are submitted automatically (Basic and above; Google does not use IndexNow)                 |
| Assistant (SAM)      | Chat with tools over the project's data                                                                                                                                                                |
| MCP server           | Tools for Claude, ChatGPT and Cursor, with OAuth                                                                                                                                                       |
| Billing              | Free, Solo, Growth, Business and Agency on Razorpay Subscriptions, with site, tracked-keyword and member limits; webhooks, cancel, monthly credits with a spend guard on every paid lookup and AI call |
| Privacy              | `cmd/erase-user` erases one account                                                                                                                                                                    |

## Not built yet

| Feature                                                                                                  | Status                                             |
| -------------------------------------------------------------------------------------------------------- | -------------------------------------------------- |
| Changing from one paid plan to another without writing to support                                        | ⏳                                                 |
| Agency: client access, and extra sites at ₹400 each                                                      | ⏳                                                 |
| AI Visibility: saved prompts, daily/weekly/monthly checks, history and in-app drop warnings              | ✅ built; real-provider acceptance pending         |
| AI Visibility sentiment classification                                                                   | ⏳                                                 |
| "Seen in AI" next to "visitors from AI" (AI Visibility joined with Analytics)                            | ⏳                                                 |
| Publishing to Shopify, Wix, Webflow, plain HTML (SFTP) and an Uplumo-hosted blog                         | ⏳                                                 |
| Content editor on an external site; delivering fixes (JS snippet, GitHub PR, copy-paste)                 | ⏳                                                 |
| Site audit: AI "how to fix" for every issue (English + Hinglish)                                         | ⏳                                                 |
| Scheduled reports, emailed to clients                                                                    | ⏳                                                 |
| Client portal: clients log in and see only their own reports                                             | ⏳                                                 |
| Plain-language and Hinglish copy pass across the app                                                     | ⏳                                                 |
| USD pricing                                                                                              | ⏳                                                 |
| Account deletion and data export from Settings                                                           | ⏳                                                 |
| SEO Change Guard: compare saved completed audits with the same start URL and show field/coverage changes | ✅ built; explicit comparison, no automatic fixing |

## Competitive scan and product opportunity (16 October 2026)

This is a shortlist for product discovery, not a claim that competitors lack these capabilities. Public feature pages are not exhaustive.

- Ahrefs advertises always-on site audits, technical issue reporting and AI-assisted patches ([Site Audit](https://ahrefs.com/use-cases/seo), [product update](https://ahrefs.com/blog/new-features-apr-2025/)). Semrush also promotes technical audits and AI-crawler checks ([Site Audit](https://www.semrush.com/features/site-audit/)). So audits, internal links and AI-bot checks alone are not a defensible unique claim.
- Ahrefs AI Content Helper offers content-gap recommendations and in-editor assistance ([product page](https://ahrefs.com/ai-content-helper)). Content scoring and AI writing are also competitive categories; Uplumo should differentiate through its India-first pricing and a simpler action workflow, not claim exclusivity.
- **Proposed: SEO Change Guard.** Save crawl snapshots and diff titles, canonicals, indexability, robots directives, schema and internal links. Show the exact URLs and dates affected, then recrawl after the owner fixes the change. Use Uplumo's crawler and stored snapshots; this needs no paid SEO-data API.
- **Potential extension:** connect the change timeline to Uplumo's first-party analytics to show whether a change coincided with visits or lead events moving. Treat this as correlation, not proof that the change caused the movement.

## Usage and saved AI tracking

- Built: project-scoped saved AI questions, topics, engine/market selection, approved per-run credit ceilings, asynchronous collection, historical answers/citations, exports and comparable-scope drop warnings.
- Built: Billing forecast for saved rank/AI schedules, plus optional server-configured one-time Razorpay credit packs. Packs remain unavailable until pricing and validity are explicitly configured. Refund/dispute reconciliation is manual.

## Next up

Owner's order (25 Oct 2026, `FOUNDER.md` section 1a). New features wait.

1. ✅ Credits audit (`docs/audits/2026-10-25-credits-audit.md`). Owner still
   decides F1 (show the max, or cap the charge) and F5.
2. ⏳ DataForSEO audit: real paid calls per click vs. each `flow.md`.
3. ⏳ Build "why not cited, and what to fix".
4. ⏳ Wiring, test-strength and code/docs-size audits.

Still open from before: verify live Razorpay plan IDs and the
subscribe/renew/cancel flow; walk every production page as a real user;
let a customer change plan from Billing.

## Owner actions

- Razorpay: after release, inspect read-only plan verification logs against
  `backend/internal/plans/catalog.go`, then complete live payment/cancel acceptance.
- Submit the Google OAuth app for verification.
- Have a lawyer read the terms, privacy, refund and shipping pages.
- Make the GitHub repo private if it is not already.

Secrets go into Railway variables only, never into the repo or chat.

## Decided

- Name: Uplumo. Domain: uplumo.com. Hosting: Railway. Redis: yes.
- Launch market: India only, prices in INR. USD pricing comes later.
- GST: not registered at launch, so prices carry no tax line.
- Credits (owner, 13 October 2026; public pricing clarified 15 October 2026).
  Every paid data lookup and AI call spends credits; customers see each plan's
  monthly grant on the pricing page and see their balance and estimated action
  costs in the app. Monthly credits: Solo 4,000, Growth 10,000, Business 15,000,
  Agency 25,000 (1 credit represents $1/1,280 of provider cost at the current
  markup). New paid work stops when the shared workspace balance cannot cover
  it. There is no per-workspace daily credit cap; `PROVIDER_DAILY_LIMIT_USD`
  (default $100) is a provider-wide emergency brake. Unused plan credits roll
  over for up to two months. Rank depth is Solo top 30 and Growth, Business,
  Agency top 40; those are plan ceilings, not a guarantee that the maximum
  keyword list can run weekly. Billing forecasts scheduled spend, and each
  provider request still passes the shared spend guard. See
  `backend/internal/billing/README.md`.
- Plans (owner, 12 October 2026, confirmed 13 October). Built: the table
  below is `backend/internal/plans/catalog.go`. Free is one site with
  analytics and a starter audit.

  | Limit               | Solo | Growth | Business | Agency             |
  | ------------------- | ---- | ------ | -------- | ------------------ |
  | Price per month     | ₹999 | ₹2,499 | ₹4,999   | ₹9,999             |
  | Sites               | 1    | 2      | 5        | 10, then ₹400 each |
  | Tracked keywords    | 100  | 500    | 1,000    | 1,500              |
  | White-label reports | no   | no     | no       | yes                |
  | Client access       | no   | no     | no       | yes                |
  | Team members        | 1    | 3      | 5        | 10                 |

  Sites and white-label are what separate Pro from Agency: a small agency
  with four client sites must not fit in Pro. Basic and Pro differ by
  feature: Basic is keywords, rankings, Google data, search briefs and AI
  writing for one site; Pro adds backlinks, competitor research and AI
  Visibility.

- The owner's and the team's own workspaces are never limited by a plan
  (`STAFF_EMAILS`).

## Open decisions for the owner

- Credits audit F1: Brand Lookup shows "about 372 credits" but may charge up
  to its hold (2,240 in the US with competitors). Show the max, or cap the
  charge at the shown cost?
- Credits audit F5: confirm the 3 `active` subscriptions in production are
  the owner's own or test workspaces.

- The typical costs on Billing (`backend/internal/billing/costs.go`) are from
  published prices; check them against the first real DataForSEO and
  OpenRouter invoices.
- Data provider: keep DataForSEO, or drop the features that need it
  (keywords, rank tracking, backlinks, AI visibility). An AI model cannot
  replace it, because it has no real search data.
- Audit page and AI-check limits per plan.
- Cloudflare's managed `robots.txt` blocks AI crawlers (GPTBot, ClaudeBot,
  Google-Extended) on a product that sells AI visibility. Keep or turn off.
- Whether billing rows are kept when an account is erased (tax records). See
  `backend/internal/gdpr/README.md`.

## Editorial measurement

- **Built:** three prioritized editorial tasks with current evidence handed to
  the existing writer only after an explicit action; duplicate brief headings
  disabled; published articles link to saved AI and rank measurement.
- **Removed:** Missed Enquiries Finder and Lead Flow Guard at the owner's
  request. Existing Analytics tracking and its event history remain available.

## Search growth analyst

- Search Performance includes an explicit GSC + Google-organic GA4 opportunity
  workflow, query branding/competing-page evidence, measurement/device checks,
  stored AI citations, saved page actions and observed 14/28-day results.
- SAM can explain verified evidence and prepare page improvement briefs in
  read-only analyst mode. Existing editor tools keep review/apply explicit.
- Before/after differences are observational; GA4 key events require correct
  customer configuration and are not automatically confirmed leads or sales.

## Goal-led search diagnosis

- Built: registered GA4 business-goal selection, goal-specific session rates,
  low-click/query diagnosis, submitted-sitemap status and explicit stored-index
  checks, actual ordered GA4 funnels, attribution investigation, weekday-aligned
  year comparisons and optional unchanged-page comparison baselines.
- Built: explicit expanded growth coverage (5,000 rows/source) and opt-in weekly
  in-app top-three actions/drop alerts, with no scheduled paid AI or email.
- Google native generative-AI reporting and optional BigQuery setup are external
  entry points; automatic ingestion is not implemented. Real-account Google
  reconciliation and production acceptance remain pending.
