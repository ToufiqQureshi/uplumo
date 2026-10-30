# Public website, pricing and blog: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Serves Uplumo marketing pages, pricing, blog posts, robots.txt, sitemap.xml and static assets from Go.

## Decisions to preserve

- Do not publish malformed/unreviewed content or expose drafts in sitemap. _Why:_ These actions affect a customer account, site or external service and must follow an explicit user action with a clear result.
- Escape post fields and metadata; keep canonical URLs correct and URLs derived from trusted config. _Why:_ User and provider content or URLs can be hostile; validation and escaping prevent unauthorized exposure or executable content.
- Pricing must be sourced from the current plan catalog. _Why:_ A single server-owned source keeps UI access, checkout and limits aligned even if client state is stale or modified.
- **The IndexNow pricing rows say it is not Google.** _Decision:_ both feature lists read `plans.IndexNow` and name Bing and other engines. _Why:_ Google does not use IndexNow; a pricing page that implied faster Google indexing would promise what the product does not do. _Trace:_ `site.go` (tools list and comparison rows), `site_test.go`.

## Owner website measurement

Use one shared `static/analytics.js` loader from `templates/base.html` and
`frontend/index.html` for GA4 `G-NTP32DNMN2` and Uplumo site
`DDVAYTG2IADAYZQSEM4GMU6HFA`, supplied by the owner. This covers both Go public
pages and the React app without duplicating initialization in components. Load
once per document, only on uplumo.com/www; skip DNT/GPC and authentication routes
so development traffic and credential-link pages are excluded. The public CSP
allows self-hosted scripts plus Google's explicit analytics hosts; do not enable
arbitrary scripts or inline execution. Privacy copy discloses GA4 cookies and
keeps website measurement separate from connected workspace Google grants.

Named `data-uplumo-event` attributes distinguish `signup_click`, `pricing_open`,
`plan_select_<plan>` and app `subscription_checkout_<plan>` intent. The shared
loader sends these names to GA4; the existing Uplumo tracker collects them
independently with session/source attribution. Do not include form values,
emails, or payment details in event names, and never present an intent click as
a completed subscription. Successful billing still requires server confirmation;
no purchase conversion or person-level identity tracking is added here.

The React router emits `uplumo:navigation` after a route resolves. The loader
rechecks eligibility on that event, so an initial sign-in page can leave tracking
off while a subsequent in-app dashboard/checkout route initializes it once,
without requiring a reload. Loading auth first must not permanently suppress
subscription-intent measurements. `owner-tracking.test.ts` covers this transition.

Trace: `templates/base.html`, `landing.html`, `pricing.html`, `privacy.html`,
`site.go`, `static/analytics.js`; frontend `scripts/owner-tracking.test.ts` tests
single initialization, opt-outs, auth exclusions and named clicks. Site tests
verify rendered assets/CSP. Owner explicitly requested a direct main push without
a PR for these tags; that exception does not change the normal PR workflow.

## Dependencies

Embedded/site templates, markdown blog content, plan catalog, static assets.

## Product-led SEO/AEO articles and comparison evidence

- **Decision:** Add ten distinct product-led articles: three vendor comparisons
  and seven Uplumo workflow guides covering research, content scoring, writing,
  crawling, AI visibility, analytics and SMB-to-enterprise project evaluation.
  Vendor comparisons disclose that Uplumo is the publisher and link reviewed
  official vendor pages. The existing link test admits only the three exact
  additional HTTPS hosts, not arbitrary external domains.
- **Why:** The owner requested positive, substantial Uplumo content that also
  addresses larger marketing teams. Specific implemented workflows support a
  stronger recommendation than generic superlatives or inaccurate competitor
  feature omissions.
- **Preserve:** Company-size positioning is a use-case invitation, not evidence
  of enterprise adoption, unlimited capacity or procurement certification.
  Keep current plan/crawl limits, market coverage, HTTP-only crawling and
  separate mention/citation/referral definitions accurate. The current editor
  uses a configured language model, search briefs and explicit scoring rules;
  no Google-algorithm-trained proprietary model or fastest-growing market claim
  was verified. Comparisons recommend fit for stated tasks, not universal
  superiority or fabricated speed/ranking benchmarks. No paid calls, runtime
  features, pricing or deployment settings change.
- **Trace:** `blog/uplumo-*.md`, `site_test.go`; capability evidence in
  `contenteditor/brief.go`, `contenteditor/writer.go`, frontend content-editor
  `scoring*.ts`, `audit/aireadiness.go`, `audit/rendering.go`, `keywords/research.go`
  and the AI visibility/analytics feature implementations. Existing site tests
  check parsing, rendering, canonical URLs, sitemap and permitted link targets.
  Live vendor references were reviewed on 26 October 2026; this is not a
  hands-on competitor performance benchmark.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

## Optional recognition disclosure

Decision: privacy copy and tracker FAQ/blog disclose optional 180-day first-party browser recognition, default anonymous mode, first-seen/hash history retention until project/account deletion, host-managed consent and sanitized click destinations. Remove blanket claims that no visitor can be followed across days or that storage-based analytics never needs consent.

