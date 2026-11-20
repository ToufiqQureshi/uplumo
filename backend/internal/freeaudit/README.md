# Free SEO and AEO audit

A public, sign-up-free readiness check of one page, in two engines: the **SEO
audit** (how well Google can crawl, index, read and rank the page) and the
**AEO audit** (how ready the page is for AI assistants to read, trust and
quote). One fresh check fetches the site once and scores it with both. It calls
no paid provider. Neither engine measures rankings, backlinks or AI mentions,
and each report says so.

## Code map

| File | What it does |
| --- | --- |
| `service.go` | `Service.Run(kind)` (fresh or cached report), `Service.Cached(kind)` (cache only, never a check), `ParseKind`. 24 h cache of both reports under `free-audit:v3:`, limits, check slots, singleflight |
| `handler.go` | `GET /free-seo-audit/report?url=&type=seo\|aeo[&cached=1]` (HTML fragment), `GET /free-seo-audit/report.docx?url=&type=` (cached report as Word) |
| `docx.go` | Builds the .docx by hand (zip + WordprocessingML); every page string is XML-escaped; the title says SEO or AEO audit report |
| `../audit/pagecheck.go` | `PageChecker.Check` (one fetch, returns a `PageSnapshot`), shared types, `scoreEngine`, `scoreChecks`, `applyCaps` |
| `../audit/seoengine.go` | `ScoreSEO`: the SEO categories, checks, thresholds and the critical cap (49) |
| `../audit/aeoengine.go` | `ScoreAEO`: the AEO categories, checks, thresholds and the critical cap (39) |
| `../audit/pagecheck_shared.go` | Helpers both engines use (check builder, length, title words, dates, generic link text) |
| `../audit/aireadiness.go`, `citationextras.go`, `pageanalyzer.go`, `technicalsignals.go` | The HTML signals both engines read (JSON-LD, answer capsules, quotes, render-blocking scripts, lazy images, `og:site_name`) |
| `../site/freeaudit.go` | View model for the report fragment, per-engine upgrade offers, FAQs |
| `../site/templates/free-audit.html` | The two audit pages (`/free-seo-audit`, `/free-aeo-audit`) |
| `../site/templates/audit-report.html` | Tabs, form, landing start buttons, report fragment |
| `../site/static/free-audit.js` | Form, progress, SEO/AEO tabs (cached switch), print hook |
| `../site/static/logo-report.png` | Logo for the Word file, rasterized from `logo.svg` (256 px) |

## Routes

- `GET /free-seo-audit` — the SEO audit page (SEO tab selected). Indexed.
- `GET /free-aeo-audit` — the AEO audit page (AEO tab selected). Indexed.
- `GET /free-seo-audit/report?url=&type=seo|aeo` — runs a check (cached 24 h) and returns the fragment for that engine. `type` defaults to `seo`; any other value is a 400.
- `GET /free-seo-audit/report?url=&type=…&cached=1` — the same, but only reads the cache. The tab switch uses it; a missing entry is a 404 and no check runs.
- `GET /free-seo-audit/report.docx?url=&type=` — returns the cached report as a Word file. It never runs a check: 404 "Run the audit first" when the page is not cached.

## The two engines

Each engine has its own categories, weights, checks, severities and caps. The
checks are listed in `business-logic.md` and `how-it-works.md`.

Per check: a severity (critical 8, high 5, medium 3, low 1 weight) and a status
(pass, warn, fail, info). Info is not scored. Found, Why and Fix are written for
the site owner.

## Data

Nothing is stored. Reports live only in Redis for 24 hours (10 minutes when
the page was unreachable). Field data is cached by the CrUX provider.
