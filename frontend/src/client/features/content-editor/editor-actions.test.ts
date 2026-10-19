import { describe, expect, it } from "vitest";
import { nextEdits, taskInstructions } from "./editor-actions";
import type { ScoreCheck } from "./scoring";
const check = (
  id: string,
  weight: number,
  earned: number,
  passed = false,
): ScoreCheck => ({
  id,
  weight,
  earned,
  passed,
  label: id,
  detail: `Evidence for ${id}`,
});
describe("focused editorial tasks", () => {
  it("never spends attention on passed or inapplicable checks", () => {
    const source = [
      check("readability", 0, 0),
      check("finished", 30, 30, true),
      check("links", 8, 4),
      check("depth", 10, 0),
      check("terms", 30, 20),
      check("minor", 1, 0),
    ];
    expect(nextEdits(source).map((c) => c.id)).toEqual([
      "depth",
      "terms",
      "links",
    ]);
    expect(source[0].id).toBe("readability");
  });
  it("passes selected evidence to the writer inside its input cap", () => {
    const instructions = taskInstructions(
      {
        label: "Sources",
        detail: "The selected paragraph needs a reliable source.",
      },
      "Keep the customer's wording.",
    );
    expect(instructions).toContain("Keep the customer's wording.");
    expect(instructions).toContain(
      "The selected paragraph needs a reliable source.",
    );
    expect(instructions).toContain("Do not invent");
    expect(
      taskInstructions(
        { label: "x".repeat(1000), detail: "y".repeat(3000) },
        "z".repeat(2000),
      ).length,
    ).toBeLessThanOrEqual(2000);
    expect(taskInstructions(undefined, "Original instructions")).toBe(
      "Original instructions",
    );
  });
});
