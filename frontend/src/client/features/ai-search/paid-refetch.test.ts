// @vitest-environment happy-dom
import {
  QueryClient,
  QueryClientProvider,
  focusManager,
  onlineManager,
} from "@tanstack/react-query";
import { act, createElement } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { BrandLookupPage } from "./BrandLookupPage";
import { PromptExplorerPage } from "./PromptExplorerPage";
import { explorePrompt, lookupBrand } from "./aiSearchApi";

vi.mock("./aiSearchApi", () => ({
  explorePrompt: vi.fn(),
  lookupBrand: vi.fn(),
  getBrandVisibilityHistory: vi.fn(),
}));
vi.mock("@/client/features/billing/HostedPlanGate", () => ({
  useHostedPlanGate: () => "paid",
}));
vi.mock("@/client/features/billing/credits/useConfirmedCost", () => ({
  useConfirmedCost: () => ({ dialog: null, request: vi.fn() }),
}));
vi.mock("@tanstack/react-router", () => ({ Link: () => null }));
vi.mock("./ResearchPageShell", () => ({ ResearchPageShell: () => null }));
vi.mock("./AiVisibilityGuide", () => ({ AiVisibilityGuide: () => null }));
vi.mock("./components/PromptExplorerForm", () => ({
  PromptExplorerForm: () => null,
}));
vi.mock("./components/PromptExplorerResults", () => ({
  PromptExplorerResults: () => null,
}));
vi.mock("./components/BrandLookupResults", () => ({
  BrandLookupResults: () => null,
}));
vi.mock("./components/BrandVisibilityHistoryCard", () => ({
  BrandVisibilityHistoryCard: () => null,
}));
vi.mock("./components/BrandLookupSearchCard", () => ({
  BrandLookupSearchCard: () => null,
}));
vi.mock("@/client/components/RecentSearches", () => ({
  RecentSearches: () => null,
}));
vi.mock("@/client/components/PageHeader", () => ({ BackLink: () => null }));
vi.mock("@/client/components/ui/badge", () => ({ Badge: () => null }));
vi.mock("@/shared/prompt-explorer-labels", () => ({
  formatModelLabel: (model: string) => model,
}));
vi.mock("@/shared/targetDetection", () => ({
  detectTarget: (value: string) => ({ type: "keyword", value }),
}));
vi.mock("@/shared/researchScope", () => ({
  parseResearchTarget: () => null,
  RESEARCH_SCOPE_LABELS: {},
  toScopeSearchParam: vi.fn(),
}));
vi.mock("@/types/schemas/ai-search", () => ({
  BRAND_LOOKUP_MAX_INPUT_LENGTH: 200,
  PROMPT_EXPLORER_MAX_PROMPT_LENGTH: 2000,
  parseCompetitorList: () => [],
}));
vi.mock("@/client/hooks/usePromptExplorerSearchHistory", () => ({
  usePromptExplorerSearchHistory: () => ({
    history: [],
    isLoaded: true,
    addSearch: vi.fn(),
    removeHistoryItem: vi.fn(),
  }),
}));
const projectMarket = vi.hoisted(() => ({
  value: undefined as { locationCode: number } | undefined,
}));
vi.mock("@/client/features/projects/projectQueries", () => ({
  useProjectMarket: () => projectMarket.value,
}));
vi.mock("@/client/hooks/useBrandLookupSearchHistory", () => ({
  useBrandLookupSearchHistory: () => ({
    history: [],
    isLoaded: true,
    addSearch: vi.fn(),
    removeHistoryItem: vi.fn(),
  }),
}));

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
let root: Root;
let host: HTMLDivElement;
let client: QueryClient;
beforeEach(() => {
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  client = new QueryClient({
    defaultOptions: {
      queries: {
        retry: 2,
        retryDelay: 1,
        refetchOnWindowFocus: true,
        refetchOnReconnect: true,
      },
    },
  });
  vi.mocked(lookupBrand).mockRejectedValue(new Error("provider unavailable"));
  vi.mocked(explorePrompt).mockRejectedValue(new Error("provider unavailable"));
  projectMarket.value = undefined;
});
afterEach(() => {
  act(() => root.unmount());
  client.clear();
  host.remove();
  vi.clearAllMocks();
  focusManager.setFocused(undefined);
  onlineManager.setOnline(true);
});

it.each(["brand", "prompt"] as const)(
  "never re-buys a failed %s lookup on window focus or reconnect",
  async (kind) => {
    const page =
      kind === "brand"
        ? createElement(BrandLookupPage, {
            projectId: "p1",
            initialQuery: "Acme",
            initialCompetitors: [],
            initialScope: undefined,
            initialLocationCode: 2356,
            onSearchChange: vi.fn(),
          })
        : createElement(PromptExplorerPage, {
            projectId: "p1",
            urlState: {
              prompt: "best hotels",
              highlightBrand: "",
              models: ["chat_gpt"],
              webSearch: true,
              webSearchCountryCode: "default",
            },
            onSubmit: vi.fn(),
          });
    const request = kind === "brand" ? lookupBrand : explorePrompt;
    act(() =>
      root.render(createElement(QueryClientProvider, { client }, page)),
    );
    await act(async () => {
      await vi.waitFor(() =>
        expect(client.getQueryCache().getAll()[0]?.state.status).toBe("error"),
      );
    });
    expect(request).toHaveBeenCalledTimes(1);
    await act(async () => {
      focusManager.setFocused(false);
      focusManager.setFocused(true);
      await new Promise((resolve) => setTimeout(resolve, 20));
      onlineManager.setOnline(false);
      onlineManager.setOnline(true);
      await new Promise((resolve) => setTimeout(resolve, 20));
    });
    expect(request).toHaveBeenCalledTimes(1);
  },
);

function renderBrandLookup(initialLocationCode: number | undefined) {
  act(() =>
    root.render(
      createElement(
        QueryClientProvider,
        { client },
        createElement(BrandLookupPage, {
          projectId: "p1",
          initialQuery: "Acme",
          initialCompetitors: [],
          initialScope: undefined,
          initialLocationCode,
          onSearchChange: vi.fn(),
        }),
      ),
    ),
  );
}

it("buys a brand lookup for the chosen country in its language", async () => {
  renderBrandLookup(2276);
  await act(async () => {
    await vi.waitFor(() => expect(lookupBrand).toHaveBeenCalledTimes(1));
  });
  expect(lookupBrand).toHaveBeenCalledWith(
    expect.objectContaining({ locationCode: 2276, languageCode: "de" }),
  );
});

it("never buys a lookup before it knows the country", async () => {
  renderBrandLookup(undefined);
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 20));
  });
  expect(lookupBrand).not.toHaveBeenCalled();

  projectMarket.value = { locationCode: 2356 };
  renderBrandLookup(undefined);
  await act(async () => {
    await vi.waitFor(() => expect(lookupBrand).toHaveBeenCalledTimes(1));
  });
  expect(lookupBrand).toHaveBeenCalledWith(
    expect.objectContaining({ locationCode: 2356, languageCode: "en" }),
  );
});
