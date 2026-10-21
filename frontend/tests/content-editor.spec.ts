import { expect, test, type TestInfo } from "@playwright/test";
import {
  project,
  paidEndpoints,
  requestBody,
  mockEditorApi,
  openEditor,
} from "./content-editor.fixtures";

test("a rapid AI writer double click sends one keyed request", async ({
  page,
}) => {
  const requests: string[] = [];
  const writerRequests: Record<string, unknown>[] = [];
  await openEditor(page, requests, writerRequests);
  await page.getByRole("tab", { name: "AI help" }).click();
  await expect(page.getByText("Up to 15 credits")).toHaveCount(10);
  const outline = page.getByRole("button", { name: "Create outline" });
  await expect(outline).toBeEnabled();
  await outline.dispatchEvent("click");
  await outline.dispatchEvent("click");
  await expect.poll(() => writerRequests.length).toBe(1);
  expect(writerRequests[0]?.idempotencyKey).toMatch(/^[0-9a-f-]{36}$/iu);
});

test("block insert menu stays beside the editor and inside the viewport", async ({
  page,
}, testInfo: TestInfo) => {
  const requests: string[] = [];
  await openEditor(page, requests);
  await page.getByText("How it looks on Google").click();

  const article = page.locator(
    '[role="textbox"][aria-labelledby="article-editor-label"]',
  );
  await article.scrollIntoViewIfNeeded();
  await article.click();
  await page.keyboard.press("/");
  const menu = page.getByRole("menu", { name: "Add a block" });
  await expect(menu).toBeVisible();

  const menuBox = await menu.boundingBox();
  const fields = await Promise.all(
    [
      "#content-draft-title",
      "#content-draft-keyword",
      "#content-draft-meta",
    ].map((selector) => page.locator(selector).boundingBox()),
  );
  expect(menuBox).not.toBeNull();
  if (!menuBox) throw new Error("Block menu has no visible bounds.");
  expect(menuBox.x).toBeGreaterThanOrEqual(0);
  expect(menuBox.y).toBeGreaterThanOrEqual(0);
  expect(menuBox.x + menuBox.width).toBeLessThanOrEqual(1280);
  expect(menuBox.y + menuBox.height).toBeLessThanOrEqual(900);
  for (const field of fields) {
    if (!field) continue;
    const overlaps =
      menuBox.x < field.x + field.width &&
      menuBox.x + menuBox.width > field.x &&
      menuBox.y < field.y + field.height &&
      menuBox.y + menuBox.height > field.y;
    expect(overlaps).toBe(false);
  }

  await page.keyboard.press("Escape");
  await expect(menu).toBeHidden();
  await article.click();
  await page.keyboard.press("/");
  await expect(menu).toBeVisible();
  await page.getByRole("heading", { name: "Content Editor" }).click();
  await expect(menu).toBeHidden();

  await testInfo.attach("content-editor-block-menu.png", {
    body: await page.screenshot(),
    contentType: "image/png",
  });
});

test("page load, 30 seconds of typing, and autosave trigger no paid lookups", async ({
  page,
}) => {
  test.setTimeout(60_000);
  const requests: string[] = [];
  await openEditor(page, requests);
  const title = page.getByLabel(/Article title/);
  for (const letter of "A practical SEO guide for small teams") {
    await title.pressSequentially(letter, { delay: 1000 });
  }
  await expect(page.getByText(/· Saved$/u)).toBeVisible({ timeout: 10_000 });

  expect(requests.filter((path) => paidEndpoints.test(path))).toEqual([]);
  expect(
    requests.filter((path) => path.endsWith("/content/drafts/save")),
  ).toHaveLength(1);
});

test("saved draft link restores writing and metadata after reload", async ({
  page,
}) => {
  const requests: string[] = [];
  await openEditor(page, requests);
  await page.route("**/api/v1/projects/*/content/authors/list", (route) =>
    route.fulfill({
      json: [
        {
          id: "author-review",
          projectId: project.id,
          name: "Review Author",
          role: "Editor",
          bio: "Writes practical guides",
          profileUrl: "",
          createdAt: "2026-10-20T00:00:00Z",
          updatedAt: "2026-10-20T00:00:00Z",
        },
      ],
    }),
  );
  await page
    .getByLabel(/Article title/)
    .fill("A useful guide that survives reload");
  await page
    .locator('[contenteditable="true"]')
    .first()
    .fill("A practical article that must stay in the saved draft.");
  await page.getByRole("tab", { name: "Publish", exact: true }).click();
  await page.getByLabel("Draft author").selectOption("author-review");
  await page.getByRole("button", { name: "Mark as reviewed today" }).click();
  const saved = page.waitForResponse(
    (r) => r.url().endsWith("/content/drafts/save") && r.status() === 200,
  );
  await page.getByRole("button", { name: "Save draft", exact: true }).click();
  await saved;
  await expect(page.getByLabel("Draft author")).toHaveValue("author-review");
  await expect(page.getByText(/^Reviewed \d{4}-\d{2}-\d{2}$/)).toBeVisible();
  await expect(page).toHaveURL(/draft=e2e-draft/u);
  await page.reload();
  await expect(page.getByLabel(/Article title/)).toHaveValue(
    "A useful guide that survives reload",
  );
  await expect(page.locator('[contenteditable="true"]').first()).toContainText(
    "A practical article that must stay in the saved draft.",
  );
  await page.getByRole("tab", { name: "Publish", exact: true }).click();
  await expect(page.getByLabel("Draft author")).toHaveValue("author-review");
  await expect(page.getByText(/^Reviewed \d{4}-\d{2}-\d{2}$/)).toBeVisible();
  expect(requests.filter((path) => paidEndpoints.test(path))).toEqual([]);
});

