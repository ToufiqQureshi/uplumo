# Uplumo blog style

Every Uplumo article must be worth ranking: it targets a keyword people search, answers it better than page one, and moves the reader to a next step in Uplumo. Follow this file for every new or edited post. Write original copy, make original visuals or use Uplumo screenshots with demo data, and keep claims verifiable.

## 1. Pick the keyword before writing

1. **Pull real data.** Use DataForSEO (keyword overview, United States first, then other target markets) for search volume, keyword difficulty (KD), intent and 12-month trend. Write the numbers in the PR. Never pick a topic on a hunch.
2. **Choose one primary keyword per post** and 2–5 close variants. Prefer KD under 40 with clear buyer relevance. A rising trend beats a flat larger number.
3. **Use US spelling** (optimization, not optimisation). The British variant usually has no recorded volume.
4. **Check the SERP and the AI Overview** (DataForSEO SERP, with People Also Ask). Copy the winning format, not the words: a "best X" query wants a list of 10+ tools with prices; "what is X" wants a definition, a comparison table and steps; "X alternatives" wants a baseline row and a price comparison.
5. **Avoid cannibalization.** One intent, one URL. Before writing, grep `backend/internal/site/blog/` for the keyword. If an existing post targets it, improve that post instead.

Current clusters (keep new posts inside one and link them):

| Cluster | Pillar | Supporting posts |
| --- | --- | --- |
| AI search basics | `seo-vs-aeo-vs-geo` | `what-is-aeo`, `how-to-rank-in-chatgpt`, `how-to-get-website-cited-by-ai` |
| AI visibility tools (commercial) | `best-aeo-tools` | `profound-alternatives`, `peec-ai-alternatives`, `best-semrush-alternatives` |
| Measurement | `how-to-track-competitors-in-ai-search` | `chatgpt-recommends-competitors-not-my-business`, `chatgpt-traffic-google-analytics`, `why-website-not-ranking-google` |

## 2. Front matter

```yaml
---
title: "Primary Keyword: Specific Promise (2027)"
description: "50–160 characters. Primary keyword early, the concrete outcome, and who it is for."
date: 2026-11-09
author: James
image: /site/blog/<slug>-cover.webp
related: pillar-slug, sibling-slug, next-step-slug
---
```

