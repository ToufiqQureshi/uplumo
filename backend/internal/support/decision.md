# Support reports: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Saves customer bug, billing and suggestion reports and notifies the support inbox.

## Decisions to preserve

- Persist first, notify later. _Why:_ a ticket number must mean the report is stored even if Loops is down or unconfigured.
- Client submission UUID plus payload hash makes retries idempotent. _Why:_ ambiguous network failures must not create duplicate tickets, and reusing an ID for different text is a client bug (409).
- Only page pathname, app version and server request ID are collected automatically. _Why:_ query strings, cookies and tokens can carry secrets.
- Screenshot access is limited to the submitter or a staff email that is verified and owns the current workspace. _Why:_ ordinary members of the staff workspace must not read other customers' attachments.
- Screenshots are PNG/JPEG, at most 1 MiB and 16 megapixels, decoded server-side. _Why:_ the browser check is advisory; the server rejects disguised or decompression-bomb images.

## Dependencies

Postgres; auth session; Loops transactional email (`LOOPS_TRANSACTIONAL_SUPPORT_REPORT_ID`).

## Verification

`support_test.go` against `TEST_DATABASE_URL`; frontend caller is `frontend/src/client/features/support/`.
