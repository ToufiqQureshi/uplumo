import { describe, expect, it } from "vitest";
import { formatAmount, refundStatus } from "./receiptsApi";

describe("formatAmount", () => {
  it.each([
    [249900, "INR", "₹2,499"],
    [12345678, "INR", "₹1,23,456.78"],
    [1900, "USD", "$19"],
    [1950, "USD", "$19.50"],
    [0, "USD", "$0"],
  ] as const)("%d %s is %s", (subunits, currency, want) => {
    expect(formatAmount(subunits, currency)).toBe(want);
  });
});

describe("refundStatus", () => {
  it.each([
    [{ amount: 100, refunded: 0 }, "Paid"],
    [{ amount: 100, refunded: 40 }, "Part refunded"],
    [{ amount: 100, refunded: 100 }, "Refunded"],
  ])("%o is %s", (payment, want) => {
    expect(refundStatus(payment)).toBe(want);
  });
});
