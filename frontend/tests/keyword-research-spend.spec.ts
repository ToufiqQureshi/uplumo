import { expect, test, type Page } from "@playwright/test";
import { installGrowthFixtures } from "./search-growth.fixtures";

const project = {
  id: "e2e-project",
  name: "Example site",
  domain: "example.com",
  locationCode: 2840,
  languageCode: "en",
  createdAt: new Date().toISOString(),
};

const row = (keyword: string, searchVolume: number, kd: number) => ({
  keyword,
  searchVolume,
  trend: [
    { year: 2026, month: 9, searchVolume: searchVolume - 100 },
    { year: 2026, month: 10, searchVolume },
  ],
  keywordDifficulty: kd,
  cpc: 2.5,
  competition: 0.4,
  intent: "commercial",
  topOfPageBidLow: 1.1,
  topOfPageBidHigh: 4.2,
  referringDomains: 320,
  backlinks: 5400,
  dofollowBacklinks: 4100,
  referringPages: 2900,
  monthlyChange: 3,
  quarterlyChange: -2,
  yearlyChange: 18,
  competitionLevel: "MEDIUM",
  lastUpdatedTime: "2026-10-14 09:12:33 +00:00",
  detectedLanguage: "en",
  wordsCount: 3,
  secondaryIntents: ["informational"],
  serpFeatures: ["ai_overview", "people_also_ask", "video"],
  coreKeyword: null,
  topPagesDomainRank: 512,
});

async function openPage(page: Page) {
  await installGrowthFixtures(page);
  await page.route("**/api/v1/projects/list", (route) =>
    route.fulfill({ json: [project] }),
  );
  const research: unknown[] = [];
  const serp: unknown[] = [];
  await page.route("**/keywords/research", async (route) => {
    research.push(route.request().postDataJSON());
    // Slow enough for a second click to land while the first is in flight.
    await new Promise((resolve) => setTimeout(resolve, 300));
    await route.fulfill({
      json: {
        rows: [
          row("seo audit tool", 12100, 64),
          row("free seo audit checklist", 1900, 24),
        ],
        source: "blended",
        usedFallback: false,
        placeFilter: null,
      },
    });
  });
  await page.route("**/keywords/serp", async (route) => {
    serp.push(route.request().postDataJSON());
    await new Promise((resolve) => setTimeout(resolve, 300));
    await route.fulfill({
      json: {
        requestedKeyword: "seo audit tool",
        depth: 20,
        items: [
          {
            rank: 1,
            title: "Audit your site",
            url: "https://audit.example/",
            domain: "audit.example",
            description: "",
            etv: 900,
            estimatedPaidTrafficCost: null,
            referringDomains: 40,
            backlinks: null,
            isNew: false,
            rankChange: null,
          },
        ],
      },
    });
  });
  await page.goto(`/p/${project.id}/keywords`);
  await expect(
    page.getByRole("heading", { name: "Keyword Research" }),
  ).toBeVisible();
  return { research, serp };
}

test("opening and typing buy nothing; one search is one purchase", async ({
  page,
}) => {
  const { research, serp } = await openPage(page);
  const keyword = page.getByRole("textbox", { name: "Keyword" });
  await keyword.fill("seo audit tool");
  await page.waitForLoadState("networkidle");
  expect(research).toEqual([]);

  const search = page.getByRole("button", { name: "Search", exact: true });
  await search.click();
  await search.click({ force: true });
  await keyword.press("Enter");
  await expect(
    page.getByRole("heading", { name: "seo audit tool", exact: true }),
  ).toBeVisible();
  expect(research).toEqual([
    {
      keywords: ["seo audit tool"],
      locationCode: 2840,
      resultLimit: 150,
      mode: "auto",
    },
  ]);

  // Every number the research returned is on the page.
  for (const text of [
    "12.1K",
    "$2.50",
    "$1.10–$4.20",
    "0.40 · medium",
    "512",
    "320",
    "5,400",
    "4,100",
    "2,900",
    "+18%",
    "AI Overview",
    "People also ask",
    "Video",
    "Oct 2026",
  ])
    await expect(page.getByText(text, { exact: false }).first()).toBeVisible();

  // Opening another keyword, filtering and sorting use rows already bought.
  await page
    .getByRole("cell", { name: "free seo audit checklist", exact: true })
    .click();
  await page.getByLabel("KD at most").fill("30");
  await page.getByRole("button", { name: "Sort by Searches" }).click();
  await page.waitForLoadState("networkidle");
  expect(research).toHaveLength(1);
  expect(serp).toEqual([]);

  const top = page.getByRole("button", { name: "Show top results" });
  await top.click();
  await top.click({ force: true }).catch(() => {});
  await expect(page.getByText("audit.example").first()).toBeVisible();
  expect(serp).toHaveLength(1);
  // Only the numbers the lookup returned get a column.
  await expect(
    page.getByRole("columnheader", { name: "Visits / mo" }),
  ).toBeVisible();
  await expect(
    page.getByRole("columnheader", { name: "Backlinks" }),
  ).toHaveCount(0);
});

test("a reload buys nothing again", async ({ page }) => {
  const { research, serp } = await openPage(page);
  await page.reload();
  await page.waitForLoadState("networkidle");
  expect(research).toEqual([]);
  expect(serp).toEqual([]);
});
