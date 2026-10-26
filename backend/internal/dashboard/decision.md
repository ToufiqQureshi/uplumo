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
