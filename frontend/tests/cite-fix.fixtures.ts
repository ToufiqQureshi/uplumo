const fix = (id: string, title: string, gain: number, pages: number) => ({
  id,
  category: "answers",
  title,
  why: "Assistants quote short, direct answers.",
  how: "Start the page with a 1 to 3 sentence answer.",
  impact: gain >= 5 ? "high" : gain >= 2 ? "medium" : "low",
  gain,
  pagesAffected: pages,
  pages: Array.from({ length: Math.min(pages, 20) }, (_, i) => ({
    url: `https://example.com/page-${i}`,
    title: `Page ${i}`,
  })),
});

export const ready = {
  state: "ready",
  running: null,
  scan: {
    auditId: "audit-1",
    startUrl: "https://example.com/",
    completedAt: "2026-10-27T10:00:00Z",
  },
  score: 44,
  categories: [
    { id: "access", label: "AI can read the site", weight: 20, score: 100 },
    { id: "answers", label: "Direct answers", weight: 20, score: 0 },
  ],
  fixes: [
    fix("answer-first", "The answer is not in the first paragraph", 10, 35),
    fix("ai-search-bots-allowed", "AI search bots are blocked", 10, 0),
  ],
  pages: [
    {
      url: "https://example.com/page-0",
      title: "Page 0",
      score: 17,
      failed: ["answer-first"],
    },
  ],
  pagesChecked: 35,
  pagesTotal: 40,
  proof: null,
  agent: {
    available: true,
    planAllowed: true,
    quickMaxCredits: 25,
    deepMaxCredits: 60,
    maxPages: 20,
    defaultPages: 5,
  },
};
