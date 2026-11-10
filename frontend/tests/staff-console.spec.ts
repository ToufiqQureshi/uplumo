import { expect, test } from "@playwright/test";
import { installGrowthFixtures } from "./search-growth.fixtures";

const customers = Array.from({ length: 40 }, (_, index) => ({
  userId: `fixture-user-${index}`,
  email: `customer-${index}@example.com`,
  name: `Customer ${index}`,
  emailVerified: true,
  blocked: false,
  signedUpAt: "2026-10-01T10:00:00Z",
  workspace: "Fixture workspace",
  role: "owner",
  plan: "pro",
  subscriptionStatus: "active",
  activeSites: 1,
  domains: ["example.com"],
  gscConnected: true,
  ga4Connected: false,
  interestedFeatures: '["AI visibility"]',
  workFor: "Clients",
  clientWebsiteCount: "1",
  foundVia: "Search",
  mcpSetupIntent: "",
  onboardingCompletedAt: "2026-10-01T10:00:00Z",
}));

for (const width of [375, 1280]) {
  test(`staff content scrolls independently of navigation at ${width}px`, async ({
    page,
  }) => {
    await installGrowthFixtures(page);
    await page.route("**/api/v1/admin/access", (route) =>
      route.fulfill({ json: { staff: true } }),
    );
    await page.route("**/api/v1/admin/customers", (route) =>
      route.fulfill({ json: { customers } }),
    );
    await page.setViewportSize({ width, height: 640 });
    await page.goto("/staff/customers");

    const main = page.getByRole("main");
    const navigation = page.getByRole("navigation", { name: "Staff console" });
    await expect(main.getByRole("row")).toHaveCount(customers.length + 1);
    await main.getByRole("textbox", { name: "Search customers" }).hover();
    await page.mouse.wheel(0, 1200);
    await expect
      .poll(() => main.evaluate((element) => element.scrollTop))
      .toBeGreaterThan(0);
    await expect(
      navigation.getByRole("link", { name: "Customers" }),
    ).toBeInViewport();
    await expect.poll(() => page.evaluate(() => window.scrollY)).toBe(0);

    await main.focus();
    await page.keyboard.press("End");
    await expect(main.getByText("customer-39@example.com")).toBeInViewport();

    if (width === 375) {
      const table = main.locator("[data-slot=table-container]");
      await expect
        .poll(() =>
          table.evaluate((element) => {
            element.scrollLeft = element.scrollWidth;
            return element.scrollLeft;
          }),
        )
        .toBeGreaterThan(0);
      await expect
        .poll(() =>
          main.evaluate((element) => element.scrollWidth - element.clientWidth),
        )
        .toBe(0);
    }
  });
}

test("staff access can recover after a temporary error without showing staff data first", async ({
  page,
}) => {
  await installGrowthFixtures(page);
  let requests = 0;
  await page.route("**/api/v1/admin/access", async (route) => {
    requests += 1;
    if (requests === 1) {
      await route.fulfill({
        status: 503,
        json: { error: { code: "unavailable", message: "Unavailable" } },
      });
      return;
    }
    await route.fulfill({ json: { staff: false } });
  });
  await page.goto("/staff/customers");
  await expect(page.getByRole("alert")).toContainText(
    "Could not load Staff Console",
  );
  await expect(
    page.getByRole("navigation", { name: "Staff console" }),
  ).toHaveCount(0);
  await page.getByRole("button", { name: "Try again" }).click();
  await expect(page.getByText("404", { exact: true })).toBeVisible();
  expect(requests).toBe(2);
});
