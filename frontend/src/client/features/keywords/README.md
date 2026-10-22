# Keyword research and saved keywords (frontend)

## What this does

Pages `/p/$projectId/keywords` (research) and `/p/$projectId/saved` (saved
keywords), plus the search-performance and domain-overview "save keyword"
actions. All data comes from the Go keywords API.

## Flow

- Research: the controls form builds a request (`keywords`, optional
  `locationCode`/`locationName`, `resultLimit`, `mode`, `clickstream`,
  `groupKeywords`) and the query cache keys on every field that changes the
  result, including the local area. Every research call is billed, so queries
  never auto-retry.
- SERP panel: a shallow top-20 snapshot by default; paging past it re-buys a
  depth-100 snapshot one level deeper than the current one. A failed deep
  fetch drops back to the cached shallow snapshot instead of re-billing on
  every retry.
  Successful snapshots stay fresh and retained in TanStack Query for 12 hours,
  matching the server cache; revisiting the same keyword does not refetch it.
- A missing exact seed shows an approximate-match notice in every mode,
  including Auto. Exact matching trims whitespace and ignores case.
- When research limited ideas to a place named in the seed (`placeFilter`),
  an info notice says so and tells the customer to search without the place
  name for wider ideas. No extra request is made.
- The "Google shows" column turns each row's `serpFeatures` into up to three
  badges (Maps, AI, Snippet, Q&A, Video), each with a tooltip saying what to do
  about it (`components/SerpFeatureBadges.tsx`). The data comes with the
  research task; showing it buys nothing.
- Research CSV/Sheets exports include both top bids, referring domains,
  quarterly change and monthly-search history alongside the core metrics.
- SERP rows and exports retain the description already present in the API.
  No additional lookup is bought for this field.
- Saved keywords: filtered/sorted/paged list (page sizes 50/100/250), bulk
  remove (max 2000 ids), tag assign (append/replace), tag rename/recolor, and
  a delete that fails with `tag_in_use` while any keyword still carries the
  tag. Refresh re-buys metrics for every saved keyword.
- Export CSV/Sheets: runs the same list filters through `saved/export` and
  hands the rows to the shared exporter.

## Go endpoints

`POST /api/v1/projects/{projectId}/keywords/...` — `research`, `serp`,
`saved/refresh|save|list|export|remove`, `saved/tags/assign|update|delete`.
Contract: `backend/api/keywords.yaml`; response shapes in
`keywordsApi.ts` (Zod) mirror `backend/internal/keywords`.

## Rules that must never break

- Response validation through `apiRequest` + Zod: a drifted server shape is an
  "unexpected answer" error, never a half-rendered table.
- Never sneak a billed refetch past the user (retries off for research/serp;
  refresh and SERP depth changes are explicit).
- Playwright runs research against `VITE_E2E_KEYWORD_FIXTURES` canned rows,
  never against DataForSEO.
- The fixture use for e2e and tests stays opt-in; production builds call the
  API.
