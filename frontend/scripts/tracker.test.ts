// @vitest-environment happy-dom
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// The tracker is served by the Go server (backend/internal/httpapi), but it
// runs in the visitor's browser, so it is tested here with a browser DOM.
const source = readFileSync(
  join(__dirname, "../../backend/internal/httpapi/tracker.js"),
  "utf8",
);
const siteKey = "site-key";
const sessionKey = `uplumo-session:${siteKey}`;

let visibility: DocumentVisibilityState;
let clock: number;
let beacons: Record<string, unknown>[];

function runTracker(options: {
  opener: boolean;
  historyLength: number;
  navigation: string;
}) {
  const script = document.createElement("script");
  script.setAttribute("data-site", siteKey);
  script.src = "https://uplumo.test/t.js";
  Object.defineProperty(document, "currentScript", {
    configurable: true,
    value: script,
  });
  Object.defineProperty(window, "opener", {
    configurable: true,
    value: options.opener ? {} : null,
  });
  Object.defineProperty(history, "length", {
    configurable: true,
    value: options.historyLength,
  });
  vi.spyOn(performance, "getEntriesByType").mockReturnValue([
    { type: options.navigation } as unknown as PerformanceEntry,
  ]);
  const before = beacons.length;
  new Function(source)();
  // The first beacon of this run is its pageview.
  return beacons[before];
}

beforeEach(() => {
  visibility = "visible";
  clock = 1_000;
  beacons = [];
  sessionStorage.clear();
  Object.defineProperty(document, "visibilityState", {
    configurable: true,
    get: () => visibility,
  });
  vi.spyOn(performance, "now").mockImplementation(() => clock);
  Object.defineProperty(navigator, "sendBeacon", {
    configurable: true,
    value: (_url: string, body: string) => {
      beacons.push(JSON.parse(body) as Record<string, unknown>);
      return true;
    },
  });
});

afterEach(() => {
  vi.restoreAllMocks();
});

describe("tracker", () => {
  it("counts the visible time before the page is hidden", () => {
    const pageview = runTracker({
      opener: false,
      historyLength: 1,
      navigation: "navigate",
    });
    clock += 12_000;
    visibility = "hidden";
    document.dispatchEvent(new Event("visibilitychange"));
    const engagement = beacons.find(
      (b) => b.p === pageview?.p && typeof b.d === "number",
    );
    expect(engagement?.d).toBe(12_000);
  });

  it("does not count time while the page was hidden", () => {
    const pageview = runTracker({
      opener: false,
      historyLength: 1,
      navigation: "navigate",
    });
    clock += 5_000;
    visibility = "hidden";
    document.dispatchEvent(new Event("visibilitychange"));
    clock += 60_000;
    window.dispatchEvent(new Event("pagehide"));
    const reports = beacons.filter(
      (b) => b.p === pageview?.p && typeof b.d === "number",
    );
    expect(reports.map((b) => b.d)).toEqual([5_000, 5_000]);
  });

  it("keeps the session across reloads of a tab that has an opener", () => {
    const stored = {
      id: "a".repeat(32),
      started: Date.now(),
      last: Date.now(),
    };
    sessionStorage.setItem(sessionKey, JSON.stringify(stored));
    const reload = runTracker({
      opener: true,
      historyLength: 1,
      navigation: "reload",
    });
    expect(reload?.s).toBe(stored.id);
    const nextPage = runTracker({
      opener: true,
      historyLength: 2,
      navigation: "navigate",
    });
    expect(nextPage?.s).toBe(stored.id);
  });

  it("starts a new session on the first page of a tab opened by another page", () => {
    const copied = {
      id: "b".repeat(32),
      started: Date.now(),
      last: Date.now(),
    };
    sessionStorage.setItem(sessionKey, JSON.stringify(copied));
    const first = runTracker({
      opener: true,
      historyLength: 1,
      navigation: "navigate",
    });
    expect(first?.s).toMatch(/^[0-9a-f]{32}$/);
    expect(first?.s).not.toBe(copied.id);
  });
});
