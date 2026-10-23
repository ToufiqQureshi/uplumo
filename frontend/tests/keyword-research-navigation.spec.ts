import { expect, test, type Page } from "@playwright/test";
import { installGrowthFixtures } from "./search-growth.fixtures";
import { z } from "zod";
import { getKeywordResearchFixture } from "./fixtures/keyword-research-fixtures";

async function getProjectId(page: Page) {
  await installGrowthFixtures(page);
  await page.route("**/api/v1/projects/*/keywords/research", (route) =>
    route.fulfill({
      json: getKeywordResearchFixture(
        z
          .object({
            projectId: z.string().default("e2e-project"),
            keywords: z.array(z.string()).min(1),
            locationCode: z.number().default(2840),
            languageCode: z.string().default("en"),
            resultLimit: z
              .union([z.literal(150), z.literal(300), z.literal(500)])
              .default(150),
            mode: z
              .enum(["auto", "related", "suggestions", "ideas"])
              .default("auto"),
            clickstream: z.boolean().default(false),
            groupKeywords: z.boolean().default(false),
            locationName: z.string().optional(),
          })
          .parse(route.request().postDataJSON()),
      ),
    }),
  );
  return "e2e-project";
}

test.describe("Keyword Research navigation", () => {
  test("Back to Recent searches clears the active keyword tab query", async ({
    page,
  }) => {
    const projectId = await getProjectId(page);

    await page.goto(
      `/p/${projectId}/keywords?q=keyword%20research&loc=2840&kLimit=150&mode=auto`,
    );
    await expect(
      page.getByRole("heading", { name: "Keyword Research", exact: true }),
    ).toBeVisible();
    await expect(
      page.getByRole("row", { name: /keyword research/i }).first(),
    ).toBeVisible({ timeout: 30_000 });
    const recentSearchesButton = page.locator(
      '[data-testid="keyword-research-recent-searches"]:visible',
    );
    await expect(recentSearchesButton).toBeVisible();

    await recentSearchesButton.click();

    await expect
      .poll(() => new URL(page.url()).searchParams.get("q"))
      .toBe(null);
    await expect(
      page.getByRole("link", { name: "keyword research US" }),
    ).toBeVisible();
    await expect(recentSearchesButton).toBeHidden();
  });

  test("clicking keyword tabs keeps the clicked tab and URL in sync", async ({
    page,
  }) => {
    const projectId = await getProjectId(page);

    await page.goto(
      `/p/${projectId}/keywords?q=keyword%20research&loc=2840&kLimit=150&mode=auto`,
    );
    await expect(
      page.getByRole("row", { name: /keyword research/i }).first(),
    ).toBeVisible({ timeout: 30_000 });

    await page.goto(
      `/p/${projectId}/keywords?q=backlinks&loc=2840&kLimit=150&mode=auto`,
    );
    await expect(
      page.getByRole("row", { name: /backlinks/i }).first(),
    ).toBeVisible({ timeout: 30_000 });

    await page.getByRole("tab", { name: /keyword research/i }).click();
    await expect
      .poll(() => new URL(page.url()).searchParams.get("q"))
      .toBe("keyword research");
    await expect(
      page.getByRole("row", { name: /keyword research/i }).first(),
    ).toBeVisible();

    await page.getByRole("tab", { name: /^backlinks/i }).click();
    await expect
      .poll(() => new URL(page.url()).searchParams.get("q"))
      .toBe("backlinks");
    await expect(
      page.getByRole("row", { name: /backlinks/i }).first(),
    ).toBeVisible();
  });

  test("closing the active middle keyword tab removes it and selects the next tab", async ({
    page,
  }) => {
    const projectId = await getProjectId(page);

    for (const keyword of ["ai seo", "backlinks", "seo tools"]) {
      await page.goto(
        `/p/${projectId}/keywords?q=${encodeURIComponent(keyword)}&loc=2840&kLimit=150&mode=auto`,
      );
      await expect(
        page.getByRole("row", { name: new RegExp(keyword, "i") }).first(),
      ).toBeVisible({ timeout: 30_000 });
    }

    await page.getByRole("tab", { name: /^backlinks/i }).click();
    await expect
      .poll(() => new URL(page.url()).searchParams.get("q"))
      .toBe("backlinks");

    const closeButton = page.getByRole("button", {
      name: "Close backlinks tab",
    });
    const closedTabId = await closeButton.getAttribute("data-search-tab-id");
    expect(closedTabId).toBeTruthy();

    await closeButton.click();

    await expect
      .poll(() => new URL(page.url()).searchParams.get("q"))
      .toBe("seo tools");
    await expect(
      page.getByRole("tab", { name: /^seo tools/i }),
    ).toHaveAttribute("aria-selected", "true");
    await expect(
      page.locator(`[data-search-tab-id="${closedTabId}"]`),
    ).toHaveCount(0);
    // Count only search tabs: the app shell has grown other tablists
    // (Browse/Chat), so a bare role=tab count would include them.
    await expect(
      page.getByRole("tablist", { name: "Search tabs" }).getByRole("tab"),
    ).toHaveCount(2);
  });
});
