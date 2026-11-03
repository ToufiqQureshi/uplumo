import { expect, it } from "vitest";
import { keywordsToTable } from "./utils";

it("exports rank change, type and intent that the table shows", () => {
  const base = {
    searchVolume: 10,
    traffic: 1,
    cpc: null,
    url: "https://example.com/a",
    relativeUrl: "/a",
    keywordDifficulty: 20,
  };
  const table = keywordsToTable([
    {
      ...base,
      keyword: "a",
      position: 3,
      previousPosition: 7,
      type: "paid",
      intent: "commercial",
    },
    { ...base, keyword: "b", position: 5, isNew: true },
  ]);
  expect(table.headers.slice(0, 5)).toEqual([
    "Keyword",
    "Rank",
    "Previous Rank",
    "Type",
    "Intent",
  ]);
  expect(table.rows[0].slice(0, 5)).toEqual(["a", 3, 7, "paid", "commercial"]);
  expect(table.rows[1].slice(0, 5)).toEqual(["b", 5, "new", null, null]);
});
