# DeepSeek streaming transport: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **Callers:** `internal/sam` (agent turns and history summaries) and
  `internal/contenteditor` (writer actions). No route of its own.
- **Source:** `deepseek.go`, `prices.go`

## Request flow

1. The caller builds a `Request` and holds credits for `MaxCostUSD(req)`
   through the spend guard (`billing.AIMeter`).
2. `Stream` sends `POST /chat/completions` with `stream: true`,
   `stream_options.include_usage`, `thinking` (disabled unless an effort is
   given) and the hashed `user_id`. The response header must arrive within
   60 seconds; the body may stream for minutes.
3. Text, reasoning and tool-call deltas go to the caller's `Handler` as they
   arrive. Usage rides on the last chunk.
4. `Stream` prices the usage (`priceUsage`), peak if the call started or
   ended in peak hours, and returns it in `Usage.CostUSD`.
5. The caller settles the hold with `Usage.BillableUSD(model, chars)`: the
   priced cost, or, if the stream broke before usage arrived, an estimate at
   the model's peak rates.

## Dependencies and cost boundary

DeepSeek API (`api.deepseek.com`), billed per token to Uplumo's account. Every
call goes through the spend guard into `go_provider_spend`; this package never
calls DeepSeek without a caller-held hold.

## Failure, security and data invariants

- A non-2xx answer returns `StatusError` with at most 4 KiB of the body; a
  mid-stream `error` object returns `ErrStreamError` with the partial result,
  so the caller can still bill what streamed.
- No API key, provider body or Uplumo id is logged or sent to the client.
- Lines over 1 MiB end the stream with an error instead of growing memory.
