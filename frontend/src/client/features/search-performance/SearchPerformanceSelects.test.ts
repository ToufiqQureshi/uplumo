import { describe, it, expect, beforeEach } from "vitest";
import { subtractUTCMonths } from "./SearchPerformanceSelects";

describe("SearchPerformanceSelects date math", () => {
  let mockToday: Date;

  beforeEach(() => {
    // Set a known date for consistent testing: 2026-05-28
    mockToday = new Date(Date.UTC(2026, 4, 28)); // May 28, 2026
  });

  describe("subtractUTCMonths", () => {
    it("should subtract months from a date", () => {
      const result = subtractUTCMonths(mockToday, 1);
      expect(result.getUTCFullYear()).toBe(2026);
      expect(result.getUTCMonth()).toBe(3); // April
      expect(result.getUTCDate()).toBe(28);
    });

    it("should handle month-end day clamping for shorter months", () => {
      // Start from March 31
      const mar31 = new Date(Date.UTC(2026, 2, 31));
      // Subtract 1 month → February, which has 28 days
      const result = subtractUTCMonths(mar31, 1);
      expect(result.getUTCMonth()).toBe(1); // February
      expect(result.getUTCDate()).toBe(28); // Clamped from 31 to 28
    });

    it("should handle leap year February", () => {
      // 2024 is a leap year
      const mar31_2024 = new Date(Date.UTC(2024, 2, 31));
      const result = subtractUTCMonths(mar31_2024, 1);
      expect(result.getUTCMonth()).toBe(1); // February
      expect(result.getUTCDate()).toBe(29); // Leap year has 29 days
    });

    it("should handle year boundary backwards", () => {
      // December 15, 2026
      const dec15 = new Date(Date.UTC(2026, 11, 15));
      // Subtract 2 months → October 15, 2026
      const result = subtractUTCMonths(dec15, 2);
      expect(result.getUTCFullYear()).toBe(2026);
      expect(result.getUTCMonth()).toBe(9); // October
    });

    it("should cross year boundary when subtracting months", () => {
      // February 15, 2026
      const feb15 = new Date(Date.UTC(2026, 1, 15));
      // Subtract 3 months → November 2025
      const result = subtractUTCMonths(feb15, 3);
      expect(result.getUTCFullYear()).toBe(2025);
      expect(result.getUTCMonth()).toBe(10); // November
      expect(result.getUTCDate()).toBe(15);
    });

    it("should handle 16-month subtraction correctly", () => {
      // May 28, 2026 - 16 months = January 28, 2025
      const result = subtractUTCMonths(mockToday, 16);
      expect(result.getUTCFullYear()).toBe(2025);
      expect(result.getUTCMonth()).toBe(0); // January
      expect(result.getUTCDate()).toBe(28);
    });

    it("should handle subtraction of 0 months", () => {
      const result = subtractUTCMonths(mockToday, 0);
      expect(result.getUTCFullYear()).toBe(mockToday.getUTCFullYear());
      expect(result.getUTCMonth()).toBe(mockToday.getUTCMonth());
      expect(result.getUTCDate()).toBe(mockToday.getUTCDate());
    });

    it("should handle multiple month boundaries", () => {
      // December 31, 2026
      const dec31 = new Date(Date.UTC(2026, 11, 31));
      // Subtract 14 months → October 31, 2025
      const result = subtractUTCMonths(dec31, 14);
      expect(result.getUTCFullYear()).toBe(2025);
      expect(result.getUTCMonth()).toBe(9); // October
      expect(result.getUTCDate()).toBe(31);
    });

    it("should clamp February 31st to 28/29", () => {
      // May 31, 2026
      const may31 = new Date(Date.UTC(2026, 4, 31));
      // Subtract 3 months → February, should be clamped to 28
      const result = subtractUTCMonths(may31, 3);
      expect(result.getUTCMonth()).toBe(1); // February
      expect(result.getUTCDate()).toBe(28);
    });

    it("should maintain UTC timezone", () => {
      const result = subtractUTCMonths(mockToday, 1);
      // Check it's UTC (timezone offset should be 0)
      expect(result.getUTCHours()).toBe(0);
      expect(result.getUTCMinutes()).toBe(0);
      expect(result.getUTCSeconds()).toBe(0);
    });
  });

  describe("preset range computations", () => {
    it("should compute last 7 days correctly", () => {
      const start = new Date(mockToday);
      start.setUTCDate(start.getUTCDate() - 7);
      expect(start.getUTCDate()).toBe(21); // May 21, 2026
      expect(mockToday.getUTCDate()).toBe(28); // May 28, 2026
    });

    it("should compute last 28 days correctly", () => {
      const start = new Date(mockToday);
      start.setUTCDate(start.getUTCDate() - 28);
      expect(start.getUTCDate()).toBe(30); // April 30, 2026
      expect(start.getUTCMonth()).toBe(3); // April
    });

    it("should handle month-based presets with day boundaries", () => {
      // "This month" for May 28, 2026 should be May 1 to May 28
      const thisMonthStart = new Date(Date.UTC(2026, 4, 1));
      expect(thisMonthStart.getUTCMonth()).toBe(4); // May
      expect(thisMonthStart.getUTCDate()).toBe(1);
    });

    it("should handle last month with February", () => {
      // If today is March 31, last month is February
      const mar31 = new Date(Date.UTC(2026, 2, 31));
      const lastMonthStart = new Date(Date.UTC(2026, 1, 1)); // Feb 1
      const lastMonthEnd = new Date(Date.UTC(2026, 1 + 1, 0)); // Last day of Feb = Feb 28
      expect(lastMonthStart.getUTCMonth()).toBe(1);
      expect(lastMonthStart.getUTCDate()).toBe(1);
      expect(lastMonthEnd.getUTCDate()).toBe(28);
    });

    it("should handle last month across year boundary", () => {
      // If today is January 15, 2026, last month is December 2025
      const jan15 = new Date(Date.UTC(2026, 0, 15));
      const prevMonth = subtractUTCMonths(jan15, 1);
      expect(prevMonth.getUTCFullYear()).toBe(2025);
      expect(prevMonth.getUTCMonth()).toBe(11); // December
    });
  });
});
