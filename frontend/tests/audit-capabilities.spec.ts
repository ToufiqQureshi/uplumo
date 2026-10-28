import { expect, test } from "@playwright/test";
import { installGrowthFixtures } from "./search-growth.fixtures";

for (const width of [390, 1280]) {
  test(`audit explains unavailable rendering at ${width}px`, async ({
    page,
  }) => {
    await page.setViewportSize({ width, height: 844 });
    const { paid } = await installGrowthFixtures(page);
    const starts: string[] = [];
    await page.route("**/api/v1/projects/*/audit/*", async (route) => {
      const path = new URL(route.request().url()).pathname;
      if (path.endsWith("/capabilities"))
        return route.fulfill({ json: { canRenderJavaScript: false } });
      if (path.endsWith("/history")) return route.fulfill({ json: [] });
      if (path.endsWith("/start")) starts.push(path);
      return route.fulfill({
        status: 500,
        json: { error: "Unexpected request" },
      });
    });
    await page.goto("/p/e2e-project/audit");
    await expect(
      page.getByRole("checkbox", { name: "Render JavaScript pages" }),
    ).toBeDisabled();
    await expect(
      page.getByText("JavaScript rendering isn't available right now."),
    ).toBeVisible();
    await expect(page.getByText(/Estimated \$.*for up to/)).toHaveCount(0);
    await expect(
      page.getByRole("heading", { name: "Start New Audit" }),
    ).toBeVisible();
    expect(starts).toEqual([]);
    expect(paid).toEqual([]);
  });
}
