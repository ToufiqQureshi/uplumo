import { beforeEach, expect, it, vi } from "vitest";
import { enquiryApi } from "./enquiryApi";

beforeEach(() =>
  vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValue(Response.json({ id: "baseline-1" })),
);

it("records a project-scoped path/note with request idempotency", async () => {
  await enquiryApi.saveChange("project/one", "/services", "Test mobile form");
  const call = vi.mocked(fetch).mock.calls[0];
  expect(call?.[0]).toBe(
    "/api/v1/projects/project%2Fone/analytics/enquiries/changes",
  );
  expect(call?.[1]).toMatchObject({
    method: "POST",
    body: JSON.stringify({ path: "/services", note: "Test mobile form" }),
  });
  const headers = new Headers(call?.[1]?.headers);
  expect(headers.get("Idempotency-Key")).toMatch(/^[\da-f-]{36}$/u);
});

it("rejects malformed evidence instead of treating it as zero enquiries", async () => {
  await expect(enquiryApi.report("project")).rejects.toMatchObject({
    code: "invalid_response",
  });
});
