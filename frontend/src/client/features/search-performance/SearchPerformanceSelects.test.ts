import { describe, expect, it } from "vitest";
import {
  monthPresetDates,
  subtractUTCMonths,
} from "@/client/features/search-performance/SearchPerformanceSelects";

const day = (year: number, month: number, date: number) =>
  new Date(Date.UTC(year, month - 1, date));

/** Search Console data lags three days behind today. */
const latestFor = (today: Date) =>
  new Date(today.getTime() - 3 * 24 * 60 * 60 * 1000);

describe("monthPresetDates", () => {
  it.each([
    // today, expected "Last month"
    [day(2026, 11, 20), "2026-10-01", "2026-10-31"], // 31-day month
    [day(2026, 3, 15), "2026-02-01", "2026-02-28"], // February
    [day(2024, 3, 15), "2024-02-01", "2024-02-29"], // leap February
    [day(2027, 1, 10), "2026-12-01", "2026-12-31"], // across a year
  ])("gives the whole previous month on %s", (today, startDate, endDate) => {
    expect(monthPresetDates("last_month", today, latestFor(today))).toEqual({
      startDate,
      endDate,
    });
  });

  it("ends last month at the newest data day when Google is still behind", () => {
    const today = day(2026, 11, 2);
    expect(monthPresetDates("last_month", today, latestFor(today))).toEqual({
      startDate: "2026-10-01",
      endDate: "2026-10-30",
    });
  });

  it("runs this month from the 1st to the newest data day", () => {
    const today = day(2026, 11, 20);
    expect(monthPresetDates("this_month", today, latestFor(today))).toEqual({
      startDate: "2026-11-01",
      endDate: "2026-11-17",
    });
  });

  it("offers no 'this month' before the month has a data day", () => {
    const today = day(2026, 11, 2);
    expect(monthPresetDates("this_month", today, latestFor(today))).toBeNull();
  });
});

describe("subtractUTCMonths", () => {
  it.each([
    [day(2026, 3, 31), 1, day(2026, 2, 28)],
    [day(2024, 3, 31), 1, day(2024, 2, 29)],
    [day(2026, 2, 15), 3, day(2025, 11, 15)],
    [day(2026, 5, 28), 16, day(2025, 1, 28)],
  ])("moves %s back %i months to %s", (from, months, expected) => {
    expect(subtractUTCMonths(from, months)).toEqual(expected);
  });
});
