# Uplumo: instructions for coding agents

Uplumo is a React/Vite SPA served by a Go API. Postgres stores product data,
Redis caches short-lived results and limits requests, and external SEO/AI
providers can incur charges. Start at [docs/INDEX.md](docs/INDEX.md) for
navigation; confirm behavior in code and tests when documentation disagrees.
Feature-specific README, flow and decision files live beside their code.

## Work safely

- Preserve unrelated changes. Work on a dedicated branch. Never commit secrets.
- Trace a user action through UI, API client, Go handler, service, storage or
  provider, and response before changing it. A page may legitimately need
  several distinct requests; avoid duplicate paid work.
- Project routes require a session and membership in the project's
  organization. Authorize before cache reads or external work. Scope cache
  keys, usage records and job access by organization and every input that
  changes the result. Do not share private data across tenants.
- Validate targets, filters, market and pagination on the server before
  sending a billable provider request. Keep DataForSEO transport through
  `backend/internal/platform/dataforseo`; preserve spend holds, reported
  cost recording and provider error handling.
- Never automatically retry a paid POST after an ambiguous failure. The
  provider may have billed it. Treat client retry policy separately from
  server transport retry. Bound timeouts, concurrency and payloads, propagate
  cancellation, and make background side effects idempotent where required.
  See [cost boundaries](docs/integrations/DATAFORSEO.md).
- Keep HTTP handlers, services and SQL repositories in their existing
  packages. Database changes use numbered goose SQL migrations under
  `backend/internal/database/migrations`; read its README and avoid duplicate
  sequence numbers. Update API YAML in `backend/api` when an applicable
  contract exists, and keep client Zod response contracts aligned.
- Preserve `LICENSE-OPENSEO-MIT` for upstream-derived material. Feature
  status and owner decisions live in `ROADMAP.md` and `DECISIONS.md`.

## Change and verify

1. Read the relevant feature's `README.md`, `flow.md`, `decision.md`,
   handler/service, frontend hook/API client and tests. See
   [docs/FILE_MAP.md](docs/FILE_MAP.md).
2. Reproduce a confirmed bug with a regression test. For provider calls,
   use a fake transport and assert operation-specific call counts, including
   concurrent calls and tenant separation when relevant. Never run live paid
   calls in tests.
3. Update affected feature docs, API contracts and the docs index in the
   same PR. Distinguish implemented behavior, known limitations and plans.
   Document non-obvious security and cost boundaries, not each line of code.
4. Run focused tests, then relevant full checks. From `frontend/`:
   `pnpm ci:check`, `pnpm test:ci`. From `backend/`:
   `go test -p 1 -race -count=1 ./...` with disposable `TEST_DATABASE_URL` and
   `TEST_REDIS_URL` (see [local setup](docs/LOCAL_DEVELOPMENT.md)).
   CI additionally runs vet, staticcheck, golangci-lint, govulncheck,
   deadcode and frontend checks. Record what actually ran.
5. Review the diff for secret exposure, tenant isolation, request
   amplification, migration safety and stale docs. A change is done when
   regression coverage, docs and applicable checks pass, or limitations
   are reported explicitly. Never merge or deploy without authorization.

Production configuration and deployment steps belong to
[docs/DEPLOY.md](docs/DEPLOY.md); do not infer live provider keys or payment
mode from source. Read [docs/security/MODEL.md](docs/security/MODEL.md) and
[docs/testing/REGRESSION.md](docs/testing/REGRESSION.md) for detailed rules.
