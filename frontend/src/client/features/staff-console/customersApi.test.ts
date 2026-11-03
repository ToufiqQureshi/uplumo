import { describe, expect, it } from "vitest";
import { interestsText } from "./customersApi";

describe("interestsText", () => {
  it("joins saved interest arrays and keeps plain or empty text", () => {
    expect(interestsText('["Rank tracking","Site audit"]')).toBe(
      "Rank tracking, Site audit",
    );
    expect(interestsText("Other text")).toBe("Other text");
    expect(interestsText("")).toBe("");
  });
});