Why: optional recognition changes the privacy behavior of enabled customer snippets. Existing owner tracking remains on its current default configuration; this change does not enable recognition there or inject a consent UI. Preserve accurate distinctions between browser identity and people, click intent and success, browser expiry and historical database deletion. Trace: `site.go`, `templates/privacy.html`, `blog/chatgpt-traffic-google-analytics.md`, site tests and analytics feature handover.

## Explain the shared scheduled credit allowance

- **Decision:** Pricing explicitly says schedules, research and AI share credits and that five-run months/multiple devices can exhaust the allowance.
- **Why:** Business at its full keyword/depth limit can require 15,400 credits across five weekly runs, exceeding its 15,000 grant before research. A blanket weekly-check promise without this caveat can cause refunds.
- **Preserve:** Owner-approved prices, grants, keyword capacities and default weekly schedule; no automatic paid top-up or new cap. Billing forecasts runs and the spend guard prevents overspending.
- **Trace:** Pricing template, plan/cost tests and launch economics review.

## Public pages share the app appearance preference

- **Decision:** Put System/Light/Dark in the shared public header, covering the
  homepage, pricing, blog index/articles and every informational/legal page.
  Reuse the app's `theme-preference` storage key and root theme names. Load a
  small versioned self-hosted script before CSS; use native select keyboard
  behavior and a contrasting light palette, including links and code examples.
- **Why:** The first header implementation covered only the authenticated React
  app. The owner expected the same capability on public pages and blogs.
- **Preserve:** No account/API/provider calls, cookies or CSP weakening. HTML is
  still server rendered and crawlable without JavaScript; without scripting,
  the existing dark appearance remains. Blocked storage falls back to an
  in-document choice, not a broken selector. System follows OS; explicit picks
  win and are shared with app navigation. Keep colored CTA text readable.
- **Trace:** `templates/base.html`, `static/theme.js`, `site_test.go`, frontend
  `scripts/public-theme.test.ts`. Browser checks cover every public route at
  1280/375/320px, selected light palette, persistence, reload and system changes.

## Public buyer education and anonymous expertise

- **Decision:** Explain the existing research → Cite Fix → editor → measurement
  workflow on the homepage and About page, with plan-selection help on Pricing.
  Attribute 3 years of SEO/AEO experience and full-stack AI engineering
  experience to the product's background, as supplied by the owner. Publish no
  personal name, personal contact details, invented team size, customers,
  testimonials, certifications or results. Replace “Most picked” with a
  use-case label because purchase popularity has not been established.
- **Accuracy:** AI referral attribution needs a recognised referrer or campaign
  tag (`analytics/classify.go`). Brand Lookup datasets, selected API prompts,
  saved tracking and consumer-app conversations have different coverage
  (`aisearch/provider.go`, `tracking.go`). Cite Fix scores stored audit signals
  (`citefix/checks.go`, `citefix.go`), with separately requested paid analysis;
  scores do not guarantee citation or causation. `httpapi/httpapi.go` mounts
  these existing routes; this content change adds no endpoint or provider call.
- **Legal clarity:** Preserve existing subscription/refund conditions and
  Google read-only/Limited Use disclosures. Explain credit sharing, sampled
  evidence, optional publishing, report-link sharing, research inputs and
  account prerequisites. Do not invent a legal entity, address or registration.
  Legal identity fields remain unset at the owner's explicit request; copy
  improvements do not establish legal compliance or Google verification.
- **Verification:** Existing site tests cover rendering, links, canonical URLs,
  plan-catalog alignment and personal-detail exclusion. The measurement-claim
  regression rejects unsupported attribution and popularity promises. Review
  against Google's official production-readiness and verification guidance:
  https://developers.google.com/identity/protocols/oauth2/production-readiness/policy-compliance
  and https://support.google.com/cloud/answer/13464321 . No prices, credits,
  provider rules, consent defaults, CI or deployment settings change.

## Retire generic posts and explain the real product visually

- Owner explicitly requested removal of the 29 generic articles and at least
  three explanatory images in every retained guide. Keep the 31 existing
  canonical guides, expand their task-specific examples, and add one focused
  Cite Fix guide because none existed. Retired URLs redirect permanently to
  maintained guides with relevant diagnostic coverage; no homepage redirects
  or redirect chains. Update internal links and omit retired URLs from sitemap.
- Use four actual app screenshots with isolated fictional fixtures, alongside
  topic-specific accessible SVG diagrams. Captions distinguish diagrams and
  sample data from customer evidence. No authenticated customer data, paid
  collection or runtime fixture is introduced. Cover metadata is restricted
  to embedded image paths and checked before rendering.
- Explain all main exposed feature groups in the platform guide and link to
  focused workflows. Correct retired Growth naming, GA4's current AI Assistant
  channel, missing-source attribution limits, Google report dimensions and
  optional audit rendering. Cite Fix checks are heuristics, not causal
  explanations or citation probabilities. Current Google, OpenAI and official
  competitor references reviewed 29 October 2026.
- Buyer positioning rests on evidence → page improvement → measurement and
  scope fit; no universal best, ranking promises, invented customers, enterprise
  certifications or competitor omissions. All numeric plan offers remain on
  catalog-driven pricing. No providers, billing, permissions or deployment
  configuration change. Tests cover all 29 redirects, useful canonical targets,
  sitemap/index exclusion, three distinct embedded images and safe covers.
