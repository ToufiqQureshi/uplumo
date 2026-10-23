# Dashboard overview

This package serves the dashboard's existing audit and backlink snapshot cards.
Overview reads use only the database; visits never trigger a paid DataForSEO request.
The backlink snapshot is refreshed only by the explicit
`dashboard/backlinks/refresh` call, authorized by the same paid Backlinks gate.

All reads are scoped to the authenticated project and organization. Latest
audits (`go_audits`, `go_audit_issues`) and snapshots use stable newest-first ordering, and stale snapshots are
still returned with an explicit `stale` flag so a provider outage cannot hide
the last known data.

`backlinkRefreshAvailable` tells the UI whether a provider is wired. A missing
provider returns 503 for a requested refresh instead of claiming success.
Missing/stale snapshots expose an explicit refresh button with credit guidance.
The mutation has no automatic retry and shows errors while retaining saved data.

An explicit refresh makes zero provider calls while the project/domain snapshot
is less than 24 hours old; otherwise it buys one summary task, not history.
Concurrent refreshes of the same organization/project/domain share one bounded
90-second operation and re-check freshness inside that operation. This is
instance-local coalescing: keep one server replica until a distributed claim is
implemented. A failed refresh never overwrites the old snapshot or reports
success. Existing provider spend/credit guards still apply.

Tests include real Postgres snapshot persistence/reuse, concurrent exact call
counts, cross-organization access, independent projects, retained stale data and
mounted HTTP plan refusal before provider spend. No live paid provider is used.

The refresh payload must contain `userInitiated: true`; legacy empty requests
from an already-open older SPA are rejected before any plan/provider work. This
is a versioned action contract, not proof of a human click or an authorization
replacement. The current project domain participates in coalescing and snapshot
lookup: a previous domain finishing later cannot hide the current domain's data.
The domain-switch regression uses real Postgres and a blocked provider; malformed
and legacy request tests assert zero provider calls without an earlier plan gate.

Cancellation during cache/project preparation is rechecked before detaching
paid work. Active callers retain cache-outage fallback; existing shared work
can finish for other waiters. See the cancellation decision and regressions.
