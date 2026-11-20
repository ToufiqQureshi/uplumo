import { describe, it, expect, beforeEach } from "vitest";
import { buildSearchInsights } from "./searchInsights";

const baseReport = {
  range: { comparisonAvailable: true },
  totals: { clicks: 1000 },
  countries: [],
  queryGains: [],
  queryLosses: [],
  pageChanges: [],
  strikingDistance: [],
};

describe("buildSearchInsights", () => {
  beforeEach(() => {
    // Reset any state
  });

  it("should return empty array for empty report", () => {
    const report = {
      ...baseReport,
      range: { comparisonAvailable: false },
      totals: { clicks: 0 },
    };
    expect(buildSearchInsights(report)).toEqual([]);
  });

  it("should return empty array when comparison is not available", () => {
    const report = {
      ...baseReport,
      range: { comparisonAvailable: false },
      queryGains: [{ query: "test", clicks: 10, prevClicks: 5 }],
    };
    expect(buildSearchInsights(report)).toEqual([]);
  });

  it("should include biggest keyword gain when comparison available", () => {
    const report = {
      ...baseReport,
      queryGains: [{ query: "test query", clicks: 15, prevClicks: 5 }],
    };
    const insights = buildSearchInsights(report);
    expect(insights[0]?.text).toContain('"test query" gained 10 clicks');
    expect(insights[0]?.link?.param).toBe("queryText");
  });

  it("should include biggest keyword loss with reason", () => {
    const report = {
      ...baseReport,
      queryLosses: [
        {
          query: "declining query",
          clicks: 5,
          prevClicks: 15,
          reason: "rank_fell" as const,
        },
      ],
    };
    const insights = buildSearchInsights(report);
    expect(insights[0]?.text).toContain('"declining query" lost 10 clicks');
    expect(insights[0]?.text).toContain("position dropped");
  });

  it("should handle different loss reasons", () => {
    const reasons: Array<
      "rank_fell" | "fewer_searches" | "ctr_fell" | "mixed"
    > = ["rank_fell", "fewer_searches", "ctr_fell", "mixed"];
    for (const reason of reasons) {
      const report = {
        ...baseReport,
        queryLosses: [
          {
            query: "test",
            clicks: 5,
            prevClicks: 10,
            reason,
          },
        ],
      };
      const insights = buildSearchInsights(report);
      expect(insights[0]?.text).toContain("lost 5 clicks");
    }
  });

  it("should include biggest page drop", () => {
    const report = {
      ...baseReport,
      pageChanges: [
        {
          page: "https://example.com/blog/article?utm=test",
          clicks: 10,
          prevClicks: 25,
        },
      ],
    };
    const insights = buildSearchInsights(report);
    expect(insights[0]?.text).toContain("/blog/article lost 15 clicks");
    expect(insights[0]?.link?.param).toBe("pageText");
  });

  it("should include top country when >= 2 countries exist", () => {
    const report = {
      ...baseReport,
      totals: { clicks: 100 },
      countries: [
        { key: "usa", clicks: 62, impressions: 500, ctr: 0.1, position: 5 },
        { key: "gbr", clicks: 25, impressions: 200, ctr: 0.1, position: 6 },
      ],
    };
    const insights = buildSearchInsights(report);
    expect(insights[0]?.text).toContain("62% of your clicks come from");
  });

  it("should not include country when only 1 country exists", () => {
    const report = {
      ...baseReport,
      countries: [
        { key: "usa", clicks: 100, impressions: 500, ctr: 0.1, position: 5 },
      ],
    };
    const insights = buildSearchInsights(report);
    expect(
      insights.find((i) => i.text.includes("of your clicks come from")),
    ).toBeUndefined();
  });

  it("should include quick wins for striking distance keywords", () => {
    const report = {
      ...baseReport,
      strikingDistance: [
        {
          query: "test",
          page: "https://example.com",
          clicks: 1,
          impressions: 10,
          position: 5,
        },
        {
          query: "test2",
          page: "https://example.com",
          clicks: 1,
          impressions: 10,
          position: 18,
        },
      ],
    };
    const insights = buildSearchInsights(report);
    expect(
      insights.find((i) =>
        i.text.includes("2 keywords rank on positions 4–20"),
      ),
    ).toBeDefined();
  });

  it("should handle unicode query text", () => {
    const report = {
      ...baseReport,
      queryGains: [{ query: "مرحبا بالعالم", clicks: 10, prevClicks: 5 }],
    };
    const insights = buildSearchInsights(report);
    expect(insights[0]?.text).toContain("مرحبا بالعالم");
  });

  it("should return at most 4 insights", () => {
    const report = {
      ...baseReport,
      queryGains: [{ query: "query1", clicks: 10, prevClicks: 5 }],
      queryLosses: [
        {
          query: "query2",
          clicks: 5,
          prevClicks: 10,
          reason: "rank_fell" as const,
        },
      ],
      pageChanges: [
        { page: "https://example.com/page1", clicks: 10, prevClicks: 20 },
      ],
      countries: [
        { key: "usa", clicks: 50, impressions: 100, ctr: 0.5, position: 5 },
        { key: "gbr", clicks: 30, impressions: 100, ctr: 0.3, position: 6 },
      ],
      strikingDistance: [
        {
          query: "test",
          page: "https://example.com",
          clicks: 1,
          impressions: 10,
          position: 5,
        },
      ],
    };
    const insights = buildSearchInsights(report);
    expect(insights.length).toBeLessThanOrEqual(4);
  });

  it("should handle invalid report data", () => {
    const insights = buildSearchInsights({ invalid: "data" });
    expect(insights).toEqual([]);
  });

  it("should use singular click when delta is 1", () => {
    const report = {
      ...baseReport,
      queryGains: [{ query: "test", clicks: 6, prevClicks: 5 }],
    };
    const insights = buildSearchInsights(report);
    expect(insights[0]?.text).toContain("gained 1 click");
  });

  it("should use plural clicks when delta > 1", () => {
    const report = {
      ...baseReport,
      queryGains: [{ query: "test", clicks: 7, prevClicks: 5 }],
    };
    const insights = buildSearchInsights(report);
    expect(insights[0]?.text).toContain("gained 2 clicks");
  });

  it("should round country percentage correctly", () => {
    const report = {
      ...baseReport,
      totals: { clicks: 100 },
      countries: [
        { key: "usa", clicks: 33, impressions: 100, ctr: 0.33, position: 5 },
        { key: "gbr", clicks: 67, impressions: 100, ctr: 0.67, position: 6 },
      ],
    };
    const insights = buildSearchInsights(report);
    expect(insights[0]?.text).toContain("33% of your clicks");
  });
});
