import { expect, test, type Page } from "@playwright/test";
import { z } from "zod";
import { installGrowthFixtures } from "./search-growth.fixtures";

async function reportsFixture(page: Page) {
  const base = await installGrowthFixtures(page);
  let saved:
    | { title: string; summary: string; html: string; definition: unknown }
    | undefined;
  let failSave = false;
  let saves = 0;
  const metadata = () => ({
    id: "visual-1",
    projectId: "e2e-project",
    title: saved?.title ?? "Report",
    summary: saved?.summary ?? "",
    skill: "Visual report",
    templateId: null,
    createdBy: "Member",
    createdByUserId: "u1",
    sizeBytes: 200,
    shareToken: null,
    sharedAt: null,
    createdAt: "2026-10-26T00:00:00Z",
    updatedAt: "2026-10-26T00:00:00Z",
  });
  await page.route("**/reports/**", async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (!path.startsWith("/api/")) return route.fallback();
    if (path.endsWith("/templates/list"))
      return route.fulfill({ json: { templates: [], remaining: 10 } });
    if (path.endsWith("/list"))
      return route.fulfill({ json: { reports: saved ? [metadata()] : [] } });
    if (path.endsWith("/get"))
      return route.fulfill({
        json: {
          report: metadata(),
          html: saved?.html,
          definition: saved?.definition,
        },
      });
    if (path.endsWith("/save")) {
      saves++;
      if (failSave)
        return route.fulfill({
          status: 503,
          json: {
            error: { code: "unavailable", message: "Try saving again." },
          },
        });
      const input: unknown = route.request().postDataJSON();
      saved = z
        .object({
          title: z.string(),
          summary: z.string(),
          html: z.string(),
          definition: z.unknown(),
        })
        .parse(input);
      return route.fulfill({
        json: {
          reportId: "visual-1",
          title: saved.title,
          created: saves === 1,
          htmlBytes: saved.html.length,
        },
      });
    }
    return route.fallback();
  });
  await page.route("**/r/visual-1", (route) =>
    route.fulfill({ contentType: "text/html", body: saved?.html ?? "" }),
  );
  return {
    ...base,
    getSaved: () => saved,
    getSaves: () => saves,
    fail: () => {
      failSave = true;
    },
    recover: () => {
      failSave = false;
    },
  };
}

async function upload(page: Page) {
  await page.getByLabel("Upload CSV file").setInputFiles({
    name: "traffic.csv",
    mimeType: "text/csv",
    buffer: Buffer.from(
      "Date,Channel,Revenue,PrivateNote\n2026-10-01,Search,100,hidden-client-detail\n2026-10-02,Search,50,private\n2026-10-03,Paid,20,private",
    ),
  });
  await expect(page.getByText("3 rows · 4 fields")).toBeVisible();
}

