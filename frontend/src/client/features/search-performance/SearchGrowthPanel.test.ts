// @vitest-environment happy-dom
import { act, createElement, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { SearchGrowthPanel } from "./SearchGrowthPanel";
import {
  analyzeGrowth,
  listGrowthActions,
  saveGrowthAction,
  type GrowthReport,
} from "./growthApi";
import { takeAnalystDraft } from "@/client/features/sam/analystDraft";
import { readSearchContext } from "@/client/features/content-editor/searchContext";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

const { navigate } = vi.hoisted(() => ({ navigate: vi.fn() }));
vi.mock("@tanstack/react-router", async (original) => ({
  ...(await original<object>()),
  useNavigate: () => navigate,
  Link: ({ children }: { children: ReactNode }) =>
    createElement("span", null, children),
}));
vi.mock("./growthApi", async (original) => ({
  ...(await original<object>()),
  analyzeGrowth: vi.fn(),
  listGrowthActions: vi.fn(),
  saveGrowthAction: vi.fn(),
}));

function fixture(): GrowthReport {
  return {
    id: "report-a",
    generatedAt: "2026-10-19T12:00:00Z",
    range: { startDate: "2026-09-18", endDate: "2026-10-15" },
    previousRange: { startDate: "2026-08-21", endDate: "2026-09-17" },
    siteUrl: "https://example.test/",
    propertyId: "properties/123",
    currency: "USD",
    timeZone: "UTC",
    brandTerms: ["acme"],
    warnings: ["Key events are not verified leads."],
    limited: false,
    truncated: false,
    matchedPages: 1,
    gscPageCount: 1,
    ga4PageCount: 1,
    pages: [
      {
        page: "https://example.test/guide",
        current: {
          clicks: 30,
          impressions: 1000,
          ctr: 0.03,
          position: 9,
          sessions: null,
          keyEvents: null,
          keyEventRate: null,
          engagementRate: null,
          revenue: null,
        },
        previous: null,
        priority: 20,
        queries: [
          {
            query: "booking guide",
            clicks: 10,
            impressions: 300,
            position: 9,
            branded: false,
            competingPages: [],
          },
        ],
        findings: [
          {
            kind: "near_top",
            title: "Queries are within reach",
            evidence: "1,000 impressions; average position 9.",
            action: "Review search intent before rewriting.",
            confidence: "Review candidate",
          },
        ],
      },
    ],
  };
}
let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
let client: QueryClient;
async function mount() {
  container = document.createElement("div");
  document.body.append(container);
  root = createRoot(container);
  client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  await act(async () =>
    root.render(
      createElement(
        QueryClientProvider,
        { client },
        createElement(SearchGrowthPanel, { projectId: "project-a" }),
      ),
    ),
  );
}
async function click(text: string) {
  const button = [...container.querySelectorAll("button")].find((b) =>
    b.textContent?.includes(text),
  );
  if (!button) throw new Error(`Missing button ${text}`);
  await act(async () => {
    button.click();
  });
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
}
beforeEach(() => {
  vi.clearAllMocks();
  sessionStorage.clear();
  vi.mocked(analyzeGrowth).mockResolvedValue(fixture());
  vi.mocked(listGrowthActions).mockResolvedValue([]);
  vi.mocked(saveGrowthAction).mockResolvedValue({ id: "action-a" });
});
afterEach(async () => {
  if (root) await act(async () => root.unmount());
  client?.clear();
  container?.remove();
});

describe("Search growth action boundaries", () => {
  it("loads no Google report or AI action on mount and keeps missing metrics unavailable", async () => {
    await mount();
    expect(analyzeGrowth).not.toHaveBeenCalled();
    expect(listGrowthActions).not.toHaveBeenCalled();
    await click("Find opportunities");
    expect(analyzeGrowth).toHaveBeenCalledOnce();
    expect(container.textContent).toContain("Unavailable");
    expect(container.textContent).toContain(
      "Review search intent before rewriting.",
    );
  });
  it("opens a scoped editable analyst draft without sending any paid message", async () => {
    await mount();
    await click("Find opportunities");
    await click("Create AI improvement brief");
    expect(navigate).toHaveBeenCalledWith({
      to: "/p/$projectId/sam",
      params: { projectId: "project-a" },
    });
    expect(takeAnalystDraft("project-a")).toContain(
      "https://example.test/guide",
    );
    expect(takeAnalystDraft("project-b")).toBe("");
    expect(saveGrowthAction).not.toHaveBeenCalled();
  });
  it("carries page and query evidence into the editor without modifying a draft", async () => {
    await mount();
    await click("Find opportunities");
    await click("Improve this page in editor");
    expect(readSearchContext("project-a")?.queries).toEqual(["booking guide"]);
    expect(readSearchContext("project-b")).toBeNull();
    expect(navigate).toHaveBeenCalledWith(
      expect.objectContaining({
        to: "/p/$projectId/content-editor",
        search: { keyword: "booking guide", growth: "1" },
      }),
    );
  });
  it("keeps provider failure visible and does not retry the analysis", async () => {
    vi.mocked(analyzeGrowth).mockRejectedValue(new Error("offline"));
    await mount();
    await click("Find opportunities");
    expect(analyzeGrowth).toHaveBeenCalledOnce();
    expect(container.textContent).toContain("offline");
  });
});
