import { expect, test, type Page } from "@playwright/test";
import { countBy } from "remeda";
import { mockAiVisibility, paidCalls } from "./ai-visibility.fixtures";
import { mockPageImport } from "./page-import.fixtures";

const base = "/p/test-project/ai-visibility";

test("every AI visibility tab and button works, and only clicks buy", async ({
  page,
}) => {
  const log = await mockAiVisibility(page);
  await mockPageImport(page, {
    id: "imported-pricing",
    projectId: "test-project",
    title: "Example pricing",
    content: "## Plans\n\nPricing text from the live page.",
  });
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.goto(base);

  // Tabs: each opens its own address and shows its own content.
  const nav = page.getByRole("navigation", { name: "AI visibility sections" });
  for (const [label, path, marker] of [
    ["Prompts", "/prompts", /click a cell to read the answer/u],
    ["Opportunities", "/opportunities", /AI leaves you out of/u],
    [
      "Sentiment",
      "/sentiment",
      /What ChatGPT, Gemini and other assistants say/u,
    ],
    ["Test a question", "/test", /Ask any prompt across/u],
    ["Market research", "/research", /See how AI search cites any brand/u],
    ["Prompt ideas", "/ideas", /The questions your customers type/u],
    ["Setup & history", "/setup", /Questions to monitor/u],
    ["Overview", "", /AI names Rival 3\.0× as often as Example\./u],
  ] as const) {
    await nav.getByRole("link", { name: label, exact: true }).click();
    await expect(page).toHaveURL(new RegExp(`${base}${path}(\\?.*)?$`, "u"));
    await expect(
      nav.getByRole("link", { name: label, exact: true }),
    ).toHaveAttribute("aria-current", "page");
    await expect(page.getByText(marker).first()).toBeVisible();
  }
  expect(paidCalls(log)).toEqual([]);

  // Overview: logos beside engine names, both verdict buttons, the alerts.
  const engines = page.locator("section").filter({ hasText: "By AI engine" });
  await expect(engines.locator("img")).toHaveCount(2);
  await page.getByText(/See the 1 question where AI leaves you out/u).click();
  await expect(page).toHaveURL(new RegExp(`${base}/prompts\\?show=gaps$`, "u"));
  await page.goBack();
  await page.getByText("Read the answers", { exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`${base}/prompts$`, "u"));
  await page.goBack();
  const changes = page.getByRole("region", {
    name: "Changes since the previous check",
  });
  await expect(changes.getByRole("listitem")).toHaveCount(2);
  await changes.getByRole("link", { name: "Read the answer" }).nth(1).click();
  await expect(
    page.getByRole("region", { name: "Saved answer" }),
  ).toContainText("What ChatGPT answered");

  // Prompts: every filter and every cell.
  await nav.getByRole("link", { name: "Prompts", exact: true }).click();
  const rows = page.locator("tbody tr");
  for (const [filter, count] of [
    [/AI leaves you out · 1/u, 1],
    [/AI names you · 1/u, 1],
    [/All questions · 2/u, 2],
  ] as const) {
    await page.getByRole("button", { name: filter }).click();
    await expect(rows).toHaveCount(count);
  }
  await expect(page.locator("thead img")).toHaveCount(2);
  const cells = page.getByRole("button", { name: /Read the answer\.$/u });
  await expect(cells).toHaveCount(4);
  for (let i = 0; i < 4; i++) {
    await cells.nth(i).click();
    const proof = page.getByRole("region", { name: "Saved answer" });
    await expect(proof).toContainText("Sources it trusted");
    await proof.getByRole("button", { name: "Close answer" }).click();
    await expect(proof).toBeHidden();
  }

  // A page of the customer's own site in an answer opens in the editor.
  const named = page.getByRole("button", { name: /^ChatGPT on "cheap crm"/u });
  await named.click();
  await page
    .getByRole("region", { name: "Saved answer" })
    .getByRole("button", { name: /Improve this page/u })
    .click();
  await expect(page).toHaveURL(/content-editor\?draft=imported-pricing/u);
  await expect(
    page.getByText("Pricing text from the live page."),
  ).toBeVisible();
  await page.goBack();

  // Opportunities: pick a gap, then the paid action plan (one call).
  await nav.getByRole("link", { name: "Opportunities", exact: true }).click();
  await page.getByRole("button", { name: /best crm for small teams/u }).click();
  await page.getByRole("button", { name: "Make my action plan" }).click();
  await expect(page.getByText("Get listed on review.example.")).toBeVisible();

  // Sentiment: classify (one call).
  await nav.getByRole("link", { name: "Sentiment", exact: true }).click();
  await page.getByRole("button", { name: /Classify \d+ answers?/u }).click();
  await expect
    .poll(() => paidCalls(log).filter((p) => p.endsWith("/sentiment")).length)
    .toBe(1);

  // Test a question: a submit buys once; a reload of that URL buys nothing.
  await nav.getByRole("link", { name: "Test a question", exact: true }).click();
  await page.getByLabel("Prompt").fill("best crm");
  await page.getByRole("button", { name: /^Run \d+ models?$/u }).click();
  await expect(
    page.getByText("Rival is popular for small teams."),
  ).toBeVisible();
  await page.reload();
  await expect(page.getByLabel("Prompt")).toHaveValue("best crm");
  await expect(
    page.getByText("Rival is popular for small teams."),
  ).toBeHidden();

  // Market research: a submit buys once.
  await nav.getByRole("link", { name: "Market research", exact: true }).click();
  await page.getByLabel("Brand name or domain").fill("example.com");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  await expect
    .poll(
      () => paidCalls(log).filter((p) => p.endsWith("/brand-lookup")).length,
    )
    .toBe(1);

  // Prompt ideas: generate (one call), pick one, add it to the tracker.
  await nav.getByRole("link", { name: "Prompt ideas", exact: true }).click();
  await page.getByLabel("Keyword").fill("crm");
  await page.getByRole("button", { name: "Generate question ideas" }).click();
  await page.getByRole("checkbox", { name: "best crm for agencies" }).click();
  await page.getByRole("button", { name: "Add 1 to the tracker" }).click();
  await expect(page.getByText(/Added 1 question/u)).toBeVisible();
  const saves = log.bodies.filter((_, i) =>
    log.calls[i]?.endsWith("/tracking/save"),
  );
  expect(saves.at(-1)?.prompts).toHaveLength(3);

  // Setup & history: add and remove a question, review, save, run, history.
  await nav.getByRole("link", { name: "Setup & history", exact: true }).click();
  await page.getByRole("button", { name: "Add question" }).click();
  await expect(
    page.getByRole("textbox", { name: /Question 3/u }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Remove question 3" }).click();
  await expect(page.getByRole("textbox", { name: /Question 3/u })).toBeHidden();
  await page.getByRole("button", { name: "Review cost" }).click();
  await expect(page.getByText(/4 credits/u).first()).toBeVisible();
  await page.getByRole("button", { name: "Run approved check" }).click();
  await expect
    .poll(
      () => paidCalls(log).filter((p) => p.endsWith("/tracking/run")).length,
    )
    .toBe(1);
  const runBody =
    log.bodies[log.calls.findIndex((p) => p.endsWith("/tracking/run"))];
  expect(runBody).toEqual({ maxCredits: 4 });
  await page.getByRole("button", { name: "Save questions & schedule" }).click();
  await expect
    .poll(() => log.calls.filter((p) => p.endsWith("/tracking/save")).length)
    .toBe(2);
  await page.getByRole("button", { name: "View answers" }).nth(1).click();
  await expect
    .poll(() => log.calls.some((p) => p.endsWith("/tracking/runs/r1")))
    .toBe(true);
  const download = page.waitForEvent("download");
  await page.getByRole("button", { name: "Export saved results" }).click();
  expect((await download).suggestedFilename()).toBe(
    "ai-visibility-results.json",
  );

  // Exactly the clicks above were paid: one each, nothing on load or reload.
  expect(
    countBy(paidCalls(log), (path) => path.split("/").slice(5).join("/")),
  ).toEqual({
    "content/assistant/generate": 2,
    "ai-search/tracking/runs/r2/sentiment": 1,
    "ai-search/prompt-explorer": 1,
    "ai-search/brand-lookup": 1,
    "ai-search/tracking/run": 1,
  });
});

test("a workspace without checks is sent to setup or a test question", async ({
  page,
}) => {
  await mockAiVisibility(page);
  await page.route("**/ai-search/tracking", (route) =>
    route.fulfill({
      json: {
        configuration: null,
        nextCheckAt: null,
        runs: [],
        providerConfigured: true,
      },
    }),
  );
  await page.goto(base);
  await expect(
    page.getByText("Pick the questions your customers ask AI"),
  ).toBeVisible();
  await page.getByText("Test one question first").click();
  await expect(page).toHaveURL(new RegExp(`${base}/test(\\?.*)?$`, "u"));
  await page.goBack();
  await page.getByText("Set up tracking").click();
  await expect(page).toHaveURL(new RegExp(`${base}/setup$`, "u"));
  await tabs(page, "Prompts");
  await expect(page.getByText("No answers saved yet")).toBeVisible();
  await page.getByText("Go to setup").click();
  await expect(page).toHaveURL(new RegExp(`${base}/setup$`, "u"));
});

async function tabs(page: Page, label: string) {
  await page
    .getByRole("navigation", { name: "AI visibility sections" })
    .getByRole("link", { name: label, exact: true })
    .click();
}
