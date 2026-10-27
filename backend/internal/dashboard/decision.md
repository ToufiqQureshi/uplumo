# Dashboard layout: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Stores each member's dashboard card layout per project. It holds no card data.

## Per-member, per-project layouts

- **Decision:** the owner asked (23 Oct 2026) for a dashboard a customer can
  shape: hide cards, add them back and reorder them. The layout is stored in
  `go_dashboard_layouts`, keyed by `(user_id, project_id)`, as an ordered list
  of `{id, hidden}`. This package replaces the unused response-types package
  that lived here (`types.go`, never imported, describing legacy tables).
- **Why:** teammates on one project look for different things (an owner wants
  leads, an SEO wants rankings), so one shared layout would make one of them
  lose. Per project, because a member with several sites wants different cards
  per site. Postgres, not browser storage, so the layout follows the member to
  another device.
- **Preserve:**
  - The server does not know the card list. It checks only the id format,
    duplicates and a 30-card cap; the client drops ids it does not know and
    appends new cards visible. Adding or retiring a card is a client-only
    change, and an old saved layout never hides a new card.
  - An empty save deletes the row ("Reset to default").
  - The user id always comes from the session; project access is checked by
    the shared middleware before the handler runs.
  - No paid work: layout reads and writes touch only this table.
  - RLS is enabled and client roles have no grants, like every table since
    migration 42.
- **Trace:** `layout.go`, `handler.go`, migration `00044_dashboard_layouts.sql`,
  `layout_test.go` (validation table), `httpapi/dashboard_layout_test.go`
  (mounted: signed out, another organization's project, two members kept
  apart, invalid bodies write nothing, reset). A mutation that loosened the
  card cap failed both tests.

## Next Actions decisions live here, shared by the project team

- **Decision:** `go_next_action_states` (migration 46) keeps which Next Actions a project's team marked done or ignored; `POST /api/v1/projects/{projectId}/next-actions/get|set` read and write it. "To do" deletes the row.
- **Why:** The Insights page builds its list from data already stored; only the team's decisions need saving. This package already holds per-project page state, so no new package.
- **Preserve:**
  - Organization comes from the project-access middleware and is matched in every query.
  - Keys match `^[a-z][a-z0-9_:.-]{0,119}$`; at most 500 decisions per project, enforced in the insert under a project row lock; changing an existing key always works.
  - No paid work; RLS on, no client grants.
- **Trace:** `actions.go`, `handler.go`, `httpapi/next_actions_test.go` (signed out, foreign project, bad key/status/extra field, teammate sees and changes a decision, reset, cap refuses a new key but not an existing one). A mutation that lifted the cap failed the test.
