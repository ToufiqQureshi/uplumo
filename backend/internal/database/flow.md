# Database and migrations: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API/trigger:** Server startup migration path; no product HTTP route.
- **Source:** `database.go`, `migrations/*.sql`

## Request or job flow

Server loads database config, opens a bounded pgx pool and applies ordered goose migrations before serving requests; feature repositories use plain SQL.

## Dependencies and cost boundary

Postgres/pgx and goose migrations.

## Failure, security and data invariants

- Migrations are forward-only and ordered uniquely; duplicate versions can prevent server startup.
- New feature tables use `go_` prefixes and deliberate indexes/FKs/cascades.
- Never use production data in tests; integration tests require explicit TEST_DATABASE_URL.

Migration 42 runs after all earlier schema changes, hardens existing and newly created server tables, and preserves trusted owner access. Supabase client roles receive no table policies. A production emergency application of the same idempotent SQL does not advance Goose history; normal startup still applies versions 37–42 in order.

Known costs after a zero estimate use a zero-amount outcome reservation (migration 43), committed before staff top-up, expiry or customer charging. The existing settlement/reconciler charges available credits and records provider spend once; charging failure cannot turn it into an ordinary refundable hold.
