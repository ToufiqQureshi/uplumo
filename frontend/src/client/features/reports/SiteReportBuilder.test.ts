import { describe, expect, it } from "vitest";
import { reportPeriod } from "./SiteReportBuilder";

describe("reportPeriod", () => {
  // Same window as the Go service: it ends three days before today (UTC).
  it.each([
    [7, "28 Oct – 3 Nov 2026"],
    [28, "7 Oct – 3 Nov 2026"],
    [90, "6 Aug – 3 Nov 2026"],
  ] as const)("%i days", (days, label) => {
    expect(reportPeriod(days, new Date("2026-11-06T23:30:00Z"))).toBe(label);
  });

  it("shows both years when the window crosses New Year", () => {
    expect(reportPeriod(28, new Date("2027-01-10T00:00:00Z"))).toBe(
      "11 Dec 2026 – 7 Jan 2027",
    );
  });
});
