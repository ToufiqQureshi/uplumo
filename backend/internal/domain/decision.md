# Domain overview and research: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Validates target scopes and returns organic estimates, ranked terms, pages, competitors and suggestions.

## Decisions to preserve

- Validate target, market and filter budget before spend; failed provider tasks may still be billed. _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.
- Keep scope semantics explicit: domain, subdomain, subfolder and exact URL do not support identical aggregates. _Why:_ A silent fallback can target the wrong market or metric and produce misleading or wasted research.
- Escape user filters for SQL/provider syntax and cap response size. _Why:_ User and provider content or URLs can be hostile; validation and escaping prevent unauthorized exposure or executable content.

## Dependencies

DataForSEO paid endpoints, project market and plan checks.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

## Explicit optional ranking suggestions

- **Decision:** Rank Tracking setup does not buy keyword suggestions on mount. Show an optional Find existing keywords action with credit guidance and a free Skip/manual path. The disabled TanStack query fetches only after that action and never retries automatically; failures show the actual API error without removing the manual path.
- **Why:** A browser trace reproduced a paid Labs suggestions request immediately after saving a tracker configuration. A cache miss costs a real task even when the customer never selected a keyword.
- **Preserve:** Keyword selection starts empty and adding selected keywords remains separate from running rank checks. Supported-country checks, existing plan/credit/spend guards and tenant cache keys remain authoritative. An explicit request uses one ranked-keywords task, not a report fan-out. Backend requires `userInitiated: true` to reject older automatic clients; this marker is compatibility protection, not proof of human action or a replacement for authorization.
- **Decision:** Coalesce identical organization/project/target/scope/market suggestions requests using the full existing cache key, recheck cache inside a bounded 90-second shared operation and respect each caller's cancellation independently. Existing nonempty results retain their 12-hour Redis cache; empty/error results are not persisted as success. Coalescing is instance-local and requires the existing single replica.
- **Trace:** `KeywordSuggestionStep.tsx`, `domainApi.ts`, Go `domain/handler.go`, `suggestions.go`; browser setup/Skip/double-click regressions, payload unit contract, legacy-request test and concurrent provider-count tests. Before the fixes, a legacy paid request bought one task and eight concurrent identical requests bought eight. No live paid provider was called.
