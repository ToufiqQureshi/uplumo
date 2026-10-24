// @vitest-environment happy-dom
import { act, createElement, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { SearchActionPlan } from "./SearchActionPlan";
import { readSearchContext } from "@/client/features/content-editor/searchContext";
import * as searchContext from "@/client/features/content-editor/searchContext";
import type { RefreshQuery } from "./contentRefreshQueue";

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
const { navigate } = vi.hoisted(() => ({ navigate: vi.fn() }));
vi.mock("@tanstack/react-router", () => ({
  useNavigate: () => navigate,
  Link: ({ children }: { children: ReactNode }) =>
    createElement("span", null, children),
}));
let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;
const range = { startDate: "2026-09-22", endDate: "2026-10-20" };
const row: RefreshQuery = {
  page: "https://example.test/guide",
  query: "things to do",
  impressions: 739,
  clicks: 2,
  position: 11,
};
async function mount(rows: RefreshQuery[]) {
  container = document.createElement("div");
  document.body.append(container);
  root = createRoot(container);
  await act(async () => {
    root.render(
      createElement(SearchActionPlan, { projectId: "project-a", rows, range }),
    );
  });
}
beforeEach(() => {
  vi.clearAllMocks();
  sessionStorage.clear();
});
afterEach(async () => {
  if (root) await act(async () => root.unmount());
  container?.remove();
});
describe("Search action plan", () => {
  it("shows bounded GSC review tasks without GA4, and preserves the existing page on handoff", async () => {
    await mount(
      Array.from({ length: 5 }, (_, i) => ({
        ...row,
        page: `${row.page}-${i}`,
      })),
    );
    expect(container.querySelectorAll("article")).toHaveLength(3);
    expect(container.textContent).toContain("739");
    expect(container.textContent).toContain("2026-09-22");
    expect(container.textContent).toContain("not revenue");
    expect(navigate).not.toHaveBeenCalled();
    const button = container.querySelector("button");
    if (!button) throw new Error("Missing page review action");
    await act(async () => button.click());
    expect(readSearchContext("project-a")).toMatchObject({
      page: "https://example.test/guide-0",
      queries: ["things to do"],
      ...range,
    });
    expect(readSearchContext("project-b")).toBeNull();
    expect(navigate).toHaveBeenCalledWith({
      to: "/p/$projectId/content-editor",
      params: { projectId: "project-a" },
      search: { keyword: "things to do", growth: "1" },
    });
  });
  it("does not invent tasks from low data or expose unsafe page links", async () => {
    await mount([
      { ...row, impressions: 10 },
      { ...row, page: "javascript:alert(1)" },
    ]);
    expect(container.querySelectorAll("article")).toHaveLength(0);
    expect(container.textContent).toContain("Not enough evidence");
    expect(container.querySelector('a[href^="javascript:"]')).toBeNull();
    expect(container.querySelector("button")).toBeNull();
  });
  it("does not navigate without preserving the source evidence when tab storage fails", async () => {
    await mount([row]);
    vi.spyOn(searchContext, "saveSearchContext").mockImplementationOnce(() => {
      throw new Error("blocked");
    });
    const button = container.querySelector("button");
    if (!button) throw new Error("Missing page review action");
    await act(async () => button.click());
    expect(navigate).not.toHaveBeenCalled();
  });
});
