import { describe, expect, it } from "vitest";
import { countryName } from "@/client/lib/countryNames";

describe("countryName", () => {
  it.each([
    ["ind", "India"],
    ["USA", "United States"],
    ["bra", "Brazil"],
    ["deu", "Germany"],
    ["nga", "Nigeria"],
  ])("names %s as %s", (code, name) => {
    expect(countryName(code)).toBe(name);
  });

  it("keeps an unknown code readable instead of blank", () => {
    expect(countryName("zzz")).toBe("Unknown");
    expect(countryName("qqq")).toBe("QQQ");
  });
});
