import { renderToStaticMarkup } from "react-dom/server";
import { beforeEach, describe, expect, it } from "vitest";
import {
  AiVisitorsCard,
  AnalyticsOverview,
  LeadPagesCard,
  LeadSourcesCard,
  TrackingProblemCard,
  changeLabel,
  leadInsight,
} from "./AnalyticsOverview";
import type { AnalyticsSummary } from "./analyticsApi";
import { leadOptions } from "./LeadActionsCard";
import { mergeEvents } from "./visitSummary";

let base: AnalyticsSummary;

beforeEach(() => {
  base = {
    visitors: 40,
    pageviews: 90,
    sessions: 45,
    engagedSessions: 20,
    bounceRate: 0.5,
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
    previousVisitors: 20,
    leads: {
      configured: false,
      count: 0,
      visitors: 0,
      previousVisitors: 0,
      actions: [],
      pages: [],
    },
    sourceLeads: [],
    aiLandings: [],
  };
});

function withLeads(
  leads: Partial<AnalyticsSummary["leads"]>,
  rest: Partial<AnalyticsSummary> = {},
): AnalyticsSummary {
  return {
    ...base,
    ...rest,
    leads: { ...base.leads, configured: true, ...leads },
  };
}

function overview(summary: AnalyticsSummary) {
  return renderToStaticMarkup(
    <AnalyticsOverview summary={summary} days={30} onPickLeads={() => {}} />,
  );
}

describe("tracking problem", () => {
  it("shows only when page views arrive without visits", () => {
    const broken = renderToStaticMarkup(
      <TrackingProblemCard
        summary={{ ...base, pageviews: 5, sessions: 0 }}
        onOpenTracking={() => {}}
      />,
    );
    expect(broken).toContain("5 page views but no visits");
    expect(broken).toContain("Get the current code");
    const healthy = renderToStaticMarkup(
      <TrackingProblemCard summary={base} onOpenTracking={() => {}} />,
    );
    expect(healthy).toBe("");
  });
});

describe("analytics overview", () => {
  it("asks for lead actions instead of guessing them", () => {
    const html = overview({
      ...base,
      events: [{ name: "whatsapp_click", count: 9, visitors: 4 }],
    });
    expect(html).toContain("Choose lead actions");
    expect(html).not.toContain("Became leads");
    expect(html).not.toMatch(/WhatsApp|call button|enquir/i);
  });

  it("shows chosen leads, the rate and the change from last period", () => {
    const html = overview(
      withLeads({
        count: 7,
        visitors: 6,
        previousVisitors: 4,
        actions: [{ name: "Book a call", count: 7, visitors: 6 }],
      }),
    );
    expect(html).toContain(">6<");
    expect(html).toContain("15%"); // 6 of 40 visitors
    expect(html).toContain("+50%"); // 6 leads vs 4
    expect(html).toContain("+100%"); // 40 visitors vs 20
    expect(html).toContain("Leads: Book a call");
  });

  it("names no lead action when none happened in the period", () => {
    expect(overview(withLeads({}))).toContain("none in this period");
  });
});

describe("changeLabel", () => {
  it.each([
    [10, 0, "new"],
    [0, 0, null],
    [5, 10, "−50%"],
    [10, 10, "no change"],
    [1001, 1000, "no change"],
    [3, 1, "+200%"],
  ])("%i vs %i is %s", (now, before, want) => {
    expect(changeLabel(now, before)).toBe(want);
  });
});

describe("leadInsight", () => {
  it("says nothing until leads are chosen", () => {
    expect(leadInsight(base)).toBeNull();
  });

  it("names the best source and the busy page that converts badly", () => {
    const summary = withLeads(
      {
        visitors: 8,
        pages: [
          { path: "/%E0%A4%B9", visitors: 20, leadVisitors: 0 },
          { path: "/contact", visitors: 5, leadVisitors: 4 },
        ],
      },
      {
        sourceLeads: [
          { channel: "direct", visitors: 25, leadVisitors: 1 },
          { channel: "search", visitors: 10, leadVisitors: 6 },
        ],
      },
    );
    const text = leadInsight(summary) ?? "";
    expect(text).toContain("Most leads came from Search engines.");
    expect(text).toContain("/ह gets 50% of visitors but converts at 0%");
  });

  it("points at the busiest page when there are no leads yet", () => {
    const text = leadInsight(
      withLeads({
        pages: [{ path: "/pricing", visitors: 12, leadVisitors: 0 }],
      }),
    );
    expect(text).toContain(
      "No leads yet. /pricing gets the most visitors (12)",
    );
  });

  it("stays quiet when no page or source stands out", () => {
    const summary = withLeads(
      {
        visitors: 4,
        pages: [{ path: "/", visitors: 40, leadVisitors: 4 }],
      },
      { sourceLeads: [] },
    );
    expect(leadInsight(summary)).toBeNull();
  });
});

