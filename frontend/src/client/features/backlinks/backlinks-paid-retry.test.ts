// @vitest-environment happy-dom
import {
  focusManager,
  onlineManager,
  QueryClient,
  QueryClientProvider,
} from "@tanstack/react-query";
import { act, createElement } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { shouldRetryApiError } from "@/client/lib/uplumoApi";
import { useBacklinksFilters } from "./useBacklinksFilters";
import { useBacklinksPageData } from "./useBacklinksPageData";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

function PaidQueries() {
  const filters = useBacklinksFilters();
  useBacklinksPageData({
    projectId: "p1",
    searchState: {
      target: "example.com",
      scope: "domain",
      tab: "backlinks",
      page: 1,
      pageSize: 50,
    },
    filters,
  });
  return null;
}

const failure = () =>
  Response.json(
    { error: { code: "internal", message: "Temporary failure" } },
    { status: 503 },
  );
const overview = (backlinks: number) =>
  Response.json({
    target: "example.com",
    displayTarget: "example.com",
    scope: "domain",
    summary: {
      rank: null,
      backlinks,
      referringPages: null,
      referringDomains: null,
      brokenBacklinks: null,
      brokenPages: null,
      backlinksSpamScore: null,
      targetSpamScore: null,
      newBacklinks: null,
      lostBacklinks: null,
      newReferringDomains: null,
      lostReferringDomains: null,
    },
    trends: [],
    newLostTrends: [],
    fetchedAt: "2026-10-21T00:00:00.000Z",
  });
describe("backlinks paid query failures", () => {
  let host: HTMLDivElement;
  let root: Root;
  let client: QueryClient;
  beforeEach(() => {
    localStorage.clear();
    host = document.createElement("div");
    document.body.append(host);
    root = createRoot(host);
    client = new QueryClient({
      defaultOptions: {
        queries: { retry: shouldRetryApiError, retryDelay: 1 },
      },
    });
  });
  afterEach(() => {
    act(() => root.unmount());
    client.clear();
    host.remove();
    vi.restoreAllMocks();
    focusManager.setFocused(undefined);
    onlineManager.setOnline(true);
  });

  const render = () =>
    act(() =>
      root.render(
        createElement(
          QueryClientProvider,
          { client },
          createElement(PaidQueries),
        ),
      ),
    );
  const overviewStatus = () =>
    client.getQueryState(["backlinksOverview", "p1", "domain", "example.com"])
      ?.status;

  it("buys no table when the overview found no links", async () => {
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockImplementation(() => Promise.resolve(overview(0)));
    render();
    await act(async () => {
      await vi.waitFor(() => expect(overviewStatus()).toBe("success"));
      await new Promise((resolve) => setTimeout(resolve, 20));
    });
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("buys no table after a failed overview", async () => {
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockImplementation(() => Promise.resolve(failure()));
    render();
    await act(async () => {
      await vi.waitFor(() => expect(overviewStatus()).toBe("error"));
      await new Promise((resolve) => setTimeout(resolve, 20));
    });
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("does not automatically resubmit either paid POST after an ambiguous server failure", async () => {
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockImplementation((input) =>
        Promise.resolve(
          String(input instanceof Request ? input.url : input).endsWith(
            "/overview",
          )
            ? overview(5)
            : failure(),
        ),
      );
    render();
    await act(async () => {
      await vi.waitFor(() => {
        expect(overviewStatus()).toBe("success");
        expect(fetchMock).toHaveBeenCalledTimes(2);
      });
    });
    // Returning to the tab or reconnecting must not buy the lookup again.
    await act(async () => {
      focusManager.setFocused(false);
      focusManager.setFocused(true);
      onlineManager.setOnline(false);
      onlineManager.setOnline(true);
      await new Promise((resolve) => setTimeout(resolve, 20));
    });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
