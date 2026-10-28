# Model Context Protocol server: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API/trigger:** POST /mcp; OAuth discovery/registration/authorize/token routes
- **Source:** `mcp.go`, `auth.go`, `registry.go`, `tools.go`

## Request or job flow

MCP client authenticates; transport validates bounded JSON-RPC; registry selects a Go tool; tool authorizes project/plan, calls the shared feature service and returns a schema-constrained response. OAuth uses PKCE and hashed/rotated credentials.

## Dependencies and cost boundary

API keys, OAuth/Postgres token tables, project auth, feature services and provider cost guards.

## Failure, security and data invariants

- Never forward an unregistered call to legacy/other backend code.
- Tool project IDs are untrusted; authorize every call and hide IDs from schemas when server-bound.
- Bound JSON bodies, tool output, call ceilings and OAuth tokens; classify read-only vs spend/mutation tools explicitly.
- `get_cite_fix_report` includes saved mentions, rival comparisons and audit
  issue summaries from its free report path. It does not trigger JavaScript
  rendering, AI mentions refresh or Google AI Overview refresh.

Search growth calls validate dates/brand terms, authorize the project, then reuse
`ga4.Service.GetGrowthReport`, its scoped ten-minute cache and standard/expanded read bounds in `ga4/README.md`.

Plan gating precedes dispatch. Handler panic/error/empty result becomes a safe error result, so SAM does not dereference a nil response or interpret null as success. Internal exception details stay in server logs. No-op timing/quota/activation bookkeeping was removed; actual provider accounting remains in billing.
