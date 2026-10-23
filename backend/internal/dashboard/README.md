# Dashboard layout

Stores each member's dashboard layout per project: which cards show and in
what order. Read it before changing the dashboard's Customize panel or the
`go_dashboard_layouts` table. Card data itself comes from each feature's own
API; this package never reads it.

Read [flow.md](flow.md) for the request path and [decision.md](decision.md)
for the invariants.

## Source map

- `layout.go`: `Card`, `Validate`, `Store.Get`, `Store.Save`.
- `handler.go`: the two routes, mounted from `internal/httpapi`.
- Table: `go_dashboard_layouts` (migration `00044`), primary key
  `(user_id, project_id)`, `cards` is a JSON array of `{id, hidden}`.

## API

Both need a session and access to the project (`requireProjectAccess`).

| Route | Body | Answer |
| --- | --- | --- |
| `POST /api/v1/projects/{projectId}/dashboard/layout/get` | `{}` | `{"cards": [...]}` or `{"cards": null}` when none is saved |
| `POST /api/v1/projects/{projectId}/dashboard/layout/save` | `{"cards": [...]}` | the saved cards; `[]` deletes the saved layout |

Limits: body 8 KiB, at most `MaxCards` (30) cards, ids match
`^[a-z][a-z0-9_]{0,39}$`, no duplicate ids. Anything else is `400
invalid_request` and nothing is written.
