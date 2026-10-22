# Uplumo: agent guide

Every coding agent (Claude, Codex, Cursor, others) follows this file.
`AGENTS.md` only points here.

## 0. Where Uplumo is today

- Live at `https://uplumo.com` on Railway, project `jubilant-kindness`,
  service `uplumo` (one Go server that also serves the React build, plus
  Railway Redis and Supabase Postgres). Merging to `main` does **not**
  deploy: see `docs/DEPLOY.md`.
- Razorpay runs with **live** keys: customers pay real money. A billing,
  credit or plan bug now takes or loses real money, so treat every billing
  change as a finance change (section 3).
- `DATAFORSEO_API_KEY` and `OPENROUTER_API_KEY` **are set in production**.
  Every DataForSEO task and every OpenRouter token is billed to Uplumo for
  real, so a wasted call is lost money (section 3, "Money first").
- `PROVIDER_DAILY_LIMIT_USD` is set as a global emergency brake. It stays
  until the known paid-call bugs are fixed; only the owner removes it.
- One person runs the company. You are the engineering team, QA, security,
  SRE and the person who thinks about money: act like it (section 3).

The current status of every feature is in `ROADMAP.md`; check it and the
code before you trust this section.

## 1. What to read, and when

| You are about to...                      | Read first                                                               |
| ---------------------------------------- | ------------------------------------------------------------------------ |
| Do anything                              | This file, then the latest week in `docs/WEEKLY.md`                      |
| Choose, scope or prioritise a feature    | `docs/PRODUCT.md`, `ROADMAP.md`                                          |
| Check what the owner already decided     | `DECISIONS.md` (decisions, Railway variable names, open asks)            |
| Touch a feature's code                   | Its `README.md` → `flow.md` → `decision.md`, then tests and API contract |
| Add or change a table                    | `backend/internal/database/README.md`                                    |
| Add or change an HTTP endpoint           | `backend/api/<feature>.yaml` if it exists, the feature README            |
| Run it locally                           | `docs/LOCAL_DEVELOPMENT.md`                                              |
| Deploy, or change a production setting   | `docs/DEPLOY.md`                                                         |
| Change email (Loops)                     | `backend/internal/email/README.md`                                       |
| Hit repo friction (flaky script, gotcha) | Log it in `.agents/PAPERCUTS.md` with the `papercuts` skill              |
| Touch anything that calls DataForSEO     | `docs/integrations/DATAFORSEO.md` (cost boundaries)                      |
| Touch auth, tenancy or caching           | `docs/security/MODEL.md`                                                 |
| Write or review tests                    | `docs/testing/REGRESSION.md`                                             |

If a doc and the code disagree, the code is the truth: fix the doc in the same
change.

### Docs map

