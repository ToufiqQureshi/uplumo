# Database and migrations: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Owns Postgres pool setup, goose migration application and persistent schema evolution.

## Decisions to preserve

- Migrations are forward-only and ordered uniquely; duplicate versions can prevent server startup. _Why:_ Migration and test mistakes can prevent startup or damage customer data, so ordering and isolation must remain explicit.
- New feature tables use `go_` prefixes and deliberate indexes/FKs/cascades. _Why:_ Consistent names prevent table collisions, and indexes/constraints keep ownership and lookup rules enforceable.
- Never use production data in tests; integration tests require explicit TEST\*DATABASE*URL. \_Why:* Migration and test mistakes can prevent startup or damage customer data, so ordering and isolation must remain explicit.

## Dependencies

Postgres/pgx and goose migrations.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

## Search Growth persistence

| Decision | Why | Code / verification |
| --- | --- | --- |
| Keep one expiring report snapshot per project and bounded durable page actions. | Reports are refreshable evidence; saved baselines must survive refresh without unbounded report history. | Migrations 31/32; `ga4/growth_repository.go`, `ga4/README.md` for bounds |
| Freeze the original baseline, selected goal/source and optional comparison page on save; preserve the first applied timestamp on replay. | A retry or later configuration change must not rewrite what a page edit is being compared against. | `ga4/growth_repository.go`, baseline/concurrency regressions |
| Store weekly opt-in and the last dated in-app digest in a project-scoped row; index only due enabled work. | The schedule must be discoverable without scanning every report or storing an unbounded alert history. | Migration 32; `ga4/growth_monitor.go` |
| Claim work atomically and fence writes with a revision. Identical configuration replays preserve the schedule. | Multiple replicas and disable/change races must not duplicate work or publish an outdated digest. | `ga4/growth_business_test.go` claim, replay and in-flight disable tests |
| Use explicit organization/project predicates, deletion cascades and RLS without public-client policies on reports, actions and monitor storage. | Server-owned evidence must not become a public Data API even when default privileges grant table access. Migration 34 closes the reports/actions gap without editing applied migrations; the trusted table owner or BYPASSRLS server role retains scoped API access. | Migrations 32/34; `TestSearchGrowthStorageDeniesPublicRole` reproduces unauthorized reads before the fix, then verifies owner access and restricted-role read/update/insert denial. Repository tenant tests preserve API isolation. |
| Preserve prerequisite migration ordering when combining feature branches. | Applied higher versions must not strand missing lower versions at later startup. Preparatory migrations do not themselves enable another feature's runtime. | `README.md`, migrations 29–32; fresh-database migration tests |
| Describe permanent migration compatibility in the README without PR-specific merge instructions. | A developer or AI needs current startup and write-compatibility constraints; merge sequencing notes become stale after integration. Preserve applied SQL, prerequisite ordering and existing ingestion behavior. | `README.md`, migrations 29/30; documentation-map check. Documentation-only clarification, with no schema or runtime change. |

Changes to these decisions update this file in the same commit. Use disposable
integration databases; real production roles and migrations require the release
checks in the handover. The decision text is not proof of production validation.

## Analytics recognition storage (migration 35)

Decision: add a server-owned site/browser-hash primary-key registry with first-observed timestamps and cascading site deletion; optional browser hashes/page hosts on pageviews; click identity and sanitized destination fields on custom events. Existing writes remain compatible through nullable fields/defaults. Site/browser/time and partial click/time indexes support the reports; existing session/time index supports observed-arrival lookups.

Why: daily salted identifiers cannot establish cross-day returns, while destination association must not inflate pageviews or count duplicate event deliveries. Public collection remains behind the Go validator; no database client policies are added. Browser registry, pageviews and custom events have RLS enabled, PUBLIC privileges revoked, and anon/authenticated privileges revoked where those roles exist, including TRUNCATE privileges that RLS alone would not protect.

Preserve: table owner/BYPASSRLS remains the trusted server role, API tenant checks remain necessary, and server hashes/first-seen records persist with project analytics history until deletion (not a claimed 180-day DB TTL). Goose version ordering stays canonical; production is not used for migration/testing. Trace: migration 35, `analytics_audience_test.go`, analytics integration tests and fresh migration checks. No production migration or role change is performed by opening the PR.

## Provider settlement recovery (migration 41)

- **Decision:** Add a pending-outcome flag and nullable settlement-credit/provider-cost fields to reservations; mark existing open feature holds as uncertain.
- **Why:** Provider side effects cannot be rolled back with a failed Postgres credit transaction. Persist enough evidence to recover a known outcome without making another provider call.
- **Preserve:** Applied migrations remain immutable. No new public database policy, pricing change or retroactive guess about unknown provider cost.
- **Trace:** Migration 41, billing `settlement_recovery_test.go`; fresh migration and real Postgres transaction checks.
