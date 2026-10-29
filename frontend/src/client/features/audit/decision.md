# Site audit: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when a behavior or invariant changes.

## Why the feature is structured this way

Starts an asynchronous crawl and presents technical SEO findings, crawl progress and optional performance data.

## Decisions to preserve

- **Audit start can consume shared crawl capacity and Google quota** — keep per-plan limits, per-organization quotas and timeouts. _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.
- **Every crawl URL must pass the SSRF guard** — never let redirect targets bypass it. _Why:_ User and provider content or URLs can be hostile; validation and escaping prevent unauthorized exposure or executable content.
- **Lighthouse/CrUX unavailability is a partial result, not proof that a page has no field data** _Why:_ Unavailable or partial data is not a measured zero; keeping status visible prevents false customer decisions.
- **At-least-once jobs must remain idempotent so retries do not duplicate audit rows** _Why:_ Retries, replays and parallel requests are normal; idempotency prevents duplicate charges, records and side effects.
- Compare only completed audits sharing the same start URL, and describe absent crawl rows as missing rather than deleted. _Why:_ Different crawl scopes are not comparable, and crawler discovery gaps can resemble customer-side removals.
- Load saved result payloads only after the user requests a comparison; keep comparison local and read-only. _Why:_ History browsing should not create extra crawl/provider work or mutate customer data.
- Compute orphan suggestions lazily from saved audit data, cap displayed orphans and candidate links, and keep the user in control of adding links. _Why:_ Bounded local work avoids repeated processing and prevents an inferred suggestion from changing a customer site without review.

## Verification pointers

- `auditApi.test.ts`

## Opt-in rendering when the real provider is wired (owner approved 28 Oct)

- **Decision:** Use backend capabilities to show the JavaScript render checkbox
  only when DataForSEO is wired. The control states the 39-credit maximum for
  up to 20 flagged app-shell pages; actual delivered cost is charged.
- **Why:** The owner approved the published On-Page JS price ($0.0015/page)
  after seeing the 20-page and 50-page-site scenarios.
- **Preserve:** Rendering stays opt-in, occurs only after the standard crawl
  identifies app shells, charges zero on failed/unusable rendering and has no
  automatic retry. Audit page load, results and poll actions do not render.
- **Trace:** `launch/LaunchFormCard.tsx`, `launch/useLaunchController.ts`,
  `auditApi.ts`; `frontend/tests/audit-capabilities.spec.ts` checks the
  capability-controlled opt-in, cost copy and no request before a confirmed
  audit start.

## Usable guidance for rendered and initial-HTML audits

- **Decision:** Results distinguish an initial-HTML-only audit from one that
  opted into rendering. If a JavaScript app shell remains flagged, the warning
  says rendered content was not confirmed for that page and recommends checking
  crawler-readable content; it does not claim Google is blocked.
- **Why:** A requested render can fail, and audits above the render cap may
  leave shells unrendered. A saved config flag alone does not prove a given URL
  produced usable rendered HTML.
- **Preserve:** Keep the failed-page evidence and avoid suggesting unavailable
  controls. Guidance reflects the stored audit state.
- **Trace:** `results/RenderingWarnings.tsx`, `shared.tsx`, their callers and
  `RenderingWarnings.test.ts` (both states assert accurate, usable copy).

## Issue detail lists

- **Decision:** `IssueDetails` joins list values with commas, except redirect `hops`, which keep arrows.
- **Why:** The new `ai-crawlers-blocked` issue lists crawler names; arrows would read as a path.
- **Preserve:** Details stay plain text from the issue's JSON; nothing is fetched.
- **Trace:** `results/IssuesView.tsx`; backend decision in `backend/internal/audit/decision.md`.

## AI readiness score card

- **Decision:** `results/AiReadinessCard.tsx` sits above the audit stats. `aiReadiness.ts` scores 6 checks from issues the audit already found. Score = 100 minus the weight of each failed check:

  | Check (issue)           | Weight |
  | ----------------------- | ------ |
  | AI bots blocked         | 30     |
  | Business schema         | 25     |
  | Question answers        | 20     |
  | JavaScript-only content | 10     |
  | Several phone numbers   | 10     |
  | `llms.txt`              | 5      |

  The card shows "Pass", "Partly" or "Weak" for three areas: can AI read it, does AI know who you are, can AI quote you. It then lists the fixes, biggest first, each with the registry's explanation and how-to-fix.

- **Why:** The owner wants every audit to say whether the site is AEO/GEO friendly, why not, and what to fix. This is artboard 3 of the approved Traffic redesign.
- **Preserve:**
  - No request and no credits.
  - Audits started before `AI_CHECKS_SINCE` (2026-10-26T15:00Z, when the crawler checks went live) are not scored. They show "run a new audit", because a missing issue there means "not checked".
  - The weights are judgement. Change them here only.
- **Trace:** `aiReadiness.ts`, `aiReadiness.test.ts`. Each of these 4 mutations fails the tests: pass check, Weak verdict, the date guard, fix order. Backend checks: #220 and #223.

## New audit results (PR 3)

Treat absent new signals as unmeasured: display “scan again” and suppress the
old AI-readiness score. Training-only bot blocks are informational; only search
bot blocks count. External-check status remains visible when coverage is
pending or partial, so unavailable targets do not look verified.
