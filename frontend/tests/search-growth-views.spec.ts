import { expect, test, type Page } from "@playwright/test";
import { installGrowthFixtures } from "./search-growth.fixtures";

const range = {
  startDate: "2026-09-18",
  endDate: "2026-10-15",
  prevStartDate: "2026-08-21",
  prevEndDate: "2026-09-17",
};
const loss = {
  impressions: 900,
  prevImpressions: 1000,
  position: 7.2,
  prevPosition: 3.1,
  ctr: 0.011,
  prevCtr: 0.05,
  reason: "rank_fell",
};

const insights = {
  connected: true,
  range,
  coverage: {
    queryPageRows: 4200,
    truncated: false,
    decayAvailable: true,
    comparisonAvailable: true,
    brandTermsUsed: false,
  },
  segments: [
    {
      kind: "all",
      queries: 900,
      clicks: 340,
      impressions: 20000,
      ctr: 0.017,
      position: 8,
    },
    {
      kind: "long_tail",
      queries: 410,
      clicks: 51,
      impressions: 6100,
      ctr: 0.008,
      position: 12,
    },
  ],
  actions: [
    {
      kind: "losing_page",
      subject: "https://example.com/rooms",
      page: "https://example.com/rooms",
      impactClicks: 80,
      basis: "measured",
    },
    {
      kind: "ctr_gap",
      subject: "pool villa lonavala",
      page: "https://example.com/villa",
      impactClicks: 16,
      basis: "estimated",
    },
    {
      kind: "cannibalization",
      subject: "resort in lonavala",
      page: "https://example.com/",
      impactClicks: 52,
      basis: "contested",
    },
  ],
  losingPages: Array.from({ length: 30 }, (_, i) => ({
    ...loss,
    page: `https://example.com/page-${i}`,
    clicks: 10,
    prevClicks: 60 - i,
    severity: i < 2 ? "urgent" : "watch",
  })),
  queryLosses: [
    {
      ...loss,
      query: "resort in lonavala",
      clicks: 10,
      prevClicks: 50,
      severity: "urgent",
    },
  ],
  queryGains: [],
  decaying: [
    {
      page: "https://example.com/old-guide",
      clicks: [100, 70, 40],
      lostClicks: 60,
    },
  ],
  ctrGaps: [
    {
      query: "pool villa lonavala",
      page: "https://example.com/villa",
      clicks: 4,
      impressions: 200,
      ctr: 0.02,
      position: 3,
      expectedCtr: 0.1,
      possibleClicks: 16,
    },
  ],
  cannibalization: [
    {
      query: "resort in lonavala",
      clicks: 52,
      impressions: 900,
      leader: "https://example.com/",
      pages: [
        {
          page: "https://example.com/",
          clicks: 40,
          impressions: 500,
          position: 4,
          share: 0.55,
        },
        {
          page: "https://example.com/rooms",
          clicks: 12,
          impressions: 400,
          position: 6,
          share: 0.45,
        },
      ],
    },
  ],
  totals: { decaying: 1, ctrGaps: 1, cannibalization: 1 },
};

async function open(page: Page, query = "") {
  const insightBodies: unknown[] = [];
  const fixtures = await installGrowthFixtures(page);
  await page.route("**/gsc/search-performance/insights", (route) => {
    insightBodies.push(route.request().postDataJSON());
    return route.fulfill({ json: insights });
  });
  await page.goto(`/p/e2e-project/search-performance${query}`);
  return { ...fixtures, insightBodies };
}

test("overview says what to do first, from the site's own numbers", async ({
  page,
}) => {
  const { paid } = await open(page);
  await page.setViewportSize({ width: 1280, height: 900 });
  const first = page.locator("#do-first");
  await expect(first.getByRole("listitem")).toHaveCount(3);
  await expect(first.getByText("Lost 80 clicks")).toBeVisible();
  await expect(
    first.getByText(/estimated from your own numbers/),
  ).toBeVisible();
  await expect(
    page.locator("#segments").getByText("Long-tail (4+ words)"),
  ).toBeVisible();
  await expect(page.getByText("Tell us your brand names")).toBeVisible();
  expect(paid).toEqual([]);
});

