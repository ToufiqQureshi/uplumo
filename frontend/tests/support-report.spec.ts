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
      page.getByRole("heading", { name: "Report a problem" }),
    ).toBeVisible();
    await page.getByLabel("Report type").click();
    await page.getByRole("option", { name: "Credits or payment" }).click();
    await page
      .getByLabel("What happened?")
      .fill("My credit balance did not update after the payment completed.");
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
    await page.screenshot({
      path: testInfo.outputPath(`support-${width}.png`),
      fullPage: true,
    });
    await page
      .getByRole("button", { name: "Send report", exact: true })
      .click();
    await expect(page.getByRole("alert")).toContainText(
      "Something went wrong on our side",
    );
    await expect(page.getByLabel("What happened?")).toHaveValue(
      "My credit balance did not update after the payment completed.",
    );
    expect(submissions).toHaveLength(1);
    await page
      .getByRole("button", { name: "Send report", exact: true })
      .click();
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
