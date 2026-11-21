import { describe, expect, it } from "vitest";
import {
  chooseReportPeriod,
  reportPeriodError,
  reportPeriodLabel,
} from "./SiteReportPeriod";

describe("reportPeriod", () => {
  // Same window as the Go service: it ends three days before today (UTC).
  it.each([
    [7, "28 Oct – 3 Nov 2026"],
    [28, "7 Oct – 3 Nov 2026"],
    [90, "6 Aug – 3 Nov 2026"],
  ] as const)("%i days", (days, label) => {
    expect(
      reportPeriodLabel({ key: days }, new Date("2026-11-06T23:30:00Z")),
    ).toBe(label);
  });

  it("shows both years when the window crosses New Year", () => {
    expect(
      reportPeriodLabel({ key: 28 }, new Date("2027-01-10T00:00:00Z")),
    ).toBe("11 Dec 2026 – 7 Jan 2027");
  });

  it("keeps the complete previous calendar month even before its data is ready", () => {
    const now = new Date("2027-01-01T18:00:00Z");
    const range = chooseReportPeriod("last-month", { key: 28 }, now);
    expect(range).toEqual({
      key: "last-month",
      startDate: "2026-12-01",
      endDate: "2026-12-31",
    });
    expect(reportPeriodError(range, now)).toContain("29 Dec 2026");
    expect(reportPeriodLabel(range)).toBe("1 Dec – 31 Dec 2026");
  });
});
