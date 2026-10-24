import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { AnalyticsOverview } from "./AnalyticsOverview";
import type { AnalyticsSummary } from "./analyticsApi";

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

describe("traffic overview measurement integrity", () => {
  it("explains received pageviews without presenting missing sessions as engagement", () => {
    const html = render();
    expect(html).toContain("no sessions were recorded");
    expect(html).toContain("Review tracking setup");
    expect(html).toContain("Source unavailable");
    expect(html).toContain("UTC · Today is partial");
  });
  it("shows observed actions without converting event counts into leads", () => {
    const html = render({
      sessions: 1,
      events: [{ name: "whatsapp", count: 3, visitors: 1 }],
    });
    expect(html).toContain("3 recorded custom events");
    expect(html).toContain("Events are not verified leads or sales");
    expect(html).toContain('href="#visitor-actions"');
    expect(html).not.toContain("Check your measurement");
  });
  it("flags differently cased event names without combining distinct records", () => {
    const html = render({
      sessions: 1,
      events: [
        { name: "WhatsApp", count: 1, visitors: 1 },
        { name: "whatsapp", count: 1, visitors: 1 },
      ],
    });
    expect(html).toContain("differ only by letter case");
    expect(html).toContain("Counts below remain separate");
  });
  it("selects observed leaders independently of response order and keeps AI referrals separate from visibility", () => {
    const html = render({
      sessions: 1,
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
    expect(html).toContain("Search engines");
    expect(html).toContain("/popular");
    expect(html).not.toContain("/small");
    expect(html).toContain("2 visitor-days");
    expect(html).toContain("does not measure how often AI mentions");
  });
});
