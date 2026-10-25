# Founder brief: read this before you build anything

Every agent (Claude, Codex, Cursor, others) reads this at the start of every
session, right after `CLAUDE.md`. It holds what the owner (Toufiq) wants
Uplumo to be, what worries him, and a map of the product as it is. `CLAUDE.md`
says _how_ to work; this file says _what for_. If your task does not serve the
vision below, say so before you write code.

## 1. The vision, in his words

> "Hum market me best baney."
>
> "User ki site AI pe cite ho rahi hai ya nahi, ye pata karna hai. Agar nahi
> ho rahi, to kaun ho raha hai? Aur kyu nahi ho raha user ka site cite? Ye
> batana hai."

So Uplumo answers three questions, in this order, for every customer:

1. **Is my site cited by AI?** (ChatGPT, Google AI Overviews, Gemini,
   Perplexity, Claude.)
2. **If not, who is cited instead?** The competitor domains and pages.
3. **Why not me, and what do I fix?** A specific, ranked list of reasons and
   actions, not a dashboard of numbers.

Question 3 is the product. Questions 1 and 2 are data any tool can buy; the
"why" and the "what to do next" is where Uplumo wins.

## 1a. Who, the goal, and what comes next (owner, 25 Oct 2026)

- **Customer:** Indian businesses, small to big. Copy, UI and pricing are
  written for them.
- **Goal now:** the first **10 paying customers**. Judge every task by
  "does this help get or keep one of those 10?"
