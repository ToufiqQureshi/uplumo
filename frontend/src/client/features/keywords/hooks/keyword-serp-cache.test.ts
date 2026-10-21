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
import { getSerpAnalysis } from "../keywordsApi";
import { useKeywordSerpAnalysis } from "./useKeywordSerpAnalysis";

vi.mock("../keywordsApi", () => ({ getSerpAnalysis: vi.fn() }));
vi.mock("@/client/lib/error-messages", () => ({
  getStandardErrorMessage: (error: Error) => error.message,
}));
Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

let root: Root;
let host: HTMLDivElement;
let client: QueryClient;
let state: ReturnType<typeof useKeywordSerpAnalysis>;
function Lookup() {
  state = useKeywordSerpAnalysis("p1", 2356, undefined);
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
  client = new QueryClient({
    defaultOptions: { queries: { retry: 2, retryDelay: 1 } },
  });
  vi.mocked(getSerpAnalysis).mockResolvedValue({
    requestedKeyword: "hotel",
    depth: 20,
    items: [],
  });
});
afterEach(() => {
  act(() => root.unmount());
  client.clear();
  host.remove();
  vi.clearAllMocks();
  focusManager.setFocused(undefined);
  onlineManager.setOnline(true);
});

it("reuses a successful SERP after another keyword and a component remount", async () => {
  mount();
  act(() => state.setSerpKeyword("hotel"));
  await act(async () => {
    await vi.waitFor(() => expect(getSerpAnalysis).toHaveBeenCalledTimes(1));
  });
  act(() => state.setSerpKeyword(null));
  act(() => state.setSerpKeyword("hotel"));
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 20));
  });
  expect(getSerpAnalysis).toHaveBeenCalledTimes(1);
  mount(false);
  mount();
  act(() => state.setSerpKeyword("hotel"));
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 20));
  });
  expect(getSerpAnalysis).toHaveBeenCalledTimes(1);
});

it("never automatically retries failed paid calls on errors, focus or reconnect", async () => {
  vi.mocked(getSerpAnalysis).mockRejectedValue(new Error("provider failed"));
  mount();
  act(() => state.setSerpKeyword("hotel"));
  await act(async () => {
    await vi.waitFor(() => expect(state.serpError).toBe("provider failed"));
  });
  await act(async () => {
    focusManager.setFocused(false);
    focusManager.setFocused(true);
    onlineManager.setOnline(false);
    onlineManager.setOnline(true);
    await new Promise((resolve) => setTimeout(resolve, 20));
  });
  expect(getSerpAnalysis).toHaveBeenCalledTimes(1);
});
