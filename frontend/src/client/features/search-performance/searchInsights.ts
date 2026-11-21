import type { Report } from "@/client/features/search-performance/SearchPerformanceColumns";
import { countryName } from "@/client/lib/countryNames";
import type { SearchPerformanceSearch } from "@/types/schemas/search-performance";

type SearchInsight = {
  text: string;
  /** The filters that show the rows behind this line. */
  open: Partial<SearchPerformanceSearch>;
};

const MAX_INSIGHTS = 4;

type Loss = Pick<
  Report["queryLosses"][number],
  "reason" | "position" | "prevPosition"
>;

function clicks(count: number): string {
  return `${count} ${count === 1 ? "click" : "clicks"}`;
}

function lossReason(loss: Loss): string {
  switch (loss.reason) {
    case "rank_fell":
      return `its Google position dropped from ${loss.prevPosition.toFixed(1)} to ${loss.position.toFixed(1)}`;
    case "fewer_searches":
      return "fewer people searched for it";
    case "ctr_fell":
      return "fewer people clicked your result";
    case "mixed":
      return "its position and searches both changed";
  }
}

function pagePath(url: string): string {
  try {
    const parsed = new URL(url);
    return `${parsed.pathname}${parsed.search}` || "/";
  } catch {
    return url;
  }
}

/** Up to four plain lines about the period, built only from the report the
 *  page already loaded (no extra request). A line is left out when the data
 *  behind it is missing, so an empty list means "nothing worth saying". */
export function buildSearchInsights(
  report: Report,
  { countryFiltered }: { countryFiltered: boolean },
): SearchInsight[] {
  const insights: SearchInsight[] = [];
  const compare = report.range.comparisonAvailable;

  const gain = compare ? report.queryGains[0] : undefined;
  if (gain && gain.clicks > gain.prevClicks) {
    insights.push({
      text: `"${gain.query}" gained ${clicks(gain.clicks - gain.prevClicks)} (${gain.prevClicks} → ${gain.clicks}).`,
      open: { tab: "queries", queryText: gain.query, queryMatch: "equals" },
    });
  }

  const loss = compare ? report.queryLosses[0] : undefined;
  if (loss && loss.prevClicks > loss.clicks) {
    insights.push({
      text: `"${loss.query}" lost ${clicks(loss.prevClicks - loss.clicks)} (${loss.prevClicks} → ${loss.clicks}): ${lossReason(loss)}.`,
      open: { tab: "queries", queryText: loss.query, queryMatch: "equals" },
    });
  }

  const page = compare ? report.pageChanges[0] : undefined;
  if (page && page.prevClicks > page.clicks) {
    insights.push({
      text: `${pagePath(page.page)} lost ${clicks(page.prevClicks - page.clicks)} (${page.prevClicks} → ${page.clicks}): ${lossReason(page)}.`,
      open: { tab: "pages", pageText: page.page, pageMatch: "equals" },
    });
  }

  // Country rows ignore the country filter, the totals do not, so a share is
  // only true when no country is chosen.
  const top = report.countries[0];
  if (
    !countryFiltered &&
    report.countries.length >= 2 &&
    top &&
    report.totals.clicks > 0
  ) {
    const share = Math.round((top.clicks / report.totals.clicks) * 100);
    insights.push({
      text: `${share}% of your clicks come from ${countryName(top.key)}.`,
      open: { tab: "countries" },
    });
  }

  const quickWins = report.strikingDistance.length;
  if (quickWins > 0) {
    insights.push({
      text: `${quickWins} ${quickWins === 1 ? "search ranks" : "searches rank"} on positions 5–20: small improvements there bring the most new clicks.`,
      open: { tab: "queries" },
    });
  }

  return insights.slice(0, MAX_INSIGHTS);
}
