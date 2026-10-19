import { expect, test, type Route } from "@playwright/test";

const metrics = { clicks: 60, impressions: 1000, ctr: 0.06, position: 9 };
const current = {
  ...metrics,
  sessions: 200,
  keyEvents: 10,
  keyEventRate: 0.05,
  engagementRate: 0.7,
  revenue: 0,
};
const range = { startDate: "2026-09-18", endDate: "2026-10-15" };
const previousRange = { startDate: "2026-08-21", endDate: "2026-09-17" };
const pageEvidence = {
  page: "https://example.com/guide",
  current,
  previous: { ...current, clicks: 100 },
  queries: [
    {
      query: "booking guide",
      clicks: 30,
      impressions: 500,
      position: 9,
      branded: false,
      competingPages: [],
    },
  ],
  findings: [
    {
      kind: "traffic_loss",
      title: "Search clicks are falling",
      evidence: "100 → 60 Google clicks across equal-length periods.",
      action: "Review lost queries and search intent.",
      confidence: "Review candidate",
    },
  ],
  priority: 30,
};
const report = {
  id: "report-a",
  generatedAt: "2026-10-19T12:00:00Z",
  range,
  previousRange,
  siteUrl: "https://example.com/",
  propertyId: "properties/123",
  currency: "USD",
  timeZone: "UTC",
  brandTerms: [],
  pages: [pageEvidence],
  warnings: [
    "Key events reflect your configuration, not automatically verified leads.",
  ],
  limited: false,
  truncated: false,
  matchedPages: 1,
  gscPageCount: 1,
  ga4PageCount: 1,
};

