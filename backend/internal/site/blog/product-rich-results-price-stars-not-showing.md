---
title: Product Schema Is Valid but Price or Stars Do Not Show
description: Separate schema validation from rich-result eligibility, then check visible product facts, review policies, feeds, variants and Search Console reports.
date: 2026-10-15
updated: 2026-10-26
---

**Short answer:** A valid structured-data test means Google can parse markup; it does not guarantee a rich result. Compare the markup with what shoppers see, check product eligibility and review policies, inspect Search Console’s product reports, and confirm Merchant Center feeds if you use them. Fix mismatched or stale data, then give Google time to recrawl.

![A practical troubleshooting flow for product schema is valid but price or stars do not show](/site/blog/product-rich-results-price-stars-not-showing.svg)

## Why this problem happens

Google can choose not to show an eligible feature. Price and availability can be wrong because of variants, currency, stock, cached markup, or feeds that disagree with the page. Star ratings have separate rules; marking up fake or invisible reviews can make the page ineligible. The specific self-serving-review restriction applies to LocalBusiness and Organization review snippets.

## Check these signals before changing the site

1. Use the Rich Results Test on a representative live product URL and URL Inspection to see the crawled version, not only a local template.
2. Compare name, image, offer price, currency, availability, and review/rating values with the visible page and checkout. Check variant-specific URLs.
3. Review Search Console’s Merchant listings and Product snippets reports for item-level errors or warnings. Compare feed and page attributes when using Merchant Center.
4. Check that ratings and review text are visible to visitors and meet Google’s review snippet policies; do not invent or aggregate ratings from unrelated items.

## Fix it in a safe order

### 1. Make page and markup agree

Use the accurate current price and availability for the specific product/variant. Remove outdated offers and ensure the structured data describes the main product shown on that URL.

### 2. Fix review eligibility

Use only legitimate, visible reviews for the reviewed item. Apply the policies for the actual reviewed entity; valid JSON-LD cannot override policy. In particular, do not apply a LocalBusiness/Organization rating to a product page as a shortcut.

### 3. Resolve competing data sources

If a Merchant Center feed is used, align identifiers, variant, currency, landing page, stock, and shipping data. Investigate feed warnings rather than adding duplicate schema fields.

### 4. Measure appearances separately

In Search Console, review search appearance and product enhancement reports over time. Capture the query and date; do not assume a visual spot check represents every user or device.

## How to tell the fix worked

After the fix, the live page should pass the relevant test and show no mismatch between visible details, JSON-LD, and product feed. Monitor report status and product search appearance; Google does not promise a specific feature display.

## Mistakes that make this worse

Do not add stars to a page without genuine eligible reviews. Do not change correct product facts simply to force a rich result. Structured data is not a direct ranking guarantee.

## Trace one product from page to search report

Choose one affected product URL and record its canonical URL, visible price, currency, availability, variant, review summary and structured data. Then compare those facts:

| Fact | Product page | Structured data | Feed, if used |
| --- | --- | --- | --- |
| Canonical product URL | The page users see | The product described | The submitted landing page |
| Price and currency | Current displayed offer | Same current offer | Same offer and currency |
| Availability | Current stock status | Same status | Same status |
| Reviews | Visible and policy-compliant | Only eligible visible reviews | Consistent where supported |

A mismatch can come from stale cache, a different product variant, a sale ending, a feed pointing at another URL or markup generated before the page updates. Fix the source that is wrong; do not paste a second JSON-LD block on top of a theme-generated one without checking for conflicting product entities.

### Stars have their own rules

A validator can confirm that markup parses, but review snippets follow separate eligibility and policy rules. Do not mark up reviews copied from another site or reviews hidden from shoppers. If you mark up ratings for a local business or organisation, Google's self-serving review rules also apply: a business cannot earn review stars for its own LocalBusiness or Organization page by publishing its own customer reviews there. See Google's [review snippet policies](https://developers.google.com/search/docs/appearance/structured-data/review-snippet) and [product snippet requirements](https://developers.google.com/search/docs/appearance/structured-data/product-snippet).

After fixing visible content and markup, validate the live URL, inspect the relevant Search Console enhancement report, and wait for recrawl. Track eligibility and impressions separately from guaranteed rich-result display: Google decides whether an eligible result is shown. For broader visibility or click changes, use our [Search Console diagnosis worksheet](/blog/search-console-impressions-clicks-down).
## Work through a minimal offer before adding ratings

This is an illustrative product-snippet example for a fictional INR-priced item. It deliberately omits ratings because no genuine reviews have been supplied:

```json
{
  "@context": "https://schema.org",
  "@type": "Product",
  "name": "750 ml insulated steel bottle",
  "image": "https://example.com/images/bottle.webp",
  "offers": {
    "@type": "Offer",
    "url": "https://example.com/products/bottle",
    "price": "899.00",
    "priceCurrency": "INR",
    "availability": "https://schema.org/InStock"
  }
}
```

For JSON-LD implementation, place the JSON in the appropriate `application/ld+json` script on your product page or let your supported CMS integration generate it. Replace every example value with the actual visible offer, then test the live URL. This example is not a complete merchant-listing implementation; selling directly, shipping/return details and product variants have additional guidance.

A successful parser test answers “can this markup be understood?” Eligibility also depends on the required fields for the chosen feature, page content and Google's policies. Appearance is a further step that Google chooses. Keep these three checkpoints separate.

If the product price changes by size, ensure the selected URL and markup identify the same offer. If a sale expires, update the visible price, markup and feed together. Do not add an imaginary `aggregateRating` because another result has stars.

For a business that publishes testimonials about itself, the LocalBusiness/Organization self-serving-review restriction is distinct from genuine Product review eligibility. Do not generalize either rule to every schema type. Confirm the entity and feature you are actually marking up, then use the corresponding official documentation.

## FAQ

**Does valid product schema guarantee stars in Google?**

No. Markup helps Google understand eligible data, but display is algorithmic and subject to content and policy requirements.

## Related Uplumo guides

- [Ecommerce product pages not ranking? Fix these basics](/blog/ecommerce-product-pages-not-ranking)
- [Out of Stock Products: Keep, Redirect or Remove the Page?](/blog/out-of-stock-product-pages-seo)
- [Google Search Console for small businesses: a simple guide](/blog/google-search-console-for-small-businesses)

## Sources

- [Google product snippets](https://developers.google.com/search/docs/appearance/structured-data/product-snippet)
- [Google product structured data](https://developers.google.com/search/docs/appearance/structured-data/product)
- [Google review snippet rules](https://developers.google.com/search/docs/appearance/structured-data/review-snippet)
