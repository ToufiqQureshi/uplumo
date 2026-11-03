import { describe, expect, it } from "vitest";
import { formatMoney } from "./staffApi";

describe("formatMoney", () => {
  it("shows paise and cents as whole rupees and dollars", () => {
    expect(formatMoney(249900, "INR")).toBe("₹2,499");
    expect(formatMoney(3900, "USD")).toBe("$39");
    expect(formatMoney(0, "INR")).toBe("₹0");
  });
});
