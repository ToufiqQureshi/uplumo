# Shared random IDs

Generates UUID v4 identifiers for Go-owned records with Go 1.27's standard-library `uuid.NewV4`.

Internal package guide: [flow.md](flow.md) describes the code path; [decision.md](decision.md) records invariants to preserve.

## Source map

- `ids.go`
- `ids_test.go`
