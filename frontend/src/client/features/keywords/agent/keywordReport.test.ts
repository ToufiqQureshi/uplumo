import { describe, expect, it } from "vitest";
import { reportFromPart, reportTotals } from "./keywordReport";

const output = {
  summary: "s",
  keywords: [
    {
      keyword: "a",
      chance: "high",
      metrics: { searchVolume: 390, keywordDifficulty: 13 },
    },
    { keyword: "b", chance: "low", metrics: null },
  ],
};

describe("keyword report", () => {
  it("reads only a finished present_keywords part", () => {
    const part = {
      type: "tool-present_keywords",
      toolCallId: "1",
      state: "output-available",
      input: {},
      output,
    };
    expect(reportFromPart(part)?.keywords).toHaveLength(2);
    expect(
      reportFromPart({ ...part, type: "tool-research_keywords" }),
    ).toBeNull();
    expect(reportFromPart({ type: "text" })).toBeNull();
    expect(
      reportFromPart({
        ...part,
        output: { keywords: [{ keyword: "a", chance: "maybe" }] },
      }),
    ).toBeNull();
  });

  it("sums only researched numbers and estimates top-3 traffic at 10%", () => {
    const report = reportFromPart({
      type: "tool-present_keywords",
      output,
    });
    expect(reportTotals(report!.keywords)).toEqual({
      volume: 390,
      averageDifficulty: 13,
      top3Visits: 39,
    });
  });
});
