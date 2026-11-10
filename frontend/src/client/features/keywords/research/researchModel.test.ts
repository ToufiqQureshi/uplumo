import { describe, expect, it } from "vitest";
import type { KeywordResearchRow } from "@/types/keywords";
import {
  filterIdeas,
  findSeedRow,
  formatChange,
  formatCount,
  formatUpdated,
  hasFilters,
  NO_FILTERS,
  serpFeatureName,
  trendOldestFirst,
} from "./researchModel";

function row(over: Partial<KeywordResearchRow>): KeywordResearchRow {
  return {
    keyword: "seo audit tool",
    searchVolume: 1000,
    trend: [],
    keywordDifficulty: 40,
    cpc: 1.5,
    competition: 0.4,
    intent: "commercial",
    ...over,
  };
}

const rows = [
  row({
    keyword: "seo audit tool",
    searchVolume: 12100,
    keywordDifficulty: 64,
  }),
  row({
    keyword: "free seo audit checklist",
    searchVolume: 1900,
    keywordDifficulty: 24,
    intent: "informational",
  }),
  row({ keyword: "seo audit", searchVolume: null, keywordDifficulty: null }),
  row({
    keyword: "Auditoría SEO gratis",
    searchVolume: 90,
    keywordDifficulty: 5,
    intent: "transactional",
  }),
];

describe("findSeedRow", () => {
  it.each([
    ["seo audit tool", "seo audit tool"],
    ["  SEO   Audit  Tool ", "seo audit tool"],
    ["auditoría seo gratis", "Auditoría SEO gratis"],
  ])("finds %j", (seed, want) => {
    expect(findSeedRow(rows, seed)?.keyword).toBe(want);
  });

  it("returns null when the data has no row for the seed", () => {
    expect(findSeedRow(rows, "crm")).toBeNull();
    expect(findSeedRow([], "seo")).toBeNull();
  });
});

describe("filterIdeas", () => {
  const keywords = (f: Parameters<typeof filterIdeas>[1]) =>
    filterIdeas(rows, f).map((r) => r.keyword);

  it("keeps everything without filters", () => {
    expect(keywords(NO_FILTERS)).toHaveLength(4);
    expect(hasFilters(NO_FILTERS)).toBe(false);
  });

  it("needs every word, in any order and case", () => {
    expect(keywords({ ...NO_FILTERS, text: "CHECKLIST free" })).toEqual([
      "free seo audit checklist",
    ]);
    expect(keywords({ ...NO_FILTERS, text: "gratis auditoría" })).toEqual([
      "Auditoría SEO gratis",
    ]);
  });

  it("filters by intent", () => {
    expect(
      keywords({ ...NO_FILTERS, intents: ["informational", "transactional"] }),
    ).toEqual(["free seo audit checklist", "Auditoría SEO gratis"]);
  });

  it("hides keywords whose number is unknown when that number is filtered", () => {
    expect(keywords({ ...NO_FILTERS, maxDifficulty: 30 })).toEqual([
      "free seo audit checklist",
      "Auditoría SEO gratis",
    ]);
    expect(keywords({ ...NO_FILTERS, minVolume: 1900 })).toEqual([
      "seo audit tool",
      "free seo audit checklist",
    ]);
  });

  it("treats 0 as a real limit", () => {
    expect(keywords({ ...NO_FILTERS, maxDifficulty: 0 })).toEqual([]);
    expect(hasFilters({ ...NO_FILTERS, maxDifficulty: 0 })).toBe(true);
  });
});

describe("labels and numbers", () => {
  it.each([
    ["ai_overview", "AI Overview"],
    ["people_also_ask", "People also ask"],
    ["local_pack", "Map pack"],
    ["top_stories", "Top stories"],
    ["some_new_type", "Some new type"],
  ])("names %s", (type, want) => {
    expect(serpFeatureName(type)).toBe(want);
  });

  it.each([
    [null, "—"],
    [0, "0"],
    [9_999, "9,999"],
    [12_100, "12.1K"],
    [2_400_000, "2.4M"],
  ])("counts %j", (value, want) => {
    expect(formatCount(value)).toBe(want);
  });

  it.each([
    [null, "—"],
    [0, "0%"],
    [18, "+18%"],
    [-7, "-7%"],
  ])("changes %j", (value, want) => {
    expect(formatChange(value)).toBe(want);
  });

  it("orders the trend oldest first with month names", () => {
    const trend = trendOldestFirst(
      row({
        trend: [
          { year: 2026, month: 1, searchVolume: 30 },
          { year: 2025, month: 12, searchVolume: 20 },
          { year: 2025, month: 11, searchVolume: 10 },
        ],
      }),
    );
    expect(trend).toEqual([
      { label: "Nov 2025", volume: 10 },
      { label: "Dec 2025", volume: 20 },
      { label: "Jan 2026", volume: 30 },
    ]);
  });

  it.each([
    ["2026-10-14 09:12:33 +00:00", "Oct 2026"],
    ["2026-10-14T09:12:33Z", "Oct 2026"],
    ["not a date", "—"],
    [null, "—"],
  ])("updated %j", (value, want) => {
    expect(formatUpdated(value)).toBe(want);
  });
});
