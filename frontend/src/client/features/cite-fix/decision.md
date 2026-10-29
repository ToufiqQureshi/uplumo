# Cite Fix (frontend): decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Own page that links to Site Audit for scans (PR 1 of 5)

- **Decision:** Cite Fix gets its own sidebar row ("Your website", badge "New") and page. The page only reads the report; to scan, it links to Site Audit instead of starting an audit itself. Fix copy comes from the API. The page polls every 15 s only while a scan is running.
- **Why:** The owner asked for its own UI from the first PR. Site Audit already owns the start form, plan page limits, capacity errors and crawler access; a second start button would duplicate that logic and its errors. Keeping copy on the server means later SAM and MCP tools say the same thing as the page.
- **Preserve:** Paid actions show cost and require confirmation before work;
  agent uses `retry: false`, and Google Overview refresh has browser retries
  disabled. Never word the score as a promise of citations. Keep empty states
  honest: an old scan says "scan again", not a low score.
- **Trace:** `CiteFixPage.tsx`, `citeFixApi.ts`, `scoreBand.ts`, `routes/_app/p/$projectId/cite-fix.tsx`, `navigation/items.ts`. Tests: `scoreBand.test.ts`, `navigation/items.test.ts` (18 routes, 12 rows), `frontend/tests/cite-fix.spec.ts` (added to the CI browser job: mocked report at 390 and 1280 px, one free request, no paid call). Not yet rendered against a real production report.

## Agent panel, results and proof

- **Decision:** Checkboxes on the weakest pages (top 5 preselected, max 20), Quick/Deep toggle with per-page caps, total max and balance shown and confirmed before the run; pages run one at a time with `retry: false`; results open as a Content Editor draft. A proof card shows who AI answers cite.
- **Preserve:** Never start paid work without the confirm dialog; stop the batch on credit/plan/rate errors.
- **Trace:** `AgentPanel.tsx`, `AgentResults.tsx`, `ProofCard.tsx`, `agentSelection.ts` (+ test). Playwright spec not yet extended for the agent.

## Browser test covers the paid agent

- **Decision:** `frontend/tests/cite-fix.spec.ts` also proves the agent buys
  nothing on load or on Cancel, and exactly one `agent/run` request per page
  after Start. The report fixture carries `proof` and `agent`, which the Zod
  schema requires.
- **Why:** the fixture lacked those fields after the agent landed, so the page
  failed to parse and CI went red; the money rule needs a browser-level count.
- **Preserve:** confirm before any paid call; `retry: false`.
- **Trace:** the spec fails if the button skips the confirm dialog (checked by
  mutation).

## Saved AI evidence and Google AI Overview (owner approved 28 Oct 2026)

- **Decision:** Add saved Brand Lookup mentions, free comparisons of up to
  three cited rival pages, Google AI Overview citations and an “Also fix for
  Google” audit-issue section. AI mentions are read from AI Visibility's saved
  result at 0 credits; the refresh link stays on that page and its existing
  price. Google Overview refresh has its own button and cost confirmation,
  sends no more than ten tracked questions, caps at 16 credits and charges
  actual cost. Results cache seven days.
- **Why:** The owner approved the $0.0012/question queue price only for Google
  AI Overview and expressly declined a new Cite Fix mentions purchase.
- **Preserve:** No paid request on load, no Cite Fix mentions refresh, no paid
  MCP trigger. Browser retries stay off. Rival pages use the free guarded audit
  fetch, respect robots.txt, are capped at three/report and never auto-render.
  App-shell rivals say so. The UI shows the paid AI Overview fields stored:
  answer text and citation source, domain, URL, title and snippet. Free audit
  issues link to Site Audit.
- **Trace:** `EvidenceSections.tsx`, `citeFixApi.ts`, Playwright
  `frontend/tests/cite-fix.spec.ts` (no load purchase, confirmation, exact one
  paid request, a failed confirmed request with no retry, rival table, mention
  reuse); mutating the refresh retry setting makes the failed-request call
  count fail. `audit-capabilities.spec.ts` covers render opt-in and its price.

## Topics and citation history (PR 3)

Show the coverage estimate as a heuristic, its page limit, cluster links and
missing-pillar draft action. Show saved citation positions and history with
scope and sample limits, not a promise of citations. Missing current signals
ask for a new scan. Opening the report or draft starts zero paid calls.
