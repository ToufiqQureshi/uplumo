# Wiring, test strength and code size audit (26 October 2026)

The owner's worries: "Frontend pe code hai, backend me logic nahi, ya dono
connected nahi"; "tests error pakadne ki jagah nahi pakadte"; "itne lines,
itne files kyu?" This audit checks each one against the running code. It
removes one dead route; nothing else changes.

## Verdict

- **Wiring: every screen's API call reaches a backend route.** One backend
  route had no caller and was removed.
- **Tests: the money and tenancy tests catch real bugs.** 13 of 13 deliberate
  breaks were caught (9 in the [credits audit](2026-10-25-credits-audit.md),
  2 in the [DataForSEO audit](2026-10-26-dataforseo-audit.md), 2 here).
- **Size: the code is big because the product is big, not because of dead
  code.** Cutting lines now means cutting features, which is the owner's call.

## Wiring (frontend ↔ backend)

Method: built the server from `main`, ran it locally against Docker Postgres
and Redis with no provider keys (no paid call possible), signed in a local
test user, and sent every `/api/v1` path found in `frontend/src` (87 paths,
plus the action names each API module passes to its path builder). A path
with no route answers 404 for both GET and POST; a route answers 200, 400,
405, 409 or 503.

| Result                                | Detail                                                                                                                                                |
| ------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| Frontend call with no backend route   | None. The only 404s were handlers saying "not found" for the placeholder ID `x` (report, draft, author delete), not missing routes.                   |
| Backend route with no frontend caller | `GET /api/v1/config/seo-api-key-status`: no caller, test or doc. **Removed** with its `DataForSEOConfigured` field.                                   |
| Static matching alone                 | Misleads: most routes are registered as `base + action` in Go and built as `` `${base}/${action}` `` in TypeScript, so only a live probe is reliable. |

Not covered: whether each screen shows every field its endpoint returns
(see the [field-delivery audit](2026-10-21-dataforseo-field-delivery.md)),
and MCP-only routes, which have no screen by design.

## Test strength

| Deliberate break                                            | Caught by                                 |
| ----------------------------------------------------------- | ----------------------------------------- |
| Any signed-in user can open any project (`m.user_id` check) | `internal/auth`, `internal/httpapi` tests |
| An archived project stays reachable                         | `internal/auth`, `internal/httpapi` tests |

Both ran against a fresh database per package after a clean baseline
passed. Seen while testing, cause not traced: on this machine, running `auth`
then `httpapi` tests against one database failed on migration 1 ("relation
projects already exists"), while CI runs every package on one database and
passes. A fresh database per package avoided it.

## Code size

- 923 source files (tests and generated files excluded); 21 are over 500
  lines, the largest 842 (`billing/credits.go`). No single file dominates.
- `deadcode -test` and knip run in CI and pass, so unreachable Go functions
  and unused frontend exports are already kept out.
- The size comes from ~40 backend packages for ~28 features
  ([`FOUNDER.md`](../../FOUNDER.md) product map). Pages marked "low vision
  fit" there (Keywords, Reports, the GSC/GA4 mirror) are where real
  reduction would come from: an owner decision, not a refactor.

## Not done

- No line-by-line simplification pass (the `review-simplify` skill) per
  package; worth doing package by package when a feature is next touched.
- Frontend tests were not mutation-checked here beyond billing and AI
  visibility.
