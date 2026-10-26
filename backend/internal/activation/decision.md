# Dashboard activation state: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Provides project-scoped dashboard progress, first-click timestamps and dismissible setup state.

## Decisions to preserve

- Scope every read by project organization and every dismissal by user. _Why:_ This keeps the behavior explicit so future changes do not silently alter customer-visible results.
- First-click timestamps are write-once; repeated clicks must not rewrite onboarding history. _Why:_ Retries, replays and parallel requests are normal; idempotency prevents duplicate charges, records and side effects.
- This package records interaction state only; it must not trigger provider refreshes. _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.

## Dependencies

Postgres activation state; dashboard UI.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

## The audit step reads go_audits

- **Decision:** `hasAudit` checks `go_audits`, the table the audit feature writes, instead of the legacy `audits` table.
- **Why:** Nothing writes `audits` any more, so "Audit your site" stayed open on the dashboard after finished crawls (seen on the owner's project 26 Oct). The dashboard audit card had the same bug, fixed in #197.
- **Preserve:** Any audit run on the project, whatever its status, completes the step; other steps and dismissals are unchanged.
- **Trace:** `repository.go`; `TestDashboardActivationSeesAuditsFromTheAuditFeature` runs the real query on migrated Postgres and failed before the fix.
