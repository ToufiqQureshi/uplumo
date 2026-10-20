import { describe, expect, it, vi } from "vitest";
import { ApiError } from "@/client/lib/uplumoApi";
import {
  checkIndexNowKeyFiles,
  indexNowSetupQueryOptions,
  submitIndexNowUrls,
} from "./indexNowApi";

function respondWith(body: unknown, status = 200) {
  return vi
    .spyOn(globalThis, "fetch")
    .mockResolvedValue(Response.json(body, { status }));
}

describe("indexnow api", () => {
  it("reads the project's setup from its route", async () => {
    const fetchMock = respondWith({
      included: true,
      domain: "example.com",
      key: "",
    });
    await expect(indexNowSetupQueryOptions("p/1").queryFn()).resolves.toEqual({
      included: true,
      domain: "example.com",
      key: "",
    });
    expect(fetchMock.mock.calls[0]?.[0]).toBe(
      "/api/v1/projects/p%2F1/indexnow/get",
    );
  });

  it("rejects a key-file status the screen cannot explain", async () => {
    respondWith({ files: [{ host: "example.com", status: "teapot" }] });
    await expect(checkIndexNowKeyFiles("p")).rejects.toBeInstanceOf(ApiError);
  });

  it("sends the URLs and keeps the per-host result", async () => {
    const result = {
      hosts: [
        {
          host: "example.com",
          urlCount: 1,
          status: "submitted",
          message: "Submitted 1 URL.",
        },
      ],
      skipped: ["https://other.com/"],
      skippedCount: 1,
    };
    const fetchMock = respondWith(result);
    await expect(
      submitIndexNowUrls("p", ["https://example.com/", "https://other.com/"]),
    ).resolves.toEqual(result);
    expect(fetchMock.mock.calls[0]?.[1]).toMatchObject({
      method: "POST",
      body: JSON.stringify({
        urls: ["https://example.com/", "https://other.com/"],
      }),
    });
  });

  it("turns the plan refusal into an ApiError the card can show", async () => {
    respondWith(
      {
        error: {
          code: "plan_required",
          message: "IndexNow is available on Basic, Pro and Agency.",
        },
      },
      402,
    );
    await expect(
      submitIndexNowUrls("p", ["https://example.com/"]),
    ).rejects.toMatchObject({ status: 402 });
  });
});
