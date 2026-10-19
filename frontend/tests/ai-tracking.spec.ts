import { expect, test } from "@playwright/test";

test("saved AI tracker reviews costs without collecting answers on load", async ({
  page,
}) => {
  const calls: string[] = [];
  await page.route("**/api/**", async (route) => {
    const path = new URL(route.request().url()).pathname;
    calls.push(path);
    const now = new Date().toISOString();
    let body: unknown = {};
    if (path === "/api/auth/get-session")
      body = {
        user: {
          id: "test-user",
          name: "Tracker Test",
          email: "tracker@example.com",
          emailVerified: true,
          createdAt: now,
          updatedAt: now,
        },
        session: {
          id: "test-session",
          userId: "test-user",
          expiresAt: new Date(Date.now() + 3600000).toISOString(),
          token: "playwright-session",
          createdAt: now,
          updatedAt: now,
        },
      };
    else if (path.endsWith("/projects/test-project/get"))
      body = {
        id: "test-project",
        name: "Test site",
        domain: "example.com",
        locationCode: 2356,
        languageCode: "en",
        createdAt: now,
      };
    else if (path.endsWith("/tracking"))
      body = {
        configuration: null,
        nextCheckAt: null,
        runs: [],
        providerConfigured: true,
      };
    else if (path.endsWith("/tracking/estimate"))
      body = { answers: 2, credits: 4, monthlyCredits: 18, ceiling: true };
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(body),
    });
  });
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.goto("/p/test-project/prompt-tracking");
  await expect(
    page.getByRole("heading", { name: "Questions to monitor" }),
  ).toBeVisible();
  expect(
    calls.some((path) => /tracking\/(save|run|estimate)$/u.test(path)),
  ).toBe(false);
  await page.getByLabel("Brand name", { exact: true }).fill("Example");
  await page.getByLabel("Website domain", { exact: true }).fill("example.com");
  await page
    .getByRole("textbox", { name: /Question 1/u })
    .fill("Which service should I choose?");
  await page.getByRole("button", { name: /Review.*cost/iu }).click();
  await expect(page.getByText(/4 credits/u).first()).toBeVisible();
  expect(
    calls.filter((path) => path.endsWith("/tracking/estimate")),
  ).toHaveLength(1);
  expect(calls.some((path) => path.endsWith("/tracking/run"))).toBe(false);
  await page.getByLabel("Brand name", { exact: true }).fill("Changed");
  await expect(
    page.getByRole("button", { name: /Save/iu }).first(),
  ).toBeDisabled();
});
