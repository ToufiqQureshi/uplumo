import { describe, expect, it } from "vitest";
import {
  aiShareOfVisits,
  actionLabel,
  aiSourceRows,
  channelRows,
  displayPath,
  formatShare,
} from "./presentation";

describe("aiSourceRows", () => {
  it("lists every named assistant at zero when there is no AI traffic", () => {
    const rows = aiSourceRows([]);
    expect(rows.map((row) => row.label)).toEqual([
      "ChatGPT",
      "Perplexity",
      "Gemini",
      "Claude",
      "Copilot",
    ]);
    expect(rows.every((row) => row.count === 0 && row.share === 0)).toBe(true);
  });

  it("sorts by visitors, keeps the named order for ties, and adds other AI only when it has visitors", () => {
    const rows = aiSourceRows([
      { source: "claude", visitors: 5 },
      { source: "other_ai", visitors: 5 },
      { source: "chatgpt", visitors: 10 },
    ]);
    expect(rows.map((row) => [row.key, row.count, row.share])).toEqual([
      ["chatgpt", 10, 0.5],
      ["claude", 5, 0.25],
      ["other_ai", 5, 0.25],
      ["perplexity", 0, 0],
      ["gemini", 0, 0],
      ["copilot", 0, 0],
    ]);
    expect(
      aiSourceRows([{ source: "other_ai", visitors: 0 }]).map((r) => r.key),
    ).not.toContain("other_ai");
  });
});

describe("channelRows", () => {
  it("drops empty sources and gives each a share of all visits", () => {
    const rows = channelRows([
      { channel: "direct", sessions: 1 },
      { channel: "social", sessions: 0 },
      { channel: "ai", sessions: 3 },
    ]);
    expect(rows.map((row) => [row.label, row.count, row.share])).toEqual([
      ["AI assistants", 3, 0.75],
      ["Direct", 1, 0.25],
    ]);
  });
});

describe("aiShareOfVisits", () => {
  it.each([
    { name: "no visits", acquisition: [], share: 0 },
    {
      name: "no AI channel",
      acquisition: [{ channel: "search" as const, sessions: 4 }],
      share: 0,
    },
    {
      name: "a mix",
      acquisition: [
        { channel: "ai" as const, sessions: 1 },
        { channel: "search" as const, sessions: 3 },
      ],
      share: 0.25,
    },
  ])("$name -> $share", ({ acquisition, share }) => {
    expect(aiShareOfVisits(acquisition)).toBe(share);
  });
});

describe("actionLabel", () => {
  it("names the tracker's own actions and keeps owner tags as typed", () => {
    expect(actionLabel("whatsapp_click")).toBe("WhatsApp");
    expect(actionLabel("call_click")).toBe("Phone call");
    expect(actionLabel("form_submit")).toBe("Form");
    expect(actionLabel("Book demo")).toBe("Book demo");
  });
});

describe("formatShare", () => {
  it.each([
    [0, "0%"],
    [0.001, "<1%"],
    [0.005, "1%"],
    [0.426, "43%"],
    [1, "100%"],
  ])("%s -> %s", (share, text) => {
    expect(formatShare(share)).toBe(text);
  });
});

describe("displayPath", () => {
  it.each([
    ["/pricing", "/pricing"],
    ["/%E0%A4%B9%E0%A4%BF%E0%A4%82%E0%A4%A6%E0%A5%80", "/हिंदी"],
    // An encoded slash stays encoded: decoding it would change the path.
    ["/a%2Fb", "/a%2Fb"],
    ["/bad%E0%A4", "/bad%E0%A4"],
  ])("%s -> %s", (path, shown) => {
    expect(displayPath(path)).toBe(shown);
  });
});
