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
  proof: null,
  agent: {
    available: true,
    planAllowed: true,
    quickMaxCredits: 25,
    deepMaxCredits: 60,
    maxPages: 20,
    defaultPages: 5,
  },
};

const agentResult = {
  url: "https://example.com/page-0",
  mode: "quick",
  cached: false,
  credits: 7,
  createdAt: "2026-10-28T10:00:00Z",
  partial: false,
  summary: "The page never answers its main question.",
  targetQuestion: "What does Page 0 cost?",
  recommendations: [
    {
      title: "Answer first",
      detail: "Lead with the price.",
      check: "answer-first",
    },
  ],
  answerFirst: "Page 0 costs Rs 999 a month.",
  faq: [],
  titleTag: "Page 0 pricing",
  metaDescription: "What Page 0 costs.",
  schemaJsonLd: "",
  verify: [],
  draft: "# Page 0",
};

test("cite fix agent buys nothing until the cost is confirmed, then one run per page", async ({
  page,
}) => {
  const { paid } = await installGrowthFixtures(page);
  const runs: unknown[] = [];
  let saved: unknown[] = [];
  await page.route("**/api/v1/projects/*/cite-fix/report", (route) =>
    route.fulfill({ json: ready }),
  );
  await page.route("**/api/v1/projects/*/cite-fix/agent/results", (route) =>
    route.fulfill({ json: { results: saved } }),
  );
  await page.route("**/api/v1/projects/*/cite-fix/agent/run", (route) => {
    runs.push(route.request().postDataJSON());
    saved = [agentResult];
    return route.fulfill({ json: agentResult });
  });
  await page.goto("/p/e2e-project/cite-fix");

  // The weakest page is picked by default and the cap is shown first.
  await expect(
    page.getByRole("checkbox", { name: "Fix Page 0 with the agent" }),
  ).toBeChecked();
  await expect(
    page.getByText("at most 25 credits", { exact: false }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Fix 1 page" }).click();
  await expect(
    page.getByText("costs at most 25 credits", { exact: false }),
  ).toBeVisible();
  expect(runs).toEqual([]);

  // Cancel buys nothing.
  await page.getByRole("button", { name: "Cancel" }).click();
  expect(runs).toEqual([]);

  await page.getByRole("button", { name: "Fix 1 page" }).click();
  await page.getByRole("button", { name: "Start" }).click();
  await expect(page.getByText("7 credits").first()).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Agent results" }),
  ).toBeVisible();
  await expect(page.getByText("Page 0 costs Rs 999 a month.")).toBeVisible();

  expect(runs).toHaveLength(1);
  expect(runs[0]).toMatchObject({
    url: "https://example.com/page-0",
    mode: "quick",
  });
  expect(paid).toEqual([]);
});