test("a missing saved draft never opens a writable blank replacement", async ({
  page,
}) => {
  const requests: string[] = [];
  await mockEditorApi(page, requests);
  await page.route("**/api/v1/projects/*/content/drafts/missing/get", (route) =>
    route.fulfill({
      status: 404,
      json: {
        error: {
          code: "not_found",
          message: "This draft was not found in this project.",
        },
      },
    }),
  );
  await page.goto(`/p/${project.id}/content-editor?draft=missing`);
  await expect(
    page.getByText("This draft was not found in this project."),
  ).toBeVisible();
  await expect(page.getByLabel(/Article title/)).toHaveCount(0);
  expect(
    requests.filter((path) => path.endsWith("/content/drafts/save")),
  ).toEqual([]);
  await page.getByRole("button", { name: "Start a new draft" }).click();
  await expect(page.getByLabel(/Article title/)).toHaveValue("");
  await expect(page).not.toHaveURL(/draft=/u);
});

test("unsaved editor navigation can be cancelled without losing writing", async ({
  page,
}) => {
  const requests: string[] = [];
  await openEditor(page, requests);
  await page.getByLabel(/Article title/).fill("Keep this unsaved writing");
  page.once("dialog", (dialog) => dialog.dismiss());
  await page.getByRole("link", { name: "Dashboard", exact: true }).click();
  await expect(page).toHaveURL(/\/content-editor$/u);
  await expect(page.getByLabel(/Article title/)).toHaveValue(
    "Keep this unsaved writing",
  );
  page.once("dialog", (dialog) => dialog.accept());
  await page.getByRole("link", { name: "Dashboard", exact: true }).click();
  await expect(page).toHaveURL(/\/p\/e2e-project$/u);
  expect(requests.filter((path) => paidEndpoints.test(path))).toEqual([]);
});

test("a slow save locks editing until the saved snapshot is applied", async ({
  page,
}) => {
  const requests: string[] = [];
  await openEditor(page, requests);
  let releaseSave: (() => void) | undefined;
  const blocked = new Promise<void>((resolve) => {
    releaseSave = resolve;
  });
  let submitted: Record<string, unknown> | undefined;
  await page.route("**/content/drafts/save", async (route) => {
    submitted = requestBody(route);
    await blocked;
    await route.fulfill({
      json: {
        ...submitted,
        id: "slow-draft",
        projectId: project.id,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      },
    });
  });
  const title = page.getByLabel(/Article title/);
  await title.fill("Original title");
  await page.getByRole("button", { name: "Save draft", exact: true }).click();
  await expect.poll(() => submitted?.title).toBe("Original title");
  await expect(title).toBeDisabled();
  releaseSave?.();
  await expect(page).toHaveURL(/draft=slow-draft/u);
  await expect(title).toBeEnabled();
  await expect(title).toHaveValue("Original title");
  await title.fill("New writing after the save completes");
  await expect(page.getByText(/Unsaved changes/u)).toBeVisible();
  expect(requests.filter((path) => paidEndpoints.test(path))).toEqual([]);
});

test("a refused autosave does not loop without a new edit or explicit retry", async ({
  page,
}) => {
  const requests: string[] = [];
  await openEditor(page, requests);
  const attempts: Record<string, unknown>[] = [];
  await page.route("**/content/drafts/save", async (route) => {
    const input = requestBody(route);
    attempts.push(input);
    if (attempts.length === 1)
      return route.fulfill({
        status: 400,
        json: {
          error: {
            code: "validation_error",
            message: "This draft could not be saved.",
          },
        },
      });
    return route.fulfill({
      json: {
        ...input,
        id: "retried-draft",
        projectId: project.id,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      },
    });
  });
  const title = page.getByLabel(/Article title/);
  await title.fill("Unsaved writing");
  await page.getByRole("button", { name: "Save draft", exact: true }).click();
  await expect.poll(() => attempts.length).toBe(1);
  await expect(title).toBeEnabled();
  await page.waitForTimeout(7000);
  expect(attempts).toHaveLength(1);
  await title.fill("Corrected writing");
  await expect(page).toHaveURL(/draft=retried-draft/u);
  await expect(title).toHaveValue("Corrected writing");
  expect(attempts).toHaveLength(2);
  expect(requests.filter((path) => paidEndpoints.test(path))).toEqual([]);
});
