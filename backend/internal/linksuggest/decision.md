# Internal link suggestions: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Builds bounded suggestions from project GSC pages, falling back to the latest audit, for a draft.

## Decisions to preserve

- No lookup while the user is merely typing; this preserves Google quota and latency. _Why:_ Passive viewing and editing should not create paid work or external side effects without the user asking for them.
- Only suggest pages from the same authorized project/site; cap input, page and segment counts. _Why:_ Callers can tamper with IDs and UI state, so server-verified ownership must prevent cross-customer access.
- Redis outage falls back to free computation; source failures return a safe empty/fallback result. _Why:_ Explicit failure behavior helps callers recover safely instead of treating an error as a valid result.

## Dependencies

Search Console reads, audit tables, Redis cache; no paid SEO provider.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

## Standard-library word splitting

- **Decision:** Use `strings.FieldsFuncSeq` instead of a manual rune-index tokenizer in `text.go`.
- **Why:** The Go standard-library iterator already splits on the existing word-boundary predicate without building an intermediate rune array.
- **Preserve:** Unicode letters/digits/combining marks, first-seen ordering, two-rune minimum, case folding and stopword/deduplication rules. Source bounds, project scope, cache identity and free-only computation remain unchanged.
- **Trace:** `text.go`, Unicode/tokenization and matching regressions in `linksuggest_test.go`, package race tests. No live Google lookup is needed for this local refactor.
