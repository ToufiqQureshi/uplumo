# Ubersuggest competitor review (26 Oct 2026)

Hands-on review of Ubersuggest (NP Digital) on the owner's trial account,
project `theshalimarhotel.com`, India / English. Read this before planning
"why not cited", the dashboard or pricing. Everything below was seen in the
app on this date; numbers are theirs, not verified.

## Their plans (India, per month, shown in the upgrade dialog)

| Plan       | Monthly | Yearly (per month) | Lifetime | Sites | Users | Credits/month | AI prompts tracked          |
| ---------- | ------- | ------------------ | -------- | ----- | ----- | ------------- | --------------------------- |
| Individual | ₹1,135  | ₹631               | ₹11,350  | 1     | 1     | 250           | 10 per project, **monthly** |
| Business   | ₹1,890  | ₹985               | ₹18,900  | 2–7   | 2     | 500           | 15 per project, biweekly    |
| Enterprise | ₹3,785  | ₹2,137             | ₹37,850  | 8–15  | 5     | 1,000         | 20 per project, weekly      |

- ChatGPT + Gemini on Individual/Business; Google AI Overview only on Enterprise.
- Per-action credits are shown on the button ("Use 5 credits to get an AI
  response"). Local SEO: "credit usage varies by grid size".
- Lifetime deals and UPI AutoPay. The trial converts to a paid plan
  automatically ("Your Pro subscription starts on October 26, 2026").

Uplumo today (`backend/internal/plans/catalog.go`): Solo ₹999 (1 site, 4,000
credits), Growth ₹2,499, Business ₹4,999, Agency ₹9,999. Price is close; we
must win on depth of the AI answer, not price.

## What they have

| Area                 | What we saw                                                                                                                                                                                    |
| -------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Dashboard            | AI summary of the business in one line, "Ask AI" box, numbers (keywords, traffic, value, backlinks, DA), "How AI sees your website" (visibility, sentiment), Next Actions to-do, trend charts. |
| Next Actions         | One prioritised to-do list (48 items) with impact, effort, done/ignored, "Solve it" (AI content). Blocked until a "business summary" setup is done.                                            |
| AI Search Visibility | Overview, Prompts, Brands, Sentiment, Citation (mostly "coming soon"/empty), Opportunities. Brand visibility by topic, share of voice, competitor trend.                                       |
| Opportunities (GEO)  | Checklist: create content from competitors' content, update own content, fix non-indexable pages, allow AI bots, create `llms.txt`, get mentioned on third-party and UGC sites.                |
| AI Keyword Overview  | Google volume/difficulty/CPC **plus** AI prompt ideas for the keyword, "Get AI response" per prompt (5 credits), brands mentioned and top sources AI used, next to the Google top 10.          |
| AI Chat              | Agent that calls their tools (shows "Called Brand Visibility Overview").                                                                                                                       |
| Local SEO            | Google Maps grid scan (rank per map point), India cities.                                                                                                                                      |
| Other                | Rank tracking, pixel rank tracking, competitor traffic, backlinks, content writer, Chrome extension, ChatGPT app, MCP server, WordPress plugin.                                                |

## Where they are weak (our opening)

1. **Wrong prompts, so a useless 0%.** For a mid-range Mumbai hotel they
   track "best luxury hotels in India" prompts; the answer is Taj and Oberoi,
   and the customer sees 0% with no way to act on it.
2. **"Why not cited" is generic.** Their AI Chat answer: "strengthen brand
   signals, schema, internal links". No page, no source, no competitor
   evidence. Opportunities showed 0 items in every row; Citation is empty.
3. **The real reason was in their own data, unused.** For "hotels near Kemps
   Corner" AI cited booking.com, tripadvisor.in, makemytrip, justdial and a
   competitor's own site. The fix for a local Indian business is "get listed
   and reviewed on these 4 sources", which they never say.
4. **Slow and monthly.** AI data updates monthly on the cheapest plan; the AI
   response panel took over 40 seconds.
5. **Setup walls.** "Solve it" is blocked behind a business-summary setup.

## What Uplumo should do (recommendation, owner to approve)

1. **"Why not cited" with evidence** (next feature): per prompt, show the
   sources AI used, the competitor page it cited, and the gap on the
   customer's page (from Site Audit and the saved answer), as at most 5 fixes
   with a button each. Include "get listed on X" when the cited sources are
   directories/OTAs (Justdial, Tripadvisor, MakeMyTrip, Google Business).
2. **Relevant prompts by default**: build starter prompts from the business's
   city, category and price level, not the broad industry.
3. **One Next Actions list** on the dashboard that merges audit issues, AI
   gaps and ranking quick wins, with done/ignore.
4. **AI technical checks** in Site Audit: AI bots allowed in `robots.txt`,
   `llms.txt`, indexable key pages (cheap, no paid call).
5. **Keep showing the credit cost on each paid button** (we already do; it is
   now table stakes).
6. Later: Google Maps grid scan for local businesses; Chrome extension.

Do not copy: lifetime deals (they lock in provider cost we pay per call) and
auto-converting trials without a clear reminder.
