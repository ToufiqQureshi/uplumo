import { describe, expect, it, vi, beforeEach } from "vitest";
import {
  summarizeTracking,
  trackingApi,
  type TrackingAnswer,
} from "./trackingApi";
import { apiRequest } from "@/client/lib/uplumoApi";
vi.mock("@/client/lib/uplumoApi", () => ({ apiRequest: vi.fn() }));
beforeEach(() => {
  vi.mocked(apiRequest).mockResolvedValue([]);
});
function answer(over: Partial<TrackingAnswer> = {}): TrackingAnswer {
  return {
    id: "a",
    prompt: "Question?",
    topic: "General",
    engine: "gemini",
    status: "completed",
    answer: "Acme",
    errorCode: "",
    collectedAt: null,
    citations: [],
    recommended: [],
    brands: [
      { name: "Acme", domain: "acme.com", mentioned: true, cited: false },
    ],
    ...over,
  };
}
describe("saved AI tracking", () => {
  it("does not turn failures or empty AI answers into measured absences", () => {
    const summary = summarizeTracking([
      answer(),
      answer({ status: "failed", brands: [] }),
      answer({ answer: "" }),
    ]);
    expect(summary).toEqual([
      {
        name: "Acme",
        domain: "acme.com",
        answers: 1,
        mentions: 1,
        citations: 0,
      },
    ]);
  });
  it("keeps mentions, citations and competitor appearances distinct", () => {
    expect(
      summarizeTracking([
        answer({
          brands: [
            { name: "Acme", domain: "acme.com", mentioned: false, cited: true },
            {
              name: "Other",
              domain: "other.com",
              mentioned: true,
              cited: false,
            },
          ],
        }),
      ]),
    ).toEqual([
      {
        name: "Acme",
        domain: "acme.com",
        answers: 1,
        mentions: 0,
        citations: 1,
      },
      {
        name: "Other",
        domain: "other.com",
        answers: 1,
        mentions: 1,
        citations: 0,
      },
    ]);
  });
  it("uses project-scoped GETs for saved evidence and an explicit ceiling for collection", async () => {
    await trackingApi.answers("p/1", "r/2");
    expect(apiRequest).toHaveBeenLastCalledWith(
      "/api/v1/projects/p%2F1/ai-search/tracking/runs/r%2F2",
      expect.anything(),
    );
    await trackingApi.run("p/1", 22);
    expect(apiRequest).toHaveBeenLastCalledWith(
      "/api/v1/projects/p%2F1/ai-search/tracking/run",
      expect.anything(),
      "POST",
      { maxCredits: 22 },
    );
  });
});
