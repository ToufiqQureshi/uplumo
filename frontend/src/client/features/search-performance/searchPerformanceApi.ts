import { apiRequest } from "@/client/lib/uplumoApi";
import {
  searchPerformanceInputSchema,
  searchPerformanceTableExportInputSchema,
  searchPerformanceTableInputSchema,
} from "@/types/schemas/search-performance";
import { z } from "zod";

const metricsSchema = z.object({
  clicks: z.number(),
  impressions: z.number(),
  ctr: z.number(),
  position: z.number(),
});

const dimensionRowSchema = z
  .object({ key: z.string() })
  .extend(metricsSchema.shape);

const urlInspectionResultSchema = z.object({
  indexStatusResult: z
    .object({
      verdict: z.string().optional(),
      coverageState: z.string().optional(),
      robotsTxtState: z.string().optional(),
      indexingState: z.string().optional(),
      pageFetchState: z.string().optional(),
      lastCrawlTime: z.string().optional(),
      googleCanonical: z.string().optional(),
      userCanonical: z.string().optional(),
    })
    .optional(),
  inspectionResultLink: z.string().optional(),
});

const urlInspectionResponseSchema = z.object({
  siteUrl: z.string(),
  results: z.array(
    z.object({
      url: z.string(),
      result: urlInspectionResultSchema.nullable().optional(),
      error: z.string().optional(),
    }),
  ),
});

const reportSchema = z.discriminatedUnion("connected", [
  z.object({ connected: z.literal(false) }),
  z.object({
    connected: z.literal(true),
    range: z.object({
      startDate: z.string(),
      endDate: z.string(),
      prevStartDate: z.string(),
      prevEndDate: z.string(),
      comparisonAvailable: z.boolean(),
    }),
    totals: metricsSchema,
    prevTotals: metricsSchema,
    daily: z.array(
      z.object({
        date: z.string(),
        clicks: z.number(),
        impressions: z.number(),
      }),
    ),
    strikingDistance: z.array(
      z.object({
        query: z.string(),
        page: z.string(),
        clicks: z.number(),
        impressions: z.number(),
        position: z.number(),
      }),
    ),
    countries: z.array(dimensionRowSchema),
    // Pages that lost the most clicks against the previous period, with why.
    pageChanges: z
      .array(
        z.object({
          page: z.string(),
          clicks: z.number(),
          prevClicks: z.number(),
          impressions: z.number(),
          prevImpressions: z.number(),
          position: z.number(),
          prevPosition: z.number(),
          ctr: z.number(),
          prevCtr: z.number(),
          reason: z.enum(["rank_fell", "fewer_searches", "ctr_fell", "mixed"]),
        }),
      )
      .default([]),
  }),
]);

const tableSchema = z.discriminatedUnion("connected", [
  z.object({ connected: z.literal(false) }),
  z.object({
    connected: z.literal(true),
    dimension: z.enum(["query", "page"]),
    page: z.number(),
    pageSize: z.number(),
    hasNextPage: z.boolean(),
    totalCount: z.number().nullable(),
    rows: z.array(dimensionRowSchema),
  }),
]);

function reportPath(projectId: string, suffix: string) {
  return `/api/v1/projects/${encodeURIComponent(projectId)}/gsc/search-performance/${suffix}`;
}

export async function inspectSearchConsoleUrls({
  projectId,
  urls,
}: {
  projectId: string;
  urls: string[];
}) {
  return apiRequest(
    `/api/v1/projects/${encodeURIComponent(projectId)}/gsc/url-inspection/inspect`,
    urlInspectionResponseSchema,
    "POST",
    { urls },
  );
}

export async function getSearchPerformanceReport({
  data,
}: {
  data: z.input<typeof searchPerformanceInputSchema>;
}) {
  const parsed = searchPerformanceInputSchema.parse(data);
  const { projectId, ...input } = parsed;
  return apiRequest(
    reportPath(projectId, "report"),
    reportSchema,
    "POST",
    input,
  );
}

export async function getSearchPerformanceTable({
  data,
}: {
  data: z.input<typeof searchPerformanceTableInputSchema>;
}) {
  const parsed = searchPerformanceTableInputSchema.parse(data);
  const { projectId, ...input } = parsed;
  return apiRequest(reportPath(projectId, "table"), tableSchema, "POST", input);
}

export async function exportSearchPerformanceTable({
  data,
}: {
  data: z.input<typeof searchPerformanceTableExportInputSchema>;
}) {
  const parsed = searchPerformanceTableExportInputSchema.parse(data);
  const { projectId, ...input } = parsed;
  return apiRequest(
    reportPath(projectId, "export"),
    z.object({
      dimension: z.enum(["query", "page"]),
      rows: z.array(dimensionRowSchema),
    }),
    "POST",
    input,
  );
}
