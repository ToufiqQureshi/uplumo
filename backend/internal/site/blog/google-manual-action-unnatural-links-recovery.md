---
title: Google Manual Action for Unnatural Links: What to Do Before Disavowing
description: Read the exact Search Console notice, document link removal attempts, use disavow only when justified, and submit a clear reconsideration request.
date: 2026-10-15
updated: 2026-10-26
---

**Short answer:** Open the Manual Actions report and read the exact issue and scope. Identify links that violate Google’s spam policies, try to remove or qualify paid/manipulative links, and document your work. Use the disavow tool only when the situation meets Google’s criteria; then request reconsideration with a clear account of what changed.

![A practical troubleshooting flow for google manual action for unnatural links: what to do before disavowing](/site/blog/google-manual-action-unnatural-links-recovery.svg)

## Why this problem happens

Third-party tools may label many links “toxic,” but those scores are not Google manual actions. A manual action is different: it appears in Search Console and explains the issue. Disavowing random links can waste time and may discard signals from legitimate sites.

## Check these signals before changing the site

1. Verify that Search Console shows a manual action. Note whether it affects the whole site or specific sections and copy the issue description for your case notes.
2. Export links from Search Console and other sources, then review them by acquisition method and context. Focus on paid links, link schemes, hacked links, and manipulative patterns—not a vendor’s red score alone.
3. Record outreach attempts to remove or add `rel="sponsored"`/`nofollow` where appropriate. Keep the contact, URL, date, response, and result.
4. If there is no manual action, check for a security issue or ranking change separately. Do not treat a decline in a third-party link metric as a penalty.

## Fix it in a safe order

### 1. Remove or qualify policy-violating links

Contact site owners or vendors who placed paid/manipulative links. Stop ongoing schemes. Keep proof of changes and explain cases where removal was impossible.

### 2. Prepare a cautious disavow file only if justified

Follow Google’s current disavow guidance and file format. Use it for links you cannot remove when the manual-action or policy situation warrants it; do not disavow legitimate links solely because an SEO tool dislikes them.

### 3. Write a useful reconsideration request

State the violation, root cause, concrete cleanup, remaining links, and prevention steps. Be factual and specific; avoid blaming a former agency without evidence.

### 4. Wait for the review and monitor

After submission, check the report for status updates. Continue preventing manipulative link acquisition and keep records for the next review.

## How to tell the fix worked

The Manual Actions report should show the action removed or provide a response. Separately monitor search performance and link acquisition. A removed action does not guarantee rankings return to a previous position.

## Mistakes that make this worse

Do not submit the same vague reconsideration request repeatedly. Do not bulk-disavow every external link or pay a vendor promising instant removal.

## Prepare evidence a reviewer can understand

Use one row per reviewed placement or acquisition pattern:

| Field | What to record |
| --- | --- |
| Source and destination | Exact linking page and your linked URL |
| Why it violates policy | Paid followed placement, controlled network or another documented practice |
| Action | Removed, appropriately qualified, removal requested or inaccessible |
| Evidence | Dated response, screenshot or current link check |
| Prevention | Contract, access or acquisition process changed |

Do not publish this sheet if it contains private correspondence. Summarize the work accurately in your reconsideration request and supply evidence through the appropriate process.

If disavow is justified, the file is a plain UTF-8 or ASCII `.txt` file. A fictional syntax example is:

```text
# Illustrative entries only; do not upload this example
https://links.example/paid-placement
domain:network.example
```

Each entry belongs on its own line with no explanation appended. A URL entry targets that page; `domain:` covers the named domain or subdomain. Do not try to disavow an entire folder using a domain entry.

**An upload replaces the property's existing disavow list.** Download and review any current list before preparing a replacement, preserve still-justified entries, and check the correct supported property. Google's tool does not support Domain properties, so follow the current instructions for the appropriate URL-prefix property.

A reconsideration summary should identify the cause, the schemes stopped, what was removed or qualified, the links that could not be removed and the prevention steps. Use actual dates and counts from your review. A generic apology or an invented “100% cleaned” claim is less useful than an honest documented account.

## FAQ

**Should I disavow toxic links if Search Console shows no manual action?**

Usually not based on a third-party score alone. Google recommends using the tool cautiously and describes specific cases for it.

## Related Uplumo guides

- [Ethical backlinks for small businesses: a practical guide](/blog/ethical-backlinks-for-small-business)
- [Japanese SEO Spam Pages Appeared on Your Site? A Recovery Checklist](/blog/japanese-seo-hack-spam-urls-recovery)
- [Google Search Console for small businesses: a simple guide](/blog/google-search-console-for-small-businesses)

## Sources

- [Google disavow links guidance](https://support.google.com/webmasters/answer/2648487)
- [Google manual actions report](https://support.google.com/webmasters/answer/9044175)
- [Google link spam policies](https://developers.google.com/search/docs/essentials/spam-policies)
