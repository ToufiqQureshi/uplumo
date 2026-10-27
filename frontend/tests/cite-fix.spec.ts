import { expect, test } from "@playwright/test";
import { installGrowthFixtures } from "./search-growth.fixtures";

const fix = (id: string, title: string, gain: number, pages: number) => ({
  id,
  category: "answers",
  title,
  why: "Assistants quote short, direct answers.",
  how: "Start the page with a 1 to 3 sentence answer.",
  impact: gain >= 5 ? "high" : gain >= 2 ? "medium" : "low",
  gain,
  pagesAffected: pages,
  pages: Array.from({ length: Math.min(pages, 20) }, (_, i) => ({
    url: `https://example.com/page-${i}`,
    title: `Page ${i}`,
  })),
});

const ready = {
  state: "ready",
  running: null,
  scan: {
    auditId: "audit-1",
    startUrl: "https://example.com/",
    completedAt: "2026-10-27T10:00:00Z",
  },
  score: 44,
  categories: [
    { id: "access", label: "AI can read the site", weight: 20, score: 100 },
    { id: "answers", label: "Direct answers", weight: 20, score: 0 },
  ],
  fixes: [
    fix("answer-first", "The answer is not in the first paragraph", 10, 35),
    fix("ai-search-bots-allowed", "AI search bots are blocked", 10, 0),
  ],
  pages: [
    {
      url: "https://example.com/page-0",
      title: "Page 0",
      score: 17,
      failed: ["answer-first"],
    },
  ],
  pagesChecked: 35,
  pagesTotal: 40,
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
