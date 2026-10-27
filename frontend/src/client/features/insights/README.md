# Insights: Next Actions, AI Opportunities, AI Sentiment, AI Prompt Ideas

Four pages that turn data Uplumo already stored into a decision and an
action. Each page has the same shape: an **insight** sentence, the
**numbers behind it** with their source, and **AI buttons** that do the
work on the page (credits shown before the click).

Read [flow.md](flow.md) for what each click calls and costs, and
[decision.md](decision.md) for the rules that protect money and trust.

## Pages

| Route                            | Page                      | Reads (free)                                                                                                               | Paid on click                                |
| -------------------------------- | ------------------------- | -------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------- |
| `/p/$projectId/next-actions`     | `NextActionsPage.tsx`     | latest completed audit, Search Console query table (100 rows, 28 days), latest AI tracker check, saved done/ignored states | AI writer action per item                    |
| `/p/$projectId/ai-opportunities` | `AiOpportunitiesPage.tsx` | latest AI tracker check                                                                                                    | action plan; page outline                    |
| `/p/$projectId/ai-sentiment`     | `AiSentimentPage.tsx`     | latest AI tracker check with saved sentiment                                                                               | classify answers; Google post                |
| `/p/$projectId/prompt-ideas`     | `PromptIdeasPage.tsx`     | tracker configuration                                                                                                      | prompt ideas (saving to the tracker is free) |

## Code map

- `AiAction.tsx`: `AiAction` (one paid writer action, `retry: false`, new
  idempotency key per click, Copy/Regenerate) and `InsightCard`.
- `nextActions.ts`: `buildNextActions`, the ranked list (pure). Click
  estimates use a fixed CTR-by-position table and are always shown as "≈".
- `opportunities.ts`: `promptGaps` (questions where AI named someone else).
  The "why not cited" reasons are `ai-search/whyNotCited.ts`, reused.
- `sentiment.ts`: `sentimentSummary`, `topComplaint`.
- `promptIdeas.ts`: `parsePromptIdeas`, `addPrompts` (tracker limit 30).
- `insightsApi.ts`: Next Actions states and the sentiment classify call.
- `useLatestAnswers.ts`: tracker state plus its latest finished check.

Backend: `internal/dashboard/actions.go` (states), `internal/aisearch/tracking_sentiment.go`
(classification), `internal/contenteditor/writer.go` (the five insight writer actions).
Tests: `insights.test.ts`.
