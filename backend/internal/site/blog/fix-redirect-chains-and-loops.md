---
title: How to Find and Fix Redirect Chains and Loops
description: Trace redirect hops, loops, and outdated targets with browser and crawler checks, then update links so visitors and crawlers reach the right page.
date: 2026-10-15
updated: 2026-10-26
---

**Short answer:** A redirect chain happens when one URL points through multiple redirects before reaching the final page; a loop never reaches a destination. Test the full response path, then update the first redirect and internal links to point directly at the final relevant URL. Keep redirects only where users or old links still need them.

![A practical troubleshooting flow for how to find and fix redirect chains and loops](/site/blog/fix-redirect-chains-and-loops.svg)

## Why this problem happens

Chains often grow after several redesigns: HTTP goes to HTTPS, a trailing slash changes, and an old slug redirects again. A loop can come from conflicting CMS, CDN, and server rules. These issues can slow visitors, confuse crawlers, and make migrations harder to debug, but a long list of harmless old redirects is not automatically the cause of a traffic drop.

## Check these signals before changing the site

1. Test the URL with redirects enabled and note every status code and Location header. Use a crawler or command-line request tool, not only the final browser address bar.
2. Test both HTTP/HTTPS, www/non-www, slash/no-slash, uppercase/lowercase, and query-string variants when your site uses them.
3. Check where the first redirect is configured: CDN, web server, CMS plugin, domain host, or application router. Duplicate rules in different layers commonly create loops.
4. Find internal links, canonicals, hreflang entries, and sitemap URLs that still point to a redirecting URL.

## Fix it in a safe order

### 1. Write the intended canonical destination

Choose one final URL format for each page. Confirm it returns 200 and contains the correct page. Do not decide based only on which URL is shortest.

### 2. Collapse the path

Update the earliest redirect so the old URL goes straight to the final destination, then remove conflicting rules. Change internal links to point to the final URL rather than relying on redirects.

### 3. Preserve useful old URLs

Keep a one-hop permanent redirect for genuinely moved content. A redirect should match the old page’s topic and purpose; return 404 or 410 when content is gone and there is no suitable replacement.

### 4. Deploy and test a redirect matrix

Test a sample of high-traffic URLs plus every rule family after deployment. Confirm there are no loops, unexpected cross-domain jumps, lost query parameters, or temporary status codes where permanent moves were intended.

## How to tell the fix worked

Save before-and-after status paths for representative URLs. The final page should return 200, old permanent URLs should reach it in one hop, and internal links/sitemaps should use the final address. Monitor crawl errors and conversions for changed URLs.

## Mistakes that make this worse

Do not remove redirects blindly: old backlinks, bookmarks, and indexed URLs may still rely on them. Avoid chains of “temporary” redirects as a substitute for deciding the proper destination.

## Inspect a redirect chain with a real GET request

Replace the example with your own public URL:

```sh
curl -sS -L --max-redirs 10 -D - -o /dev/null 'http://example.com/old-service'
```

This prints response headers for each followed hop while discarding the body. Use a GET request because some servers handle HEAD differently. A spoofed user-agent or successful local request does not reproduce a verified Googlebot crawl.

An illustrative chain might be:

```text
http://example.com/old-service     301 -> https://example.com/old-service
https://example.com/old-service   301 -> https://example.com/services/new-service
https://example.com/services/new-service   200
```

For a permanent move, configure the old URL to reach the final relevant destination directly where practical. Test both the redirect and the destination; a one-hop redirect ending at a blank 200 page is still broken.

| Test case | Expected outcome |
| --- | --- |
| Old published slug | Permanent redirect to equivalent current content |
| Canonical URL | Full page with 200, without a further redirect |
| Slash or host variant | Consistent normalization |
| URL with a required campaign parameter | Measurement data preserved where appropriate |
| Arbitrary missing slug | Real 404 rather than redirect to homepage |

If two systems disagree about HTTPS, a proxy may redirect to HTTP while the app redirects back to HTTPS. Inspect forwarded-protocol settings and rules at both layers rather than adding a third redirect plugin. For temporary experiments or moves, choose an appropriate temporary redirect instead of declaring every route permanent.

## FAQ

**Are all redirect chains a ranking emergency?**

No. Prioritise loops, broken destinations, high-value pages, and long chains that affect important crawl paths. Fix direct internal links first.

## Related Uplumo guides

- [SEO Traffic Dropped After a Site Migration? A Recovery Checklist](/blog/seo-site-migration-traffic-drop-checklist)
- [Google Chose a Different Canonical: How to Fix the URL](/blog/google-chose-different-canonical-how-to-fix)
- [SEO audit checklist: find fixes that can help your business](/blog/seo-audit-that-finds-real-problems)

## Sources

- [Google redirects and Search](https://developers.google.com/search/docs/crawling-indexing/301-redirects)
- [Google crawl budget guidance](https://developers.google.com/crawling/docs/crawl-budget)
