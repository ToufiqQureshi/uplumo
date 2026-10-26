# Reports and share links: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- **API/trigger:** POST /api/v1/projects/{projectId}/reports/...; public shared-report route mounted in HTTP API
- **Source:** `handler.go`, `service.go`, `repository.go`, `share.go`, `view.go`

## Request or job flow

Private request authorizes project and stores/updates report; rendering escapes content and uses bounded HTML; public route resolves only an enabled share token; disabling a share removes public access.

## Dependencies and cost boundary

Postgres report/template data; optional branding; project auth.

## Manual report persistence and sharing

The frontend sends manually composed, escaped HTML through the existing project-scoped report save endpoint; these reports use the normal report record and do not invoke a model/provider. Client work-log sharing is a second explicit request after persistence. The existing share-state/token path controls public access and revocation.

## Failure, security and data invariants

- Keep report body out of list queries and enforce content size limits.
- Public links require explicit share state and revocation; never expose private IDs as a substitute for auth.
- Render user/agent HTML safely and preserve ownership on update/delete.

## Visual editor persistence

Save accepts optional `definition` (version 1 JSON object). Service validation bounds HTML plus definition to the existing 500,000-byte report cap, and `size_bytes` accounts for both against organization storage. The repository inserts/replaces HTML and definition in the same statement. The get endpoint returns definition only for `includeHtml: true`, under existing session/project authorization. Ordinary metadata/list/public-share queries omit the JSONB column. Frontend validates the full definition schema before use; backend treats it as bounded opaque editor data, never executable HTML.
