# Protected-site crawler credentials

Encrypts and resolves per-project credentials used when crawling protected customer sites.

This is an internal developer/AI-agent guide, not customer-facing product copy. Read [flow.md](flow.md) for the request path and [decision.md](decision.md) for cost, data and security invariants.

## Source map

- `handler.go`
- `service.go`

## API/trigger

POST /api/v1/projects/{projectId}/crawler-access/save; GET credentials; POST delete

Saving a credential requires an owner/admin role in the active workspace that owns the authorized project. Being an admin in another workspace grants no integration-management authority here.
