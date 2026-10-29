import { expect, test } from "@playwright/test";
import { mockEditorApi } from "./content-editor.fixtures";
import { ready } from "./cite-fix.fixtures";

test("topics open a free pillar draft and saved citation history", async ({
  page,
}) => {
  const requests: string[] = [];
  await mockEditorApi(page, requests);
  await page.route("**/api/v1/projects/*/cite-fix/report", (route) =>
    route.fulfill({
      json: {
        ...ready,
        topics: {
          score: 24,
          pagesAnalyzed: 2,
          limited: false,
          recommendations: ["Write a pillar page for solar panels."],
          clusters: [
            {
              topic: "solar panels",
              pageCount: 2,
              interlinkRatio: 0.5,
              pillar: null,
              pages: [{ url: "https://example.com/solar", title: "Solar" }],
              orphans: [],
              thin: [],
              duplicates: [],
            },
          ],
        },
        proof: {
          checkedAt: "2026-10-28T10:00:00Z",
          domain: "example.com",
          answers: 2,
          citingYou: 1,
          rivals: [],
          yourPages: [],
          misses: [],
          visibility: {
            runs: 2,
            historyLimited: false,
            sourcesLimited: false,
            domains: [
              {
                name: "example.com",
                firstSeen: "2026-10-27T10:00:00Z",
                lastSeen: "2026-10-28T10:00:00Z",
                averageRank: 1.5,
                latestCount: 1,
                momentum: "up",
              },
            ],
            pages: [],
            prompts: [
              {
                prompt: "Where?",
                engine: "chat_gpt",
                answers: 2,
                citedAnswers: 2,
                averageRank: 1.5,
                latestRank: 1,
                inlineAnswers: 1,
                averageAnswerRank: 2,
                latestAnswerRank: 2,
              },
            ],
          },
        },
      },
    }),
  );
  await page.goto("/p/e2e-project/cite-fix");
  await expect(
    page.getByRole("heading", { name: "Your topics", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("Topic coverage: 24/100", { exact: false }),
  ).toBeVisible();
  await expect(
    page.getByText("Citation history", { exact: true }),
  ).toBeVisible();
  await page.getByText("Your citation position by question").click();
  await expect(
    page.getByText("Answer position:", { exact: false }),
  ).toContainText("#2.0");
  await page.getByRole("button", { name: "Start pillar draft" }).click();
  await expect(
    page.getByRole("heading", { name: "Content Editor" }),
  ).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Remove solar panels" }),
  ).toBeVisible();
  expect(
    requests.filter((path) =>
      /content\/brief|assistant\/generate|keywords\/research|ai-search\/tracking\/run|google-ai-overview\/refresh|cite-fix\/agent\/run/u.test(
        path,
      ),
    ),
  ).toEqual([]);
});
