# IndexNow screen: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **Screens:** `search-performance/SearchPerformancePage.tsx` (under the report, with or without Search Console) and `routes/_app/p/$projectId/settings/integrations.tsx` → `IndexNowCard` (one component, one cached query)
- **Source:** `IndexNowCard.tsx`, `indexNowApi.ts`

## Flow

1. Opening the page runs one query, `indexnow/get` (no outbound call, no
   limiter count).
2. "Set up IndexNow" → `indexnow/setup`; the answer replaces the cached setup.
3. "Download key file" builds `{key}.txt` in the browser; nothing is sent.
4. "Check key file" → `indexnow/check` (counts against the hourly limit).
5. "Submit pages" → `indexnow/submit` with the pasted URLs split on
   whitespace.

Mutations never retry (TanStack Query's default for mutations); the query
retries only network errors and 5xx (`shouldRetryApiError`).

## Failure paths

Server messages are shown as they come (`plan_required`, `rate_limited`,
`no_website`, ...) through `getStandardErrorMessage`.
