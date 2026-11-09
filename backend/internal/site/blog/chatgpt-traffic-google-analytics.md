---
title: "How to Track ChatGPT Traffic in Google Analytics 4 (GA4)"
description: "Find ChatGPT traffic in GA4, check AI referrals and landing pages, troubleshoot missing visits, and measure leads with a practical tracking workflow."
date: 2026-10-15
updated: 2026-11-10
author: Uplumo team
image: /site/blog/chatgpt-traffic-google-analytics-cover.webp
---

![Editorial illustration of an identifiable ChatGPT referral reaching website analytics and completing an event](/site/blog/chatgpt-traffic-google-analytics-cover.webp)

Someone clicks a link to your website in ChatGPT. Where should it show up in GA4, and how do you tell whether that visit turned into an enquiry?

**Open Reports → Acquisition → Traffic acquisition, then check Session source / medium for `chatgpt`.** Use the **AI Assistant** channel for a wider view. Once you identify sessions, break them down by landing page and the key event that matters to your business.

## In this guide

- [Know what counts as a ChatGPT referral](#know-what-counts-as-a-referral)
- [Find ChatGPT traffic in GA4](#find-chatgpt-traffic-in-ga4)
- [Break down referrals by landing page](#break-down-traffic-by-landing-page)
- [Measure leads, signups and sales](#measure-confirmed-leads-and-sales)
- [Troubleshoot missing or Direct traffic](#troubleshoot-missing-or-direct-traffic)
- [Set up a custom channel when needed](#set-up-a-custom-channel-when-needed)
- [Check Google AI Search separately](#check-google-ai-search-separately)
- [Monitor AI referrals in Uplumo](#monitor-ai-referrals-in-uplumo)
- [Frequently asked questions](#frequently-asked-questions)

## Know what counts as a referral

ChatGPT referral traffic is a person clicking a link in ChatGPT and visiting your website. GA4 needs a working measurement setup and usable attribution information to report that visit and its source.

A brand mention inside an answer is different. So is a crawler fetching a page.

| Activity | What to measure | Where to look |
| --- | --- | --- |
| Someone clicks a ChatGPT link to your website | Sessions, landing pages and configured key events | GA4 or your website analytics |
| An AI answer mentions your business | Mentions across the questions you test | AI visibility monitoring |
| An answer links to your website | Citations in the sampled answers | AI visibility monitoring |
| GPTBot or another crawler requests a page | Verified crawler requests | Server or CDN logs |

A website can receive a citation without a click. It can also receive crawler requests without any human visitor arriving. Keep these measurements separate when reporting results.

## Find ChatGPT traffic in GA4

### Check the AI Assistant channel first

Google's current [default channel documentation](https://support.google.com/analytics/answer/9756891?hl=en) includes **AI Assistant**. It covers sources such as ChatGPT, Gemini, Copilot and other assistants, while excluding Google's AI Overviews and AI Mode.

In GA4:

1. Open **Reports** and find **Traffic acquisition** under Acquisition. Your report collections may be organised differently.
2. Choose a date range that covers the visits you want to investigate.
3. Inspect **Session default channel group** for **AI Assistant**.
4. Switch to **Session source / medium** to inspect the individual sources.

![GA4 navigation diagram from Traffic acquisition to session source and the ChatGPT filter](/site/blog/ga4-chatgpt-traffic-path.webp)

*The channel groups multiple assistants. Use the session source to isolate ChatGPT.*

If you do not see an AI Assistant row, continue with the source check below. An absent row alone does not establish that nobody arrived from an AI assistant.

### Check the session source directly

With **Session source / medium** selected, search for `chatgpt`. You may find a source such as `chatgpt.com`, or the older `chat.openai.com` domain. Inspect the medium recorded alongside it rather than assuming every visit must be labelled `referral`.

For a broader AI traffic review, check the sources you actually receive from Perplexity, Gemini, Copilot and Claude as well.

Use session-scoped dimensions for this task. **First user source** answers a different question: how a person was originally acquired. Someone first acquired through Google can return through ChatGPT later.

Before exporting a result, check that a country filter, comparison or short date range has not excluded the sessions you are looking for.

## Break down traffic by landing page

Finding the source is useful. Knowing which page brought someone in gives you something to work on.

In **Explore**, create a blank free-form exploration:

1. Import **Session source / medium** and **Landing page + query string** as dimensions.
2. Import **Sessions**, **Engaged sessions** and **Key events** as metrics. Add **Total revenue** if your ecommerce measurement is configured.
3. Place the source and landing-page dimensions in Rows and your metrics in Values.
4. Filter the session source/medium to contain `chatgpt`, or use a condition matching the exact source values you observed.

For an illustrative hotel example, ChatGPT visitors might land on a family rooms page rather than the homepage. Review whether that page clearly explains occupancy, facilities, booking conditions and how to check availability.

For a software company, visits might arrive on a comparison article. A useful next step could be a link to the relevant product workflow or pricing details.

Treat landing-page paths carefully when exporting reports. Query strings can contain unnecessary identifiers or personal information if your website has been configured poorly. Do not collect or share sensitive information in page URLs.

## Measure confirmed leads and sales

Choose the action that matters to your business. It might be a completed enquiry, a trial signup or a purchase.

Then check that the event fires at the correct moment. A click on “Submit” is not necessarily a successfully submitted form. A click on “Book now” does not establish that a booking was completed.

Use a small measurement plan:

| Business question | Measurement to review |
| --- | --- |
| Are people arriving from ChatGPT? | Identified ChatGPT sessions |
| What interests them? | Landing pages and engagement |
| Do they enquire? | A verified enquiry event marked as a key event |
| Do they buy? | Correctly implemented purchase events and revenue |

If your report includes several key events, separate the one you are investigating before interpreting the total. Multiple actions or repeat events can make a general key-event count look more impressive than the number of actual enquiries.

With only a handful of sessions, report the counts alongside any conversion rate. One enquiry from two visits is too little evidence for a reliable channel forecast.

If visits are arriving but enquiries are missing, use our [guide to diagnosing clicks without leads](https://uplumo.com/blog/why-website-not-ranking-google#problem-clicks-but-no-leads--can-the-right-visitor-take-the-next-step) to review the page and its next step.

## Troubleshoot missing or direct traffic

Work through the evidence before changing your tracking setup.

| Problem | Check | Next action |
| --- | --- | --- |
| No ChatGPT source appears | Dates, filters, source values and whether an identifiable visit occurred | Broaden the reporting window and inspect the actual source rows |
| You clicked a link but cannot find the visit | Consent, ad blocking, tag installation and reporting delay | Verify collection with a controlled test and GA4's diagnostic tools |
| Campaign tags disappear | The original destination and any redirects | Preserve valid attribution parameters through redirects |
| Visits are classified as Direct | Whether a referrer or campaign information was available | Keep unknown traffic separate rather than relabelling it as AI |
| ChatGPT sessions appear but key events do not | Event implementation and the landing-page journey | Test the action, then review the offer and user experience |

Links opened from apps, copied addresses and privacy controls can affect the information analytics receives. A deep-page Direct visit may have many explanations. It is not proof of a ChatGPT referral.

![Tracking checklist separating missing source signals, missing sessions and missing lead events](/site/blog/ga4-chatgpt-attribution-check.webp)

*Confirm collection first, then check attribution and the success event.*

Use a controlled test to check the implementation: open an actual link to your website from an assistant, note the destination and any parameters, and verify whether the pageview is recorded under the consent conditions you use. Do not assume your test session represents the attribution of every visitor.

**No analytics tool can reliably identify the origin of every visit when the necessary source information is absent.** Reading supported referrers and campaign tags improves classification where those signals exist. It cannot reconstruct a missing private conversation.

## Set up a custom channel when needed

A custom channel group is useful when you want your own source list or a consistent reporting definition across a team. You do not need to create one simply because older guides say GA4 has no AI channel.

Google provides [instructions for custom channel groups](https://support.google.com/analytics/answer/13051316?hl=en). With the appropriate property permissions, open **Admin → Data display → Channel groups**, copy or create a group, and add an AI channel using a **Source** condition.

For an exploration, apply the equivalent condition to **Session source**. The following is a starting pattern for domain-style source values, not a complete list of every AI service:

```text
^(.*\.)?(chatgpt\.com|chat\.openai\.com|perplexity\.ai|gemini\.google\.com|copilot\.microsoft\.com|claude\.ai|grok\.com|chat\.deepseek\.com)$
```

Compare the pattern with actual source values in your property. It intentionally will not match a custom campaign source such as `chatgpt_newsletter`. Add a source only when you understand where it came from.

Place the channel before other rules that would otherwise capture the same traffic. Then select your custom group in a supported report. Creating it does not replace every report's existing dimension automatically.

Custom groups can reclassify historical data that already contains matching source information. They cannot recover attribution that was never collected.

## Why is GPTBot traffic not showing up in my standard GA4 setup?

GPTBot requests and human referral visits are different types of activity. A crawler fetching a page does not normally execute your website's JavaScript analytics tag like a person's browser would, so those requests should not be expected to appear as ordinary GA4 sessions.

[OpenAI's crawler documentation](https://developers.openai.com/api/docs/bots) distinguishes their roles:

- **GPTBot** relates to content that may be used for model training.
- **OAI-SearchBot** supports discovery in ChatGPT search features.
- **ChatGPT-User** can fetch pages in response to a user's action.

Use server or CDN logs for crawler requests. Verify identity using the provider's published guidance rather than trusting a user-agent name alone. Keep crawler counts out of your customer traffic and lead reports.

A crawler visit does not demonstrate that your page was cited, recommended or clicked.

## Check Google AI search separately

Google Analytics classifies traffic from AI Mode and AI Overviews as **Organic Search**. Search Console includes those experiences in overall Web search reporting. Open its separate **Generative AI performance** report to see which pages earned impressions in Google’s AI features. Use those impressions to assess visibility and GA4 organic sessions to assess visits. [Google’s Generative AI performance reports](https://developers.google.com/search/blog/2026/06/gen-ai-performance-reports)

That means a filter for `chatgpt.com` will not measure AI Mode visits. An increase in Google organic traffic also does not tell you how much came specifically from AI Mode.

Review the reporting dimensions actually available in your accounts and keep any separate visibility observations clearly labelled. Do not turn an organic-traffic trend into an exact AI Mode traffic estimate. See Google's [AI features and website measurement guidance](https://developers.google.com/search/docs/appearance/ai-features).

## How can you check ChatGPT traffic to a competitor's website?

You cannot see a competitor's exact ChatGPT sessions or conversions without authorised access to their analytics. A visibility tool can show whether the competitor appears in the questions it tests, but those mentions are not a traffic count.

For a useful competitive review:

1. Choose questions a potential customer would ask.
2. Record which businesses are named and which pages are linked.
3. Read the cited sources to identify information your own pages may lack.
4. Repeat comparable tests over time and keep them separate from your website analytics.

Uplumo's AI Visibility tools support sampled answer and citation research. That can help investigate who appears alongside your brand. It does not grant access to a competitor's visitor data.

Our guide to [why ChatGPT recommends competitors](https://uplumo.com/blog/chatgpt-recommends-competitors-not-my-business) explains how to turn those observations into a page-improvement plan.

## Do you need to connect Google Analytics to ChatGPT?

You do not need a ChatGPT integration to measure visitors arriving from ChatGPT. Your website's GA4 installation records the visits it can measure and attribute.

Connecting analytics to an AI assistant is a separate task: it lets the assistant help analyse data you authorise it to access. If you do that, choose the minimum necessary access and never paste account credentials or visitor-level personal information into a chat.

For this article's workflow, start in your analytics reports. A connector is optional and does not reveal the private prompts that sent people to your site.

## Monitor AI referrals in Uplumo

If you want to check recognised AI referrals alongside your site's other traffic, [Uplumo Analytics](https://uplumo.com/) offers a tracking script that identifies supported AI referrers and campaign tags. You can review the landing pages and select recorded events that your business counts as leads.

The practical workflow is straightforward:

1. Add your website and install the analytics script.
2. Verify that visits and your chosen events are being recorded correctly.
3. Review recognised AI referrals and the pages visitors reach.
4. Improve a relevant page, then compare later observations with your baseline.

Analytics is available on Free and can run alongside GA4. Keep the reporting definitions in mind when comparing numbers: two analytics systems can measure sessions and visitors differently.

When you need to investigate sampled mentions and citations, use AI Visibility on an eligible plan. It answers a different question from referral analytics: what the tested answers say about your business. [Compare current plans and feature access](https://uplumo.com/pricing) before choosing paid tools.

## Start with one report and one useful action

Check your ChatGPT sources, choose the landing page with the clearest business relevance, and verify the action you want visitors to take.

If you want a separate view of recognised AI referrals, [create your free Uplumo account](https://uplumo.com/sign-up), install Analytics and select your lead events. Start with the visits you can actually measure. Use that evidence to decide which page deserves your attention next.

## Frequently asked questions

### Can GA4 track ChatGPT traffic automatically?

GA4 can attribute visits when its measurement setup works and source information is available. Google's documented AI Assistant channel can group qualifying traffic. Inspect Session source / medium to understand the underlying sources.

### Can I see the exact ChatGPT prompt a visitor used?

Standard referral analytics does not reveal a visitor's private prompt. Prompts you choose to monitor in an AI visibility tool are test questions, not a log of customer conversations.

### Why do I see AI citations but no website visits?

A citation does not require anyone to click. Readers may receive enough information in the answer, or the cited link may point to another source. Review the actual answer and destination before treating visibility as traffic.

### How often should I monitor ChatGPT traffic?

Choose a cadence that fits your traffic volume. A weekly review is a practical starting point, with a longer comparison window when visits are sparse. Check sources, landing pages and meaningful outcomes together rather than reacting to each day's count.
