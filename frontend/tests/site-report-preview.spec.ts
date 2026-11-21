import { expect, test } from "@playwright/test";
import { installGrowthFixtures } from "./search-growth.fixtures";

test("preview is private and saves exactly the selected report content", async ({
  page,
}) => {
  const { paid } = await installGrowthFixtures(page);
  const path = "**/api/v1/projects/e2e-project/reports";
  const writes: string[] = [];
  const saved: string[] = [];
  const preview = `<!doctype html><html lang="en"><head><title>Report</title></head><body>
<main><section id="analytics"><h2>Analytics</h2>
<div data-report-pair="Traffic channels">
<figure class="bars" data-report-view="chart">
<div data-report-row="Internal"><span class="bar-track">Internal 900</span></div>
<div data-report-row="Organic"><span class="bar-track">Organic 100</span></div>
</figure>
<div class="table" data-report-view="table"><table><thead><tr><th>Channel</th><th>Sessions</th></tr></thead>
<tbody><tr data-report-row="Internal"><td>Internal</td><td>900</td></tr><tr data-report-row="Organic"><td>Organic</td><td>100</td></tr></tbody>
</table></div></div></section></main></body></html>`;
  await page.route(`${path}/list`, (route) =>
    route.fulfill({ json: { reports: [] } }),
  );
  await page.route(`${path}/templates/list`, (route) =>
    route.fulfill({ json: { templates: [], remaining: 0 } }),
  );
  await page.route(`${path}/preview`, (route) =>
    route.fulfill({ json: { html: preview, skipped: [] } }),
  );
  await page.route(`${path}/sharing`, (route) => {
    writes.push("share");
    return route.fulfill({ json: {} });
  });
  await page.route(`${path}/save`, (route) => {
    writes.push("save");
    const body: unknown = route.request().postDataJSON();
    if (
      !body ||
      typeof body !== "object" ||
      !("html" in body) ||
      typeof body.html !== "string"
    ) {
      throw new Error("Expected the reviewed report HTML");
    }
    saved.push(body.html);
    return route.fulfill({
      json: {
        reportId: "r1",
        title: "Report",
        created: true,
        htmlBytes: body.html.length,
      },
    });
  });
  await page.goto("/p/e2e-project/reports");
  await page
    .getByRole("button", { name: "Create client report", exact: true })
    .first()
    .click();
  await page
    .getByRole("button", { name: "Preview report", exact: true })
    .click();
  const dialog = page.getByRole("dialog");
  await expect(
    dialog.getByRole("heading", { name: "Preview client report" }),
  ).toBeVisible();
  const report = page.frameLocator('iframe[title="Client report preview"]');
  await expect(report.getByText("Internal 900")).toBeVisible();
  expect(writes).toEqual([]);
  await dialog
    .getByText("Analytics · Traffic channels", { exact: true })
    .click();
  await dialog.getByLabel("Internal", { exact: true }).uncheck();
  await dialog.getByLabel("Display", { exact: true }).selectOption("chart");
  await expect(report.getByText("Organic 100")).toBeVisible();
  await expect(report.getByText("Internal 900")).toHaveCount(0);
  await expect(report.locator("table")).toHaveCount(0);
  await expect(
    dialog.getByLabel("Create a link anyone can view"),
  ).not.toBeChecked();
  await dialog.getByRole("button", { name: "Save privately" }).click();
  await expect.poll(() => saved.length).toBe(1);
  expect(saved[0]).toContain("Organic 100");
  expect(saved[0]).not.toMatch(/Internal|900|<table/);
  expect(writes).toEqual(["save"]);
  expect(paid).toEqual([]);
});
