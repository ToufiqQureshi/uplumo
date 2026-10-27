# Cite Fix (frontend): decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Own page that links to Site Audit for scans (PR 1 of 5)

- **Decision:** Cite Fix gets its own sidebar row ("Your website", badge "New") and page. The page only reads the report; to scan, it links to Site Audit instead of starting an audit itself. Fix copy comes from the API. The page polls every 15 s only while a scan is running.
- **Why:** The owner asked for its own UI from the first PR. Site Audit already owns the start form, plan page limits, capacity errors and crawler access; a second start button would duplicate that logic and its errors. Keeping copy on the server means later SAM and MCP tools say the same thing as the page.
- **Preserve:** The only paid action is the agent (next entry): cost shown and confirmed before the click, `retry: false`. Never word the score as a promise of citations. Keep empty states honest: an old scan says "scan again", not a low score.
- **Trace:** `CiteFixPage.tsx`, `citeFixApi.ts`, `scoreBand.ts`, `routes/_app/p/$projectId/cite-fix.tsx`, `navigation/items.ts`. Tests: `scoreBand.test.ts`, `navigation/items.test.ts` (18 routes, 12 rows), `frontend/tests/cite-fix.spec.ts` (added to the CI browser job: mocked report at 390 and 1280 px, one free request, no paid call). Not yet rendered against a real production report.

## Agent panel, results and proof

- **Decision:** Checkboxes on the weakest pages (top 5 preselected, max 20), Quick/Deep toggle with per-page caps, total max and balance shown and confirmed before the run; pages run one at a time with `retry: false`; results open as a Content Editor draft. A proof card shows who AI answers cite.
- **Preserve:** Never start paid work without the confirm dialog; stop the batch on credit/plan/rate errors.
- **Trace:** `AgentPanel.tsx`, `AgentResults.tsx`, `ProofCard.tsx`, `agentSelection.ts` (+ test). Playwright spec not yet extended for the agent.
