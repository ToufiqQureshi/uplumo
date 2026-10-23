# DeepSeek streaming transport

Streams chat completions, tool-call deltas and token usage from the DeepSeek
API (OpenAI format, `https://api.deepseek.com`) and prices every response.
SAM and the content editor's writer call it; read this before changing either
model call or anything that charges AI credits.

Read [flow.md](flow.md) for the request path and [decision.md](decision.md)
for the cost and data invariants.

## Source map

- `deepseek.go`: `Client.Stream`, `Request`, `Usage`, `MaxCostUSD`,
  `Usage.BillableUSD`.
- `prices.go`: the rate card (`offPeak`), `IsPeak`, `priceUsage`.
- `deepseek_test.go`: wire format, pricing, peak hours, failures.

## Rules that must never break

- **Nothing is retried here.** A retry re-bills; callers decide.
- **Every answer has a price.** DeepSeek reports tokens, not money, so
  `Stream` fills `Usage.CostUSD` from the rate card. Cache-hit, cache-miss and
  output tokens are priced separately; a model missing from the rate card is
  priced as `deepseek-v4-pro`, the most expensive.
- **Peak or off-peak.** Peak hours are 01:00-04:00 and 06:00-10:00 UTC,
  Monday to Friday; peak rates are double. A call that starts or ends in peak
  hours is priced at peak, so Uplumo is never billed more than it charged.
  Chinese public holidays are off-peak at DeepSeek but priced peak here.
- **Holds use the ceiling.** `MaxCostUSD` counts every request byte as a
  third of a token, none as cache hits, plus the full `MaxTokens`, at peak.
- **Thinking mode is off unless asked.** `ReasoningEffort` empty or `none`
  sends `thinking: disabled`. With tools, thinking mode requires every earlier
  reasoning text to be sent back (else `400`); SAM never sends reasoning back,
  so it keeps thinking off.
- **Cache isolation.** `CacheScope` (the workspace id) is hashed into
  DeepSeek's `user_id`, so one customer's prompt cache is not shared with
  another and no Uplumo id leaves the server.

## Configuration

`DEEPSEEK_API_KEY` (required for SAM and the writer), `DEEPSEEK_MODEL`
(optional SAM model, default `deepseek-flash`). The writer always uses
`deepseek-flash`. Rates checked against
https://api-docs.deepseek.com/quick_start/pricing on 23 Oct 2026.
