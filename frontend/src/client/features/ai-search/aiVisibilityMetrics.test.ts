import { describe, expect, it } from "vitest";
import {
  citedSources,
  engineBreakdown,
  promptMatrix,
  shareOfVoice,
  visibilityVerdict,
} from "./aiVisibilityMetrics";
import type { TrackingAnswer } from "./trackingApi";

type Brands = [own: boolean, rivalA?: boolean, rivalB?: boolean];

function answer(
  prompt: string,
  engine: string,
  [own, rivalA = false, rivalB = false]: Brands,
  extra: Partial<TrackingAnswer> = {},
): TrackingAnswer {
  return {
    id: `${prompt}-${engine}`,
    prompt,
    topic: "General",
    engine,
    status: "completed",
    answer: "Some answer text.",
    errorCode: "",
    collectedAt: "2026-11-09T10:00:00Z",
    citations: [],
    brands: [
      { name: "Ownco", domain: "ownco.example", mentioned: own, cited: false },
      {
        name: "Rival A",
        domain: "rival-a.example",
        mentioned: rivalA,
        cited: false,
      },
      {
        name: "Rival B",
        domain: "rival-b.example",
        mentioned: rivalB,
        cited: false,
      },
    ],
    recommended: [],
    sentiment: "",
    sentimentQuote: "",
    sentimentReason: "",
    ...extra,
  };
}

describe("shareOfVoice", () => {
  it("is empty without measured answers", () => {
    expect(shareOfVoice([])).toEqual([]);
    expect(
      shareOfVoice([answer("q", "chat_gpt", [true], { status: "failed" })]),
    ).toEqual([]);
  });

  it("counts only completed answers with text", () => {
    const rows = shareOfVoice([
      answer("q1", "chat_gpt", [true, true]),
      answer("q2", "chat_gpt", [false, true]),
      answer("q3", "chat_gpt", [true], { answer: "   " }),
      answer("q4", "gemini", [true], { status: "pending" }),
    ]);
    expect(rows.map((r) => [r.name, r.mentions, r.share, r.own])).toEqual([
      ["Rival A", 2, 67, false],
      ["Ownco", 1, 33, true],
      ["Rival B", 0, 0, false],
    ]);
  });

  it("puts the customer first on a tie", () => {
    const rows = shareOfVoice([answer("q", "chat_gpt", [true, true])]);
    expect(rows[0]?.own).toBe(true);
  });
});

describe("engineBreakdown", () => {
  it("rates each engine on its own answers", () => {
    const rows = engineBreakdown([
      answer("q1", "chat_gpt", [true]),
      answer("q2", "chat_gpt", [false]),
      answer("q1", "gemini", [false]),
      answer("q2", "gemini", [false], { status: "failed" }),
    ]);
    expect(rows).toEqual([
      {
        engine: "chat_gpt",
        label: "ChatGPT",
        answered: 2,
        mentioned: 1,
        cited: 0,
      },
      {
        engine: "gemini",
        label: "Gemini",
        answered: 1,
        mentioned: 0,
        cited: 0,
      },
    ]);
  });
});

describe("promptMatrix", () => {
  it("marks each cell and finds questions where AI leaves the customer out", () => {
    const rows = promptMatrix([
      answer("best crm", "chat_gpt", [false, true], {
        recommended: [
          { name: "Rival A", position: 1 },
          { name: "Untracked Ltd", position: 2 },
        ],
      }),
      answer("best crm", "gemini", [false], { status: "failed" }),
      answer("cheap crm", "chat_gpt", [true]),
      answer("cheap crm", "gemini", [false, false, true]),
    ]);
    const best = rows.find((r) => r.prompt === "best crm");
    const cheap = rows.find((r) => r.prompt === "cheap crm");
    expect(best?.cells.chat_gpt?.state).toBe("absent");
    expect(best?.cells.gemini?.state).toBe("failed");
    expect(best?.gap).toBe(true);
    // A tracked rival is counted once, not again from the recommended list.
    expect(best?.namedInstead).toEqual(["Rival A", "Untracked Ltd"]);
    expect(cheap?.gap).toBe(false);
    expect(cheap?.namedInstead).toEqual(["Rival B"]);
  });

  it("does not call a question a gap when nothing was measured", () => {
    const [row] = promptMatrix([
      answer("q", "chat_gpt", [false], { status: "failed" }),
    ]);
    expect(row?.gap).toBe(false);
  });

  it("keeps unicode questions as written", () => {
    const [row] = promptMatrix([
      answer("मुंबई में सबसे अच्छा CRM?", "gemini", [true]),
    ]);
    expect(row?.prompt).toBe("मुंबई में सबसे अच्छा CRM?");
  });
});

describe("visibilityVerdict", () => {
  it("is null without measured answers", () => {
    expect(visibilityVerdict([])).toBeNull();
  });

  it("names the leading rival and the weakest engine", () => {
    const v = visibilityVerdict([
      answer("q1", "chat_gpt", [false, true]),
      answer("q2", "chat_gpt", [false, true]),
      answer("q3", "chat_gpt", [false, true]),
      answer("q1", "gemini", [true, true]),
      answer("q2", "gemini", [true]),
    ]);
    expect(v?.headline).toBe("AI names Rival A 2.0× as often as Ownco.");
    expect(v?.weakest?.engine).toBe("chat_gpt");
    expect(v?.gaps).toBe(1);
  });

  it("says so when no rival is named and the customer is", () => {
    const v = visibilityVerdict([answer("q", "chat_gpt", [true])]);
    expect(v?.headline).toBe("Ownco is named in 1 of 1 AI answers.");
    expect(v?.weakest).toBeNull();
  });

  it("does not divide by zero when the customer is never named", () => {
    const v = visibilityVerdict([answer("q", "chat_gpt", [false, true])]);
    expect(v?.headline).toBe("AI names Rival A in 1 answer and Ownco in none.");
  });

  it("credits the customer when it leads", () => {
    const v = visibilityVerdict([
      answer("q1", "chat_gpt", [true, true]),
      answer("q2", "chat_gpt", [true]),
    ]);
    expect(v?.headline).toBe(
      "Ownco is named more often than any tracked competitor: 2 of 2 answers.",
    );
  });
});

const cite = (url: string, domain: string | null = null) => ({
  url,
  domain,
  title: null,
  matchedBrand: false,
});

describe("citedSources", () => {
  it("counts each domain once per answer and flags the customer's site", () => {
    const rows = citedSources(
      [
        answer("q1", "chat_gpt", [false], {
          citations: [
            cite("https://www.review.example/a"),
            cite("https://review.example/b"),
            cite("https://blog.ownco.example/post"),
          ],
        }),
        answer("q2", "gemini", [false], {
          citations: [
            cite("https://x.example/1", "review.example"),
            cite("not a url"),
          ],
        }),
      ],
      "www.ownco.example",
    );
    expect(rows).toEqual([
      { domain: "review.example", answers: 2, own: false },
      { domain: "blog.ownco.example", answers: 1, own: true },
    ]);
  });
});
