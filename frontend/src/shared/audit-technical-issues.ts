/* Technical checks produced by the site crawler. */
export const AUDIT_TECHNICAL_ISSUE_TYPES = {
  "ai-training-bots-blocked": {
    severity: "info",
    title: "AI training crawlers are blocked",
    explanation:
      "These bots are used for training. Blocking them is a choice and does not lower your AI readiness score.",
    howToFix:
      "Review their robots.txt rules only if you want to allow training use.",
  },
  "external-checks-pending": {
    severity: "info",
    title: "External link checks are queued",
    explanation:
      "External links and image sizes are checked separately so they do not delay other site scans.",
    howToFix: "Refresh this audit after the queued checks finish.",
  },
  "external-checks-incomplete": {
    severity: "info",
    title: "Some external links or images could not be checked",
    explanation:
      "The audit could not verify every external link or image within its safety limits.",
    howToFix:
      "Review unverified destinations manually; some sites block automated checks or did not respond in time.",
  },
  "invalid-html-language": {
    severity: "warning",
    title: "Missing or invalid HTML language",
    explanation:
      "Without a valid language, browsers and search engines cannot reliably identify the language of the page.",
    howToFix:
      'Add a valid language tag such as lang="en" or lang="hi" on the html element.',
  },
  "missing-viewport": {
    severity: "warning",
    title: "Missing mobile viewport",
    explanation:
      "Mobile browsers may render the page at desktop width, making content hard to read.",
    howToFix:
      "Add a viewport meta tag with width=device-width and an appropriate initial scale.",
  },
  "multiple-titles": {
    severity: "warning",
    title: "Multiple title tags",
    explanation:
      "Search engines may select an unintended title when a page declares more than one.",
    howToFix: "Keep one descriptive title element in the page head.",
  },
  "multiple-descriptions": {
    severity: "warning",
    title: "Multiple meta descriptions",
    explanation:
      "Multiple descriptions provide conflicting snippets to search engines.",
    howToFix: "Keep one relevant meta description in the page head.",
  },
  "multiple-canonicals": {
    severity: "warning",
    title: "Multiple canonical tags",
    explanation:
      "Different canonical tags can make the preferred URL ambiguous.",
    howToFix:
      "Keep one canonical link in the head pointing to the preferred page.",
  },
  "relative-canonical": {
    severity: "warning",
    title: "Relative canonical URL",
    explanation:
      "A relative canonical can resolve differently when the page moves or a base URL changes.",
    howToFix: "Use the full absolute HTTPS URL in the canonical link.",
  },
  "bad-canonical-target": {
    severity: "warning",
    title: "Canonical points to an unavailable page",
    explanation:
      "A canonical target that redirects, errors or is noindex may not be a suitable page to index.",
    howToFix: "Point the canonical at an indexable, successful final URL.",
  },
  "sitemap-noindex": {
    severity: "warning",
    title: "Sitemap includes a noindex page",
    explanation:
      "The sitemap asks search engines to discover a page that also asks not to be indexed.",
    howToFix:
      "Remove the URL from the sitemap or remove noindex if the page should appear in search.",
  },
  "sitemap-blocked": {
    severity: "warning",
    title: "Sitemap URL blocked by robots.txt",
    explanation:
      "Search engines cannot crawl a listed URL when the site blocks it in robots.txt.",
    howToFix: "Allow the URL in robots.txt or remove it from the sitemap.",
  },
  "sitemap-noncanonical": {
    severity: "warning",
    title: "Sitemap includes a non-canonical URL",
    explanation:
      "A sitemap should list the preferred URL rather than another URL that points away.",
    howToFix: "Replace the listed URL with its canonical URL.",
  },
  "hreflang-no-self": {
    severity: "warning",
    title: "Hreflang has no self-reference",
    explanation:
      "A language group without a self-reference can be harder for search engines to interpret.",
    howToFix:
      "Add an alternate hreflang link pointing to this page in its own language.",
  },
  "hreflang-no-default": {
    severity: "info",
    title: "Hreflang has no x-default",
    explanation:
      "A default version helps visitors whose language does not match another alternate.",
    howToFix: "Consider adding an x-default alternate for the fallback page.",
  },
  "hreflang-not-reciprocal": {
    severity: "warning",
    title: "Hreflang alternate does not link back",
    explanation:
      "Language alternates need reciprocal links so search engines can verify the relationship.",
    howToFix: "Add a matching return hreflang link on the alternate page.",
  },
  "hreflang-bad-target": {
    severity: "warning",
    title: "Hreflang points to a redirect or error",
    explanation:
      "A language alternate that redirects or errors may not be usable as the intended version.",
    howToFix: "Point each alternate directly to its live final URL.",
  },
  "http-links": {
    severity: "warning",
    title: "HTTPS page links to HTTP",
    explanation:
      "Visitors following an insecure link can lose HTTPS protection.",
    howToFix: "Link directly to the HTTPS version where available.",
  },
  "insecure-form": {
    severity: "critical",
    title: "Form submits over HTTP",
    explanation:
      "Information submitted to an HTTP endpoint can be exposed in transit.",
    howToFix:
      "Change the form action to an HTTPS endpoint and confirm it handles the submission.",
  },
  "internal-nofollow": {
    severity: "info",
    title: "Internal links use nofollow",
    explanation:
      "Nofollow can prevent crawlers from following links to useful pages.",
    howToFix:
      "Remove nofollow from ordinary internal links when those pages should be crawled.",
  },
  "images-missing-size": {
    severity: "warning",
    title: "Images lack width or height",
    explanation:
      "Without dimensions the browser may shift content while images load.",
    howToFix:
      "Set appropriate width and height attributes or reserve space with an aspect ratio.",
  },
  "large-image": {
    severity: "warning",
    title: "Large image",
    explanation:
      "Large image files can slow page loading, especially on mobile connections.",
    howToFix:
      "Resize or compress the image and use an efficient format appropriate for its content.",
  },
  "missing-hsts": {
    severity: "warning",
    title: "HTTPS page has no active HSTS",
    explanation:
      "Without HSTS, a browser may allow an insecure HTTP connection to the site.",
    howToFix:
      "Configure Strict-Transport-Security with a positive max-age after confirming HTTPS works across the intended domain.",
  },
  "broken-external-link": {
    severity: "warning",
    title: "External link returns an error",
    explanation:
      "A link to an unavailable external page can frustrate readers.",
    howToFix:
      "Check the destination and update or remove the link if it is no longer valid.",
  },
} as const;
