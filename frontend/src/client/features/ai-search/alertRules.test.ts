import { describe, expect, it } from "vitest";
import { visibilityAlerts } from "./alertRules";
import type { TrackingAnswer } from "./trackingApi";

function answer(
  prompt: string,
  engine: string,
  own: boolean,
  extra: Partial<TrackingAnswer> = {},
): TrackingAnswer {
  return {
    id: `${prompt}-${engine}`,
    prompt,
    topic: "General",
    engine,
    status: "completed",
    answer: "Some answer.",
    errorCode: "",
    collectedAt: null,
    citations: [],
    brands: [
      { name: "Ownco", domain: "ownco.example", mentioned: own, cited: false },
      {
        name: "Rival",
        domain: "rival.example",
        mentioned: false,
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

const cite = (url: string) => ({
  url,
  domain: null,
  title: null,
  matchedBrand: false,
});

describe("visibilityAlerts", () => {
  it("says nothing without a previous check to compare", () => {
    expect(visibilityAlerts([], [answer("q", "chat_gpt", false)])).toEqual([]);
    expect(
      visibilityAlerts(
        [answer("q", "chat_gpt", true, { status: "failed" })],
        [answer("q", "chat_gpt", false)],
      ),
    ).toEqual([]);
  });

  it("flags a lost mention with who replaced the customer, first", () => {
    const alerts = visibilityAlerts(
      [
        answer("best crm", "chat_gpt", true),
        answer("cheap crm", "gemini", false),
      ],
      [
        answer("best crm", "chat_gpt", false, {
          recommended: [{ name: "Rival", position: 1 }],
        }),
        answer("cheap crm", "gemini", true),
      ],
    );
    expect(alerts.map((a) => [a.kind, a.title])).toEqual([
      ["lost", "ChatGPT stopped naming you for “best crm”"],
      ["won", "Gemini now names you for “cheap crm”"],
    ]);
    expect(alerts[0]?.detail).toBe("It now recommends Rival.");
  });

  it("never reads a failed or empty answer as a lost mention", () => {
    expect(
      visibilityAlerts(
        [answer("q", "chat_gpt", true)],
        [
          answer("q", "chat_gpt", false, { status: "failed" }),
          answer("q", "gemini", false, { answer: " " }),
        ],
      ),
    ).toEqual([]);
  });

  it("reports a new recommended business once", () => {
    const alerts = visibilityAlerts(
      [
        answer("q1", "chat_gpt", false, {
          recommended: [{ name: "Old Co", position: 1 }],
        }),
      ],
      [
        answer("q1", "chat_gpt", false, {
          recommended: [
            { name: "old co", position: 1 },
            { name: "Fresh Ltd", position: 2 },
          ],
        }),
        answer("q2", "gemini", false, {
          recommended: [{ name: "Fresh Ltd", position: 1 }],
        }),
      ],
    );
    expect(
      alerts.filter((a) => a.kind === "new_rival").map((a) => a.title),
    ).toEqual(["New business recommended: Fresh Ltd"]);
  });

  it("reports a negative answer, but not one that was already negative", () => {
    const alerts = visibilityAlerts(
      [
        answer("q1", "chat_gpt", true, { sentiment: "neutral" }),
        answer("q2", "chat_gpt", true, { sentiment: "negative" }),
      ],
      [
        answer("q1", "chat_gpt", true, {
          sentiment: "negative",
          sentimentQuote: "Support is slow",
        }),
        answer("q2", "chat_gpt", true, { sentiment: "negative" }),
      ],
    );
    expect(alerts).toEqual([
      expect.objectContaining({
        kind: "negative",
        title: "ChatGPT describes you negatively for “q1”",
        detail: "“Support is slow”",
      }),
    ]);
  });

  it("lists new cited sites from answers that do not link the customer", () => {
    const alerts = visibilityAlerts(
      [
        answer("q", "chat_gpt", false, {
          citations: [cite("https://old.example/a")],
        }),
      ],
      [
        answer("q", "chat_gpt", false, {
          citations: [
            cite("https://www.new.example/a"),
            cite("https://new.example/b"),
            cite("https://old.example/a"),
            cite("bad url"),
          ],
        }),
        answer("q", "gemini", false, {
          brands: [
            {
              name: "Ownco",
              domain: "ownco.example",
              mentioned: false,
              cited: true,
            },
          ],
          citations: [cite("https://linked.example/")],
        }),
      ],
    );
    expect(alerts.map((a) => a.title)).toEqual([
      "AI started citing new.example",
    ]);
    expect(alerts[0]?.detail).toMatch(/^Cited in 1 answer that/u);
  });
  it("does not call an answer newly negative when the earlier one was never classified", () => {
    expect(
      visibilityAlerts(
        [answer("q", "chat_gpt", true)],
        [answer("q", "chat_gpt", true, { sentiment: "negative" })],
      ),
    ).toEqual([]);
  });

  it("announces nothing for a question the previous check failed to collect", () => {
    expect(
      visibilityAlerts(
        [
          answer("q1", "chat_gpt", false),
          answer("q2", "chat_gpt", true, { status: "failed" }),
        ],
        [
          answer("q1", "chat_gpt", false),
          answer("q2", "chat_gpt", true, {
            sentiment: "negative",
            recommended: [{ name: "Fresh Ltd", position: 1 }],
            citations: [cite("https://new.example/")],
          }),
        ],
      ),
    ).toEqual([]);
  });
});
