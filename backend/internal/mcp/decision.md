# Model Context Protocol server: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Authenticates API-key/OAuth callers and exposes registered Go tools with per-tool project and plan checks.

## Decisions to preserve

- Never forward an unregistered call to legacy/other backend code. _Why:_ This keeps the behavior explicit so future changes do not silently alter customer-visible results.
- Tool project IDs are untrusted; authorize every call and hide IDs from schemas when server-bound. _Why:_ Callers can tamper with IDs and UI state, so server-verified ownership must prevent cross-customer access.
- Bound JSON bodies, tool output, call ceilings and OAuth tokens; classify read-only vs spend/mutation tools explicitly. _Why:_ These values or checks protect account access; exposing them or weakening validation can enable account or provider abuse.

## Dependencies

API keys, OAuth/Postgres token tables, project auth, feature services and provider cost guards.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

- Classify Search Growth as GoogleData and read-only. Its result contains
  evidence and candidates, not query-level conversion attribution or causal claims.

## Search Growth tool contract

- **Decision:** reuse `ga4.Service.GetGrowthReport` from the registered read-only
  `get_search_growth_report` tool, including goal/comparison/coverage inputs.
  **Why:** the browser and analyst must use the same validation, source semantics,
  cache identity and quota bounds rather than separate inference logic.
  **Trace:** `ga4_tools.go`, GA4 `decision.md` and `README.md`.
- **Decision:** authorize the project before dispatch, retain GoogleData feature
  classification and bind SAM's active project server-side. **Why:** a model or
  external MCP client cannot grant itself another workspace's data access.
- **Decision:** make expanded coverage explicit in the schema and describe its
  larger free-Google read bound. **Why:** read-only does not mean unlimited quota;
  schema changes must not silently turn a routine report into deeper research.
- **Decision:** expose only registered capabilities and report source limitations
  as evidence. **Why:** a UI drill-down or external report link is not proof that
  an equivalent MCP tool/API integration exists. Do not claim automatic native
  AI-report ingestion, causal attribution or query-level conversions.

Update this decision file with tool-schema, access or cost-bound changes, and
verify schema validation and project binding through the existing MCP/SAM tests.

## Use standard scope membership

- **Decision:** Replace the manual `containsScope` loop with standard-library `slices.Contains`.
- **Why:** Exact slice membership is already provided by Go; a wrapper adds no policy.
- **Preserve:** Exact case-sensitive scope checks, token authorization and tool cost gates remain unchanged.
- **Trace:** `oauth.go` and existing MCP/OAuth tests.

## Contain failures and remove unwired telemetry

- **Decision:** Return one validated non-nil tool result; use named panic recovery and safe unexpected-error messages. Remove the unused outcome struct and no-op telemetry call.
- **Why:** Recovery previously returned nil, which crashed SAM's runner or returned a null external result; unexpected errors exposed raw implementation/provider details. The telemetry fields were never persisted or consumed.
- **Preserve:** Explicit tool classification, plan gates before execution, safe app/spend messages, and actual provider ledger accounting. No activation/usage event is claimed for the removed no-op.
- **Trace:** Registry, runner, transport and `TestRunnerContainsInternalFailures` covering panic, error and missing response.

Successful external tool calls now persist the first authorized/call timestamps in organization activation state with a two-second best-effort write; internal SAM and failed calls do not complete onboarding. COALESCE preserves the first evidence across repeats. `activation_test.go` verifies external/internal/failure boundaries on real Postgres.

## Search Console text leads with the property total

- **Decision:** `get_search_console_performance` puts the property total
  (clicks, impressions, CTR, "includes anonymized queries") as the first text
  line whenever the gsc service returns `totals`.
- **Why:** models quote the first numbers they see; summing query rows gave
  SAM a wrong total (0 instead of 3). See `backend/internal/gsc/decision.md`.
- **Preserve:** structured `rows` are unchanged; `totals` and `totalsNote`
  are additional fields.
- **Trace:** `gsc_tools.go`.
