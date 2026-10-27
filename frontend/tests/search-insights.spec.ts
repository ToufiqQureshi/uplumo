import { expect, test } from "@playwright/test";
import { installGrowthFixtures } from "./search-growth.fixtures";

const range = {
  startDate: "2026-09-26",
  endDate: "2026-10-24",
  prevStartDate: "2026-08-28",
  prevEndDate: "2026-09-25",
  comparisonAvailable: true,
};
const loss = {
  impressions: 900,
  prevImpressions: 1000,
  position: 7.2,
  prevPosition: 3.1,
  ctr: 0.011,
  prevCtr: 0.05,
  reason: "rank_fell",
};

test("search performance says what fell, by how much, what is urgent, and what Google visitors did", async ({
  page,
}) => {
  const { paid } = await installGrowthFixtures(page);
  // Registered after the shared fixtures, so these answers win.
  await page.route("**/gsc/search-performance/report", (route) =>
    route.fulfill({
      json: {
        connected: true,
        range,
        totals: { clicks: 340, impressions: 20000, ctr: 0.017, position: 8 },
        prevTotals: {
          clicks: 500,
          impressions: 22000,
          ctr: 0.023,
          position: 6.5,
        },
        daily: [{ date: range.startDate, clicks: 12, impressions: 700 }],
        strikingDistance: [],
        countries: [],
        pageChanges: [
          {
            ...loss,
            page: "https://example.com/rooms",
            clicks: 40,
            prevClicks: 120,
            severity: "urgent",
          },
        ],
        queryLosses: [
          {
            ...loss,
            query: "resort in lonavala",
            clicks: 10,
            prevClicks: 50,
            severity: "urgent",
          },
          {
            ...loss,
            query: "pool villa",
            clicks: 16,
            prevClicks: 20,
            reason: "ctr_fell",
            severity: "watch",
          },
        ],
        queryGains: [
          {
            query: "weekend getaway",
            clicks: 25,
            prevClicks: 0,
            impressions: 400,
            prevImpressions: 0,
            position: 4.2,
            prevPosition: 0,
            ctr: 0.06,
            prevCtr: 0,
          },
        ],
      },
    }),
  );
  let ga4Reads = 0;
  await page.route("**/ga4/overview/organic", (route) => {
    ga4Reads++;
    return route.fulfill({
      json: {
        request: {
          resolvedDateRange: { startDate: "2026-09-27", endDate: "2026-10-24" },
        },
        current: {
          sessions: 300,
          activeUsers: 250,
          engagementRate: 0.62,
          keyEvents: 9,
        },
        previous: {
          sessions: 400,
          activeUsers: 320,
          engagementRate: 0.6,
          keyEvents: 12,
        },
        trend: [],
      },
    });
  });
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.goto("/p/e2e-project/search-performance");

  // What fell and by how much, in the verdict.
  await expect(
    page.getByRole("heading", { name: /Google clicks fell 32% \(500 → 340\)/ }),
  ).toBeVisible();
  await expect(page.getByText("2 urgent drops need a fix now")).toBeVisible();

  // Page losses say how bad.
  const pages = page.locator("#pages-to-fix");
  await expect(pages.getByText("−80 clicks (−67%)")).toBeVisible();
  await expect(pages.getByText("Urgent")).toBeVisible();

  // Lost and won searches, with the editor link carrying the search.
  const searches = page.locator("#searches-changed");
  await expect(searches.getByText("“resort in lonavala”")).toBeVisible();
  await expect(searches.getByText(/−40 clicks \(−80%\)/)).toBeVisible();
  await expect(searches.getByText("Watch")).toBeVisible();
  await expect(searches.getByText(/new: 25 clicks/)).toBeVisible();
  await expect(
    searches.getByRole("button", { name: "Win it back" }).first(),
  ).toHaveAttribute("href", /keyword=resort/);

  // What Google visitors did, from GA4, without any click.
  const ga4 = page.getByRole("region", {
    name: "What Google visitors did on your site",
  });
  await expect(ga4.getByText("300", { exact: true })).toBeVisible();
  // Visits 400 -> 300 and key events 12 -> 9 both fell 25%; engagement rose.
  await expect(ga4.getByText("▼ 25% vs before")).toHaveCount(2);
  await expect(ga4.getByText("▲ 3% vs before")).toBeVisible();
  await expect(
    ga4.getByText("3.0 key events per 100 Google visits."),
  ).toBeVisible();

  // Free Google reads only: nothing paid ran on page load, and GA4 was read once.
  expect(paid).toEqual([]);
  expect(ga4Reads).toBe(1);
});

test("phone layout keeps every number readable without sideways scrolling", async ({
  page,
}) => {
  await installGrowthFixtures(page);
  await page.setViewportSize({ width: 360, height: 800 });
  await page.goto("/p/e2e-project/search-performance");
  await expect(
    page.getByRole("heading", { name: "Search Performance" }).first(),
  ).toBeVisible();
  const overflow = await page.evaluate(
    () =>
      document.documentElement.scrollWidth -
      document.documentElement.clientWidth,
  );
  expect(overflow).toBeLessThanOrEqual(1);
});
