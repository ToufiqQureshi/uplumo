# Cite Fix: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Scores how easy a site is for AI assistants to quote, from the latest completed site audit, and ranks the fixes. Free; no provider call.

## Money ceiling and reference repositories (28 Oct 2026)

- **Decision:** Cite Fix's analysis stays rule-based and free; AI runs only
  on paid, priced, confirmed clicks. The worst-case cost per action, the
  scenarios where Uplumo could still lose money, the proposed site-wide AI
  analysis and the open-source projects that informed the checks (with
  licences) are recorded in
  [docs/audits/2026-10-28-cite-fix-costs-and-references.md](../../../docs/audits/2026-10-28-cite-fix-costs-and-references.md).
- **Why:** the owner asked whether whole-site AI analysis would cost him money
  on a ₹2,499 Pro plan (10,000 credits). At the agent's price caps a full Pro
  month is at most about $6.10 of provider cost; only free or uncapped AI loses
  money.
- **Preserve:** no free AI call; credits = provider USD × 1.28 × 1000; no code
  from repositories without a licence (geo-checker); a ported file brings its
  licence file to the repository root.
- **Trace:** `agent.go` price caps, `billing.CreditsForUSD`,
  `plans/catalog.go`. Docs only; site-wide AI analysis is not built.

## Free citation-readiness score from the existing crawl (PR 1 of 5)

- **Decision:** Cite Fix is its own sidebar page and API, but it reads the site audit instead of crawling again. The crawler stores per-page citation signals; this package turns them into a 0-100 score in six weighted categories, a fix list ranked by points gained, and the 50 weakest pages. The owner named the feature "Cite Fix" and asked for it to ship in five PRs: (1) free checks, signals and the page; (2) a paid per-page LLM citability score with cost shown first; (3) citation proof from AI Visibility; (4) one click from a fix into the Content Editor, publish and re-scan; (5) SAM, MCP and report tools. JavaScript rendering is a separate owner decision.
- **Why:** Customers want to know why ChatGPT or Perplexity quote a competitor and not them. Most of the answer is in HTML we already fetch. A second crawler would double the load on customer sites and our server for the same pages. The checks follow the public GEO Optimizer rubric's categories (MIT) and the KDD 2024 GEO study (Aggarwal et al.), whose top methods were adding citations to sources, quotations and statistics; no code was copied. Weights are our own and favour access and direct answers, because an assistant that cannot read a page or find an answer in it never quotes it.
- **Preserve:**
  - The report is free. The only paid call is the agent (next entry), with a credit hold and exact call-count tests.
  - Every query is scoped by workspace through the `projects` join; a foreign project reads as `no_scan`.
  - An audit crawled before migration 47 is `needs_rescan`, never scored as zeros the site did not earn.
  - Blocking training bots (GPTBot, ClaudeBot, Google-Extended) costs no points; only search bots that cite pages count.
  - App shells fail one check (`pages-readable`), not every page check.
  - Freshness is counted in days from the scan day, and judged only on pages with a readable date.
  - Copy never promises a citation.
- **Trace:** `checks.go`, `citefix.go`, `store.go`, `handler.go`; `audit/aireadiness.go`, `audit/pageanalyzer.go`, `audit/repository.go`, migration `00047_audit_citation_signals.sql`. Tests: `citefix_test.go` works every score and gain by hand; `store_test.go` runs a crawl through `audit.Repository` into a report on Postgres, including the workspace boundary, a legacy audit, a running audit and a failed audit; `handler_test.go` covers 400/404/500/503. Not yet run against a live customer site; weights may need tuning once real scores are seen.

## Paid agent and AI Visibility proof (owner approved 27 Oct 2026)

- **Decision:** `POST .../cite-fix/agent/run` works on one page from the latest scan: Quick (DeepSeek V4.1 Flash analysis, at most 25 credits) or Deep (plus a DeepSeek V4 Pro rewrite, at most 60 credits). Default 5 weakest pages, at most 20 per batch and 20 runs per project per 10 minutes. Spend is labelled `cite_fix`. Gated by the AI writing plan feature (Basic and above). The report also carries `proof` from the latest saved AI Visibility run (free).
- **Why:** Owner wanted an AI agent with page-specific fixes, cheap models via OpenRouter, user picks pages, cost shown first.
- **Preserve:** OpenRouter `max_price` and `data_collection: deny` on every call (one Flash provider charged $30.91/M output). One hold per run, ceiling ≤ mode cap, customer charge ≤ hold; unusable answers charge 0; failed Deep rewrite charges only the analysis. Idempotency key replays, same content + mode is cached for 0 credits, one running run per page and mode. Only pages in the workspace's own latest scan are fetched, through the SSRF guard.
- **Trace:** `agent.go`, `agent_prompts.go`, `agent_store.go`, `proof.go`, migration 48, `platform/openrouter` provider fields; tests `agent_test.go`, `proof_test.go`, `handler_test.go`. Mutation-checked. Not run against real OpenRouter yet. The owner conversation behind these choices is in `imp_sessions_talked.md` (27 Oct 2026, Cite Fix).

## Report over MCP and SAM; agent stays on the page (PR 5 of 5)

- **Decision:** the free report is exposed as MCP tool `get_cite_fix_report`
  and in SAM's tool allowlist. No tool runs the agent.
- **Why:** owner's step 5. A model calling a paid agent would skip the
  confirmed batch price the owner asked for.
- **Preserve:** report stays free and workspace-scoped; agent spend only after
  the page's confirm dialog.
- **Trace:** `backend/internal/mcp/cite_fix_report.go` and its test. Step 4
  (fix → editor → re-scan) is the "Open in Content Editor" button plus "Scan
  again"; publishing happens in the customer's own CMS.
