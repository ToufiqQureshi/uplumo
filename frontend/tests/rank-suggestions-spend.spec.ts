import { expect, test, type Page } from "@playwright/test";
import { installGrowthFixtures } from "./search-growth.fixtures";

const projectId = "11111111-1111-4111-8111-111111111111";
const configId = "22222222-2222-4222-8222-222222222222";

async function openSuggestionStep(page: Page) {
  await installGrowthFixtures(page);
  const project = {
    id: projectId,
    name: "Example site",
    domain: "example.com",
    locationCode: 2840,
    languageCode: "en",
    createdAt: new Date().toISOString(),
  };
  await page.route("**/api/v1/projects/list", (route) =>
    route.fulfill({ json: [project] }),
  );
  await page.route(`**/api/v1/projects/${projectId}/get`, (route) =>
    route.fulfill({ json: project }),
  );
  const purchases: string[] = [];
  await page.route("**/domain/keyword-suggestions", (route) => {
    purchases.push(route.request().postData() ?? "");
    return route.fulfill({
      status: 503,
      json: {
        error: {
          code: "provider_billing_issue",
          message: "Suggestions are temporarily unavailable.",
        },
      },
    });
  });
  await page.route("**/api/v1/projects/*/rank-tracking/**", (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path.endsWith("/configs/summaries"))
      return route.fulfill({ json: { summaries: [] } });
    if (path.endsWith("/configs/list"))
      return route.fulfill({ json: { configs: [] } });
    if (path.endsWith("/configs/create"))
      return route.fulfill({
        json: {
          config: {
            id: configId,
            projectId,
            domain: "example.com",
            locationCode: 2840,
            languageCode: "en",
            locationName: null,
            devices: "mobile",
            serpDepth: 40,
            scheduleInterval: "weekly",
            isActive: true,
            lastCheckedAt: null,
            nextCheckAt: null,
            lastSkipReason: null,
            createdAt: new Date().toISOString(),
          },
        },
      });
    return route.fulfill({ json: {} });
  });
  await page.goto(`/p/${projectId}/rank-tracking`);
  await page.getByRole("button", { name: "Add Domain", exact: true }).click();
  await page.getByLabel("Target Domain").fill("example.com");
  await page
    .getByRole("button", { name: "Add Domain", exact: true })
    .last()
    .click();
  await page.waitForLoadState("networkidle");
  expect(purchases).toEqual([]);
  await expect(
    page.getByRole("button", { name: "Find existing keywords", exact: true }),
  ).toBeVisible();
  return purchases;
}

test("tracker setup and skipping optional suggestions buy no lookup", async ({
  page,
}) => {
  const purchases = await openSuggestionStep(page);
  await page.getByRole("button", { name: "Skip", exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`/rank-tracking/${configId}`));
  expect(purchases).toEqual([]);
});

test("an explicit suggestions double click buys one request without retries", async ({
  page,
}) => {
  const purchases = await openSuggestionStep(page);
  await page
    .getByRole("button", { name: "Find existing keywords", exact: true })
    .click({ clickCount: 2 });
  await expect(
    page.getByRole("heading", { name: "Couldn't fetch keywords" }),
  ).toBeVisible();
  await page.waitForLoadState("networkidle");
  expect(purchases).toHaveLength(1);
  await page.getByRole("button", { name: "Try again", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Couldn't fetch keywords" }),
  ).toBeVisible();
  await page.waitForLoadState("networkidle");
  expect(purchases).toHaveLength(2);
  await page.reload();
  await page.waitForLoadState("networkidle");
  expect(purchases).toHaveLength(2);
});
