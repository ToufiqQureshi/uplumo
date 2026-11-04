import { beforeEach, describe, expect, it } from "vitest";
import type { CiteFixProof, CiteFixReport } from "./citeFixApi";
import { overviewMarkdown } from "./CitationSections";
import {
  buildMatrix,
  headline,
  listPages,
  nextSteps,
  rivalRanks,
  signalRivals,
} from "./citeFixInsights";

type Prompt = NonNullable<CiteFixProof["visibility"]>["prompts"][number];
type Comparison = CiteFixReport["rivals"][number];

function prompt(over: Partial<Prompt>): Prompt {
  return {
    prompt: "q",
    engine: "chat_gpt",
    answers: 13,
    citedAnswers: 0,
    averageRank: null,
    latestRank: null,
    inlineAnswers: 0,
    averageAnswerRank: null,
    latestAnswerRank: null,
    ...over,
  };
}

let proof: CiteFixProof;

beforeEach(() => {
  proof = {
    checkedAt: "2026-11-03T10:00:00Z",
    domain: "shop.example",
    answers: 14,
    citingYou: 2,
    rivals: [{ domain: "rival.example", count: 5 }],
    yourPages: [],
    misses: [
      {
        prompt: "best firms",
        engine: "perplexity",
        cited: ["a.example", "b.example"],
        citedPages: [],
      },
    ],
    visibility: {
      runs: 13,
      historyLimited: false,
      sourcesLimited: false,
      domains: [
        {
          name: "www.rival.example",
          firstSeen: "",
          lastSeen: "",
          averageRank: 3.3,
          latestCount: 4,
          momentum: "flat",
        },
        {
          name: "other.example",
          firstSeen: "",
          lastSeen: "",
          averageRank: 2.1,
          latestCount: 4,
          momentum: "flat",
        },
        {
          name: "gone.example",
          firstSeen: "",
          lastSeen: "",
          averageRank: 1,
          latestCount: 0,
          momentum: "down",
        },
        {
          name: "shop.example",
          firstSeen: "",
          lastSeen: "",
          averageRank: 4.6,
          latestCount: 2,
          momentum: "up",
        },
      ],
      pages: [],
      prompts: [
        prompt({ prompt: "best firms", engine: "chat_gpt" }),
        prompt({ prompt: "best firms", engine: "perplexity", citedAnswers: 6 }),
        prompt({ prompt: "firm in india", engine: "chat_gpt" }),
        prompt({
          prompt: "firm in india",
          engine: "claude",
          citedAnswers: 7,
          latestRank: 5,
        }),
        prompt({
          prompt: "best firms",
          engine: "claude",
          citedAnswers: 1,
          latestRank: 9,
        }),
      ],
    },
  };
});

describe("buildMatrix", () => {
  it("returns nothing without saved history", () => {
    expect(buildMatrix(null)).toBeNull();
    expect(buildMatrix({ ...proof, visibility: null })).toBeNull();
  });

  it("places every question and engine, in first-seen order, with states", () => {
    const matrix = buildMatrix(proof)!;
    expect(matrix.prompts).toEqual(["best firms", "firm in india"]);
    expect(matrix.engines).toEqual(["chat_gpt", "perplexity", "claude"]);
    const lost = matrix.cells.get("best firms")?.get("perplexity");
    expect(lost?.state).toBe("lost");
    expect(lost?.citedInstead).toEqual(["a.example", "b.example"]);
    expect(matrix.cells.get("firm in india")?.get("claude")?.state).toBe(
      "cited",
    );
    expect(matrix.cells.get("firm in india")?.get("chat_gpt")?.state).toBe(
      "never",
    );
    expect(
      matrix.cells.get("firm in india")?.get("perplexity"),
    ).toBeUndefined();
  });
});

describe("headline", () => {
  it("separates cited, lost and never-naming engines", () => {
    const facts = headline(buildMatrix(proof)!);
    expect(facts.cited.map((c) => c.latestRank)).toEqual([9, 5]);
    expect(facts.lost.map((c) => c.engine)).toEqual(["perplexity"]);
    expect(facts.neverEngines).toEqual(["chat_gpt"]);
    expect(facts.strongest?.citedAnswers).toBe(7);
  });
});

