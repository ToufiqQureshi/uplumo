# DeepSeek streaming transport: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Streams DeepSeek chat completions and prices them for the spend guard.

## Move AI from OpenRouter to the DeepSeek API

- **Decision:** the owner chose (23 Oct 2026) to replace OpenRouter with the
  DeepSeek API for SAM and the writer, and to self-host a DeepSeek model once
  revenue allows. The package was `platform/openrouter`; it is renamed and
  rewritten for DeepSeek's OpenAI-format API. Default model `deepseek-flash`
  (DeepSeek-V4.1-Flash).
- **Why:** cost. A SAM turn held about $0.40 on `gpt-5.6-luna` at max effort
  and a writer action up to 15 credits on `gpt-4.1-mini`; on `deepseek-flash`
  a typical turn is a few cents and an outline about 2 credits, so customers
  can afford to use the assistant. The owner accepted that DeepSeek processes
  prompts on its servers in China; the privacy page says so.
- **Preserve:**
  - DeepSeek returns no cost, so the client prices usage from `prices.go`.
    Rates change only from DeepSeek's pricing page, with the date noted.
  - A call touching peak hours is priced at peak; a model missing from the
    rate card is priced at the most expensive model. Both can overcharge a
    customer slightly but never leave Uplumo paying more than it charged.
  - Thinking mode stays off for tool-using callers: DeepSeek requires every
    earlier `reasoning_content` back with tools and answers `400` otherwise,
    and SAM's history never returns reasoning to the provider.
  - `CacheScope` is hashed; no Uplumo id is sent to DeepSeek.
  - No automatic retries.
- **Trace:** `deepseek.go`, `prices.go`, `deepseek_test.go` (wire body,
  thinking toggle, cache scope, peak boundaries and weekends, cache-hit
  pricing, unknown model, `BillableUSD` fallback, `MaxCostUSD`). Not yet
  run against the live DeepSeek API from this repository: the first staging
  turn with a real key is the acceptance check.
