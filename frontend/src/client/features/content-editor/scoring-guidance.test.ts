import { describe, expect, it } from "vitest";
import { scoreContent, type ContentDraftInput } from "./scoring";

// Rules that follow Google's guidance; each case is a bug seen live.
const draft: ContentDraftInput = {
  title: "Technical SEO guide for site owners",
  targetKeyword: "technical SEO",
  slug: "technical-seo-guide",
  metaDescription: "A practical guide to technical SEO.",
  content:
    "Technical SEO makes sites easy to crawl. Read the [Google documentation](https://developers.google.com/search/docs) for details.",
};

describe("scoring follows Google's guidance", () => {
  it("gives no question points without brief questions, instead of free points", () => {
    const result = scoreContent("aeo", draft);
    const questions = result.checks.find((c) => c.id === "question-coverage");
    expect(questions?.weight).toBe(0);
    expect(questions?.detail).toContain("Not applicable");
  });

  // Live bug: two images and no links showed "2 external sources".
  it("never counts an image as a source", () => {
    const content =
      draft.content.replace(/\[[^\]]+\]\([^)]+\)/gu, "documentation") +
      "\n\n![Teal saree](https://m.media-amazon.com/images/I/x.jpg)";
    expect(
      scoreContent("aeo", { ...draft, content }).checks.find(
        (c) => c.id === "sources",
      )?.earned,
    ).toBe(0);
    expect(
      scoreContent("seo", { ...draft, content }).checks.find(
        (c) => c.id === "sources-internal",
      )?.earned,
    ).toBe(0);
  });

  // Live bug: the title is the page's H1, yet placement wanted a body H1.
  it("counts the title as the H1 and accepts the topic's words in any order", () => {
    const result = scoreContent("seo", {
      title: "Teal Green Ethnic Wear for Women: Saree or Anarkali?",
      targetKeyword: "teal green ethnic wear",
      slug: "teal-green-ethnic-wear",
      metaDescription: "Pick teal green ethnic wear for Diwali and weddings.",
      content:
        "Teal green ethnic wear works for every festival.\n\n## Ethnic wear in teal green: what to pick\nA saree or an Anarkali both work.",
    });
    expect(result.checks.find((c) => c.id === "topic-placement")?.earned).toBe(
      15,
    );
  });

  it.each([
    ["English", "Teal is a deep blue-green shade.", 10],
    ["Hinglish", "Teal ek gehra blue-green rang hota hai.", 10],
    ["a question only", "## What is teal?\nIt suits weddings.", 0],
  ])("finds a plain definition in %s", (_name, content, earned) => {
    const result = scoreContent("aeo", {
      ...draft,
      targetKeyword: "teal green ethnic wear",
      content,
    });
    expect(result.checks.find((c) => c.id === "definition")?.earned).toBe(
      earned,
    );
  });

  it.each([
    [
      "the inserted template",
      "## FAQ\n\n### Question people ask?\n\nAnswer in two or three sentences.",
      0,
    ],
    [
      "a real question",
      "## FAQ\n\n### Is teal good for dusky skin?\n\nYes, teal is a deep jewel tone that suits darker skin.",
      5,
    ],
  ])("scores an FAQ with %s", (_name, content, earned) => {
    expect(
      scoreContent("aeo", { ...draft, content }).checks.find(
        (c) => c.id === "faq",
      )?.earned,
    ).toBe(earned);
  });

  it.each([
    [
      "first-hand English",
      "Last Diwali we tested three teal sarees in daylight.",
      10,
    ],
    [
      "first-hand Hinglish",
      "Maine khud teal saree pehna aur sabne poocha.",
      10,
    ],
    ["generic", "Teal sarees are popular for festivals.", 0],
  ])("rewards %s experience", (_name, content, earned) => {
    expect(
      scoreContent("seo", { ...draft, content }).checks.find(
        (c) => c.id === "experience",
      )?.earned,
    ).toBe(earned);
  });
});
