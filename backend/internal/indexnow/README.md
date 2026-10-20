# indexnow

Tells search engines that support IndexNow (Bing, Yandex, Seznam, Naver,
Yep) which pages on a project's website are new or changed, so they crawl
them sooner. **Google does not use IndexNow**: never say on a page that this
speeds up Google. Read this before touching IndexNow, the publish hook in
`sitepublish`, or the `indexnow` plan feature.

## Rules that must never break

- **Basic, Pro and Agency only** (`plans.IndexNow` in
  `backend/internal/plans/catalog.go`). Free gets `402 plan_required` on
  setup, check and submit; `get` answers `included: false` so the screen can
  show the plans link instead of a form.
- **60 IndexNow requests per workspace per clock hour** (`HourlyLimit`,
  Redis key `indexnow:rl:{org}:{YYYYMMDDHH}`). A submission, a key-file check
  and an automatic submission after a publish each count one. Every customer
  submits from Uplumo's servers, so one workspace flooding Bing could get
  those servers rate-limited for everyone; the limit is per workspace so a
  heavy one only stops itself. It fails closed when Redis is down
  (submitting is free, so waiting costs the customer little).
- **Plan, website, key, then limit**, in that order (`Service.ready`): a Free
  workspace is told about the plan, and a refused request never touches
  Redis or the network.
- **One key per project, never replaced.** `CreateKey` inserts with
  `ON CONFLICT DO NOTHING` and re-reads in a separate statement, so
  concurrent first set-ups all get the stored key and the site never hosts a
  key that was replaced. The key is not a secret: it proves ownership only
  because the customer hosts it at `https://{host}/{key}.txt`.
- **Key files are fetched through `audit.Guard`** (no private, loopback or
  metadata addresses; no redirects followed), at most 1 KB read.
- **No automatic retry of a failed submission.** Engines try in order (Bing,
  then Yandex, then Yep) only when one is rate-limited (429), errors (5xx) or
  does not answer, because every engine shares an accepted submission. A 403
  or 422 stops: every engine would refuse the same request.
- **Automatic submission only after a live publish** (`status: publish`), and
  only when the project has set up IndexNow. Setting up the key is how a
  customer turns it on; there is no separate switch. It never fails the
  publish.

## Routes

All `POST`, any project member (as with other project research actions):

| Route | Body | Answer |
| --- | --- | --- |
| `/api/v1/projects/{projectId}/indexnow/get` | `{}` | `{included, domain, key}`; `key` is `""` before set-up |
| `.../indexnow/setup` | `{}` | same; creates the key once |
| `.../indexnow/check` | `{}` | `{files: [{host, status}]}` for the domain and its `www.` host |
| `.../indexnow/submit` | `{urls: [...]}` | `{hosts: [{host, urlCount, status, message}], skipped, skippedCount}` |

Errors: `402 plan_required`, `413 too_large`, `400 no_website`, `409 indexnow_not_set_up`,
`429 rate_limited`, `404 not_found`, `503 indexnow_unavailable`.

## Limits

- 1 to 10,000 URLs per submission (the protocol's limit), each at most 2,048
  characters, and at most 2 MB of request body (about 10,000 typical URLs).
  A larger body answers `413 too_large` asking to split the list; the cap is
  not raised to 10,000 × 2,048 because one request would then hold 20 MB.
- URLs must be http(s) on the project's domain or a subdomain, with no
  user-info or port. Others are skipped; the answer lists the first 20 and
  counts all.
- At most 3 hosts per submission (`maxHosts`), so one call cannot fan out
  into hundreds of outbound requests. URLs on a fourth host are skipped.
- Each key-file check and engine request has 8 seconds; a check or submit
  request has 60 seconds in all (the handler lifts the 30 second write
  deadline). After a publish, IndexNow gets 35 seconds.

## Key-file statuses

`found`, `missing` (404/410), `wrong_content` (200 but not just the key),
`redirects`, `moved` (redirects to the same file on another host, such as a
bare domain forwarding to `www.`), `unreachable`, `unknown` (bot wall,
server error or timeout). `missing`, `wrong_content`, `redirects` and
`moved` stop a submission for that host with a fix-it message; `unknown` and
`unreachable` let it go ahead, because a firewall that blocks Uplumo may
still let the engines in.

## Tables

- `go_indexnow_keys`: `project_id` (primary key, deleted with the project),
  `key` (32 lowercase hex characters, checked by the table), `created_at`.

## Files

`indexnow.go` (service, engines, key-file check), `repository.go` (Postgres
store and Redis limiter), `handler.go` (routes). The publish hook is
`submitToIndexNow` in `backend/internal/sitepublish/handler.go`. The screen
is `frontend/src/client/features/indexnow/`.
