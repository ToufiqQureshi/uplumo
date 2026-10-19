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
| Use explicit organization/project predicates, project/organization deletion cascades and RLS without public-client policies on monitor storage. | Server-owned evidence must not become a public Data API or survive deletion of its owning scope. The trusted migration/server owner remains responsible for scoped access. | Migration 32, repository tenant tests; verify the production database role before release |
| Preserve prerequisite migration ordering when combining feature branches. | Applied higher versions must not strand missing lower versions at later startup. Preparatory migrations do not themselves enable another feature's runtime. | `README.md`, migrations 29–32; fresh-database migration tests |

Changes to these decisions update this file in the same commit. Use disposable
integration databases; real production roles and migrations require the release
checks in the handover. The decision text is not proof of production validation.
