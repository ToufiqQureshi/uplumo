import { describe, expect, it } from "vitest";
import type { Report } from "./SearchPerformanceColumns";
import { lossText, searchVerdict, urgentCount } from "./searchVerdict";

const metrics = (clicks: number, impressions: number, position: number) => ({
  clicks,
  impressions,
  ctr: impressions ? clicks / impressions : 0,
  position,
});

function report(
  now: ReturnType<typeof metrics>,
  before: ReturnType<typeof metrics>,
  comparisonAvailable = true,
): Report {
  return {
    connected: true,
    range: {
      startDate: "2026-09-25",
      endDate: "2026-10-23",
      prevStartDate: "2026-08-28",
      prevEndDate: "2026-09-24",
      comparisonAvailable,
    },
    totals: now,
    prevTotals: before,
    daily: [],
    strikingDistance: [],
    countries: [],
    pageChanges: [],
    queryLosses: [],
    queryGains: [],
  };
}

describe("searchVerdict", () => {
  it("blames fewer impressions when Google showed the site less (the live case)", () => {
    const v = searchVerdict(
      report(metrics(1441, 74663, 6.7), metrics(1720, 98760, 6.6)),
    );
    expect(v.cause).toBe("shown_less");
    expect(v.headline).toBe(
      "Google clicks fell 16% (1,720 → 1,441). The main reason: Google showed your site 24% less often.",
    );
  });

  it("blames rank when the average position slipped a full place", () => {
    expect(
      searchVerdict(report(metrics(800, 9000, 8.2), metrics(1000, 10000, 6.9)))
        .cause,
    ).toBe("rank_fell");
  });

  it("blames the click rate when views held but clicks fell", () => {
    const v = searchVerdict(
      report(metrics(600, 10000, 5), metrics(1000, 10000, 5)),
    );
    expect(v.cause).toBe("clicked_less");
    expect(v.headline).toContain("40% fewer people clicked");
  });

  it("calls small moves steady and rises growth", () => {
    expect(
      searchVerdict(report(metrics(1020, 10000, 5), metrics(1000, 10000, 5)))
        .cause,
    ).toBe("steady");
    expect(
      searchVerdict(report(metrics(1300, 12000, 5), metrics(1000, 10000, 5)))
        .cause,
    ).toBe("grew");
  });

  it("does not invent a comparison when there is none", () => {
    expect(
      searchVerdict(report(metrics(50, 900, 9), metrics(0, 0, 0))).cause,
    ).toBe("no_comparison");
    expect(
      searchVerdict(report(metrics(50, 900, 9), metrics(40, 800, 9), false))
        .cause,
    ).toBe("no_comparison");
  });
});

describe("loss wording and urgency", () => {
  it("says how much was lost", () => {
    expect(lossText(50, 20)).toBe("−30 clicks (−60%)");
    expect(lossText(12, 0)).toBe("lost all 12 clicks");
  });

  it("counts urgent pages and searches together", () => {
    const base = report(metrics(10, 100, 5), metrics(50, 100, 5));
    const loss = {
      clicks: 1,
      prevClicks: 10,
      impressions: 1,
      prevImpressions: 1,
      position: 1,
      prevPosition: 1,
      ctr: 0,
      prevCtr: 0,
      reason: "rank_fell" as const,
    };
    expect(
      urgentCount({
        ...base,
        pageChanges: [
          { ...loss, page: "/a", severity: "urgent" },
          { ...loss, page: "/b", severity: "watch" },
        ],
        queryLosses: [{ ...loss, query: "q", severity: "urgent" }],
      }),
    ).toBe(2);
  });
});
