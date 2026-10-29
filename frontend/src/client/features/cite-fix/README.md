# Cite Fix (frontend)

The Cite Fix page (`/p/$projectId/cite-fix`, sidebar "Your website" group):
a 0-100 score for how easy the site is for AI assistants to quote, six
category bars, the fixes ranked by points gained, and the 50 weakest pages.
Read this before changing the page or its API client.

Internal guide for developers and coding agents. Read [flow.md](flow.md) for
what happens on screen and [decision.md](decision.md) for why. Scoring rules
live in `backend/internal/citefix/README.md`.

## Source map

- `CiteFixPage.tsx`: the page; states, score card, fix rows, page table.
- `citeFixApi.ts`: `getCiteFixReport` and its Zod schema.
- `scoreBand.ts`: the words and colour beside a score.
- `AgentPanel.tsx`: page checkboxes (top 5 weakest preselected, max 20), Quick/Deep, batch max credits and balance, confirm, one-at-a-time runs.
- `AgentResults.tsx`: saved results; copy blocks; "Open in Content Editor" saves a draft and opens it.
- `ProofCard.tsx`: who AI answers cite (latest AI Visibility run).
- `EvidenceSections.tsx`: saved Brand Lookup mentions, free rival signal
  comparisons (including question-matched mention pages), and Site Audit
  critical/warning issues.
- `CitationSections.tsx`: paid-confirmed Google AI Overview citations.
- `agentSelection.ts`: default selection and batch cap.
- Route: `frontend/src/routes/_app/p/$projectId/cite-fix.tsx`. Sidebar entry:
  `frontend/src/client/navigation/items.ts`.

## Rules

- No paid action runs on Cite Fix page load. Saved mentions reuse AI Visibility
  data at 0 credits and link there to refresh at its existing price.
- Google AI Overview refresh has a confirmation dialog (up to 16 credits for
  10 tracked questions; actual provider cost is charged). Results cache for
  seven days and collection is free. Browser retry is disabled.
- Only "Fix N pages" and a confirmed Google AI Overview refresh spend credits.
  Agent runs use `retry: false`; a credit, plan or rate error stops the batch.
  JavaScript rendering is an opt-in Site Audit setting, capped at 20 app shells
  and at most 39 credits. Rival comparison and “Also fix for Google” are free.
- Never promise a citation: the score says "easier to quote".
- Fix copy (title, why, how) comes from the API, so SAM, MCP and the page
  cannot drift apart.

## Tests

`scoreBand.test.ts`; the sidebar count is checked in
`client/navigation/items.test.ts`. `frontend/tests/cite-fix.spec.ts`
(Playwright, runs in CI) renders the ready and "scan again" states at 390 and
1280 px and asserts the page sends one `{}` report request and no paid call.
It also covers saved mentions, no paid call on load, the explicit Overview
price confirmation and exactly one paid refresh, rival comparisons, and the
free Site Audit issue section. `audit-capabilities.spec.ts` covers the render
opt-in and its displayed ceiling.

`TopicsSection.tsx` shows bounded topic clusters and opens a new blank Content
Editor draft with the chosen missing pillar topic prefilled. `ProofCard.tsx`
reuses `ai-search/components/VisibilityInsights.tsx` for saved citation history.
Neither action starts an AI or provider purchase. New finding evidence is
shown as untrusted text; old signals request a new scan.
