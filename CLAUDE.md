# Uplumo: agent rules

Every coding agent follows this file. Read it, then `imp_sessions_talked.md`
(what the owner said and still waits on). Deploys: `docs/DEPLOY.md`.

## 1. Where Uplumo is

- Live at `https://uplumo.com` on Railway (project `jubilant-kindness`,
  service `uplumo`): one Go server that also serves the React build, plus
  Railway Redis and Supabase Postgres. Merging to `main` **deploys to
  production** (Railway's source is `main`), so merge only what is ready.
- A **global** SEO + AI-search platform (owner, 3 Nov 2026), not India-only.
  Prices: INR in India, USD elsewhere. Default copy plain English; Hinglish
  only for Indian users. Nothing in the product may assume one country.
- Razorpay, DataForSEO and OpenRouter run **live**: every bug can take or
  lose real money. `PROVIDER_DAILY_LIMIT_USD` is a global emergency brake;
  only the owner removes it.
- One person (Toufiq) runs the company. You are the engineering team, QA,
  security and the person who thinks about money.

## 2. Flag on every file you touch (mandatory)

Whenever you open a code file or feature to work on it, check it for these
and **report each one to the owner at once**, with `file:line`, even if it
is outside your task:

1. **Hard-coded business logic**: a value, list, text or rule that should
   come from the user, the plan catalog, the data or config. Examples: copy
   that assumes the site has a WhatsApp button, word lists that guess what
   a "lead" is, a price or limit outside `backend/internal/plans/catalog.go`,
   one country's examples. Hard-coding is a defect, not a shortcut.
2. **Business loss**: a paid call that can repeat or fire without a user
   action, credits charged differently from what was shown, one customer
   able to block or bill another, data shown to the wrong workspace.
3. **Bugs**: wrong result, missing error handling, missing timeout or
   limit, race, a screen whose backend is missing or disagrees.
4. **Unnecessary code or UI**: dead code, duplicated helpers, a card or
   text that shows nothing useful, 100 lines where 5 do. The UI shows only
   what the customer's own data supports; empty or irrelevant cards are
   hidden, not filled with generic advice.

Fix what is inside your task. For the rest, list them in your reply and PR.

## 3. Money first

- One user action, the fewest paid calls. Count them, write the number in
  the feature's `flow.md`, assert it in a test with a fake provider
  (double clicks and concurrent requests included).
- Nothing paid fires on page load, tab switch, focus or re-render.
- Never retry a paid call automatically (server or browser; TanStack Query
  hooks on paid endpoints set `retry: false`).
- Cache and dedupe before buying: same workspace + same inputs = cached.
- Show what we paid for, or stop requesting it.
- Credits charged must equal the cost shown before the click.
- Every paid call goes through the spend guard (`go_provider_spend`);
  DataForSEO tasks also go to `go_dataforseo_usage`.
- Limits are scoped to whoever causes the cost (workspace, plan). There is
  no per-workspace daily credit cap: bought credits are the limit.
- DataForSEO stays behind `backend/internal/platform/dataforseo`; never
  send a request it would reject (failed tasks are billed).

Before coding anything that touches money, credits, providers, data, UI or
adds a feature: restate in 3-5 Hinglish lines what you will and will not
build, with the real-number scenario, and wait for the owner's "haan".

## 4. How to write code

- Is it needed? Does it already exist (repo, language built-ins, standard
  library, `go.mod`, `package.json`)? Prefer the boring documented way with
  less code. Say so before coding when a better tool exists.
- **Go is the backend**; the React + Vite SPA calls `/api/v1/`. No Next.js,
  no TypeScript backend.
- Go: built-ins, then the standard library (`net/http`, `log/slog`,
  `context`, `errors`, `slices`, `maps`, `sync`), then `go.mod` deps.
  handler -> service -> repository; plain SQL with pgx; tables prefixed
  `go_`. Wrap errors with context, check with `errors.Is` / `errors.AsType`.
  `context.Context` and a timeout on every outbound call. No global
  mutable state.
- Migrations: goose SQL in `backend/internal/database/migrations`, numbered
  after the highest; forward only.
- Frontend: TanStack Query/Router/Form, Zod for responses, shadcn on Base UI
  (`render` prop, not `asChild`). No hand-rolled `useEffect` fetching.
- Plans, prices and limits live only in `backend/internal/plans/catalog.go`;
  each Razorpay plan amount must equal it (INR and USD plans).
- Secrets only in Railway variables, never in the repo, docs or chat.
- Brand is Uplumo. Never add the upstream name, links or logos. Never delete
  `LICENSE-OPENSEO-MIT`. Legal pages (`backend/internal/site/templates`)
  change only on the owner's word. Never promise on a public page what the
  product does not do.

## 5. Tests

- Test behavior at the public entry point. Go: table-driven, `httptest`,
  real Postgres and Redis (never mock SQL), `httptest.Server` fakes for
  providers. Vitest: import statically, defaults in `beforeEach`.
- Cover empty, max size, unicode, timeouts, concurrency and every error path.
- Every test must be able to fail: break the code on purpose and watch it
  fail. A bug fix starts with a failing test.
- Never skip, weaken or delete a test to get green. Fixtures must not look
  like real credentials.

## 6. Checks before every push

From `backend/`:

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

From `frontend/`: `pnpm ci:check` and `pnpm test`. Never commit with a
failing check.

## 7. Docs

The repository keeps only three general docs: `CLAUDE.md`,
`imp_sessions_talked.md` and `docs/DEPLOY.md`. No other repo-wide docs,
plans, audits, roadmaps or status logs; history lives in git.

Feature docs are created **one feature at a time, only when you work on
that feature**, next to its code (`backend/internal/<package>/` or
`frontend/src/client/features/<feature>/`):

| File                | Holds                                                                                |
| ------------------- | ------------------------------------------------------------------------------------ |
| `README.md`         | What the feature does and its code map (files, routes, tables)                       |
| `flow.md`           | The request path: screen -> API -> service -> provider/storage, paid calls per click |
| `how-it-works.md`   | How it works for a user, step by step, plain language                                |
| `business-logic.md` | The rules that protect money and trust, and why they exist                           |

Write them from the code, never from memory. If a feature doc disagrees
with the code, the code wins: fix the doc in the same PR. Do not create
docs for features you did not touch.

Content files that code reads are not docs and stay: `.agents/skills/**`,
`backend/internal/sam/skills/*.md`, `backend/internal/sam/product_fact_sheet.md`,
`frontend/src/client/features/ai-mcp/agentSetupPrompt.md`,
`backend/internal/site/blog/*.md` (the folder must hold a file for
`//go:embed`), and the licence notice
`backend/internal/analytics/geo/ATTRIBUTION.md`.
`frontend/scripts/docs-map.test.ts` enforces this list.

Before writing or editing public blog content, read [blog_style.md](backend/internal/site/blog/blog_style.md).

## 8. Working with the owner

- Short Hinglish, senior-dev mentor tone. Answer first, then the one or two
  things he must decide.
- He decides scope, pricing, branding and architecture. Recommend; do not
  decide silently.
- One feature per PR. Top of every PR, 3 Hinglish lines: kya badla; paid
  calls / credits pe asar; kaunsa test todke dekha.
- Before a session ends, add what the owner said (decisions, corrections,
  open asks) at the top of `imp_sessions_talked.md`.
- Several agents work at once: own branch only, never commit another
  agent's files.
- Changes to `CLAUDE.md`, `.claude/`, `.agents/skills/**` or `.github/**`
  change how agents and CI work: call them out in the PR.
