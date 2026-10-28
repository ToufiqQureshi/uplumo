# SAM project assistant: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Streams model turns, persists bounded history, invokes registered project tools and meters AI usage.

## Decisions to preserve


- **Decision:** Chat `mode: "keywords"` runs the Keywords page agent: only keyword, Search Console, site-reading and saving tools; at most 3 research seeds, 2 SERP queries (depth ≤20) and 1 metric lookup (≤50 keywords) per reply; and a free `present_keywords` tool whose metrics are joined server-side from that reply's research rows. **Why:** one prompt must not buy unbounded DataForSEO tasks, and the screen must never show a number the model typed. **Preserve:** limits are checked at execution, so a guessed or repeated call is refused before it reaches the provider; a counted call stays counted even if it fails (a failed task can still be billed); data ceiling = 3×$0.06 + 2×$0.004 + $0.018 ≈ $0.206 ≈ 265 credits, shown on the page. AI tokens keep the existing $1 per-reply stop. Free plans are refused by the MCP plan gate and have no credits; the page also hides the chat for Free. **Trace:** `keywords_mode.go`, `keywords_mode_test.go`, `chat.go`, `agent.go`, `api/sam.yaml`. Real-provider validation of a full reply not run.

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

## Keep skill sources aligned with shipped capability boundaries

- **Decision:** Align the two `.agents/skills` source copies with the existing embedded competitive-landscape and keyword-research prompts.
- **Why:** Source copies still promised Local SEO/Maps capabilities removed by owner decision, and the embedded-copy regression failed.
- **Preserve:** No runtime prompt expansion or new paid tool; existing shipped prompts remain the capability boundary. This updates agent skill instructions and must be called out in the PR.
- **Trace:** `skills_test.go:TestEmbeddedCopiesMatchTheirSources` and the two source/embedded pairs.
