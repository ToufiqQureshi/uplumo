// @vitest-environment happy-dom
import { beforeEach, expect, it, vi } from "vitest";
import {
  clearAnalystDraft,
  isAnalystSession,
  saveAnalystDraft,
  setAnalystSession,
  takeAnalystDraft,
} from "./analystDraft";
import {
  readSearchContext,
  saveSearchContext,
} from "@/client/features/content-editor/searchContext";
beforeEach(() => {
  sessionStorage.clear();
  vi.useRealTimers();
});
it("keeps an analyst draft readable across StrictMode mounts, scoped and expiring", () => {
  vi.useFakeTimers();
  saveAnalystDraft("a", "Analyze this page");
  expect(takeAnalystDraft("a")).toBe("Analyze this page");
  expect(takeAnalystDraft("a")).toBe("Analyze this page");
  expect(takeAnalystDraft("b")).toBe("");
  setAnalystSession("a", "s", true);
  expect(isAnalystSession("a", "s")).toBe(true);
  expect(isAnalystSession("b", "s")).toBe(false);
  setAnalystSession("a", "s", false);
  expect(isAnalystSession("a", "s")).toBe(false);
  vi.advanceTimersByTime(31 * 60 * 1000);
  expect(takeAnalystDraft("a")).toBe("");
  clearAnalystDraft("a");
  vi.useRealTimers();
});
it("rejects unsafe or expired editor evidence", () => {
  expect(() =>
    saveSearchContext("a", {
      page: "javascript:alert(1)",
      queries: [],
      reasons: [],
      startDate: "2026-09-01",
      endDate: "2026-09-28",
    }),
  ).toThrow();
  sessionStorage.setItem("growth-editor:a", '{"page":"https://example.test"}');
  expect(readSearchContext("a")).toBeNull();
});
