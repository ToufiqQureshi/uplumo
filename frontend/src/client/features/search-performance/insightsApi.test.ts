import { afterEach, describe, expect, it, vi } from "vitest";
import { QueryClient } from "@tanstack/react-query";
import { ApiError } from "@/client/lib/uplumoApi";
import { insightsQueryOptions, keepForProject } from "./insightsApi";

afterEach(() => vi.restoreAllMocks());

const fetchQuery = (options: ReturnType<typeof insightsQueryOptions>) =>
  new QueryClient({ defaultOptions: { queries: { retry: false } } }).fetchQuery(
    options,
  );

const connected = {
  connected: true,
  range: {
    startDate: "2026-04-27",
    endDate: "2026-05-25",
    prevStartDate: "2026-03-29",
    prevEndDate: "2026-04-26",
  },
  coverage: {
    queryPageRows: 3,
    truncated: false,
    decayAvailable: true,
    comparisonAvailable: true,
    brandTermsUsed: true,
  },
  segments: [
    {
      kind: "all",
      queries: 2,
      clicks: 8,
      impressions: 180,
      ctr: 0.04,
      position: 5,
    },
  ],
  actions: [
    {
      kind: "ctr_gap",
      subject: "red shoes",
      page: "https://example.test/a",
      impactClicks: 16,
      basis: "estimated",
    },
  ],
  losingPages: [],
  queryLosses: [],
  queryGains: [],
  decaying: [],
  ctrGaps: [],
  cannibalization: [],
  totals: { decaying: 0, ctrGaps: 0, cannibalization: 0 },
};

describe("search insights query", () => {
  it("sends the filters and brand names to the project's insights route", async () => {
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValue(Response.json(connected));
    const filters = {
      dateRange: "last_28_days" as const,
      queryFilter: {
        operator: "includingRegex" as const,
        expression: "^(how|what)\\b",
      },
    };
    const options = insightsQueryOptions("p/1", filters, ["Acme"]);
    await expect(fetchQuery(options)).resolves.toMatchObject({
      connected: true,
    });
    expect(fetchMock.mock.calls[0]?.[0]).toBe(
      "/api/v1/projects/p%2F1/gsc/search-performance/insights",
    );
    const body = fetchMock.mock.calls[0]?.[1]?.body;
    expect(JSON.parse(typeof body === "string" ? body : "")).toEqual({
      ...filters,
      brandTerms: ["Acme"],
    });
  });

  it("keys the cache by project, filters and brand names", () => {
    const filters = { dateRange: "last_28_days" as const };
    expect(insightsQueryOptions("p", filters, ["a"]).queryKey).not.toEqual(
      insightsQueryOptions("p", filters, ["b"]).queryKey,
    );
    expect(insightsQueryOptions("p", filters, []).queryKey).not.toEqual(
      insightsQueryOptions("q", filters, []).queryKey,
    );
  });

  it("is free to reuse for five minutes, so a tab switch never reads Google again", () => {
    expect(insightsQueryOptions("p", {}, []).staleTime).toBe(300_000);
  });

  it("accepts a project with no Search Console connection", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      Response.json({ connected: false }),
    );
    await expect(
      fetchQuery(insightsQueryOptions("p", {}, [])),
    ).resolves.toEqual({ connected: false });
  });

  it("rejects an answer the screen cannot trust", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(
      Response.json({
        ...connected,
        actions: [{ ...connected.actions[0], basis: "guaranteed" }],
      }),
    );
    await expect(
      fetchQuery(insightsQueryOptions("p", {}, [])),
    ).rejects.toBeInstanceOf(ApiError);
  });

  it("keeps old numbers while new ones load, but never from another project", () => {
    const keep = keepForProject("project-a");
    const previous = { connected: false } as const;
    expect(
      keep(previous, { queryKey: ["searchInsights", "project-a", {}, []] }),
    ).toBe(previous);
    expect(
      keep(previous, { queryKey: ["searchInsights", "project-b", {}, []] }),
    ).toBeUndefined();
    expect(keep(undefined, undefined)).toBeUndefined();
    expect(keep(previous, undefined)).toBeUndefined();
  });
});
