import { afterEach, expect, it, vi } from "vitest";
import { getDomainKeywordSuggestions } from "./domainApi";

afterEach(() => vi.restoreAllMocks());

it("marks only an explicit keyword suggestions lookup for the Go contract", async () => {
  const request = vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValue(Response.json([]));
  await getDomainKeywordSuggestions({
    data: { projectId: "p/1", domain: "example.com", locationCode: 2840 },
  });
  expect(request.mock.calls[0]).toMatchObject([
    "/api/v1/projects/p%2F1/domain/keyword-suggestions",
    {
      method: "POST",
      body: JSON.stringify({
        domain: "example.com",
        locationCode: 2840,
        userInitiated: true,
      }),
    },
  ]);
});
