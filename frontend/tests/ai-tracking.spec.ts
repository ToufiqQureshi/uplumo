import { expect, test } from "@playwright/test";
import { mockAiVisibility, paidCalls } from "./ai-visibility.fixtures";

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
        maxPrompts: 30,
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
  await page.goto("/p/test-project/ai-visibility/setup");
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

test("AI visibility tabs show saved answers and buy nothing on load", async ({
  page,
}) => {
  const log = await mockAiVisibility(page);
  await page.setViewportSize({ width: 1280, height: 900 });

  // Old addresses still land on the merged page.
  await page.goto("/p/test-project/prompt-tracking");
  await expect(page).toHaveURL(/\/p\/test-project\/ai-visibility$/u);
  await expect(page.getByRole("region", { name: "Verdict" })).toContainText(
    "AI names Rival 3.0× as often as Example.",
  );
  await expect(page.getByText("1 of 4", { exact: true }).first()).toBeVisible();
  await expect(page.getByText("▲ 25 points vs previous check")).toBeVisible();

  for (const [from, to] of [
    ["prompt-explorer?q=best%20crm", "ai-visibility/test?q=best"],
    ["brand-lookup?q=Example", "ai-visibility/research?q=Example"],
    ["ai-opportunities", "ai-visibility/opportunities"],
    ["ai-sentiment", "ai-visibility/sentiment"],
    ["prompt-ideas", "ai-visibility/ideas"],
  ] as const) {
    await page.goto(`/p/test-project/${from}`);
    await expect(page).toHaveURL(new RegExp(to.replace("?", "\\?"), "u"));
    await expect(
      page.getByRole("navigation", { name: "AI visibility sections" }),
    ).toBeVisible();
  }
  // A test question opened from a link is filled in, not bought.
  await page.goto("/p/test-project/ai-visibility/test?q=best%20crm");
  await expect(page.getByLabel("Prompt")).toHaveValue("best crm");
  await page.goto("/p/test-project/ai-visibility/research?q=example.com");
  await expect(
    page.getByRole("navigation", { name: "AI visibility sections" }),
  ).toBeVisible();

  expect(paidCalls(log)).toEqual([]);
});
