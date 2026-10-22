# dataforseo

The one authenticated transport to DataForSEO. Feature packages build their
own requests and map the results; this package sends them, records what they
cost and makes sure a workspace pays for them.

Flow of `Client.Do(ctx, organizationID, feature, method, path, body, retrySafe)`:

1. Validate the organization, the feature and the path.
2. Ask the spend guard (`Options.Guard`, required) to hold `EstimateUSD(path,
   body)` of the workspace's credits. A refusal stops here: nothing is sent,
   and the error wraps both the guard's error (`spend.ErrNoCredits`,
   `spend.ErrPaused`) and `ErrBillingIssue`.
3. Send the request. Retry only operations explicitly marked `retrySafe`;
   paid, non-idempotent tasks are not retried.
4. Record every task's reported cost in `go_dataforseo_usage`. Charge customer
   credits for successful tasks and 40102 ("No Search Results"), which the
   provider bills after processing. Other task errors, including 40501
   ("Invalid Field"), stay in provider-cost telemetry but are refunded to the
   customer. Uplumo absorbs any provider cost for our invalid request.

`DoUnmetered` is only for free calls that still carry a cost field, such as
collecting a queued task that was charged when it was posted.

Rules that must never break:

- **Every billed request names a feature and goes through `Do`.** A new
  endpoint gets a ceiling in `pricing.go`; until then it is held at one
  dollar per task (`unknownTaskUSD`), so a missing price costs the customer
  a large hold, never Uplumo money.
- **Ceilings are maximums, not averages.** They come from DataForSEO's
  published prices: SERP pages by depth (default depth 10), queued and
  priority rates, the 5x price for search operators, and July 2026 Labs and
  Backlinks rates. Labs and
  Backlinks requests without a `limit` are held as 1,000 rows.
  Labs tasks with `include_clickstream_data: true` hold twice the complete
  ordinary task price, including task and result-row charges.
  `pricing_test.go` pins them.
- Never send a request the provider would reject. If it still fails, refund
  that task's customer credits while retaining the actual provider cost.
- Never return a provider error body or the API key to a client.
- **Every `Do` is its own hold under the action's key.** The hold key comes from
  `spend.ScopedKey(ctx, method, path, body)`, so an action that buys several
  different results is not refused as a replay of its first call, while a
  replayed action is refused call by call and never sent twice.
- **A budgeted operation has a cap of twice `Options.Timeout`** (120s by
  default). Its requests run one after another, so work that is not started in
  time fails and what finished is still settled. `RequestTimeout` (60s) is the
  longest one call may take; the HTTP server's `writeTimeout` in
  `cmd/server/main.go` is derived from it, so a slow provider is answered, not a
  closed connection.
