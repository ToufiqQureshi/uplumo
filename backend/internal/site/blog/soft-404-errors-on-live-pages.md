---
title: Soft 404 on a Live Page? Find the Empty or Misleading Response
description: Fix Search Console soft 404s by checking page content, status codes, product states, templates and rendered output for URLs that look empty.
date: 2026-10-15
updated: 2026-10-26
---

**Short answer:** A soft 404 is a URL that returns a success response but looks like a missing or empty page. Check the exact URL Google inspected, its rendered content, and the HTTP status. If the page is truly gone, return a real 404/410; if it should exist, restore substantive content and useful navigation before requesting another crawl.

![A practical troubleshooting flow for soft 404 on a live page? find the empty or misleading response](/site/blog/soft-404-errors-on-live-pages.svg)

## Why this problem happens

Common causes include empty search results, a product that vanished from a template, a failed API call, thin category pages, or a branded “not found” screen served with status 200. A browser may show content after scripts or login while Google’s fetched version is blank.

## Check these signals before changing the site

1. Open the indexed URL in a private browser window and check its actual HTTP status, not just the visual page.
2. Compare live URL Inspection output with your browser: title, main content, product or service details, and internal links should be visible without a session.
3. Check whether a template displays “0 results,” “product unavailable,” or an error message for many URLs while returning 200.
4. Inspect logs and application/API errors around Googlebot’s last crawl. A transient backend failure can generate a soft-404 response for a valid URL.

## Fix it in a safe order

### 1. If the page has no replacement

Return an honest 404 or 410 status and provide helpful navigation to relevant categories. Do not keep a blank 200 response just to preserve the URL.

### 2. If the page is temporarily empty

Restore the content or explain the temporary state. For a product, show useful details, expected availability, alternatives, and clear next actions rather than an empty shell.

### 3. If the URL should be indexed

Add content that fulfills the page’s stated purpose, ensure the canonical points correctly, link it from a relevant page, and make sure no noindex or access challenge applies.

### 4. Fix the shared template

If many URLs are affected, identify the component returning empty content and test several page types before requesting validation. Fixing one URL cannot solve a template-wide bug.

## How to tell the fix worked

Re-run live URL Inspection and confirm the page returns the correct status and displays useful content. Validate a sample of URLs from each template, then monitor the Page Indexing report for recrawls.

## Mistakes that make this worse

Do not redirect every soft 404 to a category or homepage. A redirect is appropriate only when a close replacement exists. Do not request validation until the underlying response is fixed.

## Test a valid URL and an invalid URL side by side

Use GET requests against your own public routes:

```sh
curl -sS -D /tmp/valid-headers.txt -o /tmp/valid.html \
  'https://example.com/service'
curl -sS -D /tmp/missing-headers.txt -o /tmp/missing.html \
  'https://example.com/this-page-does-not-exist'
```

The valid route should contain the promised content with an appropriate success status. The missing route should return a genuine missing response, usually 404, even if its page includes your logo, navigation and helpful links. Branding does not require a 200 status.

Also test a slash variant of the valid page. A known equivalent variant should follow your canonical URL policy; returning a generic application shell with 200 can obscure a routing failure. Compare source and rendered content, because a JavaScript fallback may display differently from the initial response.

| Actual situation | Suitable response |
| --- | --- |
| Permanently deleted content, no replacement | 404/410 with helpful navigation |
| Content moved to a close equivalent | Appropriate redirect to that page |
| Temporary server outage | Suitable temporary server-error response, not a fake success |
| Product temporarily out of stock but useful details remain | Useful 200 page with accurate availability |
| Backend failure creates a blank valid page | Repair the failure and return truthful status during failure |

A short page is not automatically a soft 404; the page's purpose and Google's assessment matter. Do not pad an unavailable template with 1,000 words to hide the missing item. First determine whether the page should exist and what useful information belongs there, then inspect Google's reported result after recrawling.

## FAQ

**Is a soft 404 the same as a 404?**

No. A true 404 returns an error status for missing content; a soft 404 usually returns success while the page appears missing or empty.

## Related Uplumo guides

- [Out of Stock Products: Keep, Redirect or Remove the Page?](/blog/out-of-stock-product-pages-seo)
- [Google Cannot See Your JavaScript Content? A Debugging Guide](/blog/javascript-content-not-visible-to-google)
- [Website not showing on Google? Find and fix the cause](/blog/website-not-showing-on-google)

## Sources

- [Google troubleshooting crawling errors](https://developers.google.com/search/docs/crawling-indexing/troubleshoot-crawling-errors)
- [Google technical requirements](https://developers.google.com/search/docs/essentials/technical)
