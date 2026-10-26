---
title: How to track AI Overview visibility in Search Console
description: Use Google Search Console to review available AI search performance data, compare it with organic results and choose a useful next action.
date: 2026-10-15
updated: 2026-10-26
---

**Short answer:** Use Search Console's **Generative AI performance report (Search)** to inspect reported impressions for Google AI Overviews and AI Mode. Its documented breakdowns are Pages, Countries, Dates and Devices, with supported search-type filters. Examine queries and clicks separately in ordinary Search performance; do not present those ordinary query rows as a dedicated AI-only query report.

![A measurement loop from AI search visibility to page review and a tracked improvement](/site/blog/ai-overview-measurement.svg)

## What the current Google report measures

Google's [report documentation](https://support.google.com/webmasters/answer/16984139?hl=en) says the insights rolled out worldwide on 31 August 2026. A report can still be absent when the site has insufficient impressions in the supported features. Review eligibility and actual data rather than assuming the rollout simply has not reached your country.

The documented report includes **impressions** in AI Overviews and AI Mode, excluding Search Labs experiments. It is not a complete citation log across assistants and does not report ChatGPT or Perplexity referrals.

| Data source | What to use it for | What it does not establish |
| --- | --- | --- |
| Generative AI performance report | Reported Google AI feature impressions and supported breakdowns | Every prompt, exact answer text or cross-assistant citation |
| Ordinary Search Performance | Google clicks, queries, pages and other documented metrics | AI-only attribution for every query row |
| Analytics | Measured sessions, sources and site actions | A visit from every displayed citation |
| Prompt-monitoring sample | Answers and sources observed under a defined test | Share of all AI searches or proof of revenue |

Keep the source name on every chart. An impression, a citation sample, a visit and a sale are different observations.

## Inspect one page and one market first

1. Choose the correct verified property and a complete date range.
2. Open the generative AI report and record its total impressions.
3. Use the **Pages** tab to identify the relevant final/canonical URL.
4. Review the supported country and device breakdowns for the question you are investigating.
5. Save the filters and export date so the next comparison uses the same definition.

Google documents Pages, Countries, Dates and Devices; it does not document a dedicated Queries tab in this report. The supported search-type choices distinguish text-based and multimodal Web searches. They are not a switch that reveals every original prompt or separates every assistant.

Data is generally assigned to the canonical URL. Property-level and URL-level aggregation can produce different chart and table totals, so do not add row counts blindly and accuse the report of missing impressions. Recent data can be preliminary, and daily reporting uses Pacific Time.

## Compare with ordinary queries without inventing attribution

Open **Performance → Search results** separately and filter to the same page and a comparable period. Inspect queries, clicks and impressions to understand the page's broader search relevance. If a guide receives AI feature impressions and ordinary clicks, you cannot assume every click came from an AI Overview or associate an ordinary query row with a specific generated answer.

For a hypothetical guide, a report might show more AI feature impressions while measured site visits stay flat. Possible explanations include people receiving enough information in the answer, changes in the page/query mix or normal variation. The totals alone cannot identify the cause.

Review the actual page: does it solve a task worth clicking through to complete? A useful worksheet, original comparison, detailed procedure or verified example can add value beyond a short summary. Do not hide the answer to force a click or manufacture extra pages for each prompt variation.

## Investigate a missing report

Check the correct property, available impressions and the current documentation. If important pages are not indexed or eligible for snippets, resolve that before treating the missing report as a dashboard bug. Use URL Inspection to distinguish live access from Google's indexed state.

If there is too little data, record **not available**. Do not fill the gap with a third-party visibility score that looks like a Google metric. Continue improving page accuracy and access, then review after enough real data accumulates. A sitemap can help discovery, but it cannot create impressions or guarantee inclusion.

## A client report that keeps the sources separate

Use the following outline:

```text
Property / canonical page:
Date range, country, device and search-type filters:
Google AI feature impressions and source report:
Ordinary Search clicks and query themes, separately labelled:
Assistant referral sessions and source definition:
Qualified enquiries or sales, with measurement limits:
Page changes made and the next review:
```

For prompt sampling, also record the prompt, platform/model where known, language, available location context, date, successful observations and failures. Do not change the prompt set between months without explaining it. Do not sum its citation counts with Search Console impressions.

## Improve the source page and check the next outcome

Google's [AI optimization guidance](https://developers.google.com/search/docs/fundamentals/ai-optimization-guide) emphasizes useful, distinctive content grounded in ordinary Search practices. Fix wrong facts, show evidence readers can check, explain limitations and keep the main content publicly accessible. Special markup cannot guarantee selection.

Use the [citation comparison workflow](/blog/get-cited-in-ai-search-results) to review an actual answer sample, the [AI traffic guide](/blog/chatgpt-traffic-google-analytics) for visits and the [SEO client report worksheet](/blog/seo-client-report-template-that-proves-progress) for business reporting.

## FAQ

**Does this report show ChatGPT citations?**

No. It covers the documented Google Search generative AI features.

**Can I identify the exact AI query from its ordinary Search Performance row?**

Not from that row alone. Keep the reports' documented dimensions and attribution limits clear.

**Does no report mean my brand never appears in AI?**

No. Insufficient reported impressions or other data/eligibility limits are not a complete statement about all assistants.

**How often should a small business review it?**

A monthly review with stable filters is a reasonable routine; investigate meaningful changes with enough data rather than daily noise.

## Sources

- [Google Search Console: generative AI performance report](https://support.google.com/webmasters/answer/16984139?hl=en)
- [Google Search Central: AI optimization guide](https://developers.google.com/search/docs/fundamentals/ai-optimization-guide)
- [Google: Search Performance report](https://support.google.com/webmasters/answer/17010961?hl=en)
