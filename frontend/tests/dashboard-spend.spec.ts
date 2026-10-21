import { expect, test } from "@playwright/test";
import { installGrowthFixtures } from "./search-growth.fixtures";

test("dashboard visits never refresh paid backlinks automatically", async ({
  page,
}) => {
  await installGrowthFixtures(page);
  const refreshes: string[] = [];
  const projects = [
    {
      id: "e2e-project",
      name: "Example site",
      domain: "example.com",
      locationCode: 2840,
      languageCode: "en",
      createdAt: new Date().toISOString(),
    },
    {
      id: "second-project",
      name: "Second site",
      domain: "second.example",
      locationCode: 2840,
      languageCode: "en",
      createdAt: new Date().toISOString(),
    },
  ];
  await page.route("**/api/v1/projects/list", (route) =>
    route.fulfill({ json: projects }),
  );
  await page.route("**/api/v1/projects/second-project/get", (route) =>
    route.fulfill({ json: projects[1] }),
  );
  await page.route("**/api/v1/projects/*/activation/get", (route) =>
    route.fulfill({
      json: {
        domain: "example.com",
        ga4: {
          connected: false,
          propertyDisplayName: null,
          cardDismissedAt: null,
        },
        gsc: { connected: false, siteUrl: null },
        mcp: {
          authorizedAt: null,
          firstToolCallAt: null,
          cardDismissedAt: null,
        },
        competitorClickedAt: null,
        keywordsClickedAt: null,
        hasAudit: false,
        hasMultipleProjects: false,
        hasTeammate: false,
        dismissedSteps: [],
      },
    }),
  );
  await page.route("**/api/v1/projects/*/dashboard/**", (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path.endsWith("/overview"))
      return route.fulfill({
        json: { audit: null, backlinks: null, backlinkRefreshAvailable: true },
      });
    expect(route.request().postDataJSON()).toEqual({ userInitiated: true });
    refreshes.push(path);
    return route.fulfill({
      status: 503,
      json: {
        error: {
          code: "provider_billing_issue",
          message: "Backlinks are temporarily unavailable.",
        },
      },
    });
  });
  await page.goto("/p/e2e-project");
  await expect(
    page.getByRole("heading", { name: "Backlink pulse" }),
  ).toBeVisible();
  await page.waitForLoadState("networkidle");
  expect(refreshes).toEqual([]);
  await page.reload();
  await expect(
    page.getByRole("heading", { name: "Backlink pulse" }),
  ).toBeVisible();
  await page.waitForLoadState("networkidle");
  expect(refreshes).toEqual([]);
  await page.getByRole("button", { name: "Refresh backlink snapshot" }).click();
  await expect(
    page.getByText("Something went wrong on our side. Try again.", {
      exact: true,
    }),
  ).toBeVisible();
  await page.waitForLoadState("networkidle");
  expect(refreshes).toHaveLength(1);
  await page.getByRole("combobox", { name: "Switch project" }).click();
  await page.getByRole("option", { name: /Second site/u }).click();
  await expect(page).toHaveURL(/\/p\/second-project/u);
  await expect(
    page.getByRole("heading", { name: "Backlink pulse" }),
  ).toBeVisible();
  await expect(
    page.getByText("Something went wrong on our side. Try again.", {
      exact: true,
    }),
  ).toHaveCount(0);
  expect(refreshes).toHaveLength(1);
});