test("build, filter, save, reopen and edit without exposing private source data", async ({
  page,
}) => {
  const fixture = await reportsFixture(page);
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.goto("/p/e2e-project/reports");
  await page
    .getByRole("button", { name: "Create report", exact: true })
    .click();
  await page
    .getByLabel("Report title", { exact: true })
    .fill("October performance");
  await upload(page);
  await page.getByLabel("Report start date").fill("2026-10-01");
  await page.getByLabel("Report end date").fill("2026-10-31");
  await page.getByRole("button", { name: "Bar", exact: true }).click();
  await page.getByLabel("Break down by", { exact: true }).click();
  await page.getByRole("option", { name: "Channel", exact: true }).click();
  await page.getByLabel("Filter field", { exact: true }).click();
  await page.getByRole("option", { name: "Channel", exact: true }).click();
  await page.getByLabel("Contains", { exact: true }).fill("Search");
  const canvas = page.getByRole("main", { name: "Report canvas" });
  await expect(
    canvas.getByRole("strong").filter({ hasText: /^150$/ }),
  ).toBeVisible();
  await expect(canvas.getByText("Paid", { exact: true })).toHaveCount(0);
  await page.getByRole("button", { name: "Notes", exact: true }).click();
  await page
    .getByLabel("Findings and next steps", { exact: true })
    .fill("Search generated 150. Review landing pages next.");
  await page.getByRole("button", { name: "Preview", exact: true }).click();
  await expect(
    page.getByRole("complementary", { name: "Visual settings" }),
  ).toHaveCount(0);
  await page.screenshot({
    path: "test-results/report-preview.png",
    fullPage: true,
  });
  await page.getByRole("button", { name: "Save report", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Edit report", exact: true }),
  ).toBeVisible();
  const saved = fixture.getSaved()!;
  expect(saved.html).toContain("150");
  expect(saved.html).not.toContain("hidden-client-detail");
  expect(saved.html).not.toContain("PrivateNote");
  expect(JSON.stringify(saved.definition)).toContain("hidden-client-detail");
  await page.getByRole("button", { name: "Edit report", exact: true }).click();
  await expect(page.getByLabel("Report title", { exact: true })).toHaveValue(
    "October performance",
  );
  await expect(
    canvas.getByRole("strong").filter({ hasText: /^150$/ }),
  ).toBeVisible();
  await page
    .getByLabel("Report title", { exact: true })
    .fill("October final report");
  await page.getByRole("button", { name: "Save changes", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "October final report", exact: true }),
  ).toBeVisible();
  expect(fixture.getSaves()).toBe(2);
  expect(fixture.paid).toEqual([]);
  expect(errors).toEqual([]);
});

test("failed saves preserve the draft and leaving asks before discarding", async ({
  page,
}) => {
  const fixture = await reportsFixture(page);
  await page.goto("/p/e2e-project/reports?builder=true");
  await upload(page);
  await page.getByRole("button", { name: "KPI", exact: true }).click();
  fixture.fail();
  await page.getByRole("button", { name: "Save report", exact: true }).click();
  await expect(page.getByRole("alert")).toBeVisible();
  await expect(
    page.getByText("Unsaved changes", { exact: true }),
  ).toBeVisible();
  await expect(page.getByText("3 rows · 4 fields")).toBeVisible();
  expect(fixture.getSaves()).toBe(1);
  await page.getByRole("button", { name: "Back to reports" }).click();
  await expect(page.getByRole("alertdialog")).toBeVisible();
  await page.getByRole("button", { name: "Cancel", exact: true }).click();
  fixture.recover();
  await page.getByRole("button", { name: "Save report", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Edit report", exact: true }),
  ).toBeVisible();
});

test("mobile editor stays within the viewport and imports only on request", async ({
  page,
}) => {
  const fixture = await reportsFixture(page);
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/p/e2e-project/reports?builder=true");
  expect(
    fixture.requests.filter((url) => url.includes("search-performance/report")),
  ).toEqual([]);
  await upload(page);
  await page.getByRole("button", { name: "Table", exact: true }).click();
  await expect(page.getByRole("main", { name: "Report canvas" })).toBeVisible();
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= window.innerWidth,
    ),
  ).toBe(true);
  await page.screenshot({
    path: "test-results/report-mobile.png",
    fullPage: true,
  });
});

test("project source import is explicit and editing does not refetch", async ({
  page,
}) => {
  const fixture = await reportsFixture(page);
  await page.goto("/p/e2e-project/reports?builder=true");
  const reads = () =>
    fixture.requests.filter((url) =>
      url.includes("/gsc/search-performance/report"),
    );
  expect(reads()).toHaveLength(0);
  await page.getByLabel("Import from", { exact: true }).click();
  await page
    .getByRole("option", { name: "Search Console · daily", exact: true })
    .click();
  await page.getByLabel("Import start date").fill("2026-09-18");
  await page.getByLabel("Import end date").fill("2026-10-15");
  await page
    .getByRole("button", { name: "Add project data", exact: true })
    .click();
  await expect(
    page.getByText("1 rows · 3 fields", { exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Line", exact: true }).click();
  await page
    .getByRole("button", { name: "Change to Table", exact: true })
    .click();
  await page
    .getByLabel("Visual title", { exact: true })
    .fill("Google clicks by day");
  expect(reads()).toHaveLength(1);
  expect(fixture.paid).toEqual([]);
});
