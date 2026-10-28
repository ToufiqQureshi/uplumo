# Cite Fix: money scenarios, site-wide AI analysis and reference repos (28 Oct 2026)

Read this before adding any AI or provider call to Cite Fix, or before
borrowing an idea from an outside repository. It records the owner's question
"will this cost me money on a ₹2,499 Pro plan?", the worst-case numbers, and
which open-source projects informed the feature and under which licence.

## What is AI and what is rule-based today

| Part                                            | How it works                                        | Cost                        |
| ----------------------------------------------- | --------------------------------------------------- | --------------------------- |
| 0-100 score, 17 checks, fix list, weakest pages | Go code over the stored crawl (`citefix/checks.go`) | Free                        |
| Rival comparison, topic clusters (PR 3)         | Go code over fetched/stored HTML                    | Free                        |
| Per-page AI agent (Quick / Deep)                | DeepSeek via OpenRouter, user-picked pages          | ≤25 / ≤60 credits per page  |
| Google AI Overview, JS rendering (PR #244)      | DataForSEO, explicit button / opt-in                | ≤16 / ≤39 credits per click |

No AI reads the whole site today; the agent runs only on pages the user picks
(max 20 per batch).

## Worst-case numbers

Credits = provider USD × 1.28 × 1000 (`billing.CreditsForUSD`). Prices below are
the OpenRouter `max_price` caps set in `citefix/agent.go`
(Flash: $0.30 per million input tokens, $1.20 per million output tokens), so real
costs are lower.

| Action                  | Assumed tokens   | Max provider cost | Credits |
| ----------------------- | ---------------- | ----------------- | ------- |
| One page, Quick         | 8k in + 1.5k out | ≈ $0.0042         | ≈ 6     |
| 100 pages, Quick        | 100 × the above  | ≈ $0.42           | ≈ 540   |
| Site summary (proposed) | 25k in + 3k out  | ≈ $0.011          | ≈ 15    |

Plans (`backend/internal/plans/catalog.go`): Pro is ₹2,499 with 10,000 credits a
month. Spending all 10,000 credits means at most $7.81 / 1.28 ≈ $6.10 of provider
cost (about ₹520 at ₹85/$) against ₹2,499 of revenue. Credits are shared by every
paid feature, so this ceiling holds whatever the customer clicks.

## Where Uplumo could still lose money, and the guard for each

| Scenario                                                   | Guard                                                                                                                                       |
| ---------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| AI or provider call runs free, on page load or a re-render | Never allowed: paid calls only after an explicit, priced click (Playwright counts requests)                                                 |
| A provider charges far above list price                    | OpenRouter `max_price` + `data_collection: deny` on every call                                                                              |
| Retries, double clicks, two tabs                           | `retry: false`, idempotency keys, one running run per page/mode, advisory locks                                                             |
| Unusable AI answer / failed Deep rewrite                   | Customer pays 0 / only the analysis; Uplumo absorbs that provider cost (small, capped); at most 5 uncharged failures per workspace per 24 h |
| JS render: a later batch fails                             | Paid pages are kept and charged (fixed in PR #244); an uncertain 5xx stays held for reconciliation                                          |
| Workspace on an unlimited / not-charged plan               | Provider cost is absorbed; keep such plans internal only                                                                                    |
| A runaway bug                                              | `PROVIDER_DAILY_LIMIT_USD` global brake plus per-workspace credit holds                                                                     |
| Third-party sites (rivals, external links)                 | Free HTTP only, SSRF guard, robots.txt, timeouts, failures cached 24 h                                                                      |

Rule: a new AI feature is safe when it is charged in credits, capped, priced
before the click and never automatic. Free AI is the only way to lose money.

## Site-wide AI analysis (proposed, waits for the owner)

1. **Site summary**: send the rule-based results (scores, failed checks, topics,
   rivals), not raw pages, to one Flash call that writes the site report: top
   problems, topic gaps, order of work. About 15 credits; could run with each
   scan if the owner approves.
2. **Analyze all pages**: the existing agent over every content page, cost shown
   first (≈ 5-6 credits per page in Quick) and confirmed. Same guards as the
   per-page agent.

## Reference repositories

Ideas only; no code was copied into Uplumo. If code is ever ported, add the
project's licence file at the repository root (as with `LICENSE-OPENSEO-MIT`)
and say what was ported in the PR.

| Repository                                                                                                                                                                                                     | Licence    | Useful for Uplumo                                                                                                                                                                                                                                                                                                                                                                                                                                                |
| -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| [StJudeWasHere/seonaut](https://github.com/StJudeWasHere/seonaut) (Go)                                                                                                                                         | MIT        | ~75 SEO issue reporters: `nosnippet` / `max-snippet:0` (blocks Google AI Overview quotes), canonical to redirect/error/noindex, multiple or relative canonicals, sitemap listing noindex/blocked/non-canonical URLs, hreflang family, missing H1/lang/viewport, http links and forms, broken external links, image size, HSTS                                                                                                                                    |
| [Auriti-Labs/geo-optimizer-skill](https://github.com/Auriti-Labs/geo-optimizer-skill) (Python)                                                                                                                 | MIT        | 47 citability checks (`core/citability.py`): answer capsule, definition after heading, chunk-sized sections, authoritative sources, stale data, boilerplate ratio; topic clusters with pillar pages and interlinking (`core/topic_authority.py`); a longer AI bot list incl. Bingbot, DuckAssistBot, Amazonbot, xAI-Bot (`core/audit_robots.py`); hidden-text / prompt-injection detection (`core/injection_detector.py`); content decay (`core/audit_decay.py`) |
| [smartaces/opencite](https://github.com/smartaces/opencite) (Python)                                                                                                                                           | Apache-2.0 | AI Visibility metrics: citation rank inside an answer, first seen / last seen, momentum, averaging repeated runs of a prompt                                                                                                                                                                                                                                                                                                                                     |
| [SamurAIGPT/geo-checker](https://github.com/SamurAIGPT/geo-checker) (Next.js)                                                                                                                                  | None       | Not used: no licence, and it only forwards the page to an LLM                                                                                                                                                                                                                                                                                                                                                                                                    |
| [amplifying-ai/awesome-generative-engine-optimization](https://github.com/amplifying-ai/awesome-generative-engine-optimization), [trakkr-aisearch/awesome-geo](https://github.com/trakkr-aisearch/awesome-geo) | Lists      | Reading lists only                                                                                                                                                                                                                                                                                                                                                                                                                                               |

The GEO paper behind the citation checks: "GEO: Generative Engine Optimization"
(KDD 2024, arxiv.org/abs/2311.09735).
