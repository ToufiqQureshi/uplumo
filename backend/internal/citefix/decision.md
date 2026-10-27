# Cite Fix: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Scores how easy a site is for AI assistants to quote, from the latest completed site audit, and ranks the fixes. Free; no provider call.

## Free citation-readiness score from the existing crawl (PR 1 of 5)

- **Decision:** Cite Fix is its own sidebar page and API, but it reads the site audit instead of crawling again. The crawler stores per-page citation signals; this package turns them into a 0-100 score in six weighted categories, a fix list ranked by points gained, and the 50 weakest pages. The owner named the feature "Cite Fix" and asked for it to ship in five PRs: (1) free checks, signals and the page; (2) a paid per-page LLM citability score with cost shown first; (3) citation proof from AI Visibility; (4) one click from a fix into the Content Editor, publish and re-scan; (5) SAM, MCP and report tools. JavaScript rendering is a separate owner decision.
- **Why:** Customers want to know why ChatGPT or Perplexity quote a competitor and not them. Most of the answer is in HTML we already fetch. A second crawler would double the load on customer sites and our server for the same pages. The checks follow the public GEO Optimizer rubric's categories (MIT) and the KDD 2024 GEO study (Aggarwal et al.), whose top methods were adding citations to sources, quotations and statistics; no code was copied. Weights are our own and favour access and direct answers, because an assistant that cannot read a page or find an answer in it never quotes it.
- **Preserve:**
  - No paid call in this package. PR 2 adds the paid score with a cost estimate, credit hold and an exact call-count test.
  - Every query is scoped by workspace through the `projects` join; a foreign project reads as `no_scan`.
  - An audit crawled before migration 47 is `needs_rescan`, never scored as zeros the site did not earn.
  - Blocking training bots (GPTBot, ClaudeBot, Google-Extended) costs no points; only search bots that cite pages count.
  - App shells fail one check (`pages-readable`), not every page check.
  - Freshness is counted in days from the scan day, and judged only on pages with a readable date.
  - Copy never promises a citation.
- **Trace:** `checks.go`, `citefix.go`, `store.go`, `handler.go`; `audit/aireadiness.go`, `audit/pageanalyzer.go`, `audit/repository.go`, migration `00047_audit_citation_signals.sql`. Tests: `citefix_test.go` works every score and gain by hand; `store_test.go` runs a crawl through `audit.Repository` into a report on Postgres, including the workspace boundary, a legacy audit, a running audit and a failed audit; `handler_test.go` covers 400/404/500/503. Not yet run against a live customer site; weights may need tuning once real scores are seen.
