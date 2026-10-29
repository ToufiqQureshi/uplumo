# Keywords agent: flow

1. The route `routes/_app/p/$projectId/keywords.tsx` renders
   `agent/KeywordAgentPage.tsx`. `billingStatusQueryOptions` decides Free
   (upgrade card, no requests) or paid.
2. First send: `createSamSession` (free), then `useChat` posts the question to
   `POST /api/v1/projects/{projectId}/sam/sessions/chat` with
   `mode: "keywords"`. The reply streams in the AI SDK UI message format.
3. The server runs the Keywords-mode agent: `research_keywords` (at most 3
   seeds, each one cached auto research), optional `get_serp_results` (at most
   2 queries), free `get_search_console_performance`, then `present_keywords`
   (free).
4. Paid calls per send: 0 to 3 research seeds (2 Labs tasks each on a cache
   miss, a third related-keywords task when the blend is thin, or one Google
   Ads ideas task where Labs has no data; at most 1 seed for a city or region,
   which adds one Google Ads volume task), 0 to 2 SERP pages, 0 to 1 metric
   lookup. A cached seed costs nothing. A refused call over the limit is never
   sent. Credit worst cases per call: `backend/internal/sam/decision.md`.
5. The `tool-present_keywords` part is parsed by `agent/keywordReport.ts` and
   drawn by `KeywordReportView` + `KeywordCharts`. Save all calls
   `keywords/saved/save` (free) with the shown metrics.
6. Failure: a stopped or failed reply shows a message; the user can ask again
   and identical seeds are served from the 24-hour research cache.
