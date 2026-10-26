import { describe, expect, it } from "vitest";
import { addQuestions, starterQuestions } from "./starterQuestions";

const q = (text: string) => ({ text, topic: "General", paused: false });
const texts = (prompts: { text: string }[]) => prompts.map((p) => p.text);

describe("starterQuestions", () => {
  it("asks local questions that name the business type and city", () => {
    const questions = starterQuestions({
      brand: "Shalimar Hotel",
      business: " hotel ",
      city: "Kemps Corner, Mumbai",
      competitors: ["Hotel Kemps Corner", " ", "Trident", "Taj"],
    });
    expect(texts(questions)).toEqual([
      "Best hotel in Kemps Corner, Mumbai",
      "Top rated hotel in Kemps Corner, Mumbai with good reviews",
      "Affordable hotel in Kemps Corner, Mumbai",
      "Which hotel in Kemps Corner, Mumbai do customers recommend, and why?",
      "Is Shalimar Hotel in Kemps Corner, Mumbai good? What do customers say?",
      "What is Shalimar Hotel known for?",
      "Shalimar Hotel vs Hotel Kemps Corner: which is better?",
      "Shalimar Hotel vs Trident: which is better?",
    ]);
    expect(questions.every((p) => !p.paused && p.topic)).toBe(true);
  });

  it("works without a city or brand, and needs a business type", () => {
    expect(
      texts(
        starterQuestions({
          brand: "",
          business: "dentist",
          city: "",
          competitors: ["Rival"],
        }),
      ),
    ).toEqual([
      "Best dentist",
      "Top rated dentist with good reviews",
      "Affordable dentist",
      "Which dentist do customers recommend, and why?",
    ]);
    expect(
      starterQuestions({
        brand: "A",
        business: " ",
        city: "Pune",
        competitors: [],
      }),
    ).toEqual([]);
  });
});

describe("addQuestions", () => {
  it("replaces the blank starter row and skips questions already tracked", () => {
    expect(
      texts(
        addQuestions(
          [q(""), q("Best hotel")],
          [q("best hotel"), q("Cheap hotel")],
        ),
      ),
    ).toEqual(["Best hotel", "Cheap hotel"]);
  });

  it("never goes over the tracker's question limit", () => {
    const current = Array.from({ length: 29 }, (_, i) => q(`Question ${i}`));
    const result = addQuestions(current, [q("New 1"), q("New 2")]);
    expect(result).toHaveLength(30);
    expect(result.at(-1)?.text).toBe("New 1");
  });
});
