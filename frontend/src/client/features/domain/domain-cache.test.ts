// @vitest-environment happy-dom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, createElement } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import {
  getDomainOverview,
  getDomainKeywordsPage,
  getDomainPagesPage,
} from "./domainApi";
import { useDomainOverviewQuery } from "./hooks/useDomainOverviewQuery";
import { useDomainKeywordsQuery } from "./hooks/useDomainKeywordsQuery";
import { useDomainPagesQuery } from "./hooks/useDomainPagesQuery";
import { EMPTY_DOMAIN_FILTERS } from "./types";
import { allowPurchase } from "@/client/lib/paidIntent";

vi.mock("./domainApi", () => ({
  domainIntentKey: (projectId: string, domain: string) =>
    `domain:${projectId}:${domain}`,
  getDomainOverview: vi.fn(),
  getDomainKeywordsPage: vi.fn(),
  getDomainPagesPage: vi.fn(),
}));
vi.mock("./utils", () => ({
  toNumberOrUndefined: (value: string) =>
    value.trim() ? Number(value) : undefined,
  toPageSortMode: () => "traffic",
}));
Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

let root: Root;
let host: HTMLDivElement;
let client: QueryClient;
const input = {
  projectId: "project",
  domain: "example.com",
  scope: "domain" as const,
  locationCode: 2840,
};
function Lookup() {
  useDomainOverviewQuery(input);
  const tableInput = {
    ...input,
    page: 1,
    pageSize: 50,
    sortMode: "traffic" as const,
    sortOrder: "desc" as const,
    appliedFilters: EMPTY_DOMAIN_FILTERS,
    enabled: true,
  };
  useDomainKeywordsQuery(tableInput);
  useDomainPagesQuery(tableInput);
  return null;
}
function mount(show = true) {
  act(() =>
    root.render(
      createElement(
        QueryClientProvider,
        { client },
        show ? createElement(Lookup) : null,
      ),
    ),
  );
}
beforeEach(() => {
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  client = new QueryClient();
  vi.mocked(getDomainOverview).mockResolvedValue({
    ...input,
    displayTarget: "example.com",
    organicTraffic: 100,
    organicKeywords: 10,
    backlinks: null,
    referringDomains: null,
    hasData: true,
    fetchedAt: "2026-10-21T00:00:00Z",
  });
  vi.mocked(getDomainKeywordsPage).mockResolvedValue({
    domain: "example.com",
    page: 1,
    pageSize: 50,
    totalCount: 0,
    hasMore: false,
    keywords: [],
    fetchedAt: "2026-10-21T00:00:00Z",
  });
  vi.mocked(getDomainPagesPage).mockResolvedValue({
    domain: "example.com",
    page: 1,
    pageSize: 50,
    totalCount: 0,
    hasMore: false,
    pages: [],
    fetchedAt: "2026-10-21T00:00:00Z",
  });
});
afterEach(() => {
  act(() => root.unmount());
  client.clear();
  host.remove();
  vi.restoreAllMocks();
  vi.clearAllMocks();
});

it("does not repeat fresh paid report calls when returning after six minutes", async () => {
  const now = Date.now();
  vi.spyOn(Date, "now").mockReturnValue(now);
  mount();
  await act(async () => {
    await vi.waitFor(() => {
      expect(getDomainOverview).toHaveBeenCalledTimes(1);
      expect(getDomainKeywordsPage).toHaveBeenCalledTimes(1);
      expect(getDomainPagesPage).toHaveBeenCalledTimes(1);
    });
  });
  mount(false);
  vi.spyOn(Date, "now").mockReturnValue(now + 6 * 60_000);
  mount();
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 20));
  });
  expect(getDomainOverview).toHaveBeenCalledTimes(1);
  expect(getDomainKeywordsPage).toHaveBeenCalledTimes(1);
  expect(getDomainPagesPage).toHaveBeenCalledTimes(1);
});

it("requires an explicit retry after a failed paid report even when remounted", async () => {
  vi.mocked(getDomainOverview).mockRejectedValue(new Error("provider failed"));
  vi.mocked(getDomainKeywordsPage).mockRejectedValue(
    new Error("provider failed"),
  );
  vi.mocked(getDomainPagesPage).mockRejectedValue(new Error("provider failed"));
  mount();
  await act(async () => {
    await vi.waitFor(() =>
      expect(
        client
          .getQueryCache()
          .getAll()
          .every((query) => query.state.status === "error"),
      ).toBe(true),
    );
  });
  mount(false);
  mount();
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 20));
  });
  expect(getDomainOverview).toHaveBeenCalledTimes(1);
  expect(getDomainKeywordsPage).toHaveBeenCalledTimes(1);
  expect(getDomainPagesPage).toHaveBeenCalledTimes(1);
});

it("asks for cached reports only until the customer searches or retries", async () => {
  mount();
  await act(async () => {
    await vi.waitFor(() => {
      expect(getDomainOverview).toHaveBeenCalledTimes(1);
    });
  });
  // Opened from a link: nothing may be bought unasked.
  expect(vi.mocked(getDomainOverview).mock.calls[0]?.[0].data.cachedOnly).toBe(
    true,
  );
  expect(
    vi.mocked(getDomainKeywordsPage).mock.calls[0]?.[0].data.cachedOnly,
  ).toBe(true);

  allowPurchase("domain:project:example.com");
  await act(async () => {
    await client.refetchQueries();
  });
  expect(
    vi.mocked(getDomainOverview).mock.calls.at(-1)?.[0].data.cachedOnly,
  ).toBe(false);
});
