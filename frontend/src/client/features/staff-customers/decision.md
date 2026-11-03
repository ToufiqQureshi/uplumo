# Staff customers: decisions

- One staff page instead of an external admin tool. _Why:_ the owner needs onboarding answers for marketing now; the data is already in Postgres and this needs no new service or credentials.
- Hidden link, server-enforced access. _Why:_ hiding the link is not security; `backend/internal/customers/decision.md` holds the access rule.
- Search runs in the browser over the full list. _Why:_ user counts are small; move to server paging when the list grows past a few thousand rows.
- Saved interests are a JSON array and are shown as a comma list (`interestsText`).
