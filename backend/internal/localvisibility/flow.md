# Flow

Page load: `overview` reads the business, the newest run of each tab, the
watch and `PriceList()`. **0 paid calls.** A running run is polled with
`runs/get` (a database read, 0 paid calls) every 4 seconds until it ends.

| Click | Paid calls (DataForSEO priority queue unless noted) | Most it costs |
| --- | --- | --- |
| Search (find business) | 1 Maps live search, depth 20 | 3 credits |
| Run profile check | 1 Maps task, then 4 profile + 4 posts tasks (you and the top 3) | 40 credits |
| Scan 3 x 3 / 5 x 5 | 9 / 25 Maps tasks, depth 20 | 14 / 39 credits |
| Analyse reviews | 1 reviews task (100) + up to 3 (50 each), then 1 AI call for themes | 58 credits |
| Turn on watch | per competitor: 1 profile + 1 posts task, now and every 7 days | 10 per competitor |
| Draft a reply / Rewrite | 1 AI call | 1 credit |

The credits are `billing.CreditsForUSD` of the exact task bodies
(`dataforseo.EstimateUSD`) plus the AI ceilings; `TestPriceListMatchesTheTasksEachActionPosts`
pins them.

## Request path

1. `runs/start` -> `Service.Start`: needs a saved business; same input finished
   in the last 24 hours -> that run, `cached: true`, nothing posted.
2. `Store.CreateRun`: a unique index allows one running run per project and
   kind, so a double click or second tab gets the same run.
3. `postAll` posts the first tasks. Each post carries the idempotency key
   `local:<run>:<task>` and the task id is stored at once. A spend-guard
   refusal (no credits, paused) stops before the next task; with nothing
   posted the run fails with the guard's message and the request answers 402.
4. A `local_visibility` job runs `Process`: `task_get` (free) every 15 s; for a
   check, the profile tasks are posted once the Maps result names the top 3.
   After 10 minutes whatever is missing is listed in `missing`.
5. The result is stored on the run; failed when nothing came back.
6. `Tick` (every 15 min): due watches start, runs whose job was lost finish.

Reviews: themes are one AI call over at most 48 KB of review text; the model
returns review ids only and the server counts them.
