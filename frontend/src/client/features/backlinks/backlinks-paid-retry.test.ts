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

  it("does not automatically resubmit either paid POST after an ambiguous server failure", async () => {
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(
        Response.json(
          { error: { code: "internal", message: "Temporary failure" } },
          { status: 503 },
        ),
      );
    act(() =>
      root.render(
        createElement(
          QueryClientProvider,
          { client },
          createElement(PaidQueries),
        ),
      ),
    );
    await act(async () => {
      await vi.waitFor(() => {
        expect(
          client.getQueryState([
            "backlinksOverview",
            "p1",
            "domain",
            "example.com",
          ])?.status,
        ).toBe("error");
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
