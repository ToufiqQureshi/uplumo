# HTTP routing and middleware: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Mounts public site, auth, API routes, shared session/project guards, tracker collection and SPA fallback.

## Decisions to preserve

- Public routes must be intentionally public; every private API path uses the correct session/project/role guard. _Why:_ This keeps the behavior explicit so future changes do not silently alter customer-visible results.
- Do not bypass middleware when adding a route or mount the same route twice. _Why:_ This keeps the behavior explicit so future changes do not silently alter customer-visible results.
- Keep errors free of secrets/driver details and preserve request deadlines/cancellation. _Why:_ These values or checks protect account access; exposing them or weakening validation can enable account or provider abuse.

## Dependencies

Every feature handler, auth middleware, logging and dependency wiring.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

## Recognition and destination tracker

Decision: `/t.js` accepts optional first-party browser recognition and captures configured click destinations plus outbound links without injecting content or a banner. Random browser IDs expire 180 days from creation and require secure storage/Web Locks; default snippets keep prior anonymous behavior. Host consent revocation clears identifiers and suppresses subsequent collection, including heartbeat/navigation handlers. Storage removal broadcasts revocation to other same-origin tabs. Recognition reuses the existing bounded collector, whose new fields are validated before service access.

Preserve: DNT/GPC before storage/collection, session expiry, hidden-time exclusion, duplicate-event IDs and no external-provider calls. Destinations contain only HTTP(S) origin/path in transit and normalized host/path in storage; unsupported targets remain unknown. Trace: `tracker.js`, frontend `scripts/tracker.test.ts`, mounted analytics route tests and analytics feature decision/handover. Tracker verification uses local origins; no fabricated production analytics traffic.

## Share the Backlinks gate with dashboard refresh

- **Decision:** Pass the same Backlinks plan gate into dashboardoverview route registration.
- **Why:** The refresh was an alternate path to a paid summary and omitted the normal Backlinks feature check.
- **Preserve:** Session and project membership precede the plan/provider path. An unauthorized/Free request buys no task; saved dashboard reads remain accessible.
- **Trace:** `httpapi.go`; `backlinks_test.go` uses real authentication, Postgres plan state and provider-usage recording to check the mounted bypass is refused before another purchase.