- **Order of work** (one PR each; do not jump ahead without the owner):
  1. Credits audit: shown cost == charged cost, under retries, double
     clicks, races and provider failures.
  2. DataForSEO audit: real paid calls per click vs. each `flow.md`.
  3. Build Q3, "why not cited, and what to fix" (section 5, "The biggest
     gap").
  4. Then wiring, test-strength and code-size audits (section 5 list).
- **Stop list:** no new feature, page or provider outside this order. Ideas
  go to the owner as a proposal, not as code.

## 2. What "value" means to him

> "Generic same GSC aur GA4 dene se kya hoga? Kuch valuable cheez jisse user
> decision le sake hamare data pe."

- Every screen must end in a **decision or an action**, not a chart. Bad:
  "Clicks: 1,240." Good: "Clicks fell 32% this week because `/pricing` lost
  position 3 → 9 for 'seo tool india'. Fix: X."
- **Explain changes:** what went up or down, why, and what to do. "Ye cheez
  giri hai, kyu giri hai, ise aise solve karo, urgent."
- An AI agent (SAM) should be able to talk about the customer's own data and
  do this analysis on request.
- A page that shows the same thing GSC or GA4 already shows for free is not
  value. Either add the "why / what next" layer or drop the page.

## 3. What worries him (treat each as a standing requirement)

| Worry                                                                    | What it means for your work                                                                                                                          |
| ------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| DataForSEO bill has unnecessary calls                                    | Count paid calls per click; prove the count in a test. No paid call without a user action. See `CLAUDE.md` "Money first".                            |
| Credits may not be deducted correctly, so he loses money                 | Credits charged must equal the cost shown, every time, under retries and races. Any doubt here is a bug to prove or disprove, not a guess.           |
| Code is bloated: 100 lines where 5 do, 10 files where 3 do               | Use Go / React / TS built-ins and installed libraries first. Delete before you add. A PR that removes code is welcome.                               |
| Frontend exists but backend logic is missing, or both exist but disagree | When you touch a feature, check the screen → API → service → provider path end to end. Report any dead or half-wired piece.                          |
| Tests pass but do not catch bugs                                         | A test must fail when the code is broken (mutation check, `CLAUDE.md` section 4). A test that cannot fail is worse than none.                        |
| Security: "koi bhi kone se hack na ho"                                   | Tenancy, auth, payments and credits are the highest-risk paths. See `docs/security/MODEL.md`.                                                        |
| Scale: millions of requests                                              | Bounded queries, timeouts, caches, no per-request provider calls. Do not over-build for scale that does not exist yet; fix what breaks first at 10x. |
| Too many docs, files and folders                                         | Do not add a file when an existing one can hold it. Do not add a doc nobody will read.                                                               |
| "Docs hai phir bhi AI ko flow samajh nahi aata"                          | Docs must match code. If a doc misled you, fix it in the same PR and say so.                                                                         |
| UI is "bakwas" across all features                                       | UI work should reduce clutter and lead to the next action, not add panels.                                                                           |
| He builds something and gets the opposite of what he wanted              | **Before any code, restate in 3-5 plain Hinglish lines what you will build and what you will not. Wait for his "haan".** Then build only that.       |

The question in his head is always: **"Kya hamara app error free aur secure
hai? Kya hum kisi bug se loss me to nahi aayenge?"** Your job is to replace
that doubt with evidence: tests that can fail, counted provider calls, and
plain answers.

## 4. How to talk to him

- Hinglish, short, senior-dev mentor tone. No essays, no jargon without a
  one-line meaning.
- Lead with the answer, then the one or two things he must decide.
- Say "I don't know yet, here is how I'll check" instead of guessing.
- **When his "haan" is needed:** before anything that touches money,
  credits, providers, data, UI, or adds a feature. Typos, doc fixes and a
  small bug fix inside the current task can go ahead; mention them after.
- **Top of every PR, 3 Hinglish lines for him:** kya badla; kitni paid
  calls / credits pe asar; kaunsa test todke dekha ki bug pakadta hai.
- Record important talks in `imp_sessions_talked.md` (see that file).
- He is one person running the whole company. He has no paying customers yet
  (25 Oct 2026). Protect his money and his time first.

## 5. Product map (as of 25 Oct 2026)

Sidebar groups from `frontend/src/client/navigation/items.ts`. "Paid" = calls
a billed provider (DFS = DataForSEO, LLM = OpenRouter). "Vision fit": how much
the page serves the three questions in section 1.

| Sidebar → page                                                               | What it does today                                                                                  | Backend package                                                 | Paid     | Vision fit                                      |
| ---------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- | --------------------------------------------------------------- | -------- | ----------------------------------------------- |
| Home → Dashboard                                                             | Overview of the project                                                                             | `dashboardoverview`, `activation`                               | no       | Should lead with "are you cited, and why not"   |
| Home → SAM assistant                                                         | Chat assistant over project data                                                                    | `sam`                                                           | LLM      | **Core** for "why / what next" if fed real data |
| Customers & traffic → Traffic (Analytics, Search Perf.)                      | Own analytics script + GA4 + Search Console reports                                                 | `analytics`, `ga4`, `gsc`, `google`                             | no       | Low today: mirrors GSC/GA4, no "why it dropped" |
| Customers & traffic → AI Visibility (Tracker, Brand Lookup, Prompt Explorer) | Is the brand mentioned/cited, which pages AI cites, brands named alongside, one prompt across 4 AIs | `aisearch`                                                      | DFS      | **Core** (answers Q1 and Q2; Q3 "why" missing)  |
| Your website → Site Audit                                                    | Crawl and technical checks                                                                          | `audit`, `crawleraccess`                                        | no       | Supports Q3 (technical reasons not cited)       |
| Your website → Rank Tracking                                                 | Google positions for tracked keywords                                                               | `ranktracking`                                                  | DFS      | Medium                                          |
| Your website → Content Editor                                                | Write, score, AI-assist, publish to WordPress                                                       | `contenteditor`, `sitepublish`, `linksuggest`, `contentauthors` | DFS, LLM | Supports "fix it" step of Q3                    |
| Research → Keywords (Research, Saved)                                        | Keyword volume, difficulty, ideas                                                                   | `keywords`                                                      | DFS      | Low (commodity data)                            |
| Research → Competitors (Domain, Backlinks)                                   | Competitor domain and backlink data                                                                 | `domain`, `backlinks`                                           | DFS      | Medium: could explain "why they're cited"       |
| Reports & setup → Reports                                                    | Project reports                                                                                     | `reports`                                                       | no       | Low                                             |
| Reports & setup → Setup (Context, Agent setup)                               | Project context; MCP access for outside agents                                                      | `projectcontext`, `mcp`, `apikeys`                              | via MCP  | Context feeds SAM; MCP is a side feature        |
| Billing                                                                      | Plans, Razorpay subscription, credits                                                               | `billing`, `razorpay`, `plans`                                  | money    | Must be correct before any growth               |
| Help                                                                         | Support page                                                                                        | `site`                                                          | no       |                                                 |

Size today: about 65k lines of Go, 76k lines of frontend, 51k lines of tests,
354 Markdown files (~20k lines), 42 backend packages, 28 frontend features.

### The biggest gap

AI Visibility already buys the data for Q1 and Q2 (mentions, cited pages,
`brandsAlongside`, `sourceSites`). Nothing yet turns it into **Q3: "why not
you, and what to fix"**, which would combine those citations with Site Audit,
the cited competitor pages and content gaps. That is the most valuable next
feature, and most other pages should feed it or step back.

### Not yet verified (open audits, one PR each, owner picks the order)

1. Credits: shown cost == charged cost for every paid action, including
   retries, double clicks and provider failures.
2. DataForSEO: real calls per click vs. the counts in each `flow.md`.
3. Frontend ↔ backend wiring: every screen's API call has a working route
   and every route has a caller (or a reason, e.g. MCP).
4. Tests: mutation check on billing, credits, tenancy and paid-call tests.
5. Code and docs size: what can be deleted with no behavior change.
