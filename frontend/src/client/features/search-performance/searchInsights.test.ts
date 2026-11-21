import { beforeEach, describe, expect, it } from "vitest";
import type { Report } from "@/client/features/search-performance/SearchPerformanceColumns";
import { buildSearchInsights } from "@/client/features/search-performance/searchInsights";

type Loss = Report["queryLosses"][number];

const metrics = {
  clicks: 0,
  prevClicks: 0,
  impressions: 100,
  prevImpressions: 100,
  position: 8,
  prevPosition: 8,
  ctr: 0.1,
  prevCtr: 0.1,
};

function loss(overrides: Partial<Loss>): Loss {
  return {
    ...metrics,
    query: "blue shoes",
    reason: "rank_fell",
    severity: "watch",
    ...overrides,
  };
}

let report: Report;

beforeEach(() => {
  report = {
    connected: true,
    range: {
      startDate: "2026-10-01",
      endDate: "2026-10-31",
      prevStartDate: "2026-08-31",
      prevEndDate: "2026-09-30",
      comparisonAvailable: true,
    },
    totals: { clicks: 200, impressions: 5000, ctr: 0.04, position: 9 },
    prevTotals: { clicks: 150, impressions: 4000, ctr: 0.0375, position: 10 },
    daily: [],
    strikingDistance: [],
    countries: [],
    devices: [],
    pageChanges: [],
    queryLosses: [],
    queryGains: [],
  };
});

const noFilter = { countryFiltered: false };

describe("buildSearchInsights", () => {
  it("says nothing when the report has nothing worth saying", () => {
    expect(buildSearchInsights(report, noFilter)).toEqual([]);
  });

  it("names the biggest keyword gain with before and after clicks", () => {
    report.queryGains = [
      { ...metrics, query: "red boots", prevClicks: 4, clicks: 14 },
    ];
    expect(buildSearchInsights(report, noFilter)).toEqual([
      {
        text: '"red boots" gained 10 clicks (4 → 14).',
        open: { tab: "queries", queryText: "red boots", queryMatch: "equals" },
      },
    ]);
  });

  it.each([
    ["rank_fell", "its Google position dropped from 3.0 to 9.5"],
    ["fewer_searches", "fewer people searched for it"],
    ["ctr_fell", "fewer people clicked your result"],
    ["mixed", "its position and searches both changed"],
  ] as const)("explains a %s keyword drop in plain words", (reason, why) => {
    report.queryLosses = [
      loss({
        reason,
        prevClicks: 21,
        clicks: 20,
        prevPosition: 3,
        position: 9.5,
      }),
    ];
    expect(buildSearchInsights(report, noFilter)[0]?.text).toBe(
      `"blue shoes" lost 1 click (21 → 20): ${why}.`,
    );
  });

  it("shows the losing page by its path and filters to that exact page", () => {
    report.pageChanges = [
      {
        ...loss({ prevClicks: 30, clicks: 10 }),
        page: "https://example.test/blog/post?id=2",
      },
    ];
    const [line] = buildSearchInsights(report, noFilter);
    expect(line?.text).toMatch(/^\/blog\/post\?id=2 lost 20 clicks/);
    expect(line?.open).toEqual({
      tab: "pages",
      pageText: "https://example.test/blog/post?id=2",
      pageMatch: "equals",
    });
  });

  it("drops every period comparison when Google has no previous period", () => {
    report.range.comparisonAvailable = false;
    report.queryGains = [{ ...metrics, query: "a", prevClicks: 1, clicks: 9 }];
    report.queryLosses = [loss({ prevClicks: 9, clicks: 1 })];
    report.pageChanges = [
      { ...loss({ prevClicks: 9, clicks: 1 }), page: "https://example.test/" },
    ];
    expect(buildSearchInsights(report, noFilter)).toEqual([]);
  });

  it("gives the top country's share of clicks by its readable name", () => {
    report.countries = [
      { key: "ind", clicks: 125, impressions: 1, ctr: 0, position: 1 },
      { key: "bra", clicks: 75, impressions: 1, ctr: 0, position: 1 },
    ];
    expect(buildSearchInsights(report, noFilter)[0]?.text).toBe(
      "63% of your clicks come from India.",
    );
  });

  it("skips the country share for a single country or a chosen country", () => {
    report.countries = [
      { key: "ind", clicks: 200, impressions: 1, ctr: 0, position: 1 },
    ];
    expect(buildSearchInsights(report, noFilter)).toEqual([]);
    report.countries.push({
      key: "usa",
      clicks: 5,
      impressions: 1,
      ctr: 0,
      position: 1,
    });
    expect(buildSearchInsights(report, { countryFiltered: true })).toEqual([]);
  });

  it("keeps unicode keywords intact and stops at four lines", () => {
    report.queryGains = [
      { ...metrics, query: "जूते ऑनलाइन 👟", prevClicks: 0, clicks: 3 },
    ];
    report.queryLosses = [loss({ prevClicks: 9, clicks: 1 })];
    report.pageChanges = [
      { ...loss({ prevClicks: 9, clicks: 1 }), page: "https://example.test/a" },
    ];
    report.countries = [
      { key: "ind", clicks: 150, impressions: 1, ctr: 0, position: 1 },
      { key: "usa", clicks: 50, impressions: 1, ctr: 0, position: 1 },
    ];
    report.strikingDistance = [
      {
        query: "x",
        page: "https://example.test/x",
        clicks: 0,
        impressions: 50,
        position: 7,
      },
    ];
    const lines = buildSearchInsights(report, noFilter);
    expect(lines).toHaveLength(4);
    expect(lines[0]?.text).toContain('"जूते ऑनलाइन 👟" gained 3 clicks');
  });

  it("counts quick wins on positions 5 to 20", () => {
    report.strikingDistance = [
      {
        query: "x",
        page: "https://example.test/x",
        clicks: 0,
        impressions: 50,
        position: 7,
      },
      {
        query: "y",
        page: "https://example.test/y",
        clicks: 1,
        impressions: 40,
        position: 12,
      },
    ];
    expect(buildSearchInsights(report, noFilter)[0]?.text).toBe(
      "2 searches rank on positions 5–20: small improvements there bring the most new clicks.",
    );
  });
});
