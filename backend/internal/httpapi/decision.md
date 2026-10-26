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

## Static files must be readable by the runtime user

- **Decision:** the `Dockerfile` copies the built app into `/dist` owned by the
  image's runtime user (65532, distroless `nonroot`).
- **Why:** live on 23 Oct 2026, `/logo.svg` and `/favicon.ico` answered `403`
  from the Go origin (Railway request id, `text/plain`, 14 bytes) while
  `/site/logo.svg` and hashed `/assets/*` loaded. `spaHandler` serves files with
  `http.ServeFileFS`, which answers 403 on a permission error. Files built inside
  the image are world-readable; files Vite copies from `frontend/src/public` keep
  the mode they arrived with from the deploy upload, which can be owner-only
  for root. The app logo was a broken image on every page.
- **Preserve:** `spaHandler` keeps serving only files under `STATIC_DIR`; this
  changes ownership, not which paths are public.
- **Trace:** root `Dockerfile`. Not reproducible in Go tests (permissions come
  from the image); verify after deploy that `/logo.svg` and `/favicon.ico`
  return 200.

## Recognition and destination tracker

Decision: `/t.js` accepts optional first-party browser recognition and captures configured click destinations plus outbound links without injecting content or a banner. Random browser IDs expire 180 days from creation and require secure storage/Web Locks; default snippets keep prior anonymous behavior. Host consent revocation clears identifiers and suppresses subsequent collection, including heartbeat/navigation handlers. Storage removal broadcasts revocation to other same-origin tabs. Recognition reuses the existing bounded collector, whose new fields are validated before service access.

Preserve: DNT/GPC before storage/collection, session expiry, hidden-time exclusion, duplicate-event IDs and no external-provider calls. Destinations contain only HTTP(S) origin/path in transit and normalized host/path in storage; unsupported targets remain unknown. Trace: `tracker.js`, frontend `scripts/tracker.test.ts`, mounted analytics route tests and analytics feature decision/handover. Tracker verification uses local origins; no fabricated production analytics traffic.

## Share the Backlinks gate with dashboard refresh

- **Decision:** Pass the same Backlinks plan gate into dashboardoverview route registration.
- **Why:** The refresh was an alternate path to a paid summary and omitted the normal Backlinks feature check.
- **Preserve:** Session and project membership precede the plan/provider path. An unauthorized/Free request buys no task; saved dashboard reads remain accessible.
- **Trace:** `httpapi.go`; `backlinks_test.go` uses real authentication, Postgres plan state and provider-usage recording to check the mounted bypass is refused before another purchase.

The mounted plan regression sends the explicit `userInitiated: true` contract, so its 402 proves the plan gate, rather than failing at request validation.

## Mounted checkout verifies provider pricing

- **Decision:** Update the real-session/Postgres billing HTTP fixture to serve the provider plan read before subscription creation.
- **Why:** Checkout now refuses unverified/mismatched terms before making a plan payable. A fake subscription response cannot stand in for a plan response.
- **Preserve:** Owner-only checkout, HMAC webhook validation, duplicate delivery behavior, internal/provider error distinction and read-only billing behavior.
- **Trace:** `billing_test.go`, `TestBillingEndToEnd`; full mounted suite required before release.

## Remove the unused DataForSEO key status route

- **Decision:** Delete `GET /api/v1/config/seo-api-key-status` and the `DataForSEOConfigured` field it read.
- **Why:** The wiring audit (`docs/audits/2026-10-26-wiring-tests-size-audit.md`) found no frontend caller, test or doc for it; paid features already answer 503 without the key.
- **Preserve:** `content-writer-status` and the other config routes are unchanged; no paid path read this field.
- **Trace:** `httpapi.go`, `cmd/server/main.go`; `internal/httpapi` and `cmd/server` tests pass against Postgres.
