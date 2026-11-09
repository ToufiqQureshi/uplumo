# Keyword Research (customer side)

`/p/$projectId/keywords` (`routes/_app/p/$projectId/keywords.tsx`) renders
`research/KeywordResearchPage.tsx`: type a keyword, pick a country (default:
the project's market), press **Search**. A toggle switches to **Ask the AI
planner** (`agent/KeywordAgentPage.tsx`, the SAM keywords mode).

| File                           | What it does                                                                 |
| ------------------------------ | ---------------------------------------------------------------------------- |
| `research/KeywordResearchPage` | Search form, result layout, save, free-plan card                             |
| `research/KeywordMetrics`      | Every number one keyword has, grouped; used for the seed and any opened idea |
| `research/KeywordIdeasTable`   | All bought rows: sort, filter, page, select, save; click opens the details   |
| `research/TopResults`          | Google's top 20 for the open keyword, on click only                          |
| `research/researchModel`       | Pure helpers: seed lookup, filters, labels, number formats                   |
| `keywordsApi.ts`               | Zod schemas and calls to `/api/v1/projects/{id}/keywords/...`                |
| `agent/`                       | The AI planner chat and its report view                                      |

Backend: `backend/internal/keywords` (`research.go`, `research_serp.go`,
`saved*.go`). Saved keywords live on `/p/$projectId/saved`.
