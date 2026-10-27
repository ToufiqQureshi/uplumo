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
- Route: `frontend/src/routes/_app/p/$projectId/cite-fix.tsx`. Sidebar entry:
  `frontend/src/client/navigation/items.ts`.

## Rules

- The page reads one free endpoint. It never starts paid work; "Scan my site"
  and "Scan again" link to Site Audit, which owns plan limits and the start
  form.
- Never promise a citation: the score says "easier to quote".
- Fix copy (title, why, how) comes from the API, so SAM, MCP and the page
  cannot drift apart.

## Tests

`scoreBand.test.ts`; the sidebar count is checked in
`client/navigation/items.test.ts`. `frontend/tests/cite-fix.spec.ts`
(Playwright, runs in CI) renders the ready and "scan again" states at 390 and
1280 px and asserts the page sends one `{}` report request and no paid call.
