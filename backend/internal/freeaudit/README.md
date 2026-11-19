# Free SEO and AEO audit

A public, sign-up-free audit of one page, at `/free-seo-audit` (form also on
the landing page). It scores the page for Google and AI search and suggests
the paid tools that fix what it found.

## Code map

| File | Holds |
| --- | --- |
| `freeaudit/service.go` | Cache (Redis, 24 h; 10 min for unreachable pages), per-visitor and global limits, 4 checks at a time, one check per page at once (singleflight) |
| `freeaudit/handler.go` | `GET /free-seo-audit/report?url=` (no session), error messages |
| `audit/pagecheck.go` | `PageChecker`: fetches the page, robots.txt, llms.txt and sitemap; 28 checks in 4 categories; scoring |
| `site/freeaudit.go` | Report view, upgrade cards priced from `plans/catalog.go`, page FAQ |
| `site/templates/free-audit.html`, `audit-report.html` | Page, form, report fragment |
| `site/static/free-audit.js` | Runs the check for `?url=` and inserts the report |

Tables: none. Redis keys: `free-audit:v1:<hash>` (report), `emailauth:rl:free-audit:*` (limits).
