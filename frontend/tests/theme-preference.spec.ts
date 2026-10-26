import { expect, test } from "@playwright/test";
import { installGrowthFixtures } from "./search-growth.fixtures";

for (const width of [1280, 375]) {
  test(`header theme choice persists and matches Settings at ${width}px`, async ({
    page,
  }) => {
    const { paid } = await installGrowthFixtures(page);
    await page.setViewportSize({ width, height: 850 });
    await page.emulateMedia({ colorScheme: "dark" });
    await page.goto("/settings");
    const trigger = page
      .locator("header")
      .getByRole("button", { name: "Change color theme" });
    await expect(trigger).toBeVisible();
    await expect(page.locator("html")).toHaveAttribute(
      "data-theme",
      "uplumo-dark",
    );

    // The header menu must also work with the keyboard on a narrow screen.
    await trigger.focus();
    await trigger.press("Enter");
    await page
      .getByRole("menuitemradio", { name: "Light", exact: true })
      .click();
    await expect(
      page.locator('[data-slot="dropdown-menu-content"]'),
    ).toHaveCount(0);
    await expect(page.locator("html")).toHaveAttribute("data-theme", "uplumo");
    await expect(page.locator("html")).toHaveCSS("color-scheme", "light");
    await expect(
      page.getByRole("button", { name: "Light", exact: true }),
    ).toHaveAttribute("aria-pressed", "true");
    expect(
      await page.evaluate(() => localStorage.getItem("theme-preference")),
    ).toBe("light");

    await page.reload();
    await expect(trigger).toBeVisible();
    await expect(page.locator("html")).toHaveAttribute("data-theme", "uplumo");
    await trigger.click();
    await expect(
      page.getByRole("menuitemradio", { name: "Light", exact: true }),
    ).toHaveAttribute("aria-checked", "true");
    await page
      .getByRole("menuitemradio", { name: "Dark", exact: true })
      .click();
    await expect(page.locator("html")).toHaveAttribute(
      "data-theme",
      "uplumo-dark",
    );
    await expect(page.locator("html")).toHaveCSS("color-scheme", "dark");
    expect(paid).toEqual([]);
  });
}

test("System follows OS appearance and an explicit choice overrides it", async ({
  page,
}) => {
  await installGrowthFixtures(page);
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto("/settings");
  const trigger = page
    .locator("header")
    .getByRole("button", { name: "Change color theme" });
  await trigger.click();
  await page.getByRole("menuitemradio", { name: "Light", exact: true }).click();
  await expect(page.locator('[data-slot="dropdown-menu-content"]')).toHaveCount(
    0,
  );
  await expect(page.locator("html")).toHaveAttribute("data-theme", "uplumo");
  await trigger.click();
  await page
    .getByRole("menuitemradio", { name: "System", exact: true })
    .click();
  await expect(page.locator('[data-slot="dropdown-menu-content"]')).toHaveCount(
    0,
  );
  await expect(page.locator("html")).toHaveAttribute(
    "data-theme",
    "uplumo-dark",
  );
  expect(
    await page.evaluate(() => localStorage.getItem("theme-preference")),
  ).toBeNull();

  // Both menus are closed: the always-mounted header keeps system changes live.
  await page.emulateMedia({ colorScheme: "light" });
  await expect(page.locator("html")).toHaveAttribute("data-theme", "uplumo");
  await page.reload();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "uplumo");
  await trigger.click();
  await page.getByRole("menuitemradio", { name: "Dark", exact: true }).click();
  await page.emulateMedia({ colorScheme: "dark" });
  await page.emulateMedia({ colorScheme: "light" });
  await expect(page.locator("html")).toHaveAttribute(
    "data-theme",
    "uplumo-dark",
  );
});
