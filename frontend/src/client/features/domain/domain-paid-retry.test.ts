// @vitest-environment happy-dom
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, createElement } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { shouldRetryApiError } from "@/client/lib/uplumoApi";
import { useDomainOverviewQuery } from "./hooks/useDomainOverviewQuery";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

function DomainLookup() {
  useDomainOverviewQuery({
    projectId: "p1",
    domain: "example.com",
    scope: "domain",
    locationCode: 2840,
  });
  return null;
}

let host: HTMLDivElement;
let root: Root;
let client: QueryClient;
beforeEach(() => {
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
  client = new QueryClient({
    defaultOptions: { queries: { retry: shouldRetryApiError, retryDelay: 1 } },
  });
});
afterEach(() => {
  act(() => root.unmount());
  client.clear();
  host.remove();
  vi.restoreAllMocks();
});

it("does not automatically repeat a paid domain lookup on 503", async () => {
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
        createElement(DomainLookup),
      ),
    ),
  );
  await act(async () => {
    await vi.waitFor(() => {
      expect(
        client.getQueryState([
          "domain-overview",
          "p1",
          "example.com",
          "domain",
          2840,
        ])?.status,
      ).toBe("error");
      expect(fetchMock).toHaveBeenCalledTimes(1);
    });
  });
});
