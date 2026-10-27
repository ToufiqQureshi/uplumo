import { describe, expect, it } from "vitest";
import { batchMaxCredits, defaultSelection } from "./agentSelection";

const offer = {
  available: true,
  planAllowed: true,
  quickMaxCredits: 25,
  deepMaxCredits: 60,
  maxPages: 20,
  defaultPages: 2,
};
const page = (url: string, score: number) => ({
  url,
  title: "",
  score,
  failed: [],
});

describe("defaultSelection", () => {
  it("picks the weakest pages below 80, up to the default", () => {
    const pages = [page("a", 10), page("b", 79), page("c", 30)];
    expect(defaultSelection(pages, offer)).toEqual(["a", "b"]);
  });

  it("selects nothing when every page is already good", () => {
    expect(defaultSelection([page("a", 80), page("b", 95)], offer)).toEqual([]);
  });
});

describe("batchMaxCredits", () => {
  it("charges every page at its mode's cap", () => {
    expect(batchMaxCredits(5, "deep", offer)).toBe(300);
    expect(batchMaxCredits(5, "quick", offer)).toBe(125);
    expect(batchMaxCredits(0, "deep", offer)).toBe(0);
  });
});
