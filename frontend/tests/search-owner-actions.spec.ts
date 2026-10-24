import { expect, test } from "@playwright/test";
import { installGrowthFixtures } from "./search-growth.fixtures";

test("GSC alone provides an existing-page task and no GA4 analysis dead end", async ({
  page,
}, testInfo) => {
  const { requests, paid } = await installGrowthFixtures(page, {
    ga4Connected: false,
  });
  await page.goto("/p/e2e-project/search-performance");
  await expect(
    page.getByRole("heading", { name: "Your next search actions" }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", {
      name: "Review how this page answers “booking guide”",
    }),
  ).toBeVisible();
  await expect(
    page.getByRole("link", { name: "Connect GA4 to prioritize by enquiries" }),
  ).toHaveAttribute("href", "/p/e2e-project/settings/integrations");
  await expect(
    page.getByRole("button", { name: "Find opportunities" }),
  ).toHaveCount(0);
  await expect(
    page.getByText("Search trend", { exact: true }),
  ).not.toBeVisible();
  expect(requests.some((path) => path.includes("/ga4/growth/"))).toBe(false);
  expect(paid).toEqual([]);
  for (const width of [320, 768, 1024, 1440]) {
    await page.setViewportSize({ width, height: 900 });
    await expect(
      page.getByRole("button", { name: "Review this page in editor" }),
    ).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
    if (width === 320 || width === 1440)
      await page.screenshot({
        path: testInfo.outputPath(`search-actions-${width}.png`),
        fullPage: true,
      });
  }
  await page
    .getByText("Detailed search report and filters", { exact: true })
    .click();
  await expect(page.getByText("Search trend", { exact: true })).toBeVisible();
  await page
    .getByRole("button", { name: "Review this page in editor" })
    .click();
  await expect(
    page.getByRole("heading", { name: "Improve an existing page" }),
  ).toBeVisible();
  await expect(page.getByText("Related queries: booking guide")).toBeVisible();
  await expect(
    page.getByText(
      "GSC review candidate, ordered by returned impressions, not revenue.",
      { exact: false },
    ),
  ).toBeVisible();
  expect(paid).toEqual([]);
});

test("a linked query report opens its filters without extra disclosure steps", async ({
  page,
}) => {
  await installGrowthFixtures(page);
  await page.goto("/p/e2e-project/search-performance?tab=queries");
  await expect(
    page.getByRole("tab", { name: "Queries", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Filters", exact: true }),
  ).toBeVisible();
  await page.getByRole("tab", { name: /Striking distance/ }).click();
  await expect(
    page.getByRole("button", { name: "Filters", exact: true }),
  ).toBeVisible();
});
