# Uplumo

Live at <https://uplumo.com>. Uplumo tells a business whether AI assistants
(ChatGPT, Google AI Overviews, Gemini, Perplexity, Claude) cite its site, who
they cite instead, and why not, with what to fix. Around that: site audits,
rank tracking, keyword and competitor research, analytics with AI-referral
detection, and white-label reports. Vision and priorities: `FOUNDER.md`.

Proprietary software. See [`LICENSE`](./LICENSE).

## Where things live

| Path                     | What                                                            |
| ------------------------ | --------------------------------------------------------------- |
| `backend/`               | Go server: API, public pages, analytics ingest, billing, MCP    |
| `frontend/`              | React + TypeScript app (Vite, static SPA)                       |
| `docs/`                  | Product brief, local setup, deploy guide, dated audits          |
| `CLAUDE.md`, `AGENTS.md` | Rules for coding agents and contributors                        |
| `FOUNDER.md`             | Owner's vision, worries, priorities and the product map         |
| `ROADMAP.md`             | What is built, what is next, what the owner still has to decide |

Each feature has its own `README.md` next to its code, under
`backend/internal/<feature>/` and `frontend/src/client/features/<feature>/`.

## Development

- Start with the [engineering docs index](./docs/INDEX.md) and [source map](./docs/FILE_MAP.md).
- Run it locally: [`docs/LOCAL_DEVELOPMENT.md`](./docs/LOCAL_DEVELOPMENT.md)
- Server settings and routes: [`backend/README.md`](./backend/README.md)
- Deploy: [`docs/DEPLOY.md`](./docs/DEPLOY.md)

## Hosting

Production runs on Railway: one Go server that also serves the built React
app, with Railway Redis and a Supabase Postgres database. The root `Dockerfile` builds both. Cloudflare
sits in front for DNS and forwards `support@uplumo.com`. Payments go through
Razorpay, transactional email through Loops, SEO data through DataForSEO on
Uplumo's own account (customers never bring their own keys).

Secrets live in Railway variables, never in this repository.
