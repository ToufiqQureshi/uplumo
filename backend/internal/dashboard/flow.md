# Dashboard layout: flow

> Internal guide for developers and coding agents. Verify current behavior against source and tests before changing it.

## Entry points

- `POST .../dashboard/layout/get` and `.../layout/save` (`handler.go`).

## Request flow

1. `withSession` resolves the member; `requireProjectAccess` checks the
   project belongs to their active organization (else `404 project_not_found`).
2. The handler decodes a strict JSON body (unknown fields refused, 8 KiB cap).
3. `get` reads the row for `(member, project)`; no row answers `cards: null`.
4. `save` validates, then upserts the row; an empty list deletes it.

The dashboard client merges the saved list with its own card list
(`frontend/src/client/features/dashboard/dashboardLayout.ts`): saved order
first, unknown ids dropped, cards added later appended and shown.

## Cost boundary

No provider calls. One small Postgres read on dashboard open, one write on
Save. Opening the dashboard stays free.

## Failure, security and data invariants

- A layout belongs to the signed-in member: the user id comes from the
  session, never from the request.
- A refused request writes nothing.
- If the layout cannot load, the client shows the default layout; the
  dashboard still works.
- Deleting a user or a project deletes their layouts (`ON DELETE CASCADE`).
