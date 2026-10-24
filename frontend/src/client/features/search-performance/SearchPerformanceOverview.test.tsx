import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { SearchPerformanceOverview } from "./SearchPerformanceOverview";
import type { Report } from "./SearchPerformanceColumns";

function render(overrides: Partial<Report> = {}) {
  const report: Report = {
    connected: true,
    range: {
      startDate: "2026-10-01",
      endDate: "2026-10-14",
      prevStartDate: "2026-09-17",
      prevEndDate: "2026-09-30",
      comparisonAvailable: true,
    },
    totals: { clicks: 60, impressions: 1000, ctr: 0.06, position: 9 },
    prevTotals: { clicks: 100, impressions: 1200, ctr: 0.083, position: 8 },
    daily: [],
    strikingDistance: [],
    countries: [],
    ...overrides,
  };
  return renderToStaticMarkup(
    <SearchPerformanceOverview report={report} onReviewPages={() => {}} />,
  );
}

describe("search overview evidence boundaries", () => {
  it("shows absolute click loss and the exact comparison dates without inventing its cause", () => {
    const html = render();
    expect(html).toContain("40 fewer clicks from Google");
    expect(html).toContain("2026-09-17 – 2026-09-30");
    expect(html).toContain("do not explain the cause");
    expect(html).toContain("No candidate pages");
  });
  it("does not claim a decline against an incomplete comparison", () => {
    const html = render({
      range: {
        startDate: "2025-06-01",
        endDate: "2025-06-14",
        prevStartDate: "",
        prevEndDate: "",
        comparisonAvailable: false,
      },
    });
    expect(html).not.toContain("fewer clicks");
    expect(html).toContain("comparison is unavailable");
  });
  it("bounds candidates, labels their scope, and does not emit unsafe links", () => {
    const html = render({
      strikingDistance: Array.from({ length: 5 }, (_, i) => ({
        page: i === 0 ? "javascript:alert(1)" : `https://example.test/${i}`,
        query: `query ${i}`,
        clicks: 1,
        impressions: 500 - i,
        position: 9,
      })),
    });
    expect(html.match(/<article /g)).toHaveLength(3);
    expect(html).toContain("Review all 5 pages");
    expect(html).toContain("Counts cover those queries only");
    expect(html).not.toContain('href="javascript:');
    expect(html).toContain('href="https://example.test/1"');
    expect(html).toContain("not predicted revenue");
  });
});
