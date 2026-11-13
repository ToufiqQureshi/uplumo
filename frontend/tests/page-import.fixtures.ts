import type { Page } from "@playwright/test";

/**
 * Answers "open this page in the editor" with a draft holding the page's
 * text, and that draft's reads. Register after a suite's catch-all route.
 */
export async function mockPageImport(
  page: Page,
  draft: { id: string; projectId: string; title: string; content: string },
) {
  const now = new Date().toISOString();
  const body = {
    ...draft,
    targetKeyword: "",
    slug: "",
    metaDescription: "",
    createdAt: now,
    updatedAt: now,
  };
  await page.route(
    new RegExp(`/content/drafts/(import|${draft.id}/get)$`, "u"),
    (route) => route.fulfill({ json: body }),
  );
}
