---
title: Core Web Vitals explained: what LCP, INP and CLS mean for your site
description: A plain guide to Google's Core Web Vitals: what LCP, INP and CLS measure, the good scores to aim for, how to check them free and the fixes that help most.
date: 2026-10-15
updated: 2026-10-26
---

**Short answer:** Core Web Vitals are three measurements Google uses to judge how a page feels to real visitors: **LCP** (how fast the main content shows), **INP** (how quickly the page reacts to a tap or click) and **CLS** (how much the page jumps around while loading). A page passes when, for at least 75% of visits, LCP is 2.5 seconds or less, INP is 200 milliseconds or less and CLS is 0.1 or less.

![Core Web Vitals thresholds: LCP good up to 2.5 seconds and poor over 4, INP good up to 200 milliseconds and poor over 500, CLS good up to 0.1 and poor over 0.25](/site/blog/core-web-vitals-thresholds.svg)

## Why they matter

Core Web Vitals are used by Google's ranking systems alongside many other signals. Good scores do not guarantee a higher position or automatically break a tie between pages. They measure problems visitors can experience directly; use them to prioritize actual loading, interaction and layout issues.

Test the devices and connections your actual customers use. A page that feels fine on office Wi-Fi can behave differently on a mid-range phone or a slower network.

## The three metrics, in plain words

### LCP: Largest Contentful Paint

**What it measures:** how long it takes for the biggest thing on the screen (usually the main image or headline) to appear.

| Good | Needs work | Poor |
| --- | --- | --- |
| up to 2.5 s | 2.5 to 4 s | over 4 s |

**Common causes of a slow LCP:** a large, uncompressed hero image; a slow server or hosting plan; fonts and scripts that block the page from drawing.

### INP: Interaction to Next Paint

**What it measures:** how quickly the page responds after a visitor taps, clicks or types. It replaced First Input Delay (FID) in 2024 and looks at all interactions, not just the first.

| Good | Needs work | Poor |
| --- | --- | --- |
| up to 200 ms | 200 to 500 ms | over 500 ms |

**Common causes of a poor INP:** heavy JavaScript, too many third-party scripts (chat widgets, trackers, ad tags) and large page builders doing a lot of work on every click.

### CLS: Cumulative Layout Shift

**What it measures:** how much content moves unexpectedly while the page loads. You have felt this when you go to tap a button and an ad pushes it down.

| Good | Needs work | Poor |
| --- | --- | --- |
| up to 0.1 | 0.1 to 0.25 | over 0.25 |

**Common causes of a high CLS:** images and embeds without a set width and height, banners or pop-ups inserted at the top, and web fonts that change the size of text when they load.

## Lab data vs field data

You will see two kinds of numbers, and the difference matters:

- **Field data** comes from real Chrome users who visited your page over the last 28 days (the Chrome UX Report). **This is what Google uses.** New or low-traffic sites may not have enough visits to get any.
- **Lab data** comes from a single simulated visit by a tool such as Lighthouse. It is great for finding causes, but it is one test on one simulated device, so it can differ from what your visitors experience.

Use lab data to find what to fix, and field data to confirm that the fix worked for real visitors.

## How to check your Core Web Vitals for free

1. **PageSpeed Insights** (pagespeed.web.dev): enter a URL to see field data, when there is enough of it, and a Lighthouse lab test.
2. **Google Search Console → Core Web Vitals report:** groups your pages as Good, Needs improvement or Poor, using field data, for mobile and desktop.
3. **Uplumo Site Audit:** runs the page speed test as part of a full site audit and shows real-visitor data next to the lab results, with each issue explained in plain words. The Free plan tests your homepage on mobile and desktop; paid plans test up to ten pages, one of each page type.

## The fixes that help most

Use these as starting hypotheses, then verify the bottleneck before changing the page.

### For LCP

- Compress and resize your main image, and serve it as WebP or AVIF.
- Do not lazy-load the main image at the top of the page; lazy-load images further down instead.
- Use good hosting or a CDN, and turn on caching.
- Load only the fonts and font weights you actually use.

### For INP

- Remove plugins and third-party scripts you no longer need.
- Delay chat widgets and similar tools until the visitor scrolls or interacts.
- On WordPress, avoid heavy page-builder effects on pages that matter most.

### For CLS

- Give every image, video and embed a width and height.
- Reserve space for banners and ads instead of inserting them above content.
- Avoid pop-ups that push content down on mobile.

## Diagnose the bottleneck before installing another speed plugin

Consider a hypothetical service page with poor mobile LCP but acceptable CLS. Open the lab diagnostic and identify the LCP element. If it is the hero image, check when the browser discovers the image, its download size and the time the server takes to respond. Reducing a file helps little when most of the delay happens before the request starts.

A developer can make an important image discoverable in the initial HTML and reserve its space:

```html
<img src="/images/workshop-hero.webp"
     width="1200" height="800"
     fetchpriority="high"
     alt="Technician inspecting an air-conditioning unit">
```

Use high fetch priority only for the genuinely important image; applying it to every image defeats prioritization. Adapt dimensions and responsive sources to the real design. Do not lazy-load the main above-the-fold image.

For INP, reproduce the slow interaction itself: opening the menu, filtering products or submitting a form. A good initial-load score does not prove those interactions respond quickly. Record a performance trace and inspect long tasks around that interaction. Break up unnecessary synchronous work and retest the same action.

For CLS, watch the page while images, fonts and banners arrive, and after interaction. Reserve dimensions for media and predictable banner space. A screenshot taken after everything loads cannot show the jump that happened earlier.

When reading PageSpeed Insights, note whether the field result is for the exact URL or an origin fallback. Do not present an origin-level improvement as proof that one page passed. The 28-day field window mixes older and newer visits after a release; a same-day lab improvement is useful evidence of a fix, not immediate field confirmation.

## FAQ

**Do Core Web Vitals affect rankings?**
Google uses Core Web Vitals in its ranking systems, but a pass does not guarantee rankings or conversions. Measure the actual experience and business outcome.

**Why do my lab and field results disagree?**
Lab tests simulate a particular environment. Field assessments use real-user distributions, including the 75th percentile over the reporting window, rather than a simple average. Use lab tests to debug and eligible field data to assess real-user outcomes.

**My site has no field data. What now?**
That can happen on new or low-traffic pages. Use lab diagnostics and appropriate real-user monitoring; field data appears only when reporting eligibility and sample requirements are met.

**Do I need a developer?**
Not always. Image compression, removing unused plugins and setting image sizes are often things you can do yourself in your website builder. For deeper JavaScript work, a developer helps.
