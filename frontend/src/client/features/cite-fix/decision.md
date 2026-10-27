# Cite Fix (frontend): decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Own page that links to Site Audit for scans (PR 1 of 5)

- **Decision:** Cite Fix gets its own sidebar row ("Your website", badge "New") and page. The page only reads the report; to scan, it links to Site Audit instead of starting an audit itself. Fix copy comes from the API. The page polls every 15 s only while a scan is running.
- **Why:** The owner asked for its own UI from the first PR. Site Audit already owns the start form, plan page limits, capacity errors and crawler access; a second start button would duplicate that logic and its errors. Keeping copy on the server means later SAM and MCP tools say the same thing as the page.
- **Preserve:** No paid action on this page until PR 2, and that one must show its credit cost before the click and use `retry: false`. Never word the score as a promise of citations. Keep empty states honest: an old scan says "scan again", not a low score.
- **Trace:** `CiteFixPage.tsx`, `citeFixApi.ts`, `scoreBand.ts`, `routes/_app/p/$projectId/cite-fix.tsx`, `navigation/items.ts`. Tests: `scoreBand.test.ts`, `navigation/items.test.ts` (18 routes, 12 rows), `frontend/tests/cite-fix.spec.ts` (added to the CI browser job: mocked report at 390 and 1280 px, one free request, no paid call). Not yet rendered against a real production report.
