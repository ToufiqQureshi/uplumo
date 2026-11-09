# Keyword Research: flow

1. Page load reads billing status, the project list (for its market) and
   credit usage. Nothing paid. Free plan: an upgrade card, no form.
2. **Search** (form submit) sets one request `{keyword, locationCode}`.
   `useQuery` keyed on it posts `keywords/research` with `resultLimit: 150`,
   `mode: "auto"`; `retry: false`, no refetch on focus or reconnect, kept 24 h.
   The same keyword and country again is served from the browser cache, and
   the server caches it 24 h per workspace (`backend/internal/keywords/research.go`).
3. The server buys at most: Labs suggestions + Labs ideas (75 rows each), and
   Labs related keywords only when the blend has too few new keywords. Google
   Ads ideas instead for countries Labs does not serve.
4. Every returned row is shown. The seed's row fills the details card; clicking
   an idea swaps it in. Filters, sorting, paging and opening a row use rows
   already bought: 0 paid calls.
5. **Show top results** posts `keywords/serp` (depth 20) for the open keyword,
   once per click, `retry: false`; the server caches it 12 h.
6. **Save** posts `keywords/saved/save` with the row's metrics (no provider call).

## Paid calls per click

| Action                        | Paid calls                             |
| ----------------------------- | -------------------------------------- |
| Open page, type, filter, sort | 0                                      |
| Search (new keyword/country)  | 2 Labs tasks, 3 when the fallback runs |
| Search again within 24 h      | 0                                      |
| Show top results              | 1 SERP task (0 when cached)            |

`tests/keyword-research-spend.spec.ts` checks load, typing, double submit,
reload, filtering and a double click on top results.
