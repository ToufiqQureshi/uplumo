# AI visibility

One page, `/p/$projectId/ai-visibility`, that shows what ChatGPT, Gemini,
Google AI Overview, Perplexity and Claude answer when customers ask the
questions a workspace tracks, and who they recommend. Every number links to
the saved answer behind it.

## Tabs (each is its own route)

| Tab             | Route                         | Component                                | Reads                         |
| --------------- | ----------------------------- | ---------------------------------------- | ----------------------------- |
| Overview        | `ai-visibility/`              | `AiOverviewPage.tsx`                     | latest tracker run (free)     |
| Prompts         | `ai-visibility/prompts`       | `AiPromptsPage.tsx`                      | latest tracker run (free)     |
| Opportunities   | `ai-visibility/opportunities` | `insights/AiOpportunitiesPage.tsx`       | latest tracker run (free)     |
| Sentiment       | `ai-visibility/sentiment`     | `insights/AiSentimentPage.tsx`           | latest run; classify is paid  |
| Test a question | `ai-visibility/test`          | `PromptExplorerPage.tsx`                 | paid, on submit only          |
| Market research | `ai-visibility/research`      | `BrandLookupPage.tsx`                    | paid, on submit only          |
| Prompt ideas    | `ai-visibility/ideas`         | `insights/PromptIdeasPage.tsx`           | AI writer, paid on click      |
| Setup & history | `ai-visibility/setup`         | `PromptTrackingPage.tsx`, `TrackingForm` | tracker config, runs, answers |

The layout route `routes/_app/p/$projectId/ai-visibility.tsx` renders the
header and `AiVisibilityTabs.tsx`. The old addresses (`prompt-tracking`,
`prompt-explorer`, `brand-lookup`, `ai-opportunities`, `ai-sentiment`,
`prompt-ideas`) redirect to their tab and keep their search params.

## Code map

- `aiVisibilityMetrics.ts`: verdict sentence, share of voice, per-engine
  rates, the question × engine matrix, cited domains. Pure functions over
  saved answers.
- `userRequestedRun.ts`: marks a search the customer submitted in this page
  load, so an opened URL never buys anything.
- `trackingApi.ts`, `aiSearchApi.ts`: API clients and Zod schemas.
- `platformLabels.ts`: the one place engine names live (`engineLabel`).

## API (Go, `backend/internal/aisearch`)

- `GET  /api/v1/projects/{id}/ai-search/tracking`: config, runs, metrics
- `GET  /api/v1/projects/{id}/ai-search/tracking/runs/{runId}`: answers
- `POST .../tracking/save`, `.../tracking/estimate`: free
- `POST .../tracking/run`: paid, one credit hold per run
- `POST .../tracking/runs/{runId}/sentiment`: paid (OpenRouter)
- `POST .../ai-search/prompt-explorer`, `.../ai-search/brand-lookup`: paid

Tables: `go_ai_trackers`, `go_ai_tracking_runs`, `go_ai_tracking_answers`.
