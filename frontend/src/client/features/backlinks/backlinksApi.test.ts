import { afterEach, describe, expect, it, vi } from "vitest";
import { getBacklinksRows } from "./backlinksApi";

afterEach(() => vi.restoreAllMocks());

describe("getBacklinksRows", () => {
  it("uses the backend backlinks route", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      Response.json({
        rows: [],
        totalCount: 0,
        hasMore: false,
        page: 1,
        pageSize: 50,
        fetchedAt: "2026-10-16T08:00:00Z",
      }),
    );

    await getBacklinksRows({
      projectId: "p/1",
      target: "example.com",
      scope: "subdomains",
      page: 1,
      pageSize: 50,
      sortOrder: "desc",
      sortField: "firstSeen",
      filters: {},
      mode: "one_per_domain",
    });

    expect(fetchMock.mock.calls[0]?.[0]).toBe(
      "/api/v1/projects/p%2F1/backlinks/backlinks",
    );
  });

  it("sends cached-only as a header, never in the request body", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      Response.json({
        rows: [],
        totalCount: 0,
        hasMore: false,
        page: 1,
        pageSize: 50,
        fetchedAt: "2026-10-16T08:00:00Z",
      }),
    );

    await getBacklinksRows({
      projectId: "p",
      target: "example.com",
      page: 1,
      pageSize: 50,
      sortOrder: "desc",
      sortField: "firstSeen",
      filters: {},
      mode: "one_per_domain",
      cachedOnly: true,
    });

    const init = fetchMock.mock.calls[0]?.[1];
    expect(new Headers(init?.headers).get("X-Cached-Only")).toBe("1");
    expect(typeof init?.body).toBe("string");
    expect(
      typeof init?.body === "string" && init.body.includes("cachedOnly"),
    ).toBe(false);
  });
});
