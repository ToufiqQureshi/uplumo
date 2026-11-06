import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import type { SearchInsights } from "./insightsApi";
import { DoFirst, SegmentsCard } from "./OverviewInsights";

type Action = SearchInsights["actions"][number];

const gap: Action = {
  kind: "ctr_gap",
  subject: "red shoes",
  page: "https://example.test/a",
  impactClicks: 1234,
  basis: "estimated",
};
const lost: Action = {
  kind: "losing_page",
  subject: "https://example.test/shoes/red",
  page: "https://example.test/shoes/red",
  impactClicks: 40,
  basis: "measured",
};

describe("DoFirst", () => {
  it("is hidden when there is nothing to do", () => {
    expect(
      renderToStaticMarkup(<DoFirst actions={[]} onOpen={vi.fn()} />),
    ).toBe("");
  });

  it("says in plain words what each action is and where its number comes from", () => {
    const html = renderToStaticMarkup(
      <DoFirst actions={[lost, gap]} onOpen={vi.fn()} />,
    );
    expect(html).toContain("/shoes/red");
    expect(html).toContain("Lost 40 clicks");
    expect(html).toContain("“red shoes”");
    expect(html).toContain("estimated from your own numbers");
    expect(html).toContain("1,234");
  });
});

describe("SegmentsCard", () => {
  const base: SearchInsights = {
    connected: true,
    range: {
      startDate: "2026-04-27",
      endDate: "2026-05-25",
      prevStartDate: "2026-03-29",
      prevEndDate: "2026-04-26",
    },
    coverage: {
      queryPageRows: 10_000,
      truncated: true,
      decayAvailable: true,
      comparisonAvailable: true,
      brandTermsUsed: false,
    },
    segments: [
      {
        kind: "all",
        queries: 3,
        clicks: 55,
        impressions: 500,
        ctr: 0.11,
        position: 4,
      },
      {
        kind: "long_tail",
        queries: 1,
        clicks: 2,
        impressions: 100,
        ctr: 0.02,
        position: 7,
      },
    ],
    actions: [],
    losingPages: [],
    queryLosses: [],
    queryGains: [],
    decaying: [],
    ctrGaps: [],
    cannibalization: [],
    totals: { decaying: 0, ctrGaps: 0, cannibalization: 0 },
  };

  it("asks for brand names instead of guessing a brand", () => {
    const html = renderToStaticMarkup(
      <SegmentsCard insights={base} brandText="" onSaveBrands={vi.fn()} />,
    );
    expect(html).toContain("Tell us your brand names");
    expect(html).not.toContain("Non-branded");
    expect(html).toContain("Long-tail");
  });

  it("says when the read stopped at its row limit", () => {
    const html = renderToStaticMarkup(
      <SegmentsCard insights={base} brandText="" onSaveBrands={vi.fn()} />,
    );
    expect(html).toContain("Only the first 10,000");
  });

  it("is hidden when no search was read", () => {
    const empty = {
      ...base,
      segments: base.segments.map((segment) => ({ ...segment, queries: 0 })),
    };
    expect(
      renderToStaticMarkup(
        <SegmentsCard insights={empty} brandText="" onSaveBrands={vi.fn()} />,
      ),
    ).toBe("");
  });
});
