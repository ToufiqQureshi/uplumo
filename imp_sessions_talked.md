# Important session talks

Short record of the important things the owner (Toufiq) said to any AI agent:
feelings, business calls, priorities, corrections. Read it after
`FOUNDER.md`, so you understand him without asking again.

**Rules for every agent:**

- Before your session ends, add one entry at the top if anything important
  was said: a feeling, a worry, a business or product decision, a priority
  change, or a correction of an agent. Skip routine coding chatter.
- Max ~10 lines per entry. Use his words (Hinglish) for feelings; plain
  English for decisions. Link the PR if there is one.
- A real decision also goes into `DECISIONS.md` or `FOUNDER.md`; this file is
  the story behind it, not the rulebook.
- Never write secrets, keys or customer data here.

---

## 28 Oct 2026: Cite Fix PR 2 (owner decisions)

- Use DataForSEO On-Page JavaScript only for app-shell pages the crawler flags: explicit opt-in, at most 20 pages per audit, with the credit ceiling shown first.
- Google AI Overview citations are approved at $0.0012 per queued question, up to 10 per click and 16 credits max, actual cost charged. Results are cached for 7 days.
- AI mentions must reuse the LLM-mentions result AI Visibility already saved, at 0 credits. Refresh it from the AI Visibility page at that feature's existing price; no new Cite Fix purchase. Any Cite Fix-only refresh needs a separate proposal of at most two calls with real observed costs and another yes.
- On-Page JavaScript is approved at $0.0015 per page, up to 20 app-shell pages and 39 credits max per audit, actual cost charged.
- Build the free rival signal table and pass rival evidence into the existing Quick/Deep agent prompt without changing its 25/60 credit caps.
- Show latest critical/warning Site Audit issues as “Also fix for Google”; expose rival and free report data through `get_cite_fix_report`, with no paid MCP tools.
- A 50-page site with 20 JS shells, 10 Google AI Overview questions, and saved mentions costs at most 55 credits in these new actions.

---

## 27 Oct 2026: Cite Fix, "sabko AI me cite hona ha" (Claude)

- Owner wants a crawler that reads the whole site and shows "kyu cite nahi hore, ye fixes karo, cite honge", A to Z content quality, and opens the content in the editor in one click. "muje uplumo ko bhohat advance software banana ha jo logo ke kaam aaye".
- On cost: "paise ki chinta mat kar ... user ko batayenge ki ye costly ha ... if needed vo karega warna chod dega". Paid parts show credits before the click.
- "baar baar nai bana sakte, ek hi baar me" -> agreed: one full design, shipped in 5 PRs. Name: **Cite Fix**, its own UI. He asked for the blueprint in PR 1; it went into the feature docs (repo rule: no plan files).
- Asked whether the GitHub repos are copied: no, ideas and check lists only (GEO Optimizer MIT rubric, SEOnaut checks).
- After PR 1: "kuch value dere ha hum? isme ai hai?" -> wanted a real AI agent (not SAM) giving page-specific fixes. Chose cheap models: "not opus, Deepseek v4 through open router". Worried about cost on big sites: "50 page ha to? in urls ko hi kar" -> user picks pages, top 5 default, max 20, cost shown first.
- Approved: Quick 25 / Deep 60 credits per page, Basic+, Flash for analysis + Pro for rewrite. Found one Flash provider at $30.91/M output, so every call caps provider price.
- "har ek cheez value de, kuch bhi bekar nai" and "future agent ko samjhe kitna effort" -> decisions, flows and this story are recorded in the citefix docs, DECISIONS.md and here. He is short on tokens: keep replies very short.

## 27 Oct 2026: Ubersuggest features, AI on every page (Claude)

- Owner compared open-seo, Uplumo and Ubersuggest and wants 8 features: Next Actions, AI Opportunities, AI Sentiment, AI Prompt Ideas, Local SEO grid, Backlink Opportunity, Content Ideas, Pixel Rank. "pura kaam do pr me khatam chahiye"; "jaldi bhai".
- "ui is most important muje dekhna ha sab kaise dikhenge": a clickable prototype was approved first (Design canvas artifact).
- Correction: "ye sab hinglish me kyu ha" -> product copy for these pages in English. "har ek feature actionable insight and data backed recommendation dena chahiye ... AI dal de har ek page pe ... credit to katega hi".
- Local SEO comes back (it was removed 17-23 Oct). PR 2 needs his choice of credits per grid scan, the plan for Pixel Rank (2x rank-check cost) and credits per backlink-gap lookup.
- He asked to force-delete 10 old worktree folders, then stopped the command before it ran; nothing was deleted.

## 27 Oct 2026: three plans, credit packs, "jitna use utne credit" (Claude)

