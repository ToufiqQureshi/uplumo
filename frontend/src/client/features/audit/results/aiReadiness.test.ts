import { describe, expect, it } from "vitest";
import { aiReadiness } from "./aiReadiness";

const NEW = "2026-10-28T10:00:00Z";

describe("aiReadiness", () => {
  it("scores a site with every check passing at 100", () => {
    const result = aiReadiness(["missing-title", "thin-content"], NEW);
    expect(result?.score).toBe(100);
    expect(result?.fixes).toEqual([]);
    expect(result?.groups.map((g) => g.verdict)).toEqual([
      "Pass",
      "Pass",
      "Pass",
    ]);
  });

  it("subtracts each failed check and lists the biggest wins first", () => {
    const result = aiReadiness(
      ["missing-llms-txt", "missing-business-schema", "no-question-answers"],
      NEW,
    );
    expect(result?.score).toBe(50);
    expect(result?.fixes.map((fix) => fix.issue)).toEqual([
      "missing-business-schema",
      "no-question-answers",
      "missing-llms-txt",
    ]);
    expect(result?.groups.map((g) => g.verdict)).toEqual([
      "Partly",
      "Partly",
      "Weak",
    ]);
  });

  it("refuses to score audits that ran before the checks existed", () => {
    expect(aiReadiness([], "2026-10-20T10:00:00Z")).toBeNull();
  });
});
