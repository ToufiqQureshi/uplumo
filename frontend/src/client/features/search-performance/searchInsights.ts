import { z } from "zod";

// Schema for the report data we receive from the backend
const reportSchema = z.object({
  range: z.object({
    comparisonAvailable: z.boolean(),
  }),
  totals: z.object({
    clicks: z.number(),
  }),
  countries: z.array(
    z.object({
      key: z.string(),
      clicks: z.number(),
    }),
  ),
  queryGains: z.array(
    z.object({
      query: z.string(),
      clicks: z.number(),
      prevClicks: z.number(),
    }),
  ),
  queryLosses: z.array(
    z.object({
      query: z.string(),
      clicks: z.number(),
      prevClicks: z.number(),
      reason: z.enum(["rank_fell", "fewer_searches", "ctr_fell", "mixed"]),
    }),
  ),
  pageChanges: z.array(
    z.object({
      page: z.string(),
      clicks: z.number(),
      prevClicks: z.number(),
    }),
  ),
  strikingDistance: z.array(
    z.object({
      position: z.number(),
    }),
  ),
});

type Report = z.infer<typeof reportSchema>;

interface SearchInsight {
  text: string;
  link?: {
    param: string;
    value: string;
  };
}

/**
 * Build search insights from the report data.
 * Returns at most 4 one-line insights, each with real numbers and optional link target.
 * Returns an empty array if no insights can be built.
 */
export function buildSearchInsights(report: unknown): SearchInsight[] {
  let parsed: Report;
  try {
    parsed = reportSchema.parse(report);
  } catch {
    return [];
  }

  const insights: SearchInsight[] = [];

  // Biggest keyword gain
  if (
    parsed.range.comparisonAvailable &&
    parsed.queryGains.length > 0 &&
    parsed.queryGains[0]
  ) {
    const gain = parsed.queryGains[0];
    const delta = gain.clicks - gain.prevClicks;
    if (delta > 0) {
      insights.push({
        text: `"${gain.query}" gained ${delta} ${delta === 1 ? "click" : "clicks"} (${gain.prevClicks} → ${gain.clicks})`,
        link: { param: "queryText", value: gain.query },
      });
    }
  }

  // Biggest keyword drop
  if (
    parsed.range.comparisonAvailable &&
    parsed.queryLosses.length > 0 &&
    parsed.queryLosses[0]
  ) {
    const loss = parsed.queryLosses[0];
    const delta = loss.prevClicks - loss.clicks;
    if (delta > 0) {
      let reason = "";
      switch (loss.reason) {
        case "rank_fell":
          reason =
            " — its Google position dropped. Check if the content needs updating.";
          break;
        case "fewer_searches":
          reason =
            " — fewer people searched for it. This can be seasonal or permanent.";
          break;
        case "ctr_fell":
          reason =
            " — fewer people clicked your result. Try improving your title or description.";
          break;
        case "mixed":
          reason =
            " — position and searches both changed. Check the latest data.";
          break;
      }
      insights.push({
        text: `"${loss.query}" lost ${delta} ${delta === 1 ? "click" : "clicks"}${reason}`,
        link: { param: "queryText", value: loss.query },
      });
    }
  }

  // Biggest page drop
  if (parsed.pageChanges.length > 0 && parsed.pageChanges[0]) {
    const pageChange = parsed.pageChanges[0];
    const delta = pageChange.prevClicks - pageChange.clicks;
    if (delta > 0) {
      const pagePath = pageChange.page
        .replace(/^https?:\/\/[^/]+/, "")
        .split("?")[0];
      insights.push({
        text: `${pagePath} lost ${delta} ${delta === 1 ? "click" : "clicks"}`,
        link: { param: "pageText", value: pageChange.page },
      });
    }
  }

  // Top country (only if >= 2 countries and clicks > 0)
  if (
    parsed.countries.length >= 2 &&
    parsed.totals.clicks > 0 &&
    parsed.countries[0]
  ) {
    const topCountry = parsed.countries[0];
    const share = Math.round((topCountry.clicks / parsed.totals.clicks) * 100);
    const countryName = getCountryName(topCountry.key);
    if (countryName) {
      insights.push({
        text: `${share}% of your clicks come from ${countryName}`,
        link: { param: "country", value: topCountry.key.toUpperCase() },
      });
    }
  }

  // Quick wins: keywords in striking distance (positions 4-20)
  const strikingCount = parsed.strikingDistance.filter(
    (row) => row.position >= 4 && row.position <= 20,
  ).length;
  if (strikingCount > 0 && insights.length < 4) {
    insights.push({
      text: `${strikingCount} ${strikingCount === 1 ? "keyword" : "keywords"} rank on positions 4–20 — small improvements here bring the most new clicks`,
      link: { param: "tab", value: "queries" },
    });
  }

  // Return only the first 4 insights
  return insights.slice(0, 4);
}

/**
 * Get the display name for a country using Intl.DisplayNames.
 * GSC gives ISO alpha-3 lowercase codes (e.g., "usa", "gbr", "ind").
 */
function getCountryName(code: string): string | null {
  try {
    const displayNames = new Intl.DisplayNames(["en"], { type: "region" });
    // Convert lowercase alpha-3 to uppercase alpha-2
    const countryCode = code3to2(code.toUpperCase());
    if (!countryCode) return null;
    return displayNames.of(countryCode) || null;
  } catch {
    return null;
  }
}

/**
 * Convert ISO 3166-1 alpha-3 code to alpha-2.
 * Common conversions for testing.
 */
function code3to2(code3: string): string | null {
  const map: Record<string, string> = {
    USA: "US",
    GBR: "GB",
    CAN: "CA",
    AUS: "AU",
    IND: "IN",
    DEU: "DE",
    FRA: "FR",
    JPN: "JP",
    CHN: "CN",
    BRA: "BR",
    MEX: "MX",
    ZAF: "ZA",
    NLD: "NL",
    ESP: "ES",
    ITA: "IT",
    KOR: "KR",
    RUS: "RU",
    SGP: "SG",
    HKG: "HK",
    NZL: "NZ",
  };
  return map[code3] || null;
}
