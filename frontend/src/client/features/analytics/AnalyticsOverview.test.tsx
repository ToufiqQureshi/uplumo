import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { AnalyticsOverview } from "./AnalyticsOverview";
import type { AnalyticsSummary } from "./analyticsApi";
import { enquiries, mergeEvents } from "./visitSummary";

function render(overrides: Partial<AnalyticsSummary> = {}) {
  const summary: AnalyticsSummary = {
    visitors: 1,
    pageviews: 1,
    sessions: 0,
    engagedSessions: 0,
    bounceRate: 0,
    totalEngagementMs: 0,
    avgEngagementMs: 0,
    series: [],
    audience: [],
    topPages: [],
    channels: [],
    aiSources: [],
    referrers: [],
    devices: [],
    events: [],
    campaigns: [],
    entries: [],
    exits: [],
    acquisitionChannels: [],
    ...overrides,
  };
  return renderToStaticMarkup(
    <AnalyticsOverview
      summary={summary}
      range={{ from: "2026-10-01", to: "2026-10-24" }}
      onOpenTracking={() => {}}
    />,
  );
}

describe("analytics overview", () => {
  it("puts broken tracking first when page views arrive without visits", () => {
    const html = render();
    expect(html).toContain("Fix this first");
    expect(html).toContain("1 page views but 0 visits recorded");
    expect(html).toContain("Fix tracking in 2 minutes");
  });

  it("leads with enquiries once tracking works", () => {
    const html = render({
      visitors: 40,
      sessions: 50,
      events: [
        { name: "whatsapp", count: 3, visitors: 2 },
        { name: "scroll_75", count: 30, visitors: 20 },
      ],
      channels: [
        { channel: "direct", visitors: 1 },
        { channel: "search", visitors: 4 },
      ],
      topPages: [
        { path: "/small", pageviews: 1, avgEngagementMs: 0 },
        { path: "/popular", pageviews: 5, avgEngagementMs: 0 },
      ],
      aiSources: [{ source: "chatgpt", visitors: 2 }],
    });
    expect(html).not.toContain("Fix this first");
    expect(html).toContain(
      "40 people visited. 3 tried to contact you (whatsapp).",
    );
    expect(html).toContain("Most came from Search engines");
    expect(html).toContain("/popular");
    expect(html).toContain("AI assistants sent 2 visitors");
    expect(html).toContain("8%"); // 3 of 40 visitors
  });

  it("says plainly when nobody enquired", () => {
    const html = render({ sessions: 1, events: [] });
    expect(html).toContain("Nobody tapped WhatsApp, call or a form yet.");
  });
});

describe("visit summary", () => {
  it("merges event names that differ only by case and tells the customer", () => {
    const merged = mergeEvents([
      { name: "WhatsApp", count: 1, visitors: 1 },
      { name: "whatsapp", count: 2, visitors: 2 },
      { name: "form_submit", count: 1, visitors: 1 },
    ]);
    expect(merged[0]).toEqual({
      name: "whatsapp",
      count: 3,
      visitors: 2,
      spellings: 2,
    });
    const html = render({
      sessions: 2,
      events: [
        { name: "WhatsApp", count: 1, visitors: 1 },
        { name: "whatsapp", count: 1, visitors: 1 },
      ],
    });
    expect(html).toContain("as one action");
  });

  it("counts only contact actions as enquiries", () => {
    const { total, rows } = enquiries([
      { name: "WhatsApp click", count: 2, visitors: 2 },
      { name: "call_now", count: 1, visitors: 1 },
      { name: "book_now", count: 4, visitors: 3 },
      { name: "video_play", count: 9, visitors: 9 },
      { name: "scroll", count: 9, visitors: 9 },
    ]);
    expect(total).toBe(7);
    expect(rows.map((row) => row.name)).toEqual([
      "book_now",
      "WhatsApp click",
      "call_now",
    ]);
  });
});
