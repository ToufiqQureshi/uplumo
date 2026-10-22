import { expect, type Page, type Route } from "@playwright/test";

export const project = {
  id: "e2e-project",
  name: "Example site",
  domain: "example.com",
  locationCode: 2840,
  languageCode: "en",
  createdAt: "2026-10-17T00:00:00.000Z",
};

export const paidEndpoints =
  /content\/brief|assistant\/generate|keywords\/research|domain|backlinks|brand|prompt|rank/u;

function isObjectRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function requestBody(route: Route) {
  const body: unknown = route.request().postDataJSON();
  if (!isObjectRecord(body)) throw new Error("Expected a JSON object body");
  return body;
}

export async function mockEditorApi(
  page: Page,
  requests: string[],
  writerRequests: Record<string, unknown>[] = [],
) {
  const drafts = new Map<string, Record<string, unknown>>();
  await page.route("**/api/**", async (route: Route) => {
    const pathname = new URL(route.request().url()).pathname;
    requests.push(pathname);
    const now = new Date().toISOString();
    const status = {
      plan: "basic",
      status: "active",
      currentPeriodEnd: null,
      cancelAtPeriodEnd: false,
      unlimited: false,
      paymentsConfigured: true,
      availablePlans: ["basic", "pro", "agency"],
      usage: {
        sites: { used: 1, limit: 1, exceeded: false },
        members: { used: 1, limit: 1, exceeded: false },
        trackedKeywords: { used: 0, limit: 100, exceeded: false },
      },
    };
    const usage = {
      plan: "basic",
      unlimited: false,
      chargesCredits: true,
      balanceCredits: 3900,
      monthlyCredits: 4000,
      periodStart: now,
      periodEnd: null,
      spent: [],
      costs: [{ feature: "ai_writing", unit: "action", credits: 13 }],
    };
    let body: unknown = {};

    if (pathname === "/api/auth/get-session") {
      body = {
        user: {
          id: "e2e-user",
          name: "Editor Test",
          email: "editor@example.com",
          emailVerified: true,
          createdAt: now,
          updatedAt: now,
        },
        session: {
          id: "e2e-session",
          userId: "e2e-user",
          expiresAt: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
          token: "playwright-session",
          createdAt: now,
          updatedAt: now,
        },
      };
    } else if (pathname.endsWith("/projects/e2e-project/get")) {
      body = project;
    } else if (pathname.endsWith("/content/drafts/list")) {
      body = [...drafts.values()];
    } else if (/\/content\/drafts\/[^/]+\/get$/u.test(pathname)) {
      body = drafts.get(pathname.split("/").at(-2) ?? "");
    } else if (pathname.endsWith("/content/drafts/save")) {
      const input = requestBody(route);
      if ("lastReviewedAt" in input)
        return route.fulfill({
          status: 400,
          json: {
            error: {
              code: "validation_error",
              message: "Unknown review date field.",
            },
          },
        });
      body = {
        ...input,
        id: typeof input.id === "string" ? input.id : "e2e-draft",
        projectId: project.id,
        createdAt: now,
        updatedAt: now,
        ...(typeof input.lastReviewedDate === "string" && input.lastReviewedDate
          ? {
              lastReviewedAt: `${input.lastReviewedDate.slice(0, 10)}T00:00:00Z`,
            }
          : {}),
      };
      if (isObjectRecord(body) && typeof body.id === "string")
        drafts.set(body.id, body);
    } else if (pathname === "/api/v1/config/content-writer-status") {
      body = {
        configured: true,
        message: "",
        maxCreditsPerAction: 15,
      };
    } else if (pathname.endsWith("/content/assistant/generate")) {
      await new Promise((resolve) => setTimeout(resolve, 300));
      writerRequests.push(requestBody(route));
      body = { content: "## A useful outline" };
    } else if (pathname === "/api/v1/billing/usage") {
      body = usage;
    } else if (pathname === "/api/v1/billing/status") {
      body = status;
    } else if (pathname === "/api/v1/billing/plans") {
      body = { plans: [] };
    }

    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(body),
    });
  });
}

export async function openEditor(
  page: Page,
  requests: string[],
  writerRequests: Record<string, unknown>[] = [],
) {
  await mockEditorApi(page, requests, writerRequests);
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.goto(`/p/${project.id}/content-editor`);
  await expect(
    page.getByRole("heading", { name: "Content Editor" }),
  ).toBeVisible();
  await expect(page.getByLabel(/Article title/)).toBeVisible();
  await expect(page.getByText("Your article", { exact: true })).toBeVisible();
}
