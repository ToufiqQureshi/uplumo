# Analytics

Uplumo's own privacy-friendly visit tracking. A small script on the
customer's site sends page views and actions; the Traffic page shows leads,
visitors, sources, pages and AI-assistant visits.

## Code map

| Part | File |
| --- | --- |
| Tracker script | `backend/internal/httpapi/tracker.js` |
| Collect endpoint, summary, leads, clicks, countries handlers | `handler.go` |
| Recording and summarizing | `service.go`, `repository.go` |
| Lead actions chosen by the owner | `leads.go` |
| Channel and AI-assistant classification | `classify.go` |
| Click destinations | `journey.go` |
| Country lookup | `geo/` |
| Screen | `frontend/src/client/features/analytics/` |

## Routes

| Route | What |
| --- | --- |
| `POST /collect` | Tracker events (public, rate limited per IP and site) |
| `POST /api/v1/projects/{id}/analytics/site` | Create or return the site and its snippet |
| `GET /api/v1/projects/{id}/analytics/summary?from&to` | The period's summary |
| `GET /api/v1/projects/{id}/analytics/leads` | Lead actions the owner chose |
| `POST /api/v1/projects/{id}/analytics/leads` | Replace them (`{"names": [...]}`) |
| `GET /api/v1/projects/{id}/analytics/clicks` | Button and link destinations |
| `GET /api/v1/analytics/{siteId}/countries` | Visitors by country |

## Tables

`go_analytics_sites`, `go_analytics_events` (page views),
`go_analytics_custom_events` (actions), `go_analytics_browsers`,
`go_analytics_lead_events` (the owner's lead actions).
