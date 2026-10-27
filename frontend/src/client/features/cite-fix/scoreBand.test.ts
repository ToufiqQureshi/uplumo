import { describe, expect, it } from "vitest";
import { scoreBand } from "./scoreBand";

describe("scoreBand", () => {
  it.each([
    [0, "Hard for AI to quote"],
    [49, "Hard for AI to quote"],
    [50, "Partly quotable"],
    [79, "Partly quotable"],
    [80, "Easy for AI to quote"],
    [100, "Easy for AI to quote"],
  ])("labels %i as %s", (score, label) => {
    expect(scoreBand(score).label).toBe(label);
  });
});
