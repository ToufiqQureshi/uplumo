# Insights: flow

> Internal guide for developers and coding agents. Verify against source and tests before changing behavior.

## Page load (free, no provider call)

- Next Actions: `POST audit/history` then `POST audit/results` (latest completed), `POST gsc/search-performance/table` (`dimension=query`, 100 rows; Google API, not billed), `GET ai-search/tracking` and `GET .../runs/{runId}`, `POST next-actions/get`.
- AI Opportunities, AI Sentiment, Prompt Ideas: `GET ai-search/tracking` and the latest finished run's answers.

## Clicks

| Click                               | Request                                                    | Paid calls                                                                                                 |
| ----------------------------------- | ---------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- |
| Any "… with AI" button, Regenerate  | `POST content/assistant/generate` with an insight `action` | 1 OpenRouter call, held at its ceiling (max 15 credits), settled at actual cost                            |
| Mark done / Ignore / Restore        | `POST next-actions/set` `{key, status}`                    | none                                                                                                       |
| Classify answers (Sentiment)        | `POST ai-search/tracking/runs/{runId}/sentiment`           | 0 when no unclassified answer names the brand; else 1 OpenRouter call for up to 20 answers, max 15 credits |
| Add N to the tracker (Prompt Ideas) | `POST ai-search/tracking/save`                             | none now; tracked questions are checked on the tracker's own schedule                                      |

Double clicks: every POST carries an `Idempotency-Key`; the writer and the
sentiment hold use it, and sentiment also takes a per-run advisory lock, so two
clicks never pay twice. No paid request is retried by the browser.

## Failure paths

- No audit / Search Console / tracker: the insight says which data is missing and links to it; the list is built from what exists.
- Writer refuses missing evidence (400) before any credit is held.
- Sentiment: unreadable model output charges 0 (502 `sentiment_unusable`); out of credits is 402 before the model; no OpenRouter key is 503.
