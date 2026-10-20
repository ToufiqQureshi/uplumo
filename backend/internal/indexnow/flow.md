# IndexNow: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API:** `POST /api/v1/projects/{projectId}/indexnow/{get,setup,check,submit}` (`handler.go`)
- **Hook:** a live WordPress publish, `sitepublish` `handler.publish` → `submitToIndexNow` → `Service.SubmitPublished`
- **Source:** `indexnow.go`, `repository.go`, `handler.go`

## Request flow

1. **Set up:** member clicks "Set up IndexNow" → plan check → project domain
   → `CreateKey` (once per project) → the screen shows the key file to
   download and the URL it must open at.
2. **Check:** plan → domain → key → hourly limit → `GET https://{host}/{key}.txt`
   for the domain and its `www.` host through `audit.Guard`.
3. **Submit:** plan → domain → key → hourly limit → URLs grouped by host (up
   to 3 hosts) → per host: key-file check, then one `POST` to Bing; Yandex,
   then Yep, only when the previous engine answered 429, 5xx or nothing.
4. **After a live publish:** the same as submit with the one published URL.
   Not set up, not on the plan, or no website: nothing happens and the
   publish answer has no `indexNow` field.

## Outbound calls per user action

IndexNow is free: no provider bill, no credits, nothing in
`go_provider_spend`. Calls are still counted because they leave from Uplumo's
shared servers.

| Action | Key-file fetches | Engine requests | Limiter counts |
| --- | --- | --- | --- |
| Get, set up | 0 | 0 | 0 |
| Check | 2 (1 for a `www.` domain) | 0 | 1 |
| Submit N URLs on H hosts (H ≤ 3) | H | H, up to 3H when engines are down | 1 |
| Live publish, set up | 1 | 1, up to 3 | 1 |
| Refused (plan, website, key, limit) | 0 | 0 | 0 (1 when the limit refuses) |

`TestSubmitGroupsByHostAndSendsOneRequestPerHost`,
`TestSubmitTriesTheNextEngineOnlyWhenOneIsUnavailable` and
`TestSubmitRefusesBeforeAnyOutboundCall` assert these counts.

## Dependencies

Postgres (`projects.domain`, `go_indexnow_keys`), Redis (hourly limit),
billing feature gate (`plans.IndexNow`), customers' websites, IndexNow
engines.

## Failure paths

- Redis down → `500` for check/submit; after a publish, the publish
  succeeds and shows "error on our side".
- Billing lookup down → `500`; nothing is sent.
- Engines down → `failed` with "IndexNow did not respond".
- Customer site blocks Uplumo → key-file status `unknown`; the submission
  still goes ahead.
