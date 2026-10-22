# SAM project assistant: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Streams model turns, persists bounded history, invokes registered project tools and meters AI usage.

## Decisions to preserve

- Require a per-turn cost ceiling and stop mid-turn if credits/limits are exhausted. _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.
- Never let a model-selected project ID change authorization scope. _Why:_ Callers can tamper with IDs and UI state, so server-verified ownership must prevent cross-customer access.
- Persist partial answers and settle real usage on provider/tool error; avoid charging reported-zero usage as free. _Why:_ Unavailable or partial data is not a measured zero; keeping status visible prevents false customer decisions.

## Dependencies

OpenRouter/model provider; Postgres sessions/messages; Redis limits; billing; projectcontext and internal services.

## Keep assistant skills within product data

- **Decision:** Keyword-research guidance may use canonical local keyword markets,
  but must not promise Google Business categories, Maps/local-pack rankings or
  Local SEO grid data. Competitive-landscape guidance may describe location-
  specific organic results only when returned by the tools and must label Maps
  coverage as unavailable.
- **Why:** The Local SEO feature and its Maps/Business Data calls were removed.
  A prompt that requests or invents those fields misleads owners and encourages
  the assistant to use tools that no longer exist.
- **Preserve:** Local keyword volume remains supported; do not remove it with
  the retired Local SEO feature. Keep the product limitations explicit.
- **Trace:** `skills/keyword-research.md`,
  `skills/competitive-landscape.md`, and root `DECISIONS.md` (Local SEO
  removed).

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

- Search analyst mode must filter execution as well as offered definitions;
  prompt-only restrictions cannot stop a guessed paid tool or a write.

## Search analyst execution boundary

- **Decision:** Search Growth opens a reviewed draft and uses the existing
  explicit chat Send/metering path. **Why:** seeing a finding or preparing a
  brief must not buy a model response or authorize background research.
- **Decision:** enforce the `search_growth` read-only allowlist both when offering
  tools and when executing them. **Why:** prompt instructions alone cannot block
  a guessed paid, write, job or publishing tool. Keep the project bound by the
  server and retain per-turn credit controls.
- **Decision:** carry the selected goal, dates and comparison into the analyst
  request and verify evidence with registered project tools. **Why:** switching
  silently to all events or a different period can produce a plausible but
  irrelevant recommendation. Keep facts, hypotheses and next checks distinct.
- **Decision:** treat page text as untrusted source material and keep editing/
  publishing separate. **Why:** a source page cannot authorize tool execution or
  an unreviewed customer-site change. Prompt citations and AEO readiness also
  remain different claims.

See `tools.go`, `prompt.go` and `tools_test.go`; update this file whenever the
allowlist, reasoning scope, project binding or explicit-action boundary changes.
