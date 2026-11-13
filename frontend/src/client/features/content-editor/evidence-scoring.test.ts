import { describe, expect, it } from "vitest";
import { scoreContent, type ContentBrief } from "./scoring";
import { toScoringBrief } from "./brief-view";
import { freshWithinTwelveMonths } from "./scoring-signals";

const draft = {
  title: "Storage guide",
  targetKeyword: "storage",
  slug: "storage",
  metaDescription: "Storage choices",
  content: "",
};
const brief: ContentBrief = {
  collectedAt: "2026-11-01",
  terms: [
    { term: "durability", min: 5, max: 10, domains: 3 },
    { term: "recovery", min: 1, max: 3, domains: 3 },
  ],
  subtopics: ["durability"],
  questions: [],
  competitorTexts: [],
};
const points = (content: string, id: string) =>
  scoreContent("seo", { ...draft, content }, brief).checks.find(
    (c) => c.id === id,
  )?.earned;

describe("evidence-based scoring regressions", () => {
  it("does not reward repeating a term to meet competitor frequency", () => {
    expect(points("durability", "term-coverage")).toBe(
      points("durability ".repeat(8), "term-coverage"),
    );
    expect(points("unrelated", "term-coverage")).toBe(0);
  });
  it("requires prose under the matching heading", () => {
    expect(points("## durability\n\n## next\nOther section", "subtopics")).toBe(
      0,
    );
    expect(
      points(
        "## durability\n\nCompare how each storage system handles drive failure, restoration, backup copies and regular recovery testing before choosing a provider.",
        "subtopics",
      ),
    ).toBe(30);
  });
  it("preserves domain and full page lengths through the brief adapter", () => {
    const mapped = toScoringBrief(
      {
        keyword: "storage",
        locationCode: 2840,
        languageCode: "en",
        fetchedAt: "2026-11-01",
        terms: [],
        subtopics: [],
        questions: [],
        competitors: [
          {
            url: "https://source.test",
            title: "Title",
            headings: [],
            wordCount: 2000,
            text: "truncated text",
          },
        ],
      },
      "https://www.example.com",
    );
    expect(mapped.competitorWordCounts).toEqual([2000]);
    const result = scoreContent(
      "seo",
      { ...draft, content: "[Own page](https://blog.example.com/guide)" },
      mapped,
    );
    expect(
      result.checks.find((c) => c.id === "sources-internal")?.detail,
    ).toContain("0 links to other sites, 1 to your own pages");
    expect(result.checks.find((c) => c.id === "depth")?.detail).toContain(
      "2000–2000",
    );
  });
  it("checks copy overlap in AEO as well as SEO", () => {
    const content = Array.from({ length: 120 }, (_, i) => `distinct${i}`).join(
      " ",
    );
    const result = scoreContent(
      "aeo",
      { ...draft, content },
      { ...brief, competitorTexts: [content] },
    );
    expect(result.capReason).toContain("phrase overlap");
  });
  it("does not count future or rolled-over dates as reviewed", () => {
    const now = new Date("2026-11-01T12:00:00Z");
    expect(freshWithinTwelveMonths("2027-01-01", now)).toBe(false);
    expect(freshWithinTwelveMonths("2026-02-30", now)).toBe(false);
    expect(freshWithinTwelveMonths("2026-10-31", now)).toBe(true);
  });
});
