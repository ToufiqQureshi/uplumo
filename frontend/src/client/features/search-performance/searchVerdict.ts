import type { Report } from "./SearchPerformanceColumns";

type Cause = "shown_less" | "clicked_less" | "rank_fell" | "grew" | "steady";

/**
 * One plain sentence about what happened to Google traffic and why, from the
 * report's own totals. No AI call: the same numbers the cards show.
 */
export function searchVerdict(report: Report): {
  cause: Cause | "no_comparison";
  headline: string;
  detail: string;
} {
  const { totals, prevTotals, range } = report;
  if (!range.comparisonAvailable || prevTotals.clicks <= 0) {
    return {
      cause: "no_comparison",
      headline: `Google sent you ${count(totals.clicks)} clicks in this period.`,
      detail:
        "There is no earlier period to compare with yet. The pages and quick wins below show where to grow.",
    };
  }
  const clicks = change(totals.clicks, prevTotals.clicks);
  const shown = change(totals.impressions, prevTotals.impressions);
  const rate = change(totals.ctr, prevTotals.ctr);
  const rankSlip = totals.position - prevTotals.position;
  const cause = whyChanged(clicks, shown, rate, rankSlip);
  const verb = clicks < 0 ? "fell" : "grew";
  const head = `Google clicks ${verb} ${pct(Math.abs(clicks))} (${count(prevTotals.clicks)} → ${count(totals.clicks)})`;
  switch (cause) {
    case "shown_less":
      return {
        cause,
        headline: `${head}. The main reason: Google showed your site ${pct(-shown)} less often.`,
        detail: `People who saw you clicked ${rate >= 0 ? "as much or more" : "a little less"}, so this is a visibility problem. Start with the pages below that lost the most.`,
      };
    case "rank_fell":
      return {
        cause,
        headline: `${head}. The main reason: your average Google rank slipped by ${rankSlip.toFixed(1)} places.`,
        detail:
          "Pages that drop down the results lose both views and clicks. Refresh the pages below first.",
      };
    case "clicked_less":
      return {
        cause,
        headline: `${head}. Google showed you as often, but ${pct(-rate)} fewer people clicked.`,
        detail:
          "Your titles may look less useful, or an AI Overview now answers the question. Rewrite titles and add direct answers on the pages below.",
      };
    case "grew":
      return {
        cause,
        headline: `${head}. Google showed you ${shown >= 0 ? `${pct(shown)} more` : `${pct(-shown)} less`} often.`,
        detail:
          "Keep going: the quick wins below are searches where you are almost on top.",
      };
    default:
      return {
        cause,
        headline: `Google clicks are steady (${count(prevTotals.clicks)} → ${count(totals.clicks)}, ${clicks >= 0 ? "+" : "−"}${pct(Math.abs(clicks))}).`,
        detail:
          "Nothing dropped sharply. The quick wins below are the cheapest way to grow.",
      };
  }
}

function whyChanged(
  clicks: number,
  shown: number,
  rate: number,
  rankSlip: number,
): Cause {
  if (Math.abs(clicks) < 0.05) return "steady";
  if (clicks > 0) return "grew";
  if (rankSlip >= 1) return "rank_fell";
  // Clicks = impressions x CTR: blame whichever fell further.
  return shown <= rate ? "shown_less" : "clicked_less";
}

const change = (now: number, before: number) =>
  before > 0 ? (now - before) / before : 0;
const pct = (fraction: number) => `${Math.round(fraction * 100)}%`;
const count = (value: number) =>
  new Intl.NumberFormat("en-IN").format(Math.round(value));

/** What to do about one losing page, by why it lost clicks. */
export const PAGE_REASON = {
  rank_fell: { label: "Rank fell", action: "Refresh this page" },
  fewer_searches: {
    label: "Fewer people searched",
    action: "Target new searches",
  },
  ctr_fell: { label: "Fewer clicks per view", action: "Rewrite title & intro" },
  mixed: { label: "Small drops everywhere", action: "Review this page" },
} as const satisfies Record<string, { label: string; action: string }>;

/** How urgent a loss is, in plain words, with a color class for its badge. */
export const SEVERITY = {
  urgent: {
    label: "Urgent",
    className: "bg-red-100 text-red-900 dark:bg-red-950 dark:text-red-100",
  },
  watch: {
    label: "Watch",
    className:
      "bg-amber-100 text-amber-900 dark:bg-amber-950 dark:text-amber-100",
  },
  minor: { label: "Small dip", className: "bg-muted text-foreground" },
} as const;

/** "−30 clicks (−60%)", or "lost all 12 clicks". */
export function lossText(before: number, after: number): string {
  if (after <= 0) return `lost all ${count(before)} clicks`;
  const lost = before - after;
  return `−${count(lost)} clicks (−${pct(before > 0 ? lost / before : 0)})`;
}

/** The urgent losses, pages and searches together, for the verdict. */
export function urgentCount(report: Report): number {
  return [...report.pageChanges, ...report.queryLosses].filter(
    (row) => row.severity === "urgent",
  ).length;
}
