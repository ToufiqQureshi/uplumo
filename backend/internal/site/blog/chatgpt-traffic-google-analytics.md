---
title: How to see ChatGPT traffic in Google Analytics (and why it hides)
description: Visits from ChatGPT, Perplexity and Gemini often show up as Direct. Here is how to find them in GA4 and track AI traffic properly.
date: 2026-10-15
updated: 2026-10-29
image: /site/blog/ai-traffic-shows-as-direct.svg
---

**Short answer:** In GA4, open Traffic acquisition and inspect **Session default channel group** for **AI Assistant**, then review **Session source / medium** and landing pages. Google's current channel definitions include AI Assistant. Some visits arrive without a usable referrer or campaign tag. GA4 and Uplumo cannot reliably recover that missing source.

![AI referral classification needs a recognised referrer or campaign tag; visits without a usable signal remain unknown](/site/blog/ai-traffic-shows-as-direct.svg)

## Why AI traffic matters now

People increasingly ask an assistant before they open Google. When ChatGPT, Perplexity, Gemini or Copilot answer, they often cite a few websites and link to them. A visitor who arrives that way has usually read a summary of your page already, so they tend to land with a clear question in mind.

If those visits are hidden inside "Direct", you cannot answer simple questions:

- Is any AI assistant recommending my business at all?
- Which of my pages do they send people to?
- Is that traffic growing month on month?
- Do those visitors fill the contact form or buy?

For a small business in India, that is the difference between knowing a channel exists and guessing.

## Why it shows up as "Direct"

Analytics tools decide where a visit came from mostly by the **referrer**, the address of the page the visitor clicked on. A visit is labelled "Direct" when there is no referrer and no campaign tag.

AI visits lose the referrer in a few common ways:

- **Apps instead of browsers.** A link tapped inside the ChatGPT or Perplexity mobile app often opens without passing a referrer.
- **Privacy settings.** Some browsers and apps strip or shorten the referrer on purpose.
- **Copy and paste.** Many people copy a URL from the answer into a new tab, which carries no referrer at all.

Some assistants help by adding a tag to the link. ChatGPT, for example, adds `utm_source=chatgpt.com` to many of the links it shows, which GA4 reads as the session source. But not every assistant does this, and not on every link.

## How to find AI traffic in GA4, step by step

### 1. Start with the current default channel

1. Open **Reports → Acquisition → Traffic acquisition**.
2. Select **Session default channel group** and look for **AI Assistant**.
3. Switch to **Session source / medium** to inspect which sources make up those visits.
4. Review the relevant landing pages and your configured key events.

Google's [default channel definitions](https://support.google.com/analytics/answer/9756891) distinguish AI Assistant traffic from Google AI Overviews and AI Mode, which belong to Organic Search. Use the report's actual dimension and scope; a user-acquisition view answers a different question from session acquisition.

### 2. Inspect sources rather than assuming every assistant is identical

Search source rows for the domains you expect, such as `chatgpt.com`, `perplexity.ai` or `gemini.google.com`. Check the associated medium and landing URL. A campaign tag can provide source information even when the browser sends no referrer, but it does not independently prove which conversation generated the visit.

### 3. Save a focused exploration if your reporting needs it

Use a GA4 exploration with Session source / medium, Landing page + query string, Sessions and your configured key-event metrics. Filter the actual sources you observed. A custom channel group is optional when your business needs a different grouping; it is no longer necessary simply to create the concept of an AI channel.

Keep the original source values available so the grouping can be checked. No channel rule restores a referrer or campaign tag that was never collected.

## What GA4 still misses

Everything above depends on a referrer or a tag being present. Visits from apps and copied links still arrive as "Direct", and they can be a large share of AI traffic. Two signs that some of your "Direct" traffic is really AI:

- "Direct" visits landing on deep pages, such as a specific blog post or a pricing page, rather than your homepage. People rarely type those addresses by hand.
- "Direct" traffic rising on pages that you know are cited in AI answers.

Treat these as clues, not proof.

## How Uplumo tracks AI traffic

Uplumo Analytics is a small tracking script, built to answer this exact question. It:

- recognises visits from ChatGPT, Perplexity, Gemini, Claude and Copilot by their referrers **and** by AI tracking tags such as `utm_source=chatgpt.com`;
- shows AI traffic as its own channel, with the assistant and the landing page for each visit;
- sets no cookies and stores no IP addresses. Optional new/returning recognition uses first-party browser storage; site owners must obtain any required consent and update visitor disclosures.

It is included in every plan, including Free. You add one script tag to your site, the same way you would add GA4, and both can run side by side.

## Test one AI referral before interpreting the report

Open a public page through a link with `utm_source=chatgpt.com&utm_medium=referral`, then check the recorded source and landing URL. This tests tag handling; it is a test visit, not proof that ChatGPT recommended the business. Compare a normal untagged visit separately. If neither a recognised referrer nor a campaign tag arrives, Uplumo cannot recover the missing origin.

For a useful weekly review, record the assistant source, landing page and your chosen business outcome. A click on a WhatsApp or enquiry button is intent; confirm a submitted form, conversation or sale in the system that receives it. Keep consent settings and blocked-script effects in the measurement note. Use [Uplumo's analytics workflow](/blog/uplumo-ai-traffic-analytics-search-performance) to connect these observations without treating citations as visits.

![Identify the signal before naming the source](/site/blog/chatgpt-traffic-google-analytics-workflow.svg)

*Workflow diagram: known referrer; campaign tag; no usable signal.*

![Keep four different measurements separate](/site/blog/chatgpt-traffic-google-analytics-review.svg)

*Review diagram: answer observed; visit recorded; outcome confirmed.*

## FAQ

**Will GA4 ever label AI traffic by itself?**
Yes. Google's current default definitions include AI Assistant. Check the report's scope and inspect the underlying source rows; unidentifiable visits still cannot be recovered.

**Does traffic from Google's AI Overviews count?**
Clicks from AI Overviews arrive from Google search like any other organic click, so they appear as Organic Search, not as AI assistant traffic.

**How much AI traffic is normal?**
It varies a lot by industry and by how often assistants cite your pages. Measure your own baseline first, then watch the trend each month.

**How do I get more of it?**
Write pages that answer questions clearly, so assistants can quote them. Our guide to [answer engine optimisation](/blog/what-is-answer-engine-optimisation) covers how.

## Sources

- [Google Analytics default channel definitions](https://support.google.com/analytics/answer/9756891)
- [Google Analytics traffic acquisition report](https://support.google.com/analytics/answer/12923437)
