# Analytics: request flow

**Paid calls: 0.** Analytics reads only Uplumo's own tables. No provider is
called on any click, page load or refetch.

## Recording

Visitor's browser → tracker script → `POST /collect` → `Service.Collect`:
rejects bots and hosts that are not the project's domain, rate limits per IP
and site (Redis), hashes the visitor with a daily salt, then writes a page
view (`go_analytics_events`) or an action (`go_analytics_custom_events`).

## Traffic page

1. Page load → `GET .../analytics/summary` (TanStack Query, refetch on focus
   after a minute) → `Service.Summarize` → one Postgres batch.
2. `GET .../analytics/leads` → the saved lead actions.
3. Owner ticks actions and saves → `POST .../analytics/leads` → the set is
   replaced in one transaction → the summary is refetched.

The summary batch: totals, the previous period's visitors and lead visitors,
the daily series, top pages, channels, AI sources, referrers, devices,
actions, campaigns, visit sources, visits by source with their leads
(`sourceLeads`), AI landing pages with their leads (`aiLandings`), sessions,
landing and exit pages, and the daily new/returning audience. Lead queries
run only when the owner has chosen lead actions.
