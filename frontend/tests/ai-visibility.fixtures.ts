import type { Page } from "@playwright/test";

/**
 * A paid workspace with two comparable AI visibility checks. Every API call
 * is recorded so tests can count paid requests; paid endpoints answer with
 * small valid bodies so their buttons can be clicked through.
 */
const PAID_AI_PATH =
  /\/ai-search\/(tracking\/run$|tracking\/runs\/[^/]+\/sentiment$|brand-lookup$|prompt-explorer$)|\/content\/assistant\/generate$/u;

const now = new Date().toISOString();

const brands = (own: boolean, rival: boolean) => [
  { name: "Example", domain: "example.com", mentioned: own, cited: false },
  { name: "Rival", domain: "rival.example", mentioned: rival, cited: false },
];

const trackerConfig = {
  brand: "Example",
  domain: "example.com",
  competitors: [{ name: "Rival", domain: "rival.example" }],
  prompts: [
    { text: "best crm for small teams", topic: "General", paused: false },
    { text: "cheap crm", topic: "General", paused: false },
  ],
  engines: ["chat_gpt", "gemini"],
  locationCode: 2840,
  languageCode: "en",
  interval: "weekly",
  enabled: true,
  maxCreditsPerRun: 4,
};

const run = (id: string, mentioned: number) => ({
  id,
  configuration: trackerConfig,
  scopeKey: "scope",
  trigger: "manual",
  status: "completed",
  errorCode: "",
  createdAt: id === "r2" ? now : "2026-11-01T10:00:00Z",
  finishedAt: now,
  completed: 4,
  failed: 0,
  answered: 4,
  mentioned,
  cited: 0,
});

const answer = (
  prompt: string,
  engine: string,
  own: boolean,
  rival: boolean,
) => ({
  id: `${prompt}-${engine}`,
  prompt,
  topic: "General",
  engine,
  status: "completed",
  answer: own ? "Example is a good pick." : "Rival is the usual choice.",
  errorCode: "",
  collectedAt: now,
  citations: [
    {
      url: "https://review.example/best-crm",
      domain: "review.example",
      title: "Best CRM 2026",
      matchedBrand: false,
    },
    ...(own
      ? [
          {
            url: "https://example.com/pricing",
            domain: "example.com",
            title: "Example pricing",
            // Saved checks did not mark own pages; the domain must suffice.
            matchedBrand: false,
          },
        ]
      : []),
  ],
  brands: brands(own, rival),
  recommended: rival ? [{ name: "Rival", position: 1 }] : [],
});

const WRITER: Record<string, string> = {
  prompt_ideas:
    "best|best crm for agencies\ncompare|crm vs spreadsheet for small teams",
  action_plan: "1. Get listed on review.example.",
  business_post: "We fixed it.",
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

type ApiLog = { calls: string[]; bodies: Record<string, unknown>[] };

export async function mockAiVisibility(page: Page): Promise<ApiLog> {
  const log: ApiLog = { calls: [], bodies: [] };
  await page.route("**/api/**", async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    log.calls.push(path);
    const sent: unknown =
      request.method() === "POST" ? request.postDataJSON() : null;
    const posted: Record<string, unknown> = isRecord(sent) ? sent : {};
    log.bodies.push(posted);
    let body: unknown = {};
    if (path === "/api/auth/get-session")
      body = {
        user: {
          id: "test-user",
          name: "Tracker Test",
          email: "tracker@example.com",
          emailVerified: true,
          createdAt: now,
          updatedAt: now,
        },
        session: {
          id: "test-session",
          userId: "test-user",
          expiresAt: new Date(Date.now() + 3600000).toISOString(),
          token: "playwright-session",
          createdAt: now,
          updatedAt: now,
        },
      };
    else if (path.endsWith("/projects/test-project/get"))
      body = {
        id: "test-project",
        name: "Test site",
        domain: "example.com",
        locationCode: 2840,
        languageCode: "en",
        createdAt: now,
      };
    else if (path === "/api/v1/projects/list")
      body = [
        {
          id: "test-project",
          name: "Test site",
          domain: "example.com",
          locationCode: 2840,
          languageCode: "es",
          createdAt: now,
        },
      ];
    else if (path === "/api/v1/config/content-writer-status")
      body = { configured: true, message: "", maxCreditsPerAction: 12 };
    else if (path === "/api/v1/billing/credits")
      body = {
        plan: "business",
        status: "active",
        currentPeriodEnd: null,
        balanceCredits: 5000,
        balanceUsd: 5,
        monthlyGrantCredits: 15000,
        paymentsConfigured: true,
        availablePlans: ["basic", "pro", "business"],
      };
    else if (path.endsWith("/ai-search/tracking"))
      body = {
        configuration: trackerConfig,
        nextCheckAt: null,
        runs: [run("r2", 1), run("r1", 0)],
        providerConfigured: true,
        maxPrompts: 3,
      };
    else if (path.endsWith("/tracking/save"))
      body = {
        configuration: posted,
        nextCheckAt: null,
        runs: [run("r2", 1), run("r1", 0)],
        providerConfigured: true,
        maxPrompts: 3,
      };
    else if (path.endsWith("/tracking/estimate"))
      body = { answers: 4, credits: 4, monthlyCredits: 18, ceiling: true };
    else if (path.endsWith("/tracking/run")) body = { runId: "r3" };
    else if (path.endsWith("/tracking/runs/r1"))
      body = [
        answer("best crm for small teams", "chat_gpt", false, true),
        answer("best crm for small teams", "gemini", true, false),
        answer("cheap crm", "chat_gpt", false, false),
        answer("cheap crm", "gemini", false, true),
      ];
    else if (path.endsWith("/tracking/runs/r2"))
      body = [
        answer("best crm for small teams", "chat_gpt", false, true),
        answer("best crm for small teams", "gemini", false, true),
        answer("cheap crm", "chat_gpt", true, false),
        answer("cheap crm", "gemini", false, true),
      ];
    else if (path.endsWith("/sentiment"))
      body = { classified: 1, notMentioned: 3, remaining: 0 };
    else if (path.endsWith("/content/assistant/generate"))
      body = { content: WRITER[String(posted.action)] ?? "Done." };
    else if (path.endsWith("/ai-search/prompt-explorer"))
      body = {
        prompt: String(posted.prompt),
        highlightBrand: null,
        fetchedAt: now,
        results: [
          {
            status: "success",
            model: "chat_gpt",
            modelName: "gpt",
            text: "Rival is popular for small teams.",
            citations: [],
            fanOutQueries: [],
            brandMentioned: null,
            outputTokens: 10,
            webSearch: true,
            webSearchCountryCode: null,
          },
        ],
      };
    else if (path.endsWith("/brand-lookup/history")) body = [];
    else if (path.endsWith("/ai-search/brand-lookup"))
      body = {
        query: String(posted.query),
        detectedTargetType: "domain",
        resolvedTarget: String(posted.query),
        fetchedAt: now,
        hasData: false,
        totalMentions: null,
        totalAiSearchVolume: null,
        perPlatform: [],
        shareOfVoice: null,
        topPages: [],
        topQueries: [],
        monthlyVolume: [],
      };
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(body),
    });
  });
  return log;
}

/** The paid requests a log holds, by path. */
export const paidCalls = (log: ApiLog) =>
  log.calls.filter((path) => PAID_AI_PATH.test(path));
