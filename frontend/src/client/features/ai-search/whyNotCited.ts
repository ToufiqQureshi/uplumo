import { sortBy } from "remeda";
import type { TrackingAnswer } from "./trackingApi";

/**
 * Why AI does not cite the customer, worked out only from answers a check
 * already paid for: no provider call, no credits. Each reason names its
 * evidence (sites, pages, businesses) and one action.
 */

// Directories, marketplaces and review sites. A local business wins these
// answers by being listed and reviewed there, not by writing a blog post.
const LISTING_SITES = new Set([
  "justdial",
  "tripadvisor",
  "booking",
  "makemytrip",
  "goibibo",
  "agoda",
  "expedia",
  "hotels",
  "trivago",
  "cleartrip",
  "yatra",
  "airbnb",
  "zomato",
  "swiggy",
  "dineout",
  "eazydiner",
  "practo",
  "lybrate",
  "indiamart",
  "tradeindia",
  "sulekha",
  "urbancompany",
  "yelp",
  "trustpilot",
  "g2",
  "capterra",
  "clutch",
  "goodfirms",
  "glassdoor",
  "ambitionbox",
  "99acres",
  "magicbricks",
  "housing",
  "nobroker",
  "amazon",
  "flipkart",
  "yellowpages",
  "shiksha",
  "collegedunia",
  "carwale",
  "cardekho",
  "zigwheels",
]);

// Places where people talk. AI quotes them for "what do people say" answers.
const COMMUNITY_SITES = new Set([
  "reddit",
  "quora",
  "youtube",
  "wikipedia",
  "linkedin",
  "medium",
  "facebook",
  "instagram",
  "twitter",
  "x",
  "pinterest",
  "tiktok",
]);

type GapKind =
  | "listings"
  | "community"
  | "competitor_pages"
  | "named_not_linked"
  | "not_named";

export type GapAction =
  | { kind: "write"; label: string; keyword: string }
  | { kind: "external"; label: string; href: string }
  | { kind: "track"; label: string };

type CitationGap = {
  kind: GapKind;
  title: string;
  detail: string;
  /** Domains, page URLs or business names, strongest first. */
  evidence: string[];
  /** Answers that show this reason. */
  answers: number;
  action: GapAction;
};

type WhyNotCited = {
  answered: number;
  cited: number;
  gaps: CitationGap[];
};

const bare = (domain: string) =>
  domain
    .toLowerCase()
    .replace(/^www\./, "")
    .replace(/\.$/, "");

/** The known site a domain belongs to (in.hotels.com -> "hotels"), or "". */
const siteName = (domain: string) =>
  bare(domain)
    .split(".")
    .find((label) => LISTING_SITES.has(label) || COMMUNITY_SITES.has(label)) ??
  "";

const isOwn = (domain: string, own: string) =>
  bare(domain) === own || bare(domain).endsWith(`.${own}`);

function top<T>(counts: Map<T, number>, limit: number) {
  return sortBy([...counts], [([, n]) => n, "desc"])
    .slice(0, limit)
    .map(([key]) => key);
}

function bump<T>(counts: Map<T, number>, key: T) {
  counts.set(key, (counts.get(key) ?? 0) + 1);
}

// The content editor's keyword is at most 200 characters.
const asKeyword = (prompt: string) => prompt.trim().slice(0, 200);

