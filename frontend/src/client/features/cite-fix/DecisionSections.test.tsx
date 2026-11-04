import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import type { CiteFixProof, CiteFixReport } from "./citeFixApi";
import {
  CitationMatrix,
  CiteFixHeadline,
  ListPagesCard,
} from "./DecisionSections";
import { NextStepsCard } from "./NextStepsCard";
import { SignalsCard } from "./SignalsCard";
import { buildMatrix } from "./citeFixInsights";

const proof: CiteFixProof = {
  checkedAt: "2026-11-03T10:00:00Z",
  domain: "shop.example",
  answers: 14,
  citingYou: 2,
  rivals: [],
  yourPages: [],
  misses: [],
  visibility: {
    runs: 13,
    historyLimited: false,
    sourcesLimited: false,
    domains: [],
    pages: [],
    prompts: [
      {
        prompt: "firm in india",
        engine: "claude",
        answers: 13,
        citedAnswers: 7,
        averageRank: 4.7,
        latestRank: 5,
        inlineAnswers: 0,
        averageAnswerRank: null,
        latestAnswerRank: null,
      },
      {
        prompt: "firm in india",
        engine: "perplexity",
        answers: 12,
        citedAnswers: 6,
        averageRank: 3.5,
        latestRank: null,
        inlineAnswers: 0,
        averageAnswerRank: null,
        latestAnswerRank: null,
      },
      {
        prompt: "firm in india",
        engine: "chat_gpt",
        answers: 13,
        citedAnswers: 0,
        averageRank: null,
        latestRank: null,
        inlineAnswers: 0,
        averageAnswerRank: null,
        latestAnswerRank: null,
      },
    ],
  },
};

describe("CiteFixHeadline", () => {
  it("states the latest count and the slipping, never and working facts", () => {
    const html = renderToStaticMarkup(
      <CiteFixHeadline proof={proof} matrix={buildMatrix(proof)} />,
    );
    expect(html).toContain("AI answers name shop.example.");
    expect(html).toContain("Perplexity named you 6 times in earlier checks");
    expect(html).toContain("ChatGPT has not named you in 13 checks");
    expect(html).toContain("Claude names you in 7 of 13");
  });
});

describe("CitationMatrix", () => {
  it("labels engines by name and opens the lost cell first", () => {
    const html = renderToStaticMarkup(
      <CitationMatrix matrix={buildMatrix(proof)!} />,
    );
    expect(html).toContain("ChatGPT");
    expect(html).not.toContain("chat_gpt");
    expect(html).toContain("#5");
    expect(html).toContain("was 6×");
    expect(html).toContain(
      "Perplexity named you in 6 earlier checks, but not in the latest one.",
    );
  });
});

type Comparison = CiteFixReport["rivals"][number];
const rival: Comparison = {
  prompt: "q",
  engine: "perplexity",
  rival: {
    url: "https://rival.example/",
    domain: "rival.example",
    title: "Rival",
    appShell: false,
    fetchedAt: "",
    contentHash: "h",
    signals: {
      answerFirstWords: 13,
      author: false,
      date: "2021-03-24T09:05:14+00:00",
      schema: ["Organization", "FAQPage"],
      statistics: 9,
      faq: true,
      wordCount: 2316,
    },
  },
  yours: null,
  listsRivals: [],
  linksYou: false,
};

describe("SignalsCard", () => {
  it("compares rival signals and asks for a scan when yours are missing", () => {
    const html = renderToStaticMarkup(
      <SignalsCard rivals={[rival]} scanAction={<a href="/scan">Scan</a>} />,
    );
    expect(html).toContain("13 words");
    expect(html).toContain("2 types");
    expect(html).toContain("2021");
    expect(html).toContain('href="/scan"');
    expect(html).toContain("fills the “You” column");
  });

  it("shows nothing without rival pages", () => {
    expect(
      renderToStaticMarkup(<SignalsCard rivals={[]} scanAction={null} />),
    ).toBe("");
  });
});

describe("ListPagesCard and NextStepsCard", () => {
  it("hide when there is nothing to show", () => {
    expect(renderToStaticMarkup(<ListPagesCard pages={[]} />)).toBe("");
    expect(
      renderToStaticMarkup(<NextStepsCard steps={[]} actionFor={() => null} />),
    ).toBe("");
  });

  it("number the steps and render each action", () => {
    const html = renderToStaticMarkup(
      <NextStepsCard
        steps={[
          { kind: "scan", title: "Scan", why: "Why", proof: "Proof" },
          { kind: "get_listed", title: "List", why: "Why", proof: "Proof" },
        ]}
        actionFor={(kind) => <button type="button">{kind}</button>}
      />,
    );
    expect(html).toContain("STEP 1");
    expect(html).toContain("STEP 2");
    expect(html).toContain(">get_listed<");
  });
});
