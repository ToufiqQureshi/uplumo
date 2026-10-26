import { expect, test } from "@playwright/test";
import { installGrowthFixtures } from "./search-growth.fixtures";

test("search growth stays explicit and carries evidence through save, analyst and editor", async ({
  page,
}, testInfo) => {
  const { requests, paid, chatModes, growthInputs } =
    await installGrowthFixtures(page);
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.goto("/p/e2e-project/search-performance");
  const analyze = page.getByRole("button", { name: "Find opportunities" });
  await expect(analyze).toBeVisible();
  expect(requests.filter((p) => p.endsWith("/ga4/growth/report"))).toHaveLength(
    0,
  );
  await page.getByRole("button", { name: "Load my GA4 goals" }).click();
  await page
    .getByLabel("What matters to your business?")
    .selectOption("generate_lead");
  await analyze.click();
  expect(growthInputs[0]).toMatchObject({
    goalEvent: "generate_lead",
    comparison: "previous",
    expanded: false,
  });
  await expect(
    page.getByRole("heading", { name: "Search clicks are falling" }),
  ).toBeVisible();
  await page
    .getByText("Query evidence and save action", { exact: true })
    .click();
  await page.getByLabel("What will you change?").fill("Review the enquiry CTA");
  await page.getByRole("button", { name: "Save action and baseline" }).click();
  await expect(
    page.getByRole("button", { name: "Saved", exact: true }),
  ).toBeVisible();
  await page.getByText("Saved actions and results", { exact: true }).click();
  await expect(
    page.getByText("Review the enquiry CTA", { exact: true }),
  ).toBeVisible();
  await page
    .getByText("Investigate tracking, devices and AI citations", {
      exact: true,
    })
    .click();
  await page.getByText("Measurement setup", { exact: true }).click();
  await page
    .getByRole("button", { name: "Check GA4 measurement setup" })
    .click();
  await expect(
    page.getByText("Configured key events: 1.", { exact: false }),
  ).toBeVisible();
  await page.getByText("Performance by device", { exact: true }).click();
  await page.getByRole("button", { name: "Compare devices" }).click();
  await expect(
    page.getByRole("cell", { name: "mobile", exact: true }),
  ).toBeVisible();
  await page.getByText("AI visibility evidence", { exact: true }).click();
  await page
    .getByRole("button", { name: "Load saved AI citation evidence" })
    .click();
  await expect(
    page.getByText("No completed AI tracking run is saved yet."),
  ).toBeVisible();
  await page
    .getByText("Why are clicks low or falling?", { exact: true })
    .click();
  await page
    .getByRole("button", { name: "Investigate this page's clicks" })
    .click();
  await expect(
    page.getByText("Fewer impressions are becoming clicks", { exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Check Google's stored index status" })
    .click();
  await expect(
    page.getByText("Submitted and indexed", { exact: false }),
  ).toBeVisible();
  await page.getByText("Attribution and tracking", { exact: true }).click();
  await page.getByRole("button", { name: "Check attribution" }).click();
  await expect(
    page.getByText("Verify your completed lead event in DebugView."),
  ).toBeVisible();
  await page.getByText("Visitor journey", { exact: true }).click();
  await page.getByRole("button", { name: "Inspect tracked journey" }).click();
  await expect(
    page.getByRole("cell", { name: "1. page_view", exact: true }),
  ).toBeVisible();
  await page.getByText("Sitemap health", { exact: true }).click();
  await page
    .getByRole("button", { name: "Check submitted sitemap status" })
    .click();
  await expect(page.getByText("Errors: 2", { exact: false })).toBeVisible();
  await page
    .getByText("Weekly actions and important drop alerts", { exact: true })
    .click();
  await page
    .getByRole("button", { name: "Enable weekly checks for selected goal" })
    .click();
  await expect(
    page.getByText("Weekly checks enabled for generate_lead.", {
      exact: false,
    }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Turn off weekly checks" }).click();
  await expect(
    page.getByText("Weekly checks are off.", { exact: false }),
  ).toBeVisible();
  expect(paid).toEqual([]);
  await page.screenshot({
    path: testInfo.outputPath("search-growth-desktop.png"),
    fullPage: true,
  });
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(
    page.getByRole("button", { name: "Refresh analysis" }),
  ).toBeVisible();
  await page.screenshot({
    path: testInfo.outputPath("search-growth-mobile.png"),
    fullPage: true,
  });
  await page
    .getByRole("button", { name: "Improve this page in editor" })
    .click();
  await expect(
    page.getByRole("heading", { name: "Improve an existing page" }),
  ).toBeVisible();
  await expect(page.getByText("Related queries: booking guide")).toBeVisible();
  expect(paid).toEqual([]);
  await page.goto("/p/e2e-project/search-performance");
  await page.getByRole("button", { name: "Find opportunities" }).click();
  await page
    .getByRole("button", { name: "Create AI improvement brief" })
    .click();
  const composer = page.getByPlaceholder(
    "Ask SAM to research, analyze, or track anything…",
  );
  await expect(composer).toHaveValue(/Search growth analysis/);
  await expect(
    page.getByText(
      "SEO analyst · read-only data tools · AI reply uses credits",
    ),
  ).toBeVisible();
  expect(paid).toEqual([]);
  await composer.press("Enter");
  await expect.poll(() => chatModes).toEqual(["search_growth"]);
  expect(paid).toEqual(["/api/v1/projects/e2e-project/sam/sessions/chat"]);
});

for (const width of [320, 390]) {
  test(`mobile investigations stay readable at ${width}px without a wider page`, async ({
    page,
  }, testInfo) => {
    const { paid, requests } = await installGrowthFixtures(page);
    await page.setViewportSize({ width, height: 844 });
    await page.goto("/p/e2e-project/search-performance");
    await page.getByRole("button", { name: "Find opportunities" }).click();
    await page
      .getByText("Investigate tracking, devices and AI citations", {
        exact: true,
      })
      .click();
    await expect(page.getByLabel("Step 1 event")).toBeHidden();
    expect(
      requests.filter((path) => path.endsWith("/ga4/growth/funnel")),
    ).toHaveLength(0);
    await page.getByText("Visitor journey", { exact: true }).click();
    await expect(page.getByLabel("Step 3 event")).toHaveValue("generate_lead");
    await page.getByRole("button", { name: "Add a step", exact: true }).click();
    await page.getByLabel("Step 4 event").fill("purchase");
    await page
      .getByRole("button", { name: "Remove step 4", exact: true })
      .click();
    await expect(page.getByLabel("Step 4 event")).toHaveCount(0);
    await page.getByRole("button", { name: "Inspect tracked journey" }).click();
    const journey = page.getByLabel("Journey step results", { exact: true });
    await expect(journey).toBeVisible();
    await expect(
      journey.getByRole("heading", { name: "1. page_view", exact: true }),
    ).toBeVisible();
    await expect(
      journey.getByText("Left before next step", { exact: true }),
    ).toBeVisible();
    await expect(journey.getByText("150", { exact: true })).toBeVisible();
    await expect(journey.getByText("75.0%", { exact: true })).toBeVisible();
    await journey.scrollIntoViewIfNeeded();
    await page.screenshot({
      path: testInfo.outputPath(`search-growth-journey-${width}.png`),
      fullPage: false,
    });
    await page.getByText("Performance by device", { exact: true }).click();
    await expect(page.getByLabel("Step 1 event")).toBeHidden();
    await page
      .getByRole("button", { name: "Compare devices", exact: true })
      .click();
    await expect(
      page
        .getByLabel("Google organic device results")
        .getByRole("heading", { name: "mobile", exact: true }),
    ).toBeVisible();
    await page.getByText("Sitemap health", { exact: true }).click();
    await page
      .getByRole("button", { name: "Check submitted sitemap status" })
      .click();
    await expect(page.getByText("Errors: 2", { exact: false })).toBeVisible();
    await page
      .getByText("Why are clicks low or falling?", { exact: true })
      .click();
    await page
      .getByRole("button", { name: "Investigate this page's clicks" })
      .click();
    await expect(
      page
        .getByLabel("Query comparison results")
        .getByRole("heading", { name: "booking guide", exact: true }),
    ).toBeVisible();
    await expect
      .poll(() =>
        page.evaluate(
          () => document.documentElement.scrollWidth <= window.innerWidth,
        ),
      )
      .toBe(true);
    expect(paid).toEqual([]);
  });
}