test("moving between screens reads Google once; brand names read it again and are remembered", async ({
  page,
}) => {
  const { insightBodies } = await open(page);
  await expect(page.locator("#do-first")).toBeVisible();
  await page.getByRole("tab", { name: "Fix" }).click();
  await expect(page.locator("#losing-pages")).toBeVisible();
  await page.getByRole("tab", { name: "Grow" }).click();
  await expect(page.locator("#ctr-gaps")).toBeVisible();
  await page.getByRole("tab", { name: "Overview" }).click();
  await expect(page.locator("#segments")).toBeVisible();
  expect(insightBodies).toHaveLength(1);

  await page.getByLabel("Your brand names").fill("Acme, Acme Stays");
  await page.getByRole("button", { name: "Split branded searches" }).click();
  await expect.poll(() => insightBodies.length).toBe(2);
  expect(insightBodies[1]).toMatchObject({
    brandTerms: ["Acme", "Acme Stays"],
  });

  await page.reload();
  await expect(page.getByLabel("Your brand names")).toHaveValue(
    "Acme, Acme Stays",
  );
});

test("fix lists every lost page, 25 at a time", async ({ page }) => {
  await open(page, "?view=fix");
  const lost = page.locator("#losing-pages");
  await expect(
    lost.getByRole("heading", { name: /Pages that lost clicks \(30\)/ }),
  ).toBeVisible();
  await expect(lost.getByRole("row")).toHaveCount(26); // header + 25
  await lost.getByRole("button", { name: /Show 5 more \(5 left\)/ }).click();
  await expect(lost.getByRole("row")).toHaveCount(31);
  await expect(page.locator("#decaying").getByText("−60")).toBeVisible();
});

test("grow shows searches people skip and pages that compete, with an honest estimate", async ({
  page,
}) => {
  await open(page, "?view=grow");
  const gaps = page.locator("#ctr-gaps");
  await expect(gaps.getByText("~16")).toBeVisible();
  await expect(gaps.getByText(/estimate from your own pages/)).toBeVisible();
  await expect(
    gaps.getByRole("button", { name: "Rewrite title" }),
  ).toHaveAttribute("href", /keyword=pool/);
  const compete = page.locator("#cannibalization");
  await expect(compete.getByText("Keep", { exact: true })).toBeVisible();
  await expect(compete.getByText("55%")).toBeVisible();
});

test("a regular-expression filter reaches the analysis", async ({ page }) => {
  const { insightBodies } = await open(
    page,
    "?queryText=%5E(how%7Cwhat)%5Cb&queryMatch=includingRegex",
  );
  await expect(page.locator("#do-first")).toBeVisible();
  expect(insightBodies[0]).toMatchObject({
    queryFilter: { operator: "includingRegex", expression: "^(how|what)\\b" },
  });
});

for (const width of [320, 360, 390]) {
  test(`every screen fits a ${width}px phone without sideways scrolling`, async ({
    page,
  }) => {
    await open(page);
    await page.setViewportSize({ width, height: 800 });
    for (const view of [
      "overview",
      "fix",
      "grow",
      "explore",
      "indexing",
      "conversions",
    ]) {
      await page.goto(`/p/e2e-project/search-performance?view=${view}`);
      await expect(
        page.getByRole("heading", { name: "Search Growth" }).first(),
      ).toBeVisible();
      await page.waitForTimeout(400);
      const overflow = await page.evaluate(
        () =>
          document.documentElement.scrollWidth -
          document.documentElement.clientWidth,
      );
      expect(overflow, `${view} at ${width}px`).toBeLessThanOrEqual(1);
    }
  });
}