export function whyNotCited(
  answers: TrackingAnswer[],
  ownDomain: string,
): WhyNotCited {
  const own = bare(ownDomain);
  const listings = new Map<string, number>();
  const listingAnswers = new Set<string>();
  const community = new Map<string, number>();
  const communityAnswers = new Set<string>();
  const competitorPages = new Map<string, number>();
  const competitorPrompts = new Map<string, number>();
  const competitorAnswers = new Set<string>();
  const namedPrompts = new Map<string, number>();
  const instead = new Map<string, number>();
  let answered = 0;
  let cited = 0;
  let namedNotLinked = 0;
  let notNamed = 0;

  for (const answer of answers) {
    if (answer.status !== "completed" || !answer.answer.trim()) continue;
    answered++;
    const you = answer.brands.find((b) => isOwn(b.domain, own));
    const youCited =
      you?.cited ||
      answer.citations.some(
        (c) => c.matchedBrand || (c.domain !== null && isOwn(c.domain, own)),
      );
    if (youCited) {
      cited++;
      continue;
    }
    if (you?.mentioned) {
      namedNotLinked++;
      bump(namedPrompts, answer.prompt);
    } else {
      notNamed++;
      for (const r of answer.recommended) bump(instead, r.name);
    }
    const seen = new Set<string>();
    for (const c of answer.citations) {
      if (!c.domain || seen.has(c.url)) continue;
      seen.add(c.url);
      const name = siteName(c.domain);
      if (LISTING_SITES.has(name)) {
        bump(listings, bare(c.domain));
        listingAnswers.add(answer.id);
      } else if (COMMUNITY_SITES.has(name)) {
        bump(community, bare(c.domain));
        communityAnswers.add(answer.id);
      } else {
        bump(competitorPages, c.url);
        bump(competitorPrompts, answer.prompt);
        competitorAnswers.add(answer.id);
      }
    }
  }

  const gaps: CitationGap[] = [];
  if (listingAnswers.size) {
    const sites = top(listings, 4);
    gaps.push({
      kind: "listings",
      title: "AI trusts listing and review sites you are not winning on",
      detail: `In ${listingAnswers.size} answers AI linked to ${sites.join(", ")} instead of your site. Get listed there with the same name, address and phone as your website, and ask happy customers for reviews.`,
      evidence: sites,
      answers: listingAnswers.size,
      action: {
        kind: "external",
        label: `Open ${sites[0]}`,
        href: `https://${sites[0]}`,
      },
    });
  }
  if (competitorAnswers.size) {
    const prompt = top(competitorPrompts, 1)[0];
    gaps.push({
      kind: "competitor_pages",
      title: "Other websites have a page that answers the question; you do not",
      detail: `In ${competitorAnswers.size} answers AI cited these pages. Write one page that answers "${prompt}" better: direct answer first, prices, location and an FAQ.`,
      evidence: top(competitorPages, 5),
      answers: competitorAnswers.size,
      action: {
        kind: "write",
        label: "Write this page",
        keyword: asKeyword(prompt),
      },
    });
  }
  if (namedNotLinked) {
    const prompt = top(namedPrompts, 1)[0];
    gaps.push({
      kind: "named_not_linked",
      title: "AI knows your name but has no page of yours to link",
      detail: `In ${namedNotLinked} answers AI named you without linking your site. Give it a page to cite: a clear answer to "${prompt}" on your own site.`,
      evidence: top(namedPrompts, 3),
      answers: namedNotLinked,
      action: {
        kind: "write",
        label: "Write the answer page",
        keyword: asKeyword(prompt),
      },
    });
  }
  if (notNamed && instead.size) {
    const names = top(instead, 5);
    gaps.push({
      kind: "not_named",
      title: "AI recommends other businesses by name",
      detail: `In ${notNamed} answers AI recommended ${names.slice(0, 3).join(", ")} and never named you. Compare yourself with them on your site, and track them as competitors to see when that changes.`,
      evidence: names,
      answers: notNamed,
      action: { kind: "track", label: "Track these competitors" },
    });
  }
  if (communityAnswers.size) {
    const sites = top(community, 4);
    gaps.push({
      kind: "community",
      title: "AI quotes what people say on community sites",
      detail: `In ${communityAnswers.size} answers AI linked to ${sites.join(", ")}. Answer real questions there and post short videos or guides that mention your business.`,
      evidence: sites,
      answers: communityAnswers.size,
      action: {
        kind: "external",
        label: `Open ${sites[0]}`,
        href: `https://${sites[0]}`,
      },
    });
  }
  return {
    answered,
    cited,
    gaps: sortBy(gaps, [(gap) => gap.answers, "desc"]).slice(0, 5),
  };
}
