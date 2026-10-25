# Shared random IDs: decisions

> Internal guide for developers and coding agents. Update with code changes.

## Responsibility

Generates UUID v4 identifiers for Go-owned records.

## Decisions to preserve

- Use Go's cryptographic randomness; never silently fall back to predictable IDs. An operating-system entropy failure terminates the process, as documented by `crypto/rand.Read`. _Why:_ A predictable fallback would compromise identifier security.
- Keep UUID format compatible with existing IDs and database constraints. _Why:_ UI and external clients depend on this contract; changing it without coordinated callers creates avoidable breakage.

## Cost and risk boundary

Go standard-library `uuid`/`crypto/rand`; no external provider or stored state.

## Use the standard UUID generator

- **Decision:** Generate and format IDs with Go 1.27's `uuid.NewV4().String()` instead of setting bytes and formatting UUID groups ourselves. Retain the existing function signature to avoid unrelated caller changes.
- **Why:** The pinned Go toolchain already provides cryptographic version-4 UUIDs; the custom algorithm duplicated it.
- **Preserve:** Lowercase, hyphenated UUID v4 IDs and cryptographic randomness. The error result is nil; both the previous `crypto/rand.Read` and the standard generator terminate on unrecoverable entropy failure. No record, authorization or billing behavior changes.
- **Trace:** `ids.go`, `ids_test.go` (version, variant, format and uniqueness), full Go/Postgres/Redis tests.
