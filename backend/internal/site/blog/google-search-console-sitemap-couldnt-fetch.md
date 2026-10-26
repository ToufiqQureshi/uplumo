---
title: Google Search Console Says “Couldn’t Fetch” for Your Sitemap
description: Check sitemap access, response headers, XML validity, robots rules and URL consistency to solve a Search Console fetch error.
date: 2026-10-15
updated: 2026-10-26
---

**Short answer:** Open the exact submitted sitemap URL without a login, confirm it returns a stable 200 with valid XML, and check CDN, robots, redirects, and host consistency. Then inspect the reported error details in Search Console and resubmit after the actual cause is fixed. A valid XML file on your laptop does not prove Googlebot can fetch the public URL.

![A practical troubleshooting flow for google search console says “couldn’t fetch” for your sitemap](/site/blog/google-search-console-sitemap-couldnt-fetch.svg)

## Why this problem happens

Common causes include a redirect loop, temporary server error, blocked crawler, HTML error page returned as 200, invalid compression, wrong content type, unsupported URL host, or a sitemap index pointing to inaccessible child files.

## Check these signals before changing the site

1. Copy the exact sitemap URL from Search Console and test it from a clean external session. Check status, final URL, response headers, XML body, and whether authentication is required.
2. Validate XML syntax and namespace. Ensure every `<loc>` URL uses the canonical scheme and host and responds with an appropriate status.
3. Check robots.txt includes the correct sitemap location and does not accidentally block the sitemap or important page paths.
4. If using a sitemap index, test every child sitemap independently. Check CDN caches, server logs, size limits, and compression.

## Fix it in a safe order

### 1. Fix the response at the submitted URL

The URL should reliably serve XML or a valid supported sitemap format, not an HTML challenge or login. Remove unnecessary redirects and use the canonical host.

### 2. Clean sitemap entries

Include canonical, indexable URLs you want considered. Remove 404s, redirects, noindex pages, parameter duplicates, and staging URLs. A sitemap is a discovery hint, not an indexing guarantee.

### 3. Repair the index or split files

For a sitemap index, make sure every referenced child URL is accessible. Split very large sets within Google’s documented limits and use stable names.

### 4. Resubmit and then inspect pages

After the fetch succeeds, use Page Indexing and URL Inspection to diagnose individual URLs. Sitemap processing does not mean every listed URL will be indexed.

## How to tell the fix worked

Record the submitted URL, response code, last-modified date, child maps, and Search Console status. Recheck after processing and investigate page-level indexing separately.

## Mistakes that make this worse

Do not keep re-submitting an unchanged file when Google cannot fetch it. Do not include URLs because they exist in the CMS; only include preferred, useful, indexable URLs.

## Check the public response rather than the filename

Run a GET request against your exact submitted URL:

```sh
curl -sS -D /tmp/sitemap-headers.txt \
  -o /tmp/sitemap.xml 'https://example.com/sitemap.xml'
```

Inspect both files. A body beginning with a login form or a JavaScript challenge is not a sitemap even if the URL ends in `.xml` and the response says 200. Do not assume a correct content type proves the body is valid.

For an XML URL set, the structure should look like this illustrative example:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://example.com/services</loc>
    <lastmod>2026-10-26</lastmod>
  </url>
</urlset>
```

Use your real modification dates; do not stamp every URL with today's date on every request. Encode special characters correctly, including `&amp;` in XML URLs that contain `&`.

Google's documented limit is 50,000 URLs or 50 MB uncompressed per sitemap. Use a sitemap index with accessible child maps for larger sets. A compressed file must still be valid after decompression; a tiny `.gz` can exceed the uncompressed limit.

If public GET requests succeed but Google's last fetch failed, inspect logs around that timestamp. A later successful browser request cannot explain an earlier timeout. Check the exact host, child file, status and deployment version, then resubmit after fixing a demonstrated failure. “Success” in Sitemaps confirms processing, not that every listed page is indexed.

## FAQ

**Does “Couldn’t fetch” mean none of my pages are indexed?**

No. It means Google had trouble reading the submitted sitemap. Pages may be found another way; diagnose sitemap delivery and URL indexing separately.

## Related Uplumo guides

- [Website not showing on Google? Find and fix the cause](/blog/website-not-showing-on-google)
- [Robots.txt vs Noindex: Why a Blocked URL Can Still Appear in Google](/blog/robots-txt-vs-noindex-pages-still-indexed)
- [SEO Traffic Dropped After a Site Migration? A Recovery Checklist](/blog/seo-site-migration-traffic-drop-checklist)

## Sources

- [Google sitemap overview](https://developers.google.com/search/docs/crawling-indexing/sitemaps/overview)
- [Google sitemap format and size requirements](https://developers.google.com/search/docs/crawling-indexing/sitemaps/build-sitemap)
- [Google Search Console sitemap report](https://support.google.com/webmasters/answer/7451001)
