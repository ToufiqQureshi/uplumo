import { expect, test, type Page } from "@playwright/test";
import { installGrowthFixtures } from "./search-growth.fixtures";

async function installAnalytics(page: Page, failClicks = false) {
  await installGrowthFixtures(page);
  const requests: URL[] = [];
  await page.route("**/analytics/**", async (route) => {
    const url = new URL(route.request().url());
    requests.push(url);
    if (url.pathname.endsWith("/site"))
      return route.fulfill({
        json: {
          id: 1,
          siteKey: "demo-key",
          snippet:
            '<script defer data-site="demo-key" src="https://uplumo.test/t.js"></script>',
        },
      });
    if (url.pathname.endsWith("/summary")) {
      const date = url.searchParams.get("to")!;
      return route.fulfill({
        json: {
          visitors: 120,
          pageviews: 200,
          sessions: 140,
          engagedSessions: 70,
          bounceRate: 0.5,
          totalEngagementMs: 280000,
          avgEngagementMs: 2000,
          series: [{ date, visitors: 120, pageviews: 200 }],
          audience: [
            {
              date,
              newBrowsers: 42,
              returningBrowsers: 17,
              unknownPageviews: 5,
            },
          ],
          topPages: [],
          channels: [],
          aiSources: [],
          referrers: [],
          devices: [],
          events: [],
          campaigns: [],
          entries: [],
          exits: [],
          acquisitionChannels: [],
        },
      });
    }
    if (url.pathname.endsWith("/countries")) return route.fulfill({ json: [] });
    if (url.pathname.endsWith("/clicks")) {
      if (failClicks)
        return route.fulfill({
          status: 503,
          json: {
            error: {
              code: "unavailable",
              message: "Click data is temporarily unavailable.",
            },
          },
        });
      const next = url.searchParams.get("page") === "2";
      return route.fulfill({
        json: {
          rows: [
            next
              ? {
                  page: "/services",
                  name: "Modal",
                  host: "",
                  path: "",
                  type: "unknown",
                  clicks: 2,
                  visitorDays: 1,
                  arrivals: 0,
                }
              : {
                  page: "/services",
                  name: "Book",
                  host: "partner.test",
                  path: "/booking",
                  type: "external",
                  clicks: 8,
                  visitorDays: 5,
                  arrivals: 0,
                },
          ],
          hasMore: !next,
        },
      });
    }
    return route.fallback();
  });
  return requests;
}

test("daily audience and destination counts render with scoped paging and optional setup", async ({
  page,
}) => {
  const requests = await installAnalytics(page);
  await page.goto("/p/e2e-project/analytics?view=traffic");
  await expect(
    page.getByRole("region", { name: "Traffic overview" }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Daily new and returning" }),
  ).toBeHidden();
  await page
    .getByText("Explore audience, campaigns and page journeys", { exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "Daily new and returning" }),
  ).toBeVisible();
  const daily = page.getByRole("region", { name: "Daily audience table" });
  await expect(
    daily.getByRole("cell", { name: "42", exact: true }),
  ).toBeVisible();
  await expect(
    daily.getByRole("cell", { name: "17", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("cell", { name: "partner.test/booking", exact: false }),
  ).toContainText("External · further activity unavailable");
  await page.getByRole("button", { name: "Load more destinations" }).click();
  await expect(
    page.getByRole("cell", { name: "Unknown destination", exact: false }),
  ).toBeVisible();
  expect(
    requests
      .filter((url) => url.pathname.endsWith("/clicks"))
      .map((url) => url.searchParams.get("page")),
  ).toEqual(["1", "2"]);
  await page
    .getByRole("button", { name: "Tracking code", exact: true })
    .click();
  await page.getByRole("button", { name: "Get my tracking code" }).click();
  const snippet = page.locator("pre");
  await expect(snippet).not.toContainText("data-recognition");
  await page
    .getByRole("checkbox", {
      name: "Enable new/returning browser recognition",
      exact: false,
    })
    .check();
  await expect(snippet).toContainText('data-recognition="true"');
  await expect(
    page.getByText("window.uplumoConsent(false)", { exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "7 days", exact: true }).click();
  await expect
    .poll(
      () =>
        requests.filter(
          (url) =>
            url.pathname.endsWith("/clicks") &&
            url.searchParams.get("page") === "1",
        ).length,
    )
    .toBe(2);
});

for (const width of [320, 390]) {
  test(`mobile analytics keeps reports inside scroll regions at ${width}px`, async ({
    page,
  }) => {
    await page.setViewportSize({ width, height: 844 });
    await installAnalytics(page);
    await page.goto("/p/e2e-project/analytics?view=traffic");
    await expect(
      page.getByRole("heading", { name: "Button and link destinations" }),
    ).toBeVisible();
    const region = page.getByRole("region", {
      name: "Click destination table",
    });
    await expect(region).toBeVisible();
    expect(
      await region.evaluate((node) => node.scrollWidth > node.clientWidth),
    ).toBe(true);
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
  });
}

test("destination errors do not hide the daily audience report", async ({
  page,
}) => {
  await installAnalytics(page, true);
  await page.goto("/p/e2e-project/analytics?view=traffic");
  await page
    .getByText("Explore audience, campaigns and page journeys", { exact: true })
    .click();
  await expect(
    page.getByRole("heading", { name: "Daily new and returning" }),
  ).toBeVisible();
  await expect(
    page.getByText("Something went wrong on our side. Try again."),
  ).toBeVisible();
});
