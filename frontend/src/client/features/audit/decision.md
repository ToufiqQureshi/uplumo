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

## Show supported audit work

- **Decision:** Use the backend capability to disable JavaScript rendering; remove the unused local render-price estimate and its special credit-error calculation. Remediation describes initial-HTML coverage and recommends independent browser verification.
- **Why:** The server has no JavaScript renderer. Displaying a price or telling users to enable it implied a capability they could not use.
- **Preserve:** Keep ordinary crawl and Lighthouse controls, server error handling and existing launch/result contracts. No provider work, new charge or layout redesign.
- **Trace:** `launch/LaunchFormCard.tsx`, `launch/useLaunchController.ts`, shared `audit-issues.ts`; backend audit decision and regression tests; `frontend/tests/audit-capabilities.spec.ts` checks the disabled control, explanation and absence of automatic audit/paid work on mobile and desktop using API fixtures.

## Usable guidance for initial-HTML and historical audits

- **Decision:** Results warnings never tell users to enable the unavailable renderer. Stored render flags describe a request, not evidence of execution. Bot-block advice offers browser/host checks without inferring that Google is blocked.
- **Why:** Review caught a second results alert that contradicted launch capabilities; legacy flags could also make false execution claims.
- **Preserve:** Existing audit data stays unchanged. Keep failed-page evidence and explicit limitations; guidance must describe actions users can actually take.
- **Trace:** `results/RenderingWarnings.tsx`, `shared.tsx`, their callers and `RenderingWarnings.test.ts` (both cases failed before fixing the copy).

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
