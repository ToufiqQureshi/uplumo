// @vitest-environment happy-dom
import { act, createElement } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { brandLookupResultSchema } from "@/types/schemas/ai-search";
import { BrandLookupInsights, unbrandedLead } from "./BrandLookupInsights";

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children }: { children: unknown }) => children,
}));

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
let root: Root;
let host: HTMLDivElement;
beforeEach(() => {
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(() => {
  act(() => root.unmount());
  host.remove();
});

const query = (question: string, branded: boolean | undefined) => ({
  question,
  platform: "google",
  aiSearchVolume: 2900,
  firstSeenAt: null,
  lastSeenAt: null,
  citedSources: [],
  brandsMentioned: [],
  ...(branded === undefined ? {} : { branded }),
});

function result(extra: Record<string, unknown>) {
  return brandLookupResultSchema.parse({
    query: "dellaresorts.com",
    detectedTargetType: "domain",
    resolvedTarget: "dellaresorts.com",
    fetchedAt: "2026-10-22T00:00:00.000Z",
    hasData: true,
    totalMentions: 82,
    totalAiSearchVolume: 265490,
    perPlatform: [],
    shareOfVoice: null,
    topPages: [],
    topQueries: [],
    monthlyVolume: [],
    ...extra,
  });
}

function render(value: ReturnType<typeof result>) {
  act(() =>
    root.render(
      createElement(BrandLookupInsights, { result: value, projectId: "p1" }),
    ),
  );
  return host.textContent ?? "";
}

it("shows where AI recommends the brand, the sites it reads and the brands beside it", () => {
  const text = render(
    result({
      topQueries: [
        query("della resorts lonavala", true),
        query("best resort in lonavala", false),
      ],
      brandedQuestions: 39,
      unbrandedQuestions: 43,
      sourceSites: [
        { domain: "dellaresorts.com", mentions: 81, aiSearchVolume: null },
        { domain: "tripadvisor.in", mentions: 30, aiSearchVolume: null },
      ],
      brandsAlongside: [{ name: "The Machan", mentions: 1 }],
    }),
  );
  expect(text).toContain("best resort in lonavala");
  expect(text).not.toContain("della resorts lonavala");
  expect(text).toContain("43 of 82 questions");
  expect(text).toContain("tripadvisor.in");
  expect(text).not.toMatch(/dellaresorts\.com\d/); // the brand's own site is not a "site AI reads about you"
  expect(text).toContain("The Machan");
});

it("renders nothing new for a result cached before the split and site lists", () => {
  expect(render(result({ topQueries: [query("acme", undefined)] }))).toBe("");
});

it("words the lead for results without counts", () => {
  expect(unbrandedLead(result({}))).toBe("Questions that don't use your name.");
  expect(
    unbrandedLead(result({ brandedQuestions: 0, unbrandedQuestions: 1 })),
  ).toBe("1 of 1 question where AI named you don't use your name.");
});
