import { describe, expect, it } from "vitest";
import type { TrackingAnswer } from "@/client/features/ai-search/trackingApi";
import { DASHBOARD_CARDS } from "./dashboardLayout";
import {
  aiNextStep,
  citedInstead,
  latestChecks,
  shareChange,
} from "./aiVisibilityInsight";

const run = (id: string, status: string, answered = 3) => ({
  id,
  status,
  answered,
  mentioned: 1,
  cited: 0,
});

function answer(
  status: string,
  citations: { domain: string | null; matchedBrand: boolean }[],
): TrackingAnswer {
  return {
    id: "a",
    prompt: "q",
    topic: "t",
    engine: "chat_gpt",
    status,
    answer: "x",
    errorCode: "",
    collectedAt: null,
    brands: [],
    recommended: [],
    citations: citations.map((c) => ({
      url: `https://${c.domain ?? "x.test"}/p`,
      title: null,
      ...c,
    })),
  };
}

describe("AI visibility insight", () => {
  it("counts a partial check, which still holds real answers", () => {
    const [latest, previous] = latestChecks([
      run("running", "running"),
      run("partial", "partial"),
      run("empty", "completed", 0),
      run("old", "completed"),
    ]);
    expect(latest?.id).toBe("partial");
    expect(previous?.id).toBe("old");
  });

  it("lists the domains AI cited instead of you, once per answer", () => {
    const rows = citedInstead([
      answer("completed", [
        { domain: "www.rival.test", matchedBrand: false },
        { domain: "rival.test", matchedBrand: false },
        { domain: "you.test", matchedBrand: true },
      ]),
      answer("completed", [
        { domain: "rival.test", matchedBrand: false },
        { domain: "blog.test", matchedBrand: false },
      ]),
      answer("failed", [{ domain: "ignored.test", matchedBrand: false }]),
    ]);
    expect(rows).toEqual([
      { domain: "rival.test", answersCiting: 2 },
      { domain: "blog.test", answersCiting: 1 },
    ]);
  });

  it("points each next step at the page that fixes it", () => {
    expect(aiNextStep(0, 0).to).toBe("content-editor");
    expect(aiNextStep(0.5, 0).to).toBe("content-editor");
    expect(aiNextStep(0.5, 0.5).to).toBe("prompt-tracking");
  });

  it("shows the change against the previous check", () => {
    expect(shareChange(0.4, 0.2)).toBe("↑ from 20%");
    expect(shareChange(0.1, 0.3)).toBe("↓ from 30%");
    expect(shareChange(0.4, 0.4)).toBe("");
    expect(shareChange(0.4, undefined)).toBe("");
  });

  it("puts AI visibility first on a new dashboard", () => {
    expect(DASHBOARD_CARDS[0].id).toBe("ai_visibility");
  });
});
