import { z } from "zod";
import { analyticsSummaryQueryOptions } from "@/client/features/analytics/analyticsApi";
import { getSearchPerformanceReport } from "@/client/features/search-performance/searchPerformanceApi";
import { apiRequest } from "@/client/lib/uplumoApi";
import type { ReportDataset } from "./reportDefinition";

export const REPORT_SOURCES = [
  {
    id: "traffic",
    name: "Uplumo · daily traffic",
    description: "Daily visitors and pageviews",
  },
  {
    id: "channels",
    name: "Uplumo · traffic channels",
    description: "Visitors by acquisition channel",
  },
  {
    id: "ai",
    name: "Uplumo · AI referrals",
    description: "Visitors from AI assistants",
  },
  {
    id: "pages",
    name: "Uplumo · top pages",
    description: "Pageviews by page (top pages only)",
  },
  {
    id: "gsc",
    name: "Search Console · daily",
    description: "Google search clicks and impressions",
  },
  {
    id: "ga4",
    name: "GA4 · organic sessions",
    description: "Daily organic search sessions",
  },
] as const;

export async function loadReportSource(
  projectId: string,
  source: string,
  from: string,
  to: string,
): Promise<ReportDataset> {
  const range = { from, to };
  const id = crypto.randomUUID();
  const label = REPORT_SOURCES.find((item) => item.id === source);
  if (!label) throw new Error("Choose a data source.");
  const name = `${label.name} (${from} – ${to})`;
  if (source === "gsc") {
    const data = await getSearchPerformanceReport({
      data: { projectId, startDate: from, endDate: to },
    });
    if (!data.connected)
      throw new Error(
        "Connect Search Console from Traffic → Connections first, or upload a CSV.",
      );
    return {
      id,
      name,
      columns: ["Date", "Clicks", "Impressions"],
      rows: data.daily.map((row) => [row.date, row.clicks, row.impressions]),
      dateColumn: "Date",
    };
  }
  if (source === "ga4") {
    const data = await apiRequest(
      `/api/v1/projects/${encodeURIComponent(projectId)}/ga4/overview/organic`,
      z.object({
        trend: z.array(
          z.object({ date: z.string(), sessions: z.number().nullable() }),
        ),
      }),
      "POST",
      { startDate: from, endDate: to, trend: "daily" },
    );
    return {
      id,
      name,
      columns: ["Date", "Sessions"],
      rows: data.trend.map((row) => [
        row.date.replace(/^(\d{4})(\d{2})(\d{2})$/, "$1-$2-$3"),
        row.sessions,
      ]),
      dateColumn: "Date",
    };
  }
  const data = await analyticsSummaryQueryOptions(projectId, range).queryFn();
  if (source === "channels")
    return {
      id,
      name,
      columns: ["Channel", "Visitors"],
      rows: data.channels.map((row) => [row.channel, row.visitors]),
      dateColumn: "",
    };
  if (source === "ai")
    return {
      id,
      name,
      columns: ["Assistant", "Visitors"],
      rows: data.aiSources.map((row) => [row.source, row.visitors]),
      dateColumn: "",
    };
  if (source === "pages")
    return {
      id,
      name,
      columns: ["Page", "Pageviews"],
      rows: data.topPages.map((row) => [row.path, row.pageviews]),
      dateColumn: "",
    };
  return {
    id,
    name,
    columns: ["Date", "Visitors", "Pageviews"],
    rows: data.series.map((row) => [row.date, row.visitors, row.pageviews]),
    dateColumn: "Date",
  };
}