test("search growth stays explicit and carries evidence through save, analyst and editor", async ({
  page,
}) => {
  const requests: string[] = [];
  const paid: string[] = [];
  let saved = false;
  const chatModes: unknown[] = [];
  await page.addInitScript(() => localStorage.setItem("sam-beta-opt-in", "1"));
  await page.route("**/api/**", async (route: Route) => {
    const path = new URL(route.request().url()).pathname;
    requests.push(path);
    if (
      /sam\/sessions\/chat|assistant\/generate|content\/brief|ai-search\/tracking\/run|keywords\/research/u.test(
        path,
      )
    )
      paid.push(path);
    if (path.endsWith("/sam/sessions/chat")) {
      const input: unknown = route.request().postDataJSON();
      if (typeof input === "object" && input !== null && "mode" in input)
        chatModes.push(input.mode);
      await route.fulfill({
        status: 200,
        headers: {
          "content-type": "text/event-stream",
          "x-vercel-ai-ui-message-stream": "v1",
        },
        body: 'data: {"type":"start","messageId":"reply-a"}\n\ndata: {"type":"text-start","id":"text-a"}\n\ndata: {"type":"text-delta","id":"text-a","delta":"Review the observed traffic decline first."}\n\ndata: {"type":"text-end","id":"text-a"}\n\ndata: {"type":"finish"}\n\ndata: [DONE]\n\n',
      });
      return;
    }
    const now = new Date().toISOString();
    let body: unknown = {};
    if (path === "/api/auth/get-session")
      body = {
        user: {
          id: "e2e-user",
          name: "Growth Test",
          email: "growth@example.com",
          emailVerified: true,
          createdAt: now,
          updatedAt: now,
        },
        session: {
          id: "e2e-session",
          userId: "e2e-user",
          expiresAt: new Date(Date.now() + 3600000).toISOString(),
          token: "playwright-session",
          createdAt: now,
          updatedAt: now,
        },
      };
    else if (path.endsWith("/projects/e2e-project/get"))
      body = {
        id: "e2e-project",
        name: "Example site",
        domain: "example.com",
        locationCode: 2840,
        languageCode: "en",
        createdAt: now,
      };
    else if (path.endsWith("/gsc/search-performance/report"))
      body = {
        connected: true,
        range: {
          ...range,
          prevStartDate: previousRange.startDate,
          prevEndDate: previousRange.endDate,
          comparisonAvailable: true,
        },
        totals: metrics,
        prevTotals: metrics,
        daily: [{ date: range.startDate, clicks: 5, impressions: 100 }],
        strikingDistance: [],
        countries: [],
      };
    else if (path.endsWith("/ga4/growth/report")) body = report;
    else if (path.endsWith("/ga4/growth/list"))
      body = saved
        ? [
            {
              id: "action-a",
              page: pageEvidence.page,
              note: "Review the enquiry CTA",
              status: "planned",
              baseline: pageEvidence,
              source: report,
              baselineRange: range,
              createdAt: now,
              appliedAt: null,
            },
          ]
        : [];
    else if (path.endsWith("/ga4/growth/save")) {
      saved = true;
      body = { id: "action-a" };
    } else if (path.endsWith("/ga4/growth/devices"))
      body = {
        rows: [
          {
            deviceCategory: "mobile",
            sessions: 120,
            keyEvents: 4,
            sessionKeyEventRate: 0.03,
          },
        ],
        limited: false,
        range,
      };
    else if (path.endsWith("/ga4/measurement-health"))
      body = { issues: ["no_key_events_configured"], keyEvents: [] };
    else if (path.endsWith("/ai-search/tracking"))
      body = {
        configuration: null,
        nextCheckAt: null,
        runs: [],
        providerConfigured: true,
      };
    else if (path.endsWith("/sam/sessions/list"))
      body = [
        {
          id: "session-a",
          title: "SEO analyst",
          createdAt: now,
          updatedAt: now,
        },
      ];
    else if (
      path.endsWith("/sam/sessions/messages") ||
      path.endsWith("/content/drafts/list")
    )
      body = [];
    else if (path === "/api/v1/billing/usage")
      body = {
        plan: "pro",
        unlimited: false,
        chargesCredits: true,
        balanceCredits: 9000,
        monthlyCredits: 10000,
        periodStart: now,
        periodEnd: null,
        spent: [],
        costs: [{ feature: "ai_assistant", unit: "reply", credits: 40 }],
      };
    else if (path === "/api/v1/billing/status")
      body = {
        plan: "pro",
        status: "active",
        currentPeriodEnd: null,
        cancelAtPeriodEnd: false,
        unlimited: false,
        paymentsConfigured: false,
        availablePlans: ["basic", "pro", "agency"],
        usage: {
          sites: { used: 1, limit: 5, exceeded: false },
          members: { used: 1, limit: 5, exceeded: false },
          trackedKeywords: { used: 0, limit: 100, exceeded: false },
        },
      };
    else if (path === "/api/v1/billing/plans") body = { plans: [] };
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(body),
    });
  });
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.goto("/p/e2e-project/search-performance");
  const analyze = page.getByRole("button", { name: "Find opportunities" });
  await expect(analyze).toBeVisible();
  expect(requests.filter((p) => p.endsWith("/ga4/growth/report"))).toHaveLength(
    0,
  );
  await analyze.click();
  await expect(
    page.getByRole("heading", { name: "Search clicks are falling" }),
  ).toBeVisible();
  await page
    .getByText("Query evidence and save action", { exact: true })
    .click();
  await page.getByLabel("What will you change?").fill("Review the enquiry CTA");
  await page.getByRole("button", { name: "Save action and baseline" }).click();
  await expect(
    page.getByRole("button", { name: "Saved", exact: true }),
  ).toBeVisible();
  await page.getByText("Saved actions and results", { exact: true }).click();
  await expect(
    page.getByText("Review the enquiry CTA", { exact: true }),
  ).toBeVisible();
  await page
    .getByText("Investigate tracking, devices and AI citations", {
      exact: true,
    })
    .click();
  await page
    .getByRole("button", { name: "Check GA4 measurement setup" })
    .click();
  await expect(
    page.getByText("No key events are configured.", { exact: false }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Compare Google organic traffic by device" })
    .click();
  await expect(
    page.getByRole("cell", { name: "mobile", exact: true }),
  ).toBeVisible();
  await page
    .getByRole("button", { name: "Load saved AI citation evidence" })
    .click();
  await expect(
    page.getByText("No completed AI tracking run is saved yet."),
  ).toBeVisible();
  await page.screenshot({
    path: "/workspace/search-growth-desktop.png",
    fullPage: true,
  });
  await page.setViewportSize({ width: 390, height: 844 });
  await expect(
    page.getByRole("button", { name: "Refresh analysis" }),
  ).toBeVisible();
  await page.screenshot({
    path: "/workspace/search-growth-mobile.png",
    fullPage: true,
  });
  await page
    .getByRole("button", { name: "Improve this page in editor" })
    .click();
  await expect(
    page.getByRole("heading", { name: "Improve an existing page" }),
  ).toBeVisible();
  await expect(page.getByText("Related queries: booking guide")).toBeVisible();
  expect(paid).toEqual([]);
  await page.goto("/p/e2e-project/search-performance");
  await page.getByRole("button", { name: "Find opportunities" }).click();
  await page
    .getByRole("button", { name: "Create AI improvement brief" })
    .click();
  const composer = page.getByPlaceholder(
    "Ask SAM to research, analyze, or track anything…",
  );
  await expect(composer).toHaveValue(/Search growth analysis/);
  await expect(
    page.getByText(
      "SEO analyst · read-only data tools · AI reply uses credits",
    ),
  ).toBeVisible();
  expect(paid).toEqual([]);
  await composer.press("Enter");
  await expect.poll(() => chatModes).toEqual(["search_growth"]);
  expect(paid).toEqual(["/api/v1/projects/e2e-project/sam/sessions/chat"]);
});