describe("rivalRanks", () => {
  it("ranks rivals of the latest check and adds the customer last", () => {
    const rows = rivalRanks(proof);
    expect(rows.map((r) => r.domain)).toEqual([
      "other.example",
      "rival.example",
      "shop.example",
    ]);
    expect(rows.at(-1)?.you).toBe(true);
  });

  it("falls back to the latest check's counts without history", () => {
    const rows = rivalRanks({ ...proof, visibility: null });
    expect(rows).toEqual([
      {
        domain: "rival.example",
        averageRank: null,
        latestCount: 5,
        you: false,
      },
    ]);
  });
});

function comparison(over: Partial<Comparison> & { url: string }): Comparison {
  return {
    prompt: "best firms",
    engine: "perplexity",
    rival: {
      url: over.url,
      domain: new URL(over.url).hostname,
      title: "T",
      appShell: false,
      fetchedAt: "",
      contentHash: "h",
      signals: {
        answerFirstWords: 13,
        author: false,
        date: "",
        schema: [],
        statistics: 9,
        faq: true,
        wordCount: 785,
      },
    },
    yours: null,
    listsRivals: [],
    linksYou: false,
    ...over,
  };
}

describe("listPages and signalRivals", () => {
  const rows = () => [
    comparison({
      url: "https://dir.example/top",
      listsRivals: ["a.example", "b.example", "c.example"],
    }),
    comparison({
      url: "https://dir.example/top",
      engine: "gemini",
      listsRivals: ["a.example", "b.example", "c.example"],
    }),
    comparison({
      url: "https://lists-you.example/",
      listsRivals: ["a.example", "b.example"],
      linksYou: true,
    }),
    comparison({
      url: "https://one-link.example/",
      listsRivals: ["a.example"],
    }),
    comparison({ url: "https://a.example/" }),
    comparison({ url: "https://a.example/other" }),
    comparison({ url: "https://b.example/" }),
  ];

  it("keeps lists of two or more rivals that skip the customer, merged per page", () => {
    const pages = listPages(rows());
    expect(pages).toHaveLength(1);
    expect(pages[0].engines).toEqual(["perplexity", "gemini"]);
  });

  it("compares company pages only, one per site", () => {
    expect(signalRivals(rows()).map((c) => c.rival.url)).toEqual([
      "https://one-link.example/",
      "https://a.example/",
    ]);
    expect(signalRivals(rows().slice(3), 3).map((c) => c.rival.domain)).toEqual(
      ["one-link.example", "a.example", "b.example"],
    );
  });
});

const name = (engine: string) => engine.toUpperCase();

function report(over: Partial<CiteFixReport>): CiteFixReport {
  return {
    state: "ready",
    running: null,
    scan: null,
    score: null,
    categories: [],
    fixes: [],
    pages: [],
    pagesChecked: 0,
    pagesTotal: 0,
    topics: null,
    proof,
    mentions: null,
    googleAiOverview: null,
    rivals: [],
    seoIssues: [],
    agent: {
      available: false,
      planAllowed: false,
      quickMaxCredits: 25,
      deepMaxCredits: 60,
      maxPages: 3,
      defaultPages: 1,
    },
    ...over,
  };
}

describe("nextSteps", () => {
  it("asks for a check and a scan before anything else", () => {
    const steps = nextSteps(
      report({ proof: null, state: "no_scan" }),
      null,
      [],
      name,
    );
    expect(steps.map((s) => s.kind)).toEqual(["track", "scan"]);
  });

  it("orders the data's own problems, with their evidence", () => {
    const matrix = buildMatrix(proof);
    const lists = listPages([
      comparison({
        url: "https://dir.example/top",
        listsRivals: ["a.example", "b.example"],
      }),
    ]);
    const steps = nextSteps(
      report({ state: "needs_rescan" }),
      matrix,
      lists,
      name,
    );
    expect(steps.map((s) => s.kind)).toEqual([
      "scan",
      "win_back",
      "get_listed",
    ]);
    expect(steps[1].title).toBe("Win back PERPLEXITY on “best firms”");
    expect(steps[1].proof).toBe(
      "In the latest check it cited a.example and b.example.",
    );
    expect(steps[2].proof).toBe(
      "dir.example links to a.example, b.example and not to you.",
    );
  });

  it("shows nothing when nothing is wrong", () => {
    expect(nextSteps(report({}), null, [], name)).toEqual([]);
  });
});

describe("overviewMarkdown", () => {
  it.each([
    ["`Lead sentence` then text", "**Lead sentence** then text"],
    ["```\ncode\n```", "```\ncode\n```"],
    ["no marks", "no marks"],
    ["`a`, `b`", "**a**, **b**"],
  ])("%j", (input, want) => {
    expect(overviewMarkdown(input)).toBe(want);
  });
});