`docs/INDEX.md` lists every repo-wide document (the docs-map test checks it),
and `docs/FILE_MAP.md` maps source and test files. Feature docs stay next to
their code: each `backend/internal/<package>/` and
`frontend/src/client/features/<feature>/` has `README.md`, `flow.md` and
`decision.md`. Read only the docs for the feature being changed and its direct
dependencies. Keep special references such as
`backend/internal/analytics/geo/ATTRIBUTION.md` (licence) and
`backend/internal/sam/product_fact_sheet.md` (SAM's approved product facts)
intact. Skills (`.agents/skills/`, mirrored in `.claude/skills/`) are listed
by the harness and are not repeated here.

## 2. Layout

```
backend/    Go server. cmd/server (the site), cmd/erase-user (privacy erasure),
            internal/<feature>/, internal/platform/ (shared pieces), api/ (OpenAPI).
frontend/   React + Vite SPA (pnpm). Built to frontend/dist and served by the Go server.
docs/       PRODUCT.md, DEPLOY.md, LOCAL_DEVELOPMENT.md, audits/.
Dockerfile, railway.json    one image: the Go server plus frontend/dist.
```

Migrations are goose SQL in `backend/internal/database/migrations`, applied by
the server at startup. Number a new one after the highest that exists; two
files with the same number stop the server from starting.

Every Go package and frontend feature folder has three internal docs next to
its code. `README.md` is the short entry point and code map; `flow.md` traces
the user action/request through APIs, providers, storage and the response;
`decision.md` records why the design exists and which cost, security and data
invariants must survive. Read all three before editing that feature, then its
tests and API contract. Update the affected docs in the same PR, using the code
as truth when docs disagree. Do not repeat every implementation detail: point
to the source files and explain behavior, dependencies, failure paths and why.

## 3. How to write code

### Before you write a line

1. **Is it needed?** Build what a real customer or a real failure needs,
   nothing for an imagined future. No unused parameters, options, layers,
   files or "just in case" branches. Under-engineering is a defect too: a
   missing timeout, limit, check or test is not "simple".
2. **Does it already exist?** Look in this repo, then language built-ins and
   standard libraries, then `go.mod` and `package.json`. Keep those layers
   distinct: Go predeclared functions/types such as `len`, `append` and `make`
   need no import; `net/http` and `context` are standard-library packages;
   `pgx` and TanStack are installed dependencies. Match docs to the exact
   versions in `go.mod` and `pnpm-lock.yaml`; live docs can describe newer APIs.
   Read current docs:
   Go release notes (`go.dev/doc/go1.27`, e.g. `errors.AsType`,
   `sync.WaitGroup.Go`, `testing.T.Context`, `slices`, `maps`,
   `strings.SplitSeq`, the `uuid` package) and `go doc`; for the frontend the
   React 19, TanStack Query/Router/Form, Zod 4, Base UI and Tailwind 4 docs.
   Search the web for the established way when you are not sure. A
   hand-rolled copy of something that exists is a defect.
3. **Is this the right way?** Prefer the boring, documented approach. If two
   ways work, pick the one with less code and fewer moving parts, and say
   why in the PR.
4. **What can it cost us?** For every change that touches money, data,
   providers or other customers, write down the scenarios and how the code
   stops each one from losing money or trust, then test them:
   - a bug or loop calls a paid provider again and again;
   - one customer's heavy use slows, blocks or bills another;
   - a provider, Postgres or Redis is down, slow or answers garbage;
   - the same request, webhook or job runs twice (retries, double clicks);
   - two requests race for the same row, credit or limit;
   - a user reaches another workspace's data;
   - a public page promises something the product does not do.
     Put the list, and what you did about each, in the PR.
5. **Is every line production-ready?** Errors wrapped and handled, inputs
   validated, timeouts and limits on every outside call, idempotent where it
   can repeat, logged where an on-call person needs it, and a way back if it
   goes wrong (migrations only move forward, so design for that).

Then delete every line that does not earn its place.

### Money first: no bug may cost the owner money

DataForSEO and OpenRouter are live and billed per call. Customers have
already seen one click buy many tasks and drain their credits, and Uplumo
has paid for provider work on a Pro workspace that was within its limits.
Treat any change that can add, repeat or widen a paid call as a finance
change.

- **One user action, the fewest paid calls.** Count the provider calls each
  click makes, write the number in the feature's `flow.md`, and assert that
  exact count in a test with a fake provider (including double clicks and
  concurrent requests). Nothing paid fires on page load, tab switch, focus
  or re-render without an explicit user action or an approved schedule.
- **Never retry a paid call automatically.** Not on the server, not in the
  browser: TanStack Query hooks that reach a paid endpoint set
  `retry: false`. A failed response does not prove the provider did not
  bill it.
- **Cache and dedupe before buying.** Same workspace, same inputs: serve the
  cached result. Concurrent identical requests share one provider call.
- **Show what we paid for.** If the provider returns a field the customer
  paid for, show it or stop requesting it. Dropping paid data on the way to
  the screen is a bug.
- **The customer pays what we showed them.** Never charge more credits than
  the cost shown before the action without the owner's approval.
- **Every paid call is recorded.** DataForSEO tasks go to
  `go_dataforseo_usage` (workspace, endpoint, cost); every provider's spend,
  OpenRouter included, goes through the spend guard into `go_provider_spend`
  (workspace, feature, cost). The owner must be able to answer "where did
  this dollar go" with one query; a paid call outside the spend guard is a
  bug.

### Mistakes we already made (do not repeat them)

- **One shared limit for every customer.** A global $25/day stop meant one
  heavy workspace paused all the others, and they would leave. A limit that
  protects Uplumo must be scoped to whoever causes the cost (per workspace,
  per plan); a global stop is only an emergency brake, set far above normal
  use. Before you ship any limit, work it with real numbers: "5 customers on
  Pro, one uses 10x the others: who gets blocked, when, and what do they see?"
- **Building before recommending.** Writing code first and explaining later
  hides the decision from the owner. When there is an established tool or
  pattern (TanStack Query for server state instead of `useEffect` fetches,
  `errors.AsType`, a provider's own limit or webhook), say so **before**
  coding, with the trade-off, and let the owner choose.
- **Rules that look safe but punish paying users.** A check that fails
  closed is right; a check that blocks a paying customer for something they
  did not do is a bug. Ask "who does this stop, and is that the person who
  caused the problem?"

**Go is the backend.** There is no TypeScript backend: the React app is a
static SPA that the Go server serves and that calls `/api/v1/`. No Next.js.

- **Go:** use language built-ins, then the standard library (`net/http`,
  `log/slog`, `context`, `encoding/json`, `errors`, `slices`, `maps`,
  `strings`, `sync`, `time`, `testing`, `net/http/httptest`), then dependencies
  already in `go.mod`. Hand-rolling a standard-library feature is a defect.
  Check `go doc` before writing a helper. A new module needs a reason in the PR.
- **Layering:** handler -> service -> repository. Handlers validate every
  input and map errors to HTTP. Services hold the rules. Repositories hold
  plain SQL (pgx, no ORM). New tables are prefixed `go_`.
- **Errors:** never ignore one. Wrap with context
  (`fmt.Errorf("load project %s: %w", id, err)`), check with `errors.Is` and
  `errors.AsType`. No panics for control flow.
- **Context and time:** pass `context.Context` through every request path and
  set a timeout on every outbound call.
- **No global mutable state.** Make concurrency explicit (`sync`, channels)
  and race-free.
- **Simple, flat, readable.** Delete any line that does not earn its place.
  Abstract only when it prevents real drift.
- **Data:** Postgres is the primary store, normalized. Redis holds rate limits
  and short-lived caches. Secrets live in Railway variables, never in the repo
  or in chat.
- **External data:** DataForSEO stays behind `backend/internal/platform/dataforseo`.
  Never send a request the provider would reject, because failed tasks are billed.
- **Plans:** prices, limits and what each plan includes live in one place,
  `backend/internal/plans/catalog.go`. The pricing page, the Billing screen
  and every plan check read it; never copy a price or a limit anywhere else.
  The amount on each Razorpay plan must equal the catalog's price.
- **Brand:** the product is Uplumo. Never add the upstream name, links,
  logos or copy. The only mention is the license attribution
  (`LICENSE-OPENSEO-MIT`, never delete it).
- **Copy:** plain language, Hinglish where the market is India, no raw jargon
  without a one-line explanation. Never promise on a public page what the
  product does not do.

**Frontend** (`frontend/src/client` and `frontend/src/routes`): React with
Vite, TanStack Query/Router/Form patterns, shadcn on Base UI (compose with the
`render` prop, not `asChild`). React hooks are built-in React APIs; routing,
server-state caching and forms come from the installed TanStack packages. Use
TanStack Query for server state instead of hand-rolled `useEffect` fetch/cache
logic. Screens call the Go API under `/api/v1/` through `apiRequest` and
validate responses with Zod.

## 4. How to write tests

A test exists to **catch a bug**, not to turn CI green.

1. **Test behavior at the public entry point.** Go: table-driven, `httptest`
   for handlers, a real Postgres and Redis for repositories (never mock SQL),
   an in-process fake (`httptest.Server`) for outside providers.
2. **Cover what can actually happen:** empty, nil, max size, unicode, timeouts,
   concurrency, and every error path (including the provider failing, the cache
   being down, the caller going away).
3. **Every test must be able to fail.** After writing tests, break the code
   under test on purpose (flip a condition, drop a check) and confirm a test
   fails. A mutation that survives means a missing test: add it.
4. **A failing test is a real bug until proven otherwise.** Find out whether
   the code or the test is wrong, and fix the cause. Never skip, weaken,
   delete or loosen a test to get green.
5. **Bug fix:** write the test that reproduces it first, watch it fail, then fix.
6. A negative test must fail for the reason its name gives, not because an
   earlier guard rejected first.
7. Test fixtures must not look like real credentials (secret scanners flag
   them, and rewriting history is the only cure).
8. Vitest (frontend): import the module under test statically, set default mock
   values in `beforeEach` only, keep fixtures to the fields asserted on, and do
   not export a function only so a test can reach it (knip fails on it).

## 5. Checks before every push

Go, from `backend/` (all must pass; CI runs the same against a real Postgres
and Redis):

```sh
go mod tidy && git diff --exit-code -- go.mod go.sum
test -z "$(gofmt -l .)"
go vet ./...
staticcheck ./...
golangci-lint run ./...
TEST_DATABASE_URL=... TEST_REDIS_URL=... go test -p 1 -race -count=1 -cover ./...
govulncheck ./...
test -z "$(deadcode -test ./...)"
```

Frontend, from `frontend/`: `pnpm ci:check` (prettier, knip, tsc, oxlint) and
`pnpm test`. CI runs both.

Do not commit with a failing check, and do not commit secrets.

## 6. Working with the owner

The owner (Toufiq) decides product scope, pricing, branding and architecture.
Propose with a recommendation; do not decide silently.

- Speak up first. If you see a better tool, a risk, or a cheaper way, say it
  before you build, even when nobody asked. Silence about a known problem is
  a defect.
- Anything that touches money, limits, pricing or every customer at once:
  post the plan and the real-number scenario to the owner and wait for a yes
  before coding.

- Keep answers short, in Hinglish, senior-dev tone. No long essays.
- Before planning a new feature, search the web for current competitor
  features, pricing and user complaints, and cite the sources in the PR.
- Ship each feature as its own PR. The description says what changed, the
  decisions the owner must confirm, how it was validated and what was _not_ run.
- Several agents work in this repository at once. Work on your own branch,
  and never commit another agent's uncommitted files.
- Changes to `CLAUDE.md`, `AGENTS.md`, `.claude/`, `.agents/skills/**` and
  `.github/**` change how agents and CI work. Call them out in the PR.
- Merging to `main` does not deploy. See `docs/DEPLOY.md`.

## 7. Documentation rules

- A feature's notes live in its own `README.md`, next to the code.
- `docs/` holds only what covers the whole product: the product brief, how to
  run it and how to deploy it. Dated reviews go in `docs/audits/`.
- Do not keep plans, migration history or status logs as documents. Status
  lives in `ROADMAP.md`, history lives in git.
- The exceptions are `DECISIONS.md` and `docs/WEEKLY.md`. `DECISIONS.md`
  holds the owner's decisions, which Railway variables are set (names only)
  and what waits on the owner; update it in the same change when one of those
  changes. `docs/WEEKLY.md` holds one short summary per week (below).
- Never put secrets in a document.

### Keep this file and every feature's docs true

- **`CLAUDE.md` is maintained, not frozen.** When a change makes any line
  here untrue (production state, a provider, a rule, a check, the layout),
  fix that line in the same PR. Section 0 must always match production; the
  owner's word in chat beats this file, so update the file when they correct it.
- **Feature docs are read against the code.** Before you edit a feature's
  `README.md`, `flow.md` or `decision.md`, read the code it describes. Never
  write a doc from memory, another doc or a PR title. If you find a doc that
  contradicts the code, fix it even when it is outside your task, and say so
  in the PR.
- **Every change records its decision** in the feature's `decision.md`
  (next section). There is no change too small for this.

### Weekly summary (`docs/WEEKLY.md`)

The Saturday workflow should add one section to the top of `docs/WEEKLY.md`
covering Saturday to Friday, in a PR of its own. Read the latest week first:
it is the fastest way to learn what moved. The schedule lives outside this
repository; if it is not active, prepare the weekly update manually.

- Build it from `git log origin/main` (merged PRs) and the `decision.md`
  and `DECISIONS.md` changes of that week, never from memory or chat.
- Per week, at most about 40 lines: what shipped (PR numbers), decisions
  the owner took, money or provider-cost changes, what broke or was
  reverted, open owner asks, and which docs the week changed.
- Keep the last 8 weeks; delete older sections (git keeps them).
- A week with no merges gets one line saying so.

### Mandatory decision record for every change

For **every task that changes repository files**, update the relevant feature or
package `decision.md` **in the same commit**. This includes UI/UX, calculations,
APIs, database/schema, jobs, provider/cost controls, configuration, tests,
refactors and documentation changes. It is part of finishing the task, not an
optional follow-up. Read the current decisions before editing code.

Each entry must make sense to a developer or AI with no conversation history:

- **Decision:** what was chosen or changed, and the affected user behavior/scope.
- **Why:** the problem or evidence behind it, and any meaningful trade-off or
  alternative rejected. Do not invent alternatives just to fill a template.
- **Preserve:** the data, money, authorization, UX and failure invariants that
  future edits must respect. For a behavior-preserving refactor or docs/test
  change, explain its purpose and which constraints remain in force.
- **Trace:** point to the implementing files/contracts and the relevant
  verification. State limitations and unfinished real-provider validation.

Update or replace an existing decision when it is superseded; do not append a
contradiction or a generic "docs updated" line. Keep exact operational values in
one canonical document and link to it. Update each affected layer's decision file
when a change crosses frontend/backend boundaries; use links instead of copying
one explanation across unrelated features. If a feature lacks `decision.md`,
create it next to its README/flow and check the documentation map rules.

For repository-wide rules or owner preferences with no feature folder, update
root `DECISIONS.md`. It records why this documentation rule exists; `CLAUDE.md`
is the authoritative instruction. Changes to this rule update both files.

Before committing, check the diff against the affected decision files. A change
without its relevant decision update is unfinished. Summarize material decisions
and changes to agent rules in the PR description; a handover TXT or chat message
does not replace the maintained decision records.

**Docs change with the code, in the same commit.** A PR that changes
behaviour, a rule, an env var, a limit, an endpoint or a table also updates
the README of that feature (and `ROADMAP.md`, `docs/DEPLOY.md` or the Docs
map when they are affected). A stale doc is a bug: the next agent trusts it.
Before pushing, ask "which doc says something this diff just made untrue?"

**Write docs for an agent that has never seen the code:**

- Lead with what the thing does and when to read the doc, in one or two
  sentences.
- Then the rules that must never break, the flow, env vars and edge cases,
  as short bullets or tables, not essays.
- Say **why** a rule exists; the code already says what it does.
- Name exact files, functions, tables, routes and env vars, so they can be
  searched. Give real numbers (limits, timeouts, prices), not "a few".
- One fact in one place; link to it instead of copying it.
- No history, dates or "we changed X" in a README; git keeps that.

## 8. Legal

`LICENSE` is proprietary (all rights reserved) for Uplumo's own work.
Upstream-derived code stays under MIT. Never delete `LICENSE-OPENSEO-MIT` or
the copyright notice it carries.

The public legal pages (`backend/internal/site/templates`) name the operator,
address and support contact. Change them only on the owner's word.
