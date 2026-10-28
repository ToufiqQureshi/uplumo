import type { Page, Route } from "@playwright/test";

const metrics = { clicks: 60, impressions: 1000, ctr: 0.06, position: 9 };
const current = {
  ...metrics,
  sessions: 200,
  keyEvents: 10,
  keyEventRate: 0.05,
  engagementRate: 0.7,
  revenue: 0,
  goalRate: 0.03,
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
  goalEvent: "generate_lead",
  comparison: "previous",
  rowLimit: 1000,
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

export async function installGrowthFixtures(page: Page) {
  const requests: string[] = [];
  const paid: string[] = [];
  let saved = false;
  let monitorEnabled = false;
  const growthInputs: unknown[] = [];
  const chatModes: unknown[] = [];
  await page.addInitScript(() => localStorage.setItem("sam-beta-opt-in", "1"));
  await page.route("**/api/**", async (route: Route) => {
    const path = new URL(route.request().url()).pathname;
    requests.push(path);
    if (
      /sam\/sessions\/chat|assistant\/generate|content\/brief|ai-search\/tracking\/run|keywords\/research|cite-fix\/google-ai-overview\/refresh/u.test(
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
    else if (path.endsWith("/ga4/growth/report")) {
      growthInputs.push(route.request().postDataJSON());
      body = report;
    } else if (path.endsWith("/ga4/growth/goals"))
      body = ["generate_lead", "purchase"];
    else if (path.endsWith("/ga4/growth/clicks"))
      body = {
        page: pageEvidence.page,
        range,
        previousRange,
        truncated: false,
        message: "Returned query evidence; missing queries are unknown.",
        findings: [
          {
            kind: "ctr_drop",
            title: "Fewer impressions are becoming clicks",
            evidence: "CTR 10% → 6%.",
            action: "Review the same queries and devices before editing.",
            confidence: "Cause unverified",
          },
        ],
        queries: [
          {
            query: "booking guide",
            current,
            previous: { ...current, clicks: 100 },
          },
        ],
      };
    else if (path.endsWith("/gsc/url-inspection/inspect"))
      body = {
        siteUrl: report.siteUrl,
        results: [
          {
            url: pageEvidence.page,
            result: {
              indexStatusResult: {
                verdict: "PASS",
                coverageState: "Submitted and indexed",
              },
            },
          },
        ],
      };
    else if (path.endsWith("/ga4/growth/tracking"))
      body = {
        range,
        limited: false,
        rows: [],
        findings: [],
        checklist: ["Verify your completed lead event in DebugView."],
      };
    else if (path.endsWith("/ga4/growth/funnel"))
      body = {
        range,
        steps: ["page_view", "form_start", "generate_lead"],
        limited: false,
        message: "Ordered user funnel; missing events are unknown.",
        rows: [
          {
            funnelStepName: "1. page_view",
            activeUsers: 200,
            funnelStepCompletionRate: 0.25,
            funnelStepAbandonments: 150,
            funnelStepAbandonmentRate: 0.75,
          },
        ],
      };
    else if (path.endsWith("/gsc/search-performance/sitemaps"))
      body = {
        siteUrl: report.siteUrl,
        truncated: false,
        message: "Sitemap submission does not prove indexing.",
        sitemaps: [
          {
            path: "https://example.com/sitemap.xml",
            lastSubmitted: "",
            lastDownloaded: "",
            pending: false,
            isIndex: true,
            errors: 2,
            warnings: 1,
            submitted: 42,
          },
        ],
      };
    else if (/\/ga4\/growth\/monitor(?:-save)?$/u.test(path)) {
      if (path.endsWith("monitor-save")) {
        const input: unknown = route.request().postDataJSON();
        if (
          typeof input !== "object" ||
          input === null ||
          !("enabled" in input) ||
          typeof input.enabled !== "boolean"
        )
          throw new Error("Expected an explicit monitoring toggle");
        monitorEnabled = input.enabled;
      }
      body = {
        enabled: monitorEnabled,
        goalEvent: monitorEnabled ? "generate_lead" : "",
        nextCheckAt: null,
        checkedAt: null,
        digest: null,
        lastError: null,
      };
    } else if (path.endsWith("/ga4/growth/list"))
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
      body = { issues: [], keyEvents: [{ eventName: "generate_lead" }] };
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

  return { requests, paid, chatModes, growthInputs };
}
