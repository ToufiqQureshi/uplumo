import type { UseQueryResult } from "@tanstack/react-query";
import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { FixView } from "./GrowthViews";
import type { InsightsResult, SearchInsights } from "./insightsApi";
import type { Report } from "./SearchPerformanceColumns";

const insights: SearchInsights = {
  connected: true,
  range: {
    startDate: "2026-10-06",
    endDate: "2026-11-03",
    prevStartDate: "2026-09-07",
    prevEndDate: "2026-10-05",
  },
  coverage: {
    queryPageRows: 20,
    truncated: false,
    decayAvailable: true,
    comparisonAvailable: true,
    brandTermsUsed: false,
  },
  segments: [],
  actions: [],
  losingPages: [],
  queryLosses: [],
  queryGains: [],
  decaying: [],
  ctrGaps: [],
  cannibalization: [],
  totals: { decaying: 0, ctrGaps: 0, cannibalization: 0 },
};

const fields: Partial<UseQueryResult<InsightsResult>> = {
  isPending: false,
  isPlaceholderData: false,
  data: insights,
};
// oxlint-disable-next-line typescript/no-unsafe-type-assertion -- only the fields InsightsGate reads
const query = fields as UseQueryResult<InsightsResult>;

function reportWithPreviousClicks(clicks: number): Report {
  const metrics = { clicks: 7, impressions: 109, ctr: 0.064, position: 62 };
  return {
    connected: true,
    range: { ...insights.range, comparisonAvailable: true },
    totals: metrics,
    prevTotals: { ...metrics, clicks },
    daily: [],
    strikingDistance: [],
    countries: [],
    pageChanges: [],
    queryLosses: [],
    queryGains: [],
  };
}

describe("FixView", () => {
  it("does not claim a comparison when the earlier period had no clicks", () => {
    const html = renderToStaticMarkup(
      <FixView
        projectId="p1"
        report={reportWithPreviousClicks(0)}
        query={query}
      />,
    );
    expect(html).toContain("nothing to compare yet");
    expect(html).not.toContain("against the previous period");
  });

  it("says nothing was lost when the earlier period had clicks", () => {
    const html = renderToStaticMarkup(
      <FixView
        projectId="p1"
        report={reportWithPreviousClicks(12)}
        query={query}
      />,
    );
    expect(html).toContain("No page or search lost clicks");
  });
});
