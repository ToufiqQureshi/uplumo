import { expect, test, type Page } from "@playwright/test";
import { installGrowthFixtures } from "./search-growth.fixtures";

const range = {
  startDate: "2026-09-26",
  endDate: "2026-10-24",
  prevStartDate: "2026-08-28",
  prevEndDate: "2026-09-25",
  comparisonAvailable: true,
};
const row = { impressions: 1, ctr: 0.1, position: 4 };
const change = {
  impressions: 900,
  prevImpressions: 1000,
  position: 7.2,
  prevPosition: 3.1,
  ctr: 0.011,
  prevCtr: 0.05,
};

/** The Google Search page with one report; returns every API path read. */
async function openPage(page: Page, search = "") {
  const fixtures = await installGrowthFixtures(page);
  const tableRequests: unknown[] = [];
  // Registered after the shared fixtures, so these answers win.
  await page.route("**/gsc/search-performance/report", (route) =>
    route.fulfill({
      json: {
        connected: true,
        range,
        totals: { clicks: 200, impressions: 20000, ctr: 0.01, position: 8 },
        prevTotals: {
          clicks: 150,
          impressions: 18000,
          ctr: 0.008,
          position: 9,
        },
        daily: [{ date: range.startDate, clicks: 12, impressions: 700 }],
        strikingDistance: [
          {
            query: "villa with pool",
            page: "https://example.com/villa",
            clicks: 2,
            impressions: 300,
            position: 8,
          },
        ],
        countries: [
          { key: "ind", clicks: 150, ...row },
          { key: "bra", clicks: 50, ...row },
        ],
        devices: [
          { key: "MOBILE", clicks: 160, ...row },
          { key: "DESKTOP", clicks: 40, ...row },
        ],
        pageChanges: [
          {
            ...change,
            reason: "rank_fell",
            severity: "urgent",
            page: "https://example.com/rooms",
            clicks: 40,
            prevClicks: 120,
          },
        ],
        queryLosses: [
          {
            ...change,
            reason: "fewer_searches",
            severity: "watch",
            query: "resort in lonavala",
            clicks: 10,
            prevClicks: 50,
          },
        ],
        queryGains: [
          { ...change, query: "pool villa", clicks: 30, prevClicks: 5 },
        ],
      },
    }),
  );
  await page.route("**/gsc/search-performance/table", (route) => {
    const input: unknown = route.request().postDataJSON();
    tableRequests.push(input);
    const pages =
      typeof input === "object" &&
      input !== null &&
      "dimension" in input &&
      input.dimension === "page";
    return route.fulfill({
      json: {
        connected: true,
        dimension: pages ? "page" : "query",
        page: 1,
        pageSize: 25,
        hasNextPage: false,
        totalCount: 1,
        rows: [
          pages
            ? { key: "https://example.com/rooms", clicks: 40, ...row }
            : { key: "pool villa", clicks: 30, ...row },
        ],
      },
    });
  });
  await page.route("**/indexnow/get", (route) =>
    route.fulfill({ json: { included: true, domain: "example.com", key: "" } }),
  );
  await page.goto(`/p/e2e-project/search-performance${search}`);
  await expect(
    page.getByRole("heading", { name: "Google Search" }),
  ).toBeVisible();
  return { ...fixtures, tableRequests };
}

test("shows totals, plain insights and keywords, with no paid call", async ({
  page,
}) => {
  const { paid } = await openPage(page);

  const changed = page.locator("section", { hasText: "What changed" });
  await expect(
    changed.getByText('"pool villa" gained 25 clicks (5 → 30).'),
  ).toBeVisible();
  await expect(
    changed.getByText(
      '"resort in lonavala" lost 40 clicks (50 → 10): fewer people searched for it.',
    ),
  ).toBeVisible();
  await expect(
    changed.getByText(
      "/rooms lost 80 clicks (120 → 40): its Google position dropped from 3.1 to 7.2.",
    ),
  ).toBeVisible();
  await expect(
    changed.getByText("75% of your clicks come from India."),
  ).toBeVisible();
  // Four lines at most: the quick-win line is the fifth and is left out.
  await expect(changed.getByRole("listitem")).toHaveCount(4);

  await expect(page.getByRole("cell", { name: "pool villa" })).toBeVisible();
  // The old screens are gone.
  for (const name of ["Fix", "Grow", "Explore", "Conversions"])
    await expect(page.getByRole("tab", { name, exact: true })).toHaveCount(0);
  await expect(page.getByRole("button", { name: /Ask SAM/u })).toHaveCount(0);
  expect(paid).toEqual([]);
});

test("countries and devices show names and shares, and a row filters the page", async ({
  page,
}) => {
  await openPage(page);

  await page.getByRole("tab", { name: "Countries" }).click();
  const india = page.getByRole("row", { name: /India/u });
  await expect(india).toContainText("75%");
  await expect(page.getByRole("row", { name: /Brazil/u })).toContainText("25%");
  await india.getByRole("button", { name: "India" }).click();
  await expect(page).toHaveURL(/country=ind/u);

  await page.getByRole("tab", { name: "Devices" }).click();
  await expect(page.getByRole("row", { name: /Mobile/u })).toContainText("80%");
  await page.getByRole("button", { name: "Desktop" }).click();
  await expect(page).toHaveURL(/device=DESKTOP/u);
});

test("pages have one-click Edit, and old links still open the page", async ({
  page,
}) => {
  await openPage(page, "?view=fix&tab=striking");

  await page.getByRole("tab", { name: "Pages" }).click();
  await expect(
    page.getByRole("button", { name: "Edit: https://example.com/rooms" }),
  ).toBeVisible();

  await page.getByRole("tab", { name: "IndexNow" }).click();
  await expect(page.getByText(/IndexNow/u).first()).toBeVisible();
});

test("Last month sets the whole previous calendar month", async ({ page }) => {
  await openPage(page);

  await page.getByRole("combobox", { name: "Date range" }).click();
  await page.getByRole("option", { name: "Last month" }).click();
  await expect(page).toHaveURL(/startDate=\d{4}-\d{2}-01/u);
  await expect(page.getByRole("combobox", { name: "Date range" })).toHaveText(
    /Last month/u,
  );
});
