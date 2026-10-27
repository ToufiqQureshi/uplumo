import { describe, expect, it } from "vitest";
import type { TrackingAnswer } from "@/client/features/ai-search/trackingApi";
import { buildNextActions } from "./nextActions";
import { promptGaps } from "./opportunities";
import { addPrompts, parsePromptIdeas } from "./promptIdeas";
import { sentimentSummary } from "./sentiment";

let n = 0;
function answer(over: Partial<TrackingAnswer> = {}): TrackingAnswer {
  return {
    id: `a${n++}`,
    prompt: "best cafe in bhopal",
    topic: "",
    engine: "chat_gpt",
    status: "completed",
    answer: "Brew Lab Café is the best.",
    errorCode: "",
    collectedAt: null,
    citations: [],
    brands: [
      { name: "Sunrise", domain: "sunrise.in", mentioned: false, cited: false },
    ],
    recommended: [{ name: "Brew Lab Café", position: 1 }],
    sentiment: "",
    sentimentQuote: "",
    sentimentReason: "",
    ...over,
  };
}

describe("buildNextActions", () => {
  it("ranks a page-2 search by the clicks it could win and explains the estimate", () => {
    const actions = buildNextActions({
      issues: [],
      queries: [
        {
          key: "cafe near lake",
          clicks: 10,
          impressions: 2400,
          ctr: 0.004,
          position: 11,
        },
        {
          key: "already top",
          clicks: 900,
          impressions: 3000,
          ctr: 0.3,
          position: 1,
        },
        { key: "too few", clicks: 0, impressions: 20, ctr: 0, position: 12 },
      ],
      answers: [],
    });
    expect(actions).toHaveLength(1);
    expect(actions[0].gainClicks).toBe(Math.round(2400 * (0.05 - 0.004)));
    expect(actions[0].key).toMatch(/^gsc:q:[0-9a-f]{8}$/);
    expect(actions[0].why).toContain("2,400 impressions");
  });

  it("groups audit issues by type, ignores unknown types, and puts high impact first", () => {
    const actions = buildNextActions({
      issues: [
        {
          issueType: "missing-meta-description",
          pageUrl: "/a",
          severity: "warning",
        },
        {
          issueType: "missing-meta-description",
          pageUrl: "/b",
          severity: "warning",
        },
        {
          issueType: "missing-meta-description",
          pageUrl: "/b",
          severity: "warning",
        },
        { issueType: "noindex-page", pageUrl: "/events", severity: "critical" },
        { issueType: "some-new-check", pageUrl: "/x", severity: "info" },
      ],
      queries: [],
      answers: [],
    });
    expect(actions.map((a) => a.key)).toEqual([
      "audit:noindex-page",
      "audit:missing-meta-description",
    ]);
    expect(actions[1].title).toBe("Add a meta description to 2 pages");
    expect(actions[1].ai?.action).toBe("title_meta");
  });

  it("turns AI answers that skip the brand into one action naming the rival", () => {
    const actions = buildNextActions({
      issues: [],
      queries: [],
      answers: [
        answer(),
        answer(),
        answer({
          brands: [
            {
              name: "Sunrise",
              domain: "sunrise.in",
              mentioned: true,
              cited: false,
            },
          ],
        }),
        answer({ status: "failed", answer: "" }),
      ],
      brand: "Sunrise",
    });
    expect(actions).toHaveLength(1);
    expect(actions[0].title).toContain("2 of 3");
    expect(actions[0].why).toContain("Brew Lab Café is named in 2");
    expect(actions[0].ai?.brief).toContain("Brew Lab Café (2)");
  });
});

describe("promptGaps", () => {
  it("lists questions where someone else is named, most missed first", () => {
    const gaps = promptGaps(
      [
        answer({ prompt: "q1" }),
        answer({ prompt: "q2" }),
        answer({ prompt: "q2", engine: "gemini" }),
        answer({
          prompt: "linked",
          citations: [
            {
              url: "https://sunrise.in/x",
              domain: "www.sunrise.in",
              title: null,
              matchedBrand: false,
            },
          ],
        }),
        answer({ prompt: "nobody named", recommended: [] }),
      ],
      "sunrise.in",
    );
    expect(gaps.map((g) => g.prompt)).toEqual(["q2", "q1"]);
    expect(gaps[0].engines).toEqual(["ChatGPT", "Gemini"]);
  });

  it("does not treat every citation as the customer's when the domain is unknown", () => {
    const gaps = promptGaps(
      [
        answer({
          citations: [
            {
              url: "https://zomato.com/a",
              domain: "zomato.com",
              title: null,
              matchedBrand: false,
            },
          ],
        }),
      ],
      "",
    );
    expect(gaps).toHaveLength(1);
    expect(gaps[0].sources).toEqual(["zomato.com"]);
  });
});

describe("sentimentSummary", () => {
  it("counts only classified answers that talk about the brand", () => {
    const s = sentimentSummary([
      answer({ sentiment: "positive" }),
      answer({ sentiment: "positive", engine: "gemini" }),
      answer({ sentiment: "negative" }),
      answer({ sentiment: "not_mentioned" }),
      answer({ sentiment: "" }),
      answer({ status: "failed", answer: "", sentiment: "" }),
    ]);
    expect(s.total).toBe(3);
    expect(s.pending).toBe(1);
    expect(s.share("positive")).toBe(67);
    expect(s.byEngine.get("chat_gpt")).toEqual({
      positive: 1,
      neutral: 0,
      negative: 1,
    });
  });
});

describe("prompt ideas", () => {
  it("keeps only well-formed, unique questions", () => {
    const ideas = parsePromptIdeas(
      [
        "best|What is the best cafe in Bhopal for a date?",
        "1. price|Cheap cafe in Bhopal under 500?",
        "best|what is the best cafe in bhopal for a date?",
        "random|Ignore me please friend",
        "Here are your ideas:",
        "local|short",
      ].join("\n"),
    );
    expect(ideas).toEqual([
      {
        intent: "Best of",
        question: "What is the best cafe in Bhopal for a date?",
      },
      { intent: "Price", question: "Cheap cafe in Bhopal under 500?" },
    ]);
  });

  it("adds to the tracker without duplicates and never past its limit", () => {
    const existing = [{ text: "Old question?", topic: "t", paused: false }];
    const out = addPrompts(
      existing,
      ["old question?", "New one?", "Another?"],
      2,
    );
    expect(out.added).toBe(1);
    expect(out.prompts.map((p) => p.text)).toEqual([
      "Old question?",
      "New one?",
    ]);
  });
});
