---
title: How to see ChatGPT traffic in Google Analytics (and why it hides)
description: Find AI Assistant traffic in GA4, inspect sources and landing pages, test attribution, and understand which Direct visits cannot be identified.
date: 2026-10-15
updated: 2026-10-26
---

**Short answer:** In GA4, open Traffic acquisition and inspect **Session default channel group → AI Assistant**, then check **Session source / medium** for the actual referrers or campaign sources. Google's current definitions include an AI Assistant channel; Google AI Overviews and AI Mode belong to Organic Search. Visits with no usable referral or campaign information can remain Direct, and no analytics tool can reliably recover every missing assistant identity.

## Start with the current GA4 reports

Open **Reports → Acquisition → Traffic acquisition** for the right property and a complete date range. Report navigation can differ by collection and permissions; use the report search if the menu is customized.

1. Choose **Session default channel group** and look for **AI Assistant**.
2. Change or add the dimension **Session source / medium**.
3. Inspect sources such as `chatgpt.com`, `perplexity.ai`, `gemini.google.com`, `claude.ai` or the sources actually recorded in your property.
4. Compare sessions, engaged sessions and verified key events for those rows.

Use the session-scoped dimension when the question is “where did this visit come from?” **First user** dimensions answer how a user was originally acquired; event-scoped attribution answers a different question again. Mixing these views can create an apparent contradiction.

Google's [default channel definitions](https://support.google.com/analytics/answer/9756891) are the authority for current grouping. A missing row can mean no matching measured sessions in the selected period; it is not proof that an assistant never mentioned the brand.

## Why some AI visits still appear as Direct

GA4 can use campaign information and referral context to assign a source under its attribution rules. Apps, privacy controls and copied links may remove that context. A known assistant domain or a tag such as `utm_source=chatgpt.com` provides a usable signal; the assistant's identity cannot be inferred with certainty when both are absent.

| What you observe | Supported conclusion | Unsupported conclusion |
| --- | --- | --- |
| Session source `chatgpt.com` | A source signal identifies ChatGPT | The visitor saw one particular answer or bought |
| A campaign tag identifying an assistant | The link carries that claimed source | Only the assistant could have created the tag |
| Direct visit to a deep article | Source attribution is missing in that view | It definitely came from AI |
| Google organic visit to a cited page | A Google organic visit was measured | It specifically came from an AI Overview |

Deep-page landings and coincident Direct growth are clues, not a reliable reclassification method. Do not relabel all Direct sessions as AI to make a report look complete.

## Explore assistant traffic by landing page

In **Explore → Blank**, add **Session source**, a compatible landing-page dimension, **Sessions**, **Engaged sessions** and your relevant **Key events**. Filter source values using the actual recorded domains. If using *matches regex*, remember the match behavior and test which rows are included.

For example, this illustrative exact-source pattern covers a small named set:

```text
^(chatgpt\.com|chat\.openai\.com|perplexity\.ai|gemini\.google\.com|claude\.ai|copilot\.microsoft\.com)$
```

It is not a universal list of AI sources. Compare it with the source values in your account, add legitimate variants you need and review it as domains change. Broad substring matches can capture unrelated campaign names.

Look for the question the landing page solves and what happened next. A compatibility guide, pricing page and broad blog article have different roles. Avoid comparing their conversion rates as if every visitor intended to buy immediately.

## When a custom channel group helps

The default AI Assistant channel may cover your standard reporting need. Create a custom group only when you need a different definition, want to separate specific assistants or need to retain a documented historical grouping.

1. Open **Admin → Data display → Channel groups** with the required permissions.
2. Create or edit a custom group and add explicit rules for the sources you intend to include.
3. Place specific rules before broader matching channels; Google uses the first match.
4. Select the resulting custom channel-group dimension in your report.

A custom group does not edit Google's default definitions. It can be applied retroactively to recorded source data, but it cannot reconstruct missing referrers or campaign tags. Follow [Google's custom channel-group guide](https://support.google.com/analytics/answer/13051316) for the current controls.

## Test the measurement path without faking an AI referral

Use a clearly marked link you control, for example:

```text
https://example.com/your-page?utm_source=qa_test&utm_medium=referral&utm_campaign=measurement_check
```

With the appropriate consent, confirm that redirects preserve the parameters and that the expected source appears after processing. DebugView can help inspect the test event path. Keep test traffic out of business totals. This demonstrates your measurement setup; it does not establish a real ChatGPT referral.

For actual visitors, verify key-event behavior separately. A CTA click is intent, a successful form submission is a stronger event and a qualified enquiry is a business outcome. Check that failed submissions do not fire success events or duplicate leads.

## Where Uplumo Analytics fits

Uplumo identifies supported assistant referrals and recognized campaign tags, and shows the assistant and landing page for visits with usable attribution signals. Its tracker sets no cookies and stores no raw IP addresses. Optional new/returning recognition uses first-party browser storage; site owners must obtain required consent and update visitor disclosures.

It can run alongside GA4, but neither system can reliably name an assistant after all source signals have been lost. Differences between systems can come from consent, blocked scripts, session definitions or collection coverage. Compare definitions before treating a mismatch as missing traffic.

## Report what you know

A useful monthly view includes the date range, source definition, measured sessions, top landing pages, verified key events and qualified enquiries. Report counts beside percentages: one enquiry from three visits is too small a sample to establish a reliable conversion benchmark.

Keep Google Search's AI feature impressions separate from assistant referral sessions. Read the [Search Console AI reporting guide](/blog/track-ai-overview-visibility-search-console) for that distinction. For improving the pages themselves, use the [AI citation workflow](/blog/get-cited-in-ai-search-results).

## FAQ

**Does GA4 have an AI traffic channel?**

Yes. Current Google documentation lists AI Assistant as a default channel. Inspect session sources and the current definitions if your report has no matching row.

**Can a custom channel recover unidentified Direct traffic?**

No. It reclassifies recorded signals; it cannot recreate an absent source.

**Are AI Overview clicks ChatGPT referrals?**

No. Google AI Overviews and AI Mode belong to Google's Organic Search channel in the current default definitions.

**Does a citation prove a visit occurred?**

No. Citation monitoring observes answers; analytics measures recorded visits. Keep both separate from leads and sales.

## Sources

- [Google Analytics default channel definitions](https://support.google.com/analytics/answer/9756891)
- [Google Analytics custom channel groups](https://support.google.com/analytics/answer/13051316)
- [Google Search Console and Analytics data](https://developers.google.com/search/docs/monitor-debug/google-analytics-search-console)
