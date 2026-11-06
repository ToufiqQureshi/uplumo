import { describe, expect, it } from "vitest";
import {
  MAX_BRAND_NAME_LENGTH,
  MAX_BRAND_NAMES,
  parseBrandNames,
} from "./brandNames";

describe("parseBrandNames", () => {
  it("splits on commas and new lines, trims, and drops empties", () => {
    expect(parseBrandNames(" Acme , acme shop\n\n, ")).toEqual([
      "Acme",
      "acme shop",
    ]);
  });

  it("drops repeats that differ only by case, keeping the first", () => {
    expect(parseBrandNames("Acme, ACME, acme")).toEqual(["Acme"]);
  });

  it("keeps non-latin names whole", () => {
    expect(parseBrandNames("आइबी, Café")).toEqual(["आइबी", "Café"]);
  });

  it("stops at the number of names the server accepts", () => {
    const text = Array.from({ length: 30 }, (_, i) => `brand${i}`).join(",");
    expect(parseBrandNames(text)).toHaveLength(MAX_BRAND_NAMES);
  });

  it("cuts a long name by characters, not bytes, so the server accepts it", () => {
    const [name] = parseBrandNames("ब".repeat(MAX_BRAND_NAME_LENGTH + 40));
    expect(Array.from(name ?? "")).toHaveLength(MAX_BRAND_NAME_LENGTH);
  });

  it("returns nothing for blank text", () => {
    expect(parseBrandNames("")).toEqual([]);
    expect(parseBrandNames(" , \n ")).toEqual([]);
  });
});
