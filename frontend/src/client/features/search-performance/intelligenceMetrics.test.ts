import { describe, expect, it } from "vitest";
import { intelligenceMetrics } from "./intelligenceMetrics";
import type { GrowthReport, GrowthPage } from "./growthApi";

const page = (sessions: number, rate: number): GrowthPage => ({
  page: `https://example.test/${sessions}`,
  priority: 1,
  queries: [],
  findings: [],
  previous: null,
  current: {
    clicks: 10,
    impressions: 100,
    ctr: 0.1,
    position: 9,
    sessions,
    keyEvents: 3,
    keyEventRate: rate,
    engagementRate: 0.6,
    revenue: 10,
  },
});
function report(
  pages: GrowthPage[],
  overrides: Partial<GrowthReport> = {},
): GrowthReport {
  return {
    id: "r",
    generatedAt: "2026-10-24",
    range: { startDate: "2026-09-01", endDate: "2026-09-28" },
    previousRange: { startDate: "2026-08-04", endDate: "2026-08-31" },
    siteUrl: "https://example.test",
    propertyId: "properties/1",
    currency: "USD",
    timeZone: "UTC",
    brandTerms: [],
    warnings: [],
    limited: false,
    truncated: false,
    matchedPages: pages.length,
    gscPageCount: pages.length,
    ga4PageCount: pages.length,
    pages,
    ...overrides,
  };
}
describe("intelligence metric denominators and coverage", () => {
  it("weights rates by sessions, not pages or event counts", () => {
    const result = intelligenceMetrics(
      report([page(100, 0.5), page(900, 0.1)]),
    );
    expect(result.outcomeRate).toBeCloseTo(0.14);
    expect(result.sessions).toBe(1000);
    expect(result.events).toBe(6);
  });
  it("does not substitute all-event rate when the selected goal rate is missing", () => {
    expect(
      intelligenceMetrics(
        report([page(100, 0.5)], { goalEvent: "generate_lead" }),
      ).outcomeRate,
    ).toBeNull();
  });
  it("preserves unavailable GA4 values while retaining GSC evidence", () => {
    const item = page(100, 0.1);
    item.current.sessions = null;
    const result = intelligenceMetrics(report([item]));
    expect(result.sessions).toBeNull();
    expect(result.events).toBeNull();
    expect(result.clicks).toBe(10);
  });
  it("withholds rates, events and revenue from limited reports", () => {
    const result = intelligenceMetrics(
      report([page(100, 0.2)], { limited: true }),
    );
    expect(result.outcomeRate).toBeNull();
    expect(result.events).toBeNull();
    expect(result.revenue).toBeNull();
  });
  it("does not compare partial prior page coverage", () => {
    const result = intelligenceMetrics(report([page(100, 0.2)]), true);
    expect(result.clicks).toBeNull();
    expect(result.comparableAnalytics).toBe(false);
  });
  it("blocks GA4 deltas when matched page membership changes", () => {
    const item = page(100, 0.2);
    item.previous = { ...item.current, sessions: null };
    expect(intelligenceMetrics(report([item])).comparableAnalytics).toBe(false);
  });
  it("keeps zero sessions and empty reports from creating a false rate", () => {
    expect(intelligenceMetrics(report([page(0, 0)])).outcomeRate).toBeNull();
    expect(intelligenceMetrics(report([])).clicks).toBeNull();
  });
});
