import { z } from "zod";
import { apiRequest } from "@/client/lib/uplumoApi";

const range = z.object({ startDate: z.string(), endDate: z.string() });
const metrics = z.object({
  clicks: z.number(),
  impressions: z.number(),
  ctr: z.number(),
  position: z.number(),
  sessions: z.number().nullable(),
  keyEvents: z.number().nullable(),
  keyEventRate: z.number().nullable(),
  engagementRate: z.number().nullable(),
  revenue: z.number().nullable(),
});
const growthPageSchema = z.object({
  page: z.string(),
  current: metrics,
  previous: metrics.nullable(),
  priority: z.number(),
  queries: z.array(
    z.object({
      query: z.string(),
      clicks: z.number(),
      impressions: z.number(),
      position: z.number(),
      branded: z.boolean(),
      competingPages: z.array(z.string()),
    }),
  ),
  findings: z.array(
    z.object({
      kind: z.string(),
      title: z.string(),
      evidence: z.string(),
      action: z.string(),
      confidence: z.string(),
    }),
  ),
});
const sourceSchema = z.object({
  id: z.string(),
  generatedAt: z.string(),
  range,
  previousRange: range,
  siteUrl: z.string(),
  propertyId: z.string(),
  currency: z.string(),
  timeZone: z.string(),
  brandTerms: z.array(z.string()),
  warnings: z.array(z.string()),
  limited: z.boolean(),
  truncated: z.boolean(),
  matchedPages: z.number(),
  gscPageCount: z.number(),
  ga4PageCount: z.number(),
});
const reportSchema = sourceSchema.extend({ pages: z.array(growthPageSchema) });
const actionSchema = z.object({
  id: z.string(),
  page: z.string(),
  note: z.string(),
  status: z.enum(["planned", "applied", "dismissed"]),
  baseline: growthPageSchema,
  source: sourceSchema,
  baselineRange: range,
  createdAt: z.string(),
  appliedAt: z.string().nullable(),
});
export type GrowthReport = z.infer<typeof reportSchema>;
export type GrowthPage = z.infer<typeof growthPageSchema>;
export type GrowthAction = z.infer<typeof actionSchema>;
type GrowthInput = {
  startDate?: string;
  endDate?: string;
  brandTerms?: string[];
};

function path(project: string, action: string) {
  return `/api/v1/projects/${encodeURIComponent(project)}/ga4/growth/${action}`;
}
export function analyzeGrowth(project: string, input: GrowthInput) {
  return apiRequest(path(project, "report"), reportSchema, "POST", input);
}
export function listGrowthActions(project: string) {
  return apiRequest(path(project, "list"), z.array(actionSchema), "POST", {});
}
export function saveGrowthAction(
  project: string,
  reportId: string,
  page: string,
  note: string,
) {
  return apiRequest(
    path(project, "save"),
    z.object({ id: z.string() }),
    "POST",
    { reportId, page, note },
  );
}
export function updateGrowthAction(
  project: string,
  id: string,
  status: GrowthAction["status"],
) {
  return apiRequest(
    path(project, "update"),
    z.object({ saved: z.boolean() }),
    "POST",
    { id, status },
  );
}
export function deleteGrowthAction(project: string, id: string) {
  return apiRequest(
    path(project, "delete"),
    z.object({ deleted: z.boolean() }),
    "POST",
    { id },
  );
}
export function compareGrowthAction(project: string, id: string, days: number) {
  return apiRequest(
    path(project, "compare"),
    z.object({
      status: z.enum(["waiting", "ready"]),
      baselineClicksPerDay: z.number().nullable(),
      afterClicksPerDay: z.number().nullable(),
      baseline: metrics,
      baselineRange: range,
      after: metrics.nullable(),
      afterRange: range,
      message: z.string(),
      limited: z.boolean().optional(),
    }),
    "POST",
    { id, days },
  );
}

export function getGrowthDevices(
  project: string,
  input: { startDate: string; endDate: string },
) {
  return apiRequest(
    path(project, "devices"),
    z.object({
      rows: z.array(
        z.object({
          deviceCategory: z.string(),
          sessions: z.number().nullable(),
          keyEvents: z.number().nullable(),
          sessionKeyEventRate: z.number().nullable(),
        }),
      ),
      limited: z.boolean(),
      range,
    }),
    "POST",
    input,
  );
}
export function getGrowthHealth(project: string) {
  return apiRequest(
    `/api/v1/projects/${encodeURIComponent(project)}/ga4/measurement-health`,
    z.object({
      issues: z.array(z.string()),
      keyEvents: z.array(z.object({ eventName: z.string() })),
    }),
    "POST",
    {},
  );
}
