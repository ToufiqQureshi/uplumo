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

const brands = (own: boolean, rival: boolean) => [
  { name: "Example", domain: "example.com", mentioned: own, cited: false },
  { name: "Rival", domain: "rival.example", mentioned: rival, cited: false },
];

// Calls that spend DataForSEO or model credits. None may fire unless the
// customer clicks a button that showed its cost first.
const PAID =
  /\/ai-search\/(tracking\/run$|tracking\/runs\/[^/]+\/sentiment$|brand-lookup$|prompt-explorer$)|\/writer|\/content-editor/u;

test("AI visibility tabs show saved answers and buy nothing on load", async ({
  page,
}) => {
  const calls: string[] = [];
  const now = new Date().toISOString();
  const config = {
    brand: "Example",
    domain: "example.com",
    competitors: [{ name: "Rival", domain: "rival.example" }],
    prompts: [
      { text: "best crm for small teams", topic: "General", paused: false },
      { text: "cheap crm", topic: "General", paused: false },
    ],
    engines: ["chat_gpt", "gemini"],
    locationCode: 2840,
    languageCode: "en",
    interval: "weekly",
    enabled: true,
    maxCreditsPerRun: 4,
  };
  const run = (id: string, mentioned: number) => ({
    id,
    configuration: config,
    scopeKey: "scope",
    trigger: "manual",
    status: "completed",
    errorCode: "",
    createdAt: id === "r2" ? now : "2026-11-01T10:00:00Z",
    finishedAt: now,
    completed: 4,
    failed: 0,
    answered: 4,
    mentioned,
    cited: 0,
  });
  const answer = (
    prompt: string,
    engine: string,
    own: boolean,
    rival: boolean,
  ) => ({
    id: `${prompt}-${engine}`,
    prompt,
    topic: "General",
    engine,
    status: "completed",
    answer: own ? "Example is a good pick." : "Rival is the usual choice.",
    errorCode: "",
    collectedAt: now,
    citations: [
      {
        url: "https://review.example/best-crm",
        domain: "review.example",
        title: "Best CRM 2026",
        matchedBrand: false,
      },
    ],
    brands: brands(own, rival),
    recommended: [],
  });
  await page.route("**/api/**", async (route) => {
    const path = new URL(route.request().url()).pathname;
    calls.push(path);
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
        locationCode: 2840,
        languageCode: "en",
        createdAt: now,
      };
    else if (path === "/api/v1/billing/credits")
      body = {
        plan: "business",
        status: "active",
        currentPeriodEnd: null,
        balanceCredits: 5000,
        balanceUsd: 5,
        monthlyGrantCredits: 15000,
        paymentsConfigured: true,
        availablePlans: ["basic", "pro", "business"],
      };
    else if (path.endsWith("/ai-search/tracking"))
      body = {
        configuration: config,
        nextCheckAt: null,
        runs: [run("r2", 1), run("r1", 0)],
        providerConfigured: true,
      };
    else if (path.endsWith("/tracking/runs/r2"))
      body = [
        answer("best crm for small teams", "chat_gpt", false, true),
        answer("best crm for small teams", "gemini", false, true),
        answer("cheap crm", "chat_gpt", true, false),
        answer("cheap crm", "gemini", false, true),
      ];
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(body),
    });
  });
  await page.setViewportSize({ width: 1280, height: 900 });

  // Old addresses still land on the merged page.
  await page.goto("/p/test-project/prompt-tracking");
  await expect(page).toHaveURL(/\/p\/test-project\/ai-visibility$/u);
  await expect(page.getByRole("region", { name: "Verdict" })).toContainText(
    "AI names Rival 3.0× as often as Example.",
  );
  await expect(page.getByText("1 of 4", { exact: true }).first()).toBeVisible();
  await expect(page.getByText("▲ 25 points vs previous check")).toBeVisible();

  // The verdict's button opens the gaps, and a cell opens the saved answer.
  await page.getByText(/where AI leaves you out/u).click();
  await expect(page).toHaveURL(/ai-visibility\/prompts\?show=gaps/u);
  await page
    .getByRole("button", { name: /ChatGPT on "best crm for small teams"/u })
    .click();
  const proof = page.getByRole("region", { name: "Saved answer" });
  await expect(proof).toContainText("Rival is the usual choice.");
  await expect(proof).toContainText("Best CRM 2026");
  await proof.getByRole("button", { name: "Close answer" }).click();
  await expect(proof).toBeHidden();

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

  expect(calls.filter((path) => PAID.test(path))).toEqual([]);
});
