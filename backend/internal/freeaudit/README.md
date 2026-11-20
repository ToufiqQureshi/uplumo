# Free SEO and AEO audit

A public, sign-up-free readiness check of one page. It reads the page, its
`robots.txt`, `llms.txt` and sitemap, and (when a CrUX key is set) Chrome's
real-visitor field data. It calls no paid provider. The score says how ready
the page is for Google and AI search; it does not measure rankings, backlinks
or AI mentions, and the report says so.

## Code map

| File | What it does |
| --- | --- |
| `service.go` | `Service.Run` (fresh or cached report), `Service.Cached` (cache only, never a check), 24 h cache under `free-audit:v2:`, limits, check slots, singleflight |
| `handler.go` | `GET /free-seo-audit/report?url=` (HTML fragment), `GET /free-seo-audit/report.docx?url=` (cached report as Word) |
| `docx.go` | Builds the .docx by hand (zip + WordprocessingML); every page string is XML-escaped |
| `../audit/pagecheck.go` | `PageChecker`, fetch, scoring, caps (`scoreChecks`, `applyCaps`) |
| `../audit/pagecheck_rules.go` | The individual checks, thresholds, severities |
| `../site/freeaudit.go` | View model for the report fragment, grades, upgrade offers |
| `../site/templates/audit-report.html` | Report fragment (score, caps, Download PDF and Word buttons) |
| `../site/static/free-audit.js` | Form, progress, print hook (`beforeprint` opens every check) |
| `../site/static/logo-report.png` | Logo for the Word file, rasterized from `logo.svg` (256 px) |

## Routes

- `GET /free-seo-audit` — the page (site).
- `GET /free-seo-audit/report?url=` — runs a check (cached 24 h) and returns the fragment.
- `GET /free-seo-audit/report.docx?url=` — returns the cached report as a Word file. It never runs a check: 404 "Run the audit first" when the page is not cached.

## Checks

Each check has a severity (critical 8, high 5, medium 3, low 1 weight) and a
status (pass, warn, fail, info). The full list is in `pagecheck_rules.go`.
Scoring and caps are described in `business-logic.md`.

## Data

Nothing is stored. Reports live only in Redis for 24 hours (10 minutes when
the page was unreachable). Field data is cached by the CrUX provider.