- **Slug** is the primary keyword in lower-case words joined by hyphens (`what-is-aeo`, `best-aeo-tools`). Never change a published slug. Change slugs freely before the first merge to `main`.
- **Title** up to 70 characters, primary keyword at the start. Add the year only for comparisons and "best" lists.
- **Author** is `James` (the owner's chosen pen name, schema type Person). Never invent credentials, job titles or a biography for James.
- **related** lists up to 3 published slugs shown as "Related guides" under the post. List the pillar, a sibling and the natural next step. An unknown slug, the post itself, a duplicate or more than 3 entries fail the build. Without `related`, the three newest other posts appear.
- Keep `date` when editing. Set `updated` for substantial revisions of a published post.

## 3. Article structure

1. **Answer first.** The first paragraph answers the query in 40–60 words, in bold, as a self-contained definition or recommendation an AI Overview or ChatGPT can quote without rewriting it.
2. **Concrete scenario.** The second paragraph shows the problem with a specific, global example (see section 6).
3. **"Best X" and "alternatives" posts** put a **best-for list** right after the opening: one line per tool, starting "best for ...". Then a disclosure line in italics: Uplumo publishes the post, where prices come from, and the date they were checked.
4. **`## In this guide`** linking every major H2. Anchors are the renderer's generated IDs (lower case, spaces to hyphens, punctuation dropped). Tests check that every link has a heading.
5. **Short H2/H3 headings that match questions people ask.** Paragraphs of 1–4 sentences. Numbered steps where order matters.
6. **Show the work:** comparison tables, worked calculations (state the denominator), before/after examples, exact menu paths. Label illustrative data.
7. **Bring in Uplumo at the step where it helps.** Name the feature (Prompt Explorer, AI Visibility, Cite Fix, Site Audit, Content Editor, Rank Tracking, Uplumo Analytics), what the reader sees and what they decide next. Check plan access in `backend/internal/plans/catalog.go` before writing it.
8. **`## Frequently asked questions`** with 3–4 `###` questions taken from People Also Ask and the keyword's related searches. Answers are 1–3 plain sentences. Close the section with another `##` heading (for example "Start with one page") before the call to action, because everything under the last question becomes its FAQPage answer.
9. **End with one clear action** and the links `[Start free with Uplumo](https://uplumo.com/sign-up) · [Compare current plans](https://uplumo.com/pricing)`.

Length: pillar 2,500–4,000 words; comparison and how-to posts 1,500–2,500. Write enough to beat page one; do not pad.

## 4. Internal links

- Every post links to its cluster pillar in the body, and the pillar links to every post in its cluster.
- Every post gets at least 2 inbound links from other posts (body or `related`).
- Use descriptive anchor text with the target's keyword ("compare the [best AEO tools](/blog/best-aeo-tools)"), never "click here".
- Use relative `/blog/<slug>` links. `TestPagesHaveNoDeadLinks` fails on a slug that does not exist.

## 5. Prices and claims

- **Never write an Uplumo price in a post or image.** Prices live only in `catalog.go` and change. Write "Free plan; AI Visibility from Pro" and link [Uplumo pricing](https://uplumo.com/pricing), which shows US dollars or Indian rupees per visitor. The owner's rule: Uplumo prices are never shown in rupees only.
- **Competitor prices**: take them from the provider's own pricing page on the day you write (Firecrawl scrape with `maxAge: 0`), in the provider's currency (normally USD), billed monthly, and state the check date. If a price is not published, write "Not published" or "Custom". Never estimate.
- Link each provider's pricing or documentation page under the table. Add new external hosts to `blogSourceHosts` in `site_test.go`.
- Give the disclosure line when Uplumo is in a comparison. Say honestly what other tools do better.
- Never promise rankings, citations or traffic. Separate measured visits, sampled AI answers and estimates.

## 6. Global audience

Uplumo is a global product. Examples must not assume one country: no single-country towns, festivals, seasons, payment methods or currencies as the running example. Use neutral settings ("a mountain resort", "a dental clinic", "a SaaS company") and placeholders such as `[town]`, `[country]` or `[local payment method]` when location matters. Default copy is plain US English.

## 7. Visuals

- Every substantial post has a **cover** (`<slug>-cover.webp`, 1440×900) and at least **two in-body visuals** placed next to the paragraph they explain.
- Diagram style: light green grid background (`#f3f8e6`), dark green headings (`#24391f`), white rounded cards with a `#e1efc6` border, a mono "eyebrow" label, an optional dark footer bar, DejaVu Sans. Size 1200 wide, 640–734 tall. Covers use a large title, a subtitle and 3–5 chips, with "Uplumo guide" bottom left.
- Make them as HTML cards, screenshot with the pre-installed Playwright (Chromium, `deviceScaleFactor: 1`), and convert to WebP at quality ~82 with Pillow. Look at every image before committing: no overflow, no empty half, no single-country text.
- Good diagram types: price or price-per-unit bar charts from verified data, before/after quotes, numbered step cards, "pick by the job" cards.
- Use real Uplumo screenshots from `/site/product/` (dashboard, AI Visibility, Prompt Explorer, Cite Fix) with demo data, captioned "with demo data".
- Every image needs descriptive alt text that includes what it shows (and the keyword where natural) and a short italic caption.

## 8. Structured data (automatic)

The renderer emits, per post: BlogPosting (author James as Person), BreadcrumbList (Home > Blog > post) and, when the post has a FAQ section, FAQPage. Pages without visible prices (blog index, info pages) describe plans in US dollars. You do not add JSON-LD by hand; keep the FAQ heading rules in section 3 so the FAQPage is right.

## 9. Publish checklist

1. Add the slug, dates, section-link count and image count to `TestPublishedBlogContent` and `TestGuideNavigationAndImages` in `blog_test.go`.
2. Add new external hosts to `blogSourceHosts` in `site_test.go`.
3. From `backend/`: `gofmt -l .`, `go vet ./internal/site`, `go test -race -count=1 ./internal/site`.
4. In the PR, list the target keyword with volume and KD, the cluster, and the inbound links you added.
