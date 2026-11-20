# How the free audits work

1. Open the SEO audit or the AEO audit, paste a page address (for example `yourwebsite.com`) and press Run. The two audits read the same page once, so you can switch between them without a second check.
2. We read that one page the way a search engine or AI crawler does: the HTML the server sends, your robots.txt, llms.txt and sitemap.
3. We score the page twice, with two separate engines:
   - **SEO** (how well Google can crawl, index, read and rank it): crawl and index, on-page SEO, performance and mobile, content and structured data.
   - **AEO** (how ready it is for ChatGPT, Perplexity, Claude and Google AI Overviews): AI access, answers the AI can quote, evidence and trust, entity clarity.
4. Each check says what we found, why it matters and how to fix it.
5. Each score is a weighted average of its groups. Serious problems cap it, and the report lists the reasons under "Why the score is limited":
   - SEO: a critical failure (the page is noindex, Googlebot is blocked by robots.txt, or it is not on HTTPS) holds the score at 49 or lower.
   - AEO: a critical failure (an AI search crawler is blocked, the text only appears with JavaScript, or the page hides instructions for AI) holds the score at 39 or lower.
   - Both: a high-impact failure holds the score at 79 or lower, and six or more warnings at 89.
6. Download the report as a PDF (your browser's print window; choose "Save as PDF") or as a Word file. The Word file is only available for a report you have already run in the last day.

The SEO score says how well the page is built for Google. The AEO score says how
ready the page is to be read, trusted and quoted by AI assistants. Neither
measures rankings, backlinks or whether ChatGPT or other assistants mention you
today.

Results are kept for 24 hours, so running the same page again shows the same
result. If a page is unreachable we say why and do not score it.