- Owner: the whole SaaS runs on credits; sell only Basic, Pro, Business (Agency removed, no customers on it). Pro 10 sites, Business 20 sites; above that a +5 sites add-on (Rs 999, owner chose; built as 30 days from payment so a late buy is not wasted).
- Credit packs only for Pro and Business (Rs 1,499 / 5,000 and Rs 3,999 / 15,000); pack credits last while the plan is active and end with it (owner's choice). Must be stated on legal pages.
- White-label moves to Business. Limits must stay locked in code.
- "credit sahi se cut hone chahiye ... double payment issue solve it": found late UPI authorizations not captured (12-minute auto-capture) and second-subscription charges needing manual refunds; both fixed in the PR.
- Owner wants it fast, one PR; asked to wait while a deploy/merge ran, then continue.

## 27 Oct 2026: analytics must be accurate and better than GA4 (Claude)

- Owner asked to test every number and button on Analytics live, then fix all of it and make tracking smarter than GA4, after reading GA4 docs, without building useless features. He wants speed ("jaldi").
- Shipped in PR #234. Owner still has to set TRUSTED_PROXY_CIDRS on Railway and confirm that sign-ups and checkouts are not enquiries.

## 26 Oct 2026: theme means public pages too (Codex)

- Owner corrected that the theme control was missing from blogs and public
  pages; the earlier app-only implementation did not meet his expectation.
- Add it to the shared public header, share the saved app preference, and
  finish the release with actual public-page browser checks.

## 26 Oct 2026: header light-mode access (Codex)

- Owner asked to add light mode in the header to the existing PR.
- Reuse the implemented light/dark palettes and saved theme preference;
  expose System/Light/Dark visibly in the authenticated app header, including
  mobile, rather than creating a separate appearance system.

## 26 Oct 2026: ten Uplumo product and comparison blogs (Codex)

- Owner requested ten positive, substantive Uplumo articles, including Semrush,
  Ahrefs and Ubersuggest comparisons, with positioning from SMBs to enterprises.
- Articles use verified code capabilities and official competitor references.
  Google-algorithm-trained model and fastest-growing claims were not evidenced;
  use accurate descriptions of AI writing, briefs and scoring instead.
- Scope is new editorial content for review, not runtime features or deployment.

## 26 Oct 2026 (later): "teeno ek saath complete kar" (Claude)

- Owner wants the three open items done together, reported short.
- Done: max credits shown (PR #210, CI green, merge left to him: the merge
  was refused by the permission check); wiring, test-strength and code-size
  audit (this PR): no broken screen wiring, one dead route removed, tenancy
  and money tests catch real bugs.
- Not started on purpose: the "why not cited" feature. It is the core
  product and `FOUNDER.md` asks for a plan and his "haan" first.

## 26 Oct 2026: money rules, refresh, "why not cited" (Claude)

- **Money rule (his words):** "jitna kharcha ha utna hi kaato… hold ki kya
  zaroorat ha?" Approved ("haa"): hold only the typical cost (~372 for Brand
  Lookup, not the 2,240 worst case), charge the actual cost; if it exceeds
  the balance, Uplumo absorbs the rest (~₹50 worst case). PR pending.
- "unnessery data for seo ko call nai karna": saved reports never rebuy
  without a click (Refresh/Retry). PR #216.
- Test workspace's daily AI tracker **stays daily**: "test account me use
  karra hu taaki pura app bug free banau".
- Paid-call browser test must run in CI: PR #215.
- Next feature: "Site AI pe cite kyu nahi ho rahi" must be made ready.
- Reviews PRs one by one, wants to know what each is before merge; "tests
  brake nai hone chahiye"; "jaldi karna bhai time nai ha".
- Merged + deployed today: #197, #198 (AI card shows who is cited instead),
  #208, #209. Waiting for him: #210–#216.

## 26 Oct 2026: owner answers, DataForSEO audit (Claude)

- Owner: "max dikhao" — show the maximum credits before a paid action
  (credits audit F1). Separate PR, not built yet.
- Owner: the 3 `active` subscriptions are his test accounts.
- Asked again "isse mujhe kya milega?" before the audit: he wants the payoff
  first, in one breath, then the work.
- Found and fixed: AI Overview tracker answers were bought daily and never
  shown. Open for him: the test workspace's daily tracker costs ~$5.7/month
  real DataForSEO money; pause or weekly?
- An agent's `git stash pop` mistake pulled another agent's stash into the
  tree; restored on his go. Stash `keyword field-delivery WIP` is intact.

## 25 Oct 2026: credits audit result (Claude, PR #208)

- Owner said "haan" to the credits audit and asked first "isse mujhe kya
  milega?" — he wants a plain yes/no on "loss me jayenge?", not a report.
- **Answer given:** loss risk low today: $4.24 provider spend, all charged
  back at 1.28x; 9/9 deliberate code breaks caught by tests.
- **Open for him:** customer is shown "about 372 credits" for a Brand Lookup
  but can be charged up to 2,240. Decide: show the max, or cap the charge.
  Also confirm the 3 `active` subscriptions are his own/test accounts.
- Next in order: DataForSEO audit.

## 25 Oct 2026: "Dar lagta hai" — vision reset (Claude, PR #208)

- **Feeling:** "Ab dar lagta hai. Kuch samajh nahi aa raha ki hum value de
  rahe hain ya nahi. AI ko kuch banane bolta hoon, ulta ho raha hai. Codebase
  itna bada ho gaya ki kya kahan hai samajh nahi aata." Tired of explaining
  himself again in every session.
- **Vision:** "Market me best baney." Uplumo tells a business: is your site
  cited by AI, who is cited instead, and **why not you, what to fix**.
- **Value complaint:** Traffic page just repeats GSC/GA4. He wants "ye giri
  hai, kyu giri, aise solve karo", plus an AI agent that analyses his data.
  "UI ek dum bakwas", "value nahi de rahe hum."
- **Constant doubt:** "Kya app error free aur secure hai? Kya kisi bug se
  loss me to nahi aayenge?" Fears wasted DataForSEO calls, wrong credit
  deduction, bloated code, half-wired features, tests that never fail.
- **Facts found:** ~65k Go, ~76k frontend, ~51k test lines, 354 Markdown
  files. AI Visibility already buys "cited?" and "who instead"; "why not" is
  not built.
- **Decided:** customer = Indian businesses, small to big; goal = first 10
  paying customers (none yet); order = credits audit → DataForSEO audit →
  build "why not cited" → other audits. Agents restate the plan and wait for
  "haan" before money/UI/feature work.
- **His ask to all AIs:** "Jab tak founder ke saath vision aur feelings nahi
  samjhoge, koi saath milke kaam nahi kar payega." Read `FOUNDER.md` and this
  file every session.
