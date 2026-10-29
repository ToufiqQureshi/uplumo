import { expect, test } from "@playwright/test";
import { installGrowthFixtures } from "./search-growth.fixtures";

import { ready } from "./cite-fix.fixtures";

const googleOverviewResults = Array.from({ length: 10 }, (_, index) => ({
  prompt:
    index === 0
      ? "Which booking tool is best?"
      : `Which booking tool is best ${index + 1}?`,
  topic: "Booking",
  status: "not_checked",
  requestedAt: null,
  cachedUntil: null,
  answer: "",
  citations: [],
  customerCited: false,
}));

const googleOverviewReady = {
  ...ready,
  googleAiOverview: {
    available: true,
    providerConfigured: true,
    domain: "example.com",
    questions: googleOverviewResults.map((result) => result.prompt),
    maxQuestions: 10,
    maxCredits: 16,
    results: googleOverviewResults,
  },
  seoIssues: [
    {
      type: "missing-title",
      title: "Page title is missing",
      severity: "critical",
      pageUrl: "https://example.com/guide",
      howToFix: "Add a clear title to the page.",
    },
  ],
};

for (const width of [390, 1280]) {
  test(`cite fix reads one free report at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    const { paid } = await installGrowthFixtures(page);
    const reports: string[] = [];
    await page.route("**/api/v1/projects/*/cite-fix/report", async (route) => {
      reports.push(route.request().postData() ?? "");
      return route.fulfill({ json: ready });
    });
    await page.goto("/p/e2e-project/cite-fix");

    await expect(page.getByRole("heading", { name: "Cite Fix" })).toBeVisible();
    await expect(page.getByText("Hard for AI to quote")).toBeVisible();
    await expect(
      page.getByText("35 of 40 pages checked", { exact: false }),
    ).toBeVisible();
    // The first fix opens with its why, how and pages; a site fix says so.
    await expect(page.getByText("First 20 of 35 pages")).toBeVisible();
    await expect(page.getByText("Whole site")).toBeVisible();
    // Base UI keeps the button role on a Button rendered as a link.
    await expect(
      page.getByRole("button", { name: "Scan again" }),
    ).toHaveAttribute("href", "/p/e2e-project/audit");

    expect(reports).toEqual(["{}"]);
    expect(paid).toEqual([]);
  });
}

test("cite fix asks for a new scan instead of scoring an old one", async ({
  page,
}) => {
  const { paid } = await installGrowthFixtures(page);
  await page.route("**/api/v1/projects/*/cite-fix/report", (route) =>
    route.fulfill({
      json: {
        ...ready,
        state: "needs_rescan",
        score: null,
        categories: [],
        fixes: [],
        pages: [],
      },
    }),
  );
  await page.goto("/p/e2e-project/cite-fix");
  await expect(page.getByText("Scan your site again")).toBeVisible();
  await expect(page.getByText("/100")).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Scan again" }),
  ).toHaveAttribute("href", "/p/e2e-project/audit");
  expect(paid).toEqual([]);
});

test("Google AI Overview is free to read and buys only after cost confirmation", async ({
  page,
}) => {
  const { paid } = await installGrowthFixtures(page);
  const refreshes: string[] = [];
  await page.route("**/api/v1/projects/*/cite-fix/report", (route) =>
    route.fulfill({ json: googleOverviewReady }),
  );
  await page.route(
    "**/api/v1/projects/*/cite-fix/google-ai-overview/refresh",
    async (route) => {
      refreshes.push(route.request().headers()["idempotency-key"] ?? "");
      return route.fulfill({
        json: {
          ...googleOverviewReady.googleAiOverview,
          results: googleOverviewReady.googleAiOverview.results.map(
            (result, index) => ({
              ...result,
              status: "completed" as const,
              requestedAt: "2026-10-28T10:05:00Z",
              cachedUntil: "2026-11-04T10:05:00Z",
              answer:
                index === 0
                  ? "Choose the tool that fits your team's booking workflow."
                  : "",
              customerCited: index === 0,
              citations:
                index === 0
                  ? [
                      {
                        url: "https://example.com/guide",
                        domain: "example.com",
                        title: "Booking guide",
                        matchedBrand: true,
                        source: "Example Journal",
                        publishedAt: "2026-10-20",
                      },
                    ]
                  : [],
            }),
          ),
        },
      });
    },
  );

  await page.goto("/p/e2e-project/cite-fix");
  await expect(
    page.getByRole("heading", { name: "Google AI Overview citations" }),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Also fix for Google" }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Open Site Audit" }),
  ).toHaveAttribute("href", "/p/e2e-project/audit");
  await expect(
    page.getByRole("button", { name: "Check citations" }),
  ).toBeVisible();
  expect(refreshes).toEqual([]);
  expect(paid).toEqual([]);

  await page.getByRole("button", { name: "Check citations" }).click();
  await expect(page.getByRole("alertdialog")).toContainText("up to 16 credits");
  expect(refreshes).toEqual([]);
  await page
    .getByRole("button", { name: "Continue · up to 16 credits" })
    .click();
  await expect(
    page.getByText("Your domain is cited in the AI Overview."),
  ).toBeVisible();
  await expect(
    page.getByText("example.com · Example Journal · 2026-10-20"),
  ).toBeVisible();
  await expect(
    page.getByRole("link", { name: "Booking guide" }),
  ).toHaveAttribute("href", "https://example.com/guide");
  expect(refreshes).toHaveLength(1);
  expect(refreshes[0]).not.toBe("");
  expect(paid).toEqual([]);
});

test("Google AI Overview does not retry a failed confirmed refresh", async ({
  page,
}) => {
  await installGrowthFixtures(page);
  let refreshes = 0;
  await page.route("**/api/v1/projects/*/cite-fix/report", (route) =>
    route.fulfill({ json: googleOverviewReady }),
  );
  await page.route(
    "**/api/v1/projects/*/cite-fix/google-ai-overview/refresh",
    async (route) => {
      refreshes += 1;
      return route.fulfill({
        status: 503,
        json: {
          error: {
            code: "provider_error",
            message: "The Google AI Overview refresh is unavailable.",
          },
        },
      });
    },
  );

  await page.goto("/p/e2e-project/cite-fix");
  await page.getByRole("button", { name: "Check citations" }).click();
  await page
    .getByRole("button", { name: "Continue · up to 16 credits" })
    .click();
  await expect.poll(() => refreshes).toBe(1);
  await page.waitForTimeout(1_200);
  expect(refreshes).toBe(1);
});

test("rival comparison shows the cited page signals for each missed question", async ({
  page,
}) => {
  const { paid } = await installGrowthFixtures(page);
  const report = {
    ...ready,
    proof: {
      checkedAt: "2026-10-28T10:00:00Z",
      domain: "example.com",
      answers: 1,
      citingYou: 0,
      rivals: [{ domain: "rival.example", count: 1 }],
      yourPages: [],
      misses: [
        {
          prompt: "Which booking tool is best?",
          engine: "google_ai_overview",
          cited: ["rival.example"],
          citedPages: [
            {
              url: "https://rival.example/guide",
              domain: "rival.example",
              title: "Rival booking guide",
            },
          ],
        },
      ],
    },
    rivals: [
      {
        prompt: "Which booking tool is best?",
        engine: "AI Visibility Brand Lookup · google",
        rival: {
          url: "https://rival.example/guide",
          domain: "rival.example",
          title: "Rival booking guide",
          appShell: false,
          fetchedAt: "2026-10-28T10:00:00Z",
          contentHash: "hash-a",
          signals: {
            answerFirstWords: 24,
            author: true,
            date: "2026-10-21",
            schema: ["Article"],
            statistics: 3,
            faq: true,
            wordCount: 840,
          },
        },
        yours: {
          url: "https://example.com/guide",
          title: "Our guide",
          signals: {
            answerFirstWords: 0,
            author: false,
            date: "",
            schema: [],
            statistics: 0,
            faq: false,
            wordCount: 430,
          },
        },
      },
    ],
  };
  await page.route("**/api/v1/projects/*/cite-fix/report", (route) =>
    route.fulfill({ json: report }),
  );
  await page.goto("/p/e2e-project/cite-fix");
  await expect(
    page.getByRole("heading", { name: "Why they got cited, you didn't" }),
  ).toBeVisible();
  await expect(
    page.getByRole("link", { name: "Compare with rival" }),
  ).toHaveAttribute("href", "#cite-fix-rival-0");
  await expect(
    page.getByRole("link", { name: "Rival booking guide" }),
  ).toHaveAttribute("href", "https://rival.example/guide");
  const rivalCard = page.locator("#cite-fix-rival-0");
  await expect(
    rivalCard.getByText("Evidence from AI Visibility Brand Lookup · google"),
  ).toBeVisible();
  await expect(
    rivalCard.getByRole("heading", { name: "Theirs" }),
  ).toBeVisible();
  await expect(rivalCard.getByText("24 words")).toBeVisible();
  await expect(rivalCard.getByText("3", { exact: true })).toBeVisible();
  await expect(rivalCard.getByText("840")).toBeVisible();
  expect(paid).toEqual([]);
});

test("Site Audit offers paid rendering as an opt-in with its cap shown", async ({
  page,
}) => {
  const { paid } = await installGrowthFixtures(page);
  const starts: unknown[] = [];
  await page.route("**/api/v1/projects/*/audit/*", async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (path.endsWith("/capabilities"))
      return route.fulfill({
        json: {
          canRenderJavaScript: true,
          maxRenderedPages: 20,
          maxJavaScriptCredits: 39,
        },
      });
    if (path.endsWith("/history")) return route.fulfill({ json: [] });
    if (path.endsWith("/start")) {
      starts.push(route.request().postDataJSON());
      return route.fulfill({ json: { auditId: "audit-render" } });
    }
    return route.fulfill({
      status: 500,
      json: { error: "Unexpected audit request" },
    });
  });
  await page.goto("/p/e2e-project/audit");
  const render = page.getByRole("checkbox", {
    name: "Render JavaScript pages (at most 39 credits)",
  });
  await expect(render).toBeEnabled();
  await expect(render).not.toBeChecked();
  await expect(
    page.getByText("up to 20 per audit", { exact: false }),
  ).toBeVisible();
  await page.getByLabel("Site URL").fill("https://example.com");
  await render.check();
  await page.getByRole("button", { name: "Start Audit" }).click();
  await expect.poll(() => starts.length).toBe(1);
  expect(starts[0]).toMatchObject({ renderJavaScript: true, maxPages: 50 });
  expect(paid).toEqual([]);
});
