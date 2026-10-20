export function recommendedNextStep(index: {
  robotsTxtState?: string;
  indexingState?: string;
  pageFetchState?: string;
  googleCanonical?: string;
  userCanonical?: string;
  verdict?: string;
}): string {
  if (index.indexingState === "BLOCKED_BY_META_TAG") {
    return "If this page should appear in Google, remove its noindex meta tag.";
  }
  if (index.indexingState === "BLOCKED_BY_HTTP_HEADER") {
    return "If this page should appear in Google, remove the X-Robots-Tag noindex header.";
  }
  if (
    index.robotsTxtState === "DISALLOWED" ||
    index.pageFetchState === "BLOCKED_ROBOTS_TXT"
  ) {
    return "If this page should appear in Google, allow Googlebot to crawl it in robots.txt.";
  }
  if (
    index.googleCanonical &&
    index.userCanonical &&
    index.googleCanonical !== index.userCanonical
  ) {
    return "Google selected a different canonical URL. Review your intended page, canonical tag, sitemap and internal links before changing anything.";
  }
  if (index.verdict === "PASS")
    return "Google's stored result indicates this URL is indexed. Review search performance; no indexing fix is indicated by this result.";
  switch (index.pageFetchState) {
    case "NOT_FOUND":
      return "Restore the page or redirect its URL to the closest relevant page.";
    case "SOFT_404":
      return "Check that the page has useful content and returns the right HTTP status.";
    case "SERVER_ERROR":
      return "Fix the server error so Google can fetch this page.";
    case "ACCESS_DENIED":
    case "ACCESS_FORBIDDEN":
      return "Check that the page is publicly accessible to Googlebot.";
    case "REDIRECT_ERROR":
      return "Fix the redirect destination or redirect chain.";
    default:
      return "Open the full URL Inspection report in Search Console to review Google's reason.";
  }
}
