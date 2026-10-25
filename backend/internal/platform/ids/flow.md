# Shared random IDs: flow

> Internal guide for developers and coding agents.

## Current behavior

A repository/service requests a new ID; Go's `uuid.NewV4().String()` supplies cryptographic randomness, UUID v4/variant bits and canonical formatting. The existing `(string, error)` signature remains for callers; its error is nil.

## Dependencies

Go 1.27 standard-library `uuid` (backed by `crypto/rand`); no external provider or stored state.

## Relevant code

- `ids.go`
- `ids_test.go`
