// Directories and listing sites usually take a free sign-up, so they are the
// fastest links to win. The list mirrors ai-search/whyNotCited.ts listings.
const DIRECTORIES = [
  "justdial",
  "sulekha",
  "indiamart",
  "tradeindia",
  "yellowpages",
  "zomato",
  "swiggy",
  "magicpin",
  "tripadvisor",
  "yelp",
  "practo",
  "clutch",
  "goodfirms",
  "g2",
  "capterra",
  "trustpilot",
];
const COMMUNITY = ["reddit", "quora", "medium", "linkedin", "youtube"];
const NEWS = ["news", "times", "samachar", "patrika", "bhaskar", "herald"];

/** A rough kind of linking site, from its domain name, to suggest the approach. */
export function siteKind(domain: string) {
  const labels = domain.toLowerCase().split(".");
  if (labels.some((l) => DIRECTORIES.includes(l))) return "Directory";
  if (labels.some((l) => COMMUNITY.includes(l))) return "Community";
  if (NEWS.some((n) => domain.toLowerCase().includes(n))) return "News";
  if (labels.some((l) => l === "blog" || l.endsWith("blog"))) return "Blog";
  return "Website";
}
