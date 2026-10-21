import { describe, expect, it } from "vitest";
import { parseTitleMetaOptions } from "./title-meta";

describe("parseTitleMetaOptions", () => {
  it("reads the format the AI returned live", () => {
    const text = `### Titles and Meta Descriptions for "Teal Green Ethnic Wear for Women"

1. **Title:** Why Choose Teal Green Ethnic Wear for Festivals?
   **Meta Description:** Discover why teal green ethnic wear is perfect for Indian festivals.

2. **Title:** Saree vs Anarkali: Which Teal Look Suits You?
   **Meta Description:** Compare teal silk sarees and teal Anarkali suits.`;
    expect(parseTitleMetaOptions(text)).toEqual([
      {
        title: "Why Choose Teal Green Ethnic Wear for Festivals?",
        description:
          "Discover why teal green ethnic wear is perfect for Indian festivals.",
      },
      {
        title: "Saree vs Anarkali: Which Teal Look Suits You?",
        description: "Compare teal silk sarees and teal Anarkali suits.",
      },
    ]);
  });

  it("reads the plain format and ignores a description without a title", () => {
    expect(
      parseTitleMetaOptions(
        "Description: orphan\nTitle: Teal saree guide\nDescription: Pick a teal saree.",
      ),
    ).toEqual([
      { title: "Teal saree guide", description: "Pick a teal saree." },
    ]);
  });

  it("returns nothing for free text", () => {
    expect(parseTitleMetaOptions("Here are some ideas for you.")).toEqual([]);
  });
});
