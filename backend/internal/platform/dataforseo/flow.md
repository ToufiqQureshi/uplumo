# Shared DataForSEO flow and the value-accounting boundary

Static review date: 6 November 2026. The documentation PR makes **0 paid calls**, charges **0 credits** and changes no runtime behavior.

## Existing request path

```text
Customer's explicit action, or an authorized scheduled job
  -> feature HTTP handler / service / workspace and plan checks
  -> semantic cache + in-flight dedupe where implemented by that feature
  -> feature provider builds endpoint-specific task payload
  -> shared dataforseo.Client.Do / DoBudgeted (client.go)
  -> spend guard reservation (go_provider_spend)
  -> provider HTTP request; paid requests are not retry-safe
  -> read response and record per-task costs (go_dataforseo_usage)
  -> Results validates task envelope (envelope.go)
  -> feature decodes selected fields into typed models
  -> feature-specific normalized cache / database / analysis
  -> API response -> frontend schema -> UI / details / export
```

Do not infer a universal full-response archive from this transport. `usage.go` records organization, path and cost, not every field returned. Different feature models and persistence paths decide what survives. Do not infer a fixed paid-call count from a screen name: cache hits, fallbacks, pagination, filters, the number of keywords/devices/models and the authorized action matter.

## Reviewed examples

| Feature path | Data consumption / limitation | Paid-call accounting |
| --- | --- | --- |
| `domain/provider.go` -> `domain/service.go` -> domain frontend | Selected organic fields; normalized overview snapshot; paid metrics/lower bands omitted | Relevant cache miss can purchase a Labs endpoint; subsequent features need their own endpoint counts |
| `keywords/research_provider.go` -> research service -> SAM Keywords -> agent report | Richer research model -> up to 20 final recommendations; Ads ideas local truncation | Actual research sources/fallbacks/metric/SERP choices determine purchases; 20 displayed picks is not 20 provider tasks |
| `ranktracking/serp.go` or queued path -> `parseSerp` -> snapshots -> rank frontend | Target position/features + up to 20 organic summaries; hover shows 5 | A live keyword/device check uses a paid SERP task; queued path separates task-setting and retrieval; stop-early already exists |
| `backlinks/provider.go` -> service -> backlinks frontend/export | Selected row/summary fields; CSV carries additional mapped context | Cache reuse/local view can avoid a purchase; uncached wider server queries can buy data again |

For each runtime feature PR, maintain its own `flow.md` with exact calls per click/run, including cache miss, empty result, double click, concurrent work, fallback, and failed/unknown outcome. Use fake-provider call assertions; do not hit DataForSEO to prove this documentation.

## Before changing the flow

Read [README.md](README.md), [business-logic.md](business-logic.md) and the exact endpoint in [reference.md](reference.md). Give the owner the cost/value report and follow `CLAUDE.md` runtime authorization rules. Verify every proposed field is both supported and useful, and identify any field or row lost between response decoding and the customer's actual access path.