describe("lead sources", () => {
  it("lists only sources with visitors, with lead columns once chosen", () => {
    const rows: AnalyticsSummary["sourceLeads"] = [
      { channel: "search", visitors: 10, leadVisitors: 2 },
      { channel: "social", visitors: 0, leadVisitors: 0 },
    ];
    const chosen = renderToStaticMarkup(
      <LeadSourcesCard summary={withLeads({}, { sourceLeads: rows })} />,
    );
    expect(chosen).toContain("Search engines");
    expect(chosen).not.toContain("Social media");
    expect(chosen).toContain("20%");
    const unchosen = renderToStaticMarkup(
      <LeadSourcesCard summary={{ ...base, sourceLeads: rows }} />,
    );
    expect(unchosen).toContain("Where visitors come from");
    expect(unchosen).not.toContain("Lead rate");
    expect(renderToStaticMarkup(<LeadSourcesCard summary={base} />)).toBe("");
  });
});

describe("lead pages", () => {
  it("is hidden until leads are chosen", () => {
    expect(renderToStaticMarkup(<LeadPagesCard summary={base} />)).toBe("");
  });

  it("shows each busy page's leads", () => {
    const html = renderToStaticMarkup(
      <LeadPagesCard
        summary={withLeads({
          pages: [{ path: "/pricing", visitors: 10, leadVisitors: 3 }],
        })}
      />,
    );
    expect(html).toContain("/pricing");
    expect(html).toContain("10 visitors · 3 leads");
  });
});

describe("AI visitors", () => {
  it("says plainly when no AI assistant sent anyone, and links on", () => {
    const html = renderToStaticMarkup(
      <AiVisitorsCard summary={base} citeFixLink={<a href="/x">Why</a>} />,
    );
    expect(html).toContain("No visitor came from an AI assistant");
    expect(html).toContain('href="/x"');
  });

  it("lists where AI visitors landed and their leads", () => {
    const html = renderToStaticMarkup(
      <AiVisitorsCard
        summary={withLeads(
          {},
          {
            aiSources: [{ source: "chatgpt", visitors: 3 }],
            aiLandings: [
              {
                source: "chatgpt",
                path: "/guide",
                visitors: 3,
                leadVisitors: 1,
              },
            ],
          },
        )}
        citeFixLink={null}
      />,
    );
    expect(html).toContain("ChatGPT");
    expect(html).toContain("/guide");
    expect(html).toContain("1 became a lead");
    expect(html).toContain("3 · 1 lead<");
  });
});

describe("leadOptions", () => {
  it("keeps saved picks first, even when unseen, without duplicates", () => {
    const options = leadOptions(
      ["Book a call", "Old signup"],
      [
        { name: "Download", count: 9, visitors: 5 },
        { name: "Book a call", count: 2, visitors: 2 },
      ],
    );
    expect(options.map((o) => [o.name, o.seen])).toEqual([
      ["Book a call", true],
      ["Old signup", false],
      ["Download", true],
    ]);
  });

  it("is empty when nothing was recorded and nothing saved", () => {
    expect(leadOptions([], [])).toEqual([]);
  });
});

describe("mergeEvents", () => {
  it("merges actions that differ only by letter case", () => {
    expect(
      mergeEvents([
        { name: "Signup", count: 3, visitors: 2 },
        { name: "signup", count: 1, visitors: 1 },
      ]),
    ).toEqual([{ name: "Signup", count: 4, visitors: 2, spellings: 2 }]);
  });
});
