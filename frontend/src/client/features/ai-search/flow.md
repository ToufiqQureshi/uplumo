# AI visibility: request flow and paid calls

## Opening any tab: 0 paid calls

Screen → `GET .../ai-search/tracking` (config, runs) → if a run finished,
`GET .../tracking/runs/{runId}` (answers). Overview, Prompts,
Opportunities, Sentiment and Ideas compute everything in the browser from
those answers (`aiVisibilityMetrics.ts`, `insights/*.ts`). Share of voice
comes from the tracked brands in those answers, not from a provider call.

Test a question and Market research read their search from the URL but buy
nothing until the customer submits the form in this page load
(`userRequestedRun.ts`). A reload, bookmark, shared link, redirect from an
old URL or a "recent searches" link only fills the form.

## Changes since the previous check: 0 paid calls

Overview finds the previous run with the same `scopeKey`, reads its saved
answers with `GET .../tracking/runs/{previousId}` and compares
(`visibilityAlerts.ts`): lost and won mentions per question and engine, a
new negative answer, a business AI now recommends that is not tracked, and
sites AI started citing in answers that do not link the customer. Each
change links to `ai-visibility/prompts?q=…&e=…`, which opens that answer.

## Run a tracker check: 1 click = 1 run

Setup → **Review cost** → `POST .../tracking/estimate` (free) → **Run** →
`POST .../tracking/run` with the approved credits and an Idempotency-Key.
The server holds credits once for the whole run, then a worker sends one
DataForSEO task per question per engine:

| Engine             | DataForSEO endpoint                                | Price used                    |
| ------------------ | -------------------------------------------------- | ----------------------------- |
| ChatGPT, Gemini    | `ai_optimization/{engine}/llm_scraper` (standard)  | $0.0012 per answer            |
| Google AI Overview | `serp/google/organic` task (standard, AI Overview) | $0.0012 per answer            |
| Claude, Perplexity | `ai_optimization/{engine}/llm_responses/live`      | hold $0.10, settled at actual |

Example: 20 questions × ChatGPT + Gemini + AI Overview = 60 answers =
$0.072 per check; weekly ≈ $0.31 a month. The hold settles to the real cost.
A second click while a run is queued or running gets `409`.

## Test a question: 1 submit = 1 to 4 answers

Form → `POST .../ai-search/prompt-explorer` → one `llm_responses/live` per
chosen model (ceiling `PromptExplorerCeilingUSD`). Answers are cached per
workspace, prompt, models and country for 7 days; a repeat is free.

## Market research: 1 submit = up to 7 LLM Mentions tasks

Form → cost dialog (about the action's credits) → `POST .../brand-lookup` →
`llm_mentions/target_metrics`, then per platform a mentions sample and top
pages, plus one comparison per platform with competitors. $0.10 per task +
$0.001 per row. Cached 24 h per workspace and inputs; one flight per key.

## AI buttons: up to 15 credits per click

Sentiment **Classify**, Opportunities **Make my action plan**, Prompt ideas
**Suggest questions**: one OpenRouter call each, never retried by the
browser, charged at actual cost up to 15 credits.

Tests: `paid-refetch.test.ts` (no purchase from an opened URL, focus or
reconnect), `tests/ai-tracking.spec.ts` (every tab and old URL, zero paid
requests on load).
