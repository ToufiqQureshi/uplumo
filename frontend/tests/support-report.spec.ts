import { expect, test } from "@playwright/test";
import { installGrowthFixtures } from "./search-growth.fixtures";

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null;

for (const width of [1280, 375]) {
  test(`support report preserves failed input and retries one ticket at ${width}px`, async ({
    page,
  }, testInfo) => {
    const { paid } = await installGrowthFixtures(page);
    const submissions: Array<Record<string, unknown>> = [];
    await page.route("**/api/v1/support/reports", async (route) => {
      const body: unknown = route.request().postDataJSON();
      submissions.push(isRecord(body) ? body : {});
      await route.fulfill({
        status: submissions.length === 1 ? 503 : 201,
        contentType: "application/json",
        body: JSON.stringify(
          submissions.length === 1
            ? { error: { code: "internal", message: "Try again" } }
            : { ticketId: "12345678-1234-4234-8234-123456789012" },
        ),
      });
    });
    await page.setViewportSize({ width, height: 960 });
    await page.goto("/support");
    await expect(
      page.getByRole("heading", { name: "Bugs & Feedback" }),
    ).toBeVisible();
    await expect(
      page.getByRole("heading", { name: "Report a problem" }),
    ).toBeVisible();
    await page.getByLabel("Report type").click();
    await page.getByRole("option", { name: "Credits or payment" }).click();
    const description = page.getByLabel("What happened?");
    const send = page.getByRole("button", { name: "Send report", exact: true });
    await description.fill("404");
    await page
      .getByLabel("Affected page")
      .fill("/billing/plan?token=secret#private");
    await page.getByLabel("Screenshot (optional)").setInputFiles({
      name: "balance.png",
      mimeType: "image/png",
      buffer: Buffer.from(
        "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jRZkAAAAASUVORK5CYII=",
        "base64",
      ),
    });
    await send.click();
    await expect(page.getByRole("alert")).toContainText(
      "at least 10 characters",
    );
    await expect(description).toBeFocused();
    expect(submissions).toHaveLength(0);
    await description.fill(
      "My credit balance did not update after the payment completed.",
    );
    await expect(page.getByRole("alert")).toHaveCount(0);
    await page.screenshot({
      path: testInfo.outputPath(`support-${width}.png`),
      fullPage: true,
    });
    await send.click();
    await expect(page.getByRole("alert")).toContainText(
      "Something went wrong on our side",
    );
    await expect(page.getByLabel("What happened?")).toHaveValue(
      "My credit balance did not update after the payment completed.",
    );
    expect(submissions).toHaveLength(1);
    await send.click();
    await expect(page.getByRole("status")).toContainText(
      "12345678-1234-4234-8234-123456789012",
    );
    expect(submissions).toHaveLength(2);
    expect(submissions[0]?.submissionId).toBe(submissions[1]?.submissionId);
    expect(submissions[1]).toMatchObject({
      kind: "billing",
      pagePath: "/billing/plan",
    });
    expect(submissions[1]?.screenshot).toMatch(/^data:image\/png;base64,/u);
    expect(paid).toEqual([]);
    expect(
      await page
        .locator("body")
        .evaluate((body) => body.scrollWidth <= window.innerWidth),
    ).toBe(true);
  });
}

test("personal settings asks for confirmation before account deletion", async ({
  page,
}) => {
  await installGrowthFixtures(page);
  const submitted: unknown[] = [];
  await page.route("**/api/auth/delete-account", async (route) => {
    submitted.push(route.request().postDataJSON() as unknown);
    await route.fulfill({
      status: 409,
      json: {
        code: "SHARED_WORKSPACE",
        message: "Transfer ownership or leave shared workspaces first.",
      },
    });
  });
  await page.goto("/settings");
  const deleteButton = page.getByRole("button", { name: "Delete my account" });
  await deleteButton.click();
  const dialog = page.getByRole("alertdialog");
  await expect(dialog).toBeVisible();
  expect(submitted).toEqual([]);
  await dialog.getByRole("button", { name: "Cancel" }).click();
  await expect(dialog).toHaveCount(0);
  expect(submitted).toEqual([]);
  await deleteButton.click();
  await dialog.getByRole("button", { name: "Delete my account" }).click();
  await expect(dialog.getByRole("alert")).toContainText("shared workspaces");
  expect(submitted).toEqual([{ confirm: true }]);
});
