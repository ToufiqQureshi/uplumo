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
  goalRate: z.number().nullable().optional(),
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
      competingPageCount: z.number().optional(),
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
  goalEvent: z.string().optional(),
  comparison: z.string().optional(),
  rowLimit: z.number().optional(),
  control: growthPageSchema.nullable().optional(),
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
  goalEvent?: string;
  comparison?: string;
  expanded?: boolean;
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
  controlPage?: string,
) {
  return apiRequest(
    path(project, "save"),
    z.object({ id: z.string() }),
    "POST",
    { reportId, page, note, controlPage },
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
      controlPage: z.string().optional(),
      controlAdjustedClickChange: z.number().nullable().optional(),
      controlMessage: z.string().optional(),
    }),
    "POST",
    { id, days },
  );
}

export function getGrowthDevices(
  project: string,
  input: { startDate: string; endDate: string; goalEvent?: string },
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
          goalRate: z.number().nullable().optional(),
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

export function getGrowthGoals(project: string) {
  return apiRequest(path(project, "goals"), z.array(z.string()), "POST", {});
}
export function getGrowthTracking(project: string, input: GrowthInput) {
  return apiRequest(
    path(project, "tracking"),
    z.object({
      range,
      limited: z.boolean(),
      checklist: z.array(z.string()),
      findings: z.array(
        z.object({
          kind: z.string(),
          title: z.string(),
          evidence: z.string(),
          action: z.string(),
          confidence: z.string(),
        }),
      ),
      rows: z.array(
        z.object({
          hostName: z.string(),
          sessionSource: z.string(),
          sessionMedium: z.string(),
          sessions: z.number().nullable(),
          keyEvents: z.number().nullable(),
        }),
      ),
    }),
    "POST",
    input,
  );
}
export function getGrowthFunnel(
  project: string,
  input: {
    startDate: string;
    endDate: string;
    steps: string[];
    byDevice: boolean;
  },
) {
  return apiRequest(
    path(project, "funnel"),
    z.object({
      range,
      steps: z.array(z.string()),
      limited: z.boolean(),
      message: z.string(),
      rows: z.array(
        z.object({
          funnelStepName: z.string(),
          deviceCategory: z.string().optional(),
          activeUsers: z.number().nullable(),
          funnelStepCompletionRate: z.number().nullable(),
          funnelStepAbandonments: z.number().nullable(),
          funnelStepAbandonmentRate: z.number().nullable(),
        }),
      ),
    }),
    "POST",
    input,
  );
}
const monitorSchema = z.object({
  enabled: z.boolean(),
  goalEvent: z.string(),
  nextCheckAt: z.string().nullable(),
  checkedAt: z.string().nullable(),
  lastError: z.string().nullable(),
  digest: z
    .object({
      generatedAt: z.string(),
      range,
      goalEvent: z.string(),
      pages: z.array(growthPageSchema),
      alerts: z.array(
        z.object({
          kind: z.string(),
          title: z.string(),
          evidence: z.string(),
          action: z.string(),
          confidence: z.string(),
        }),
      ),
      limited: z.boolean(),
      truncated: z.boolean(),
    })
    .nullable(),
});
export function getGrowthMonitor(project: string) {
  return apiRequest(path(project, "monitor"), monitorSchema, "POST", {});
}
export function saveGrowthMonitor(
  project: string,
  input: { enabled: boolean; goalEvent: string },
) {
  return apiRequest(
    path(project, "monitor-save"),
    monitorSchema,
    "POST",
    input,
  );
}

export function getGrowthSitemaps(project: string) {
  return apiRequest(
    `/api/v1/projects/${encodeURIComponent(project)}/gsc/search-performance/sitemaps`,
    z.object({
      siteUrl: z.string(),
      truncated: z.boolean(),
      message: z.string(),
      sitemaps: z.array(
        z.object({
          path: z.string(),
          lastSubmitted: z.string(),
          lastDownloaded: z.string(),
          pending: z.boolean(),
          isIndex: z.boolean(),
          errors: z.number(),
          warnings: z.number(),
          submitted: z.number(),
          contents: z
            .array(z.object({ type: z.string(), submitted: z.number() }))
            .optional(),
        }),
      ),
    }),
    "POST",
    {},
  );
}
export function diagnoseGrowthClicks(
  project: string,
  reportId: string,
  page: string,
) {
  return apiRequest(
    path(project, "clicks"),
    z.object({
      page: z.string(),
      range,
      previousRange: range,
      truncated: z.boolean(),
      message: z.string(),
      findings: z.array(
        z.object({
          kind: z.string(),
          title: z.string(),
          evidence: z.string(),
          action: z.string(),
          confidence: z.string(),
        }),
      ),
      queries: z.array(
        z.object({
          query: z.string(),
          current: metrics.nullable(),
          previous: metrics.nullable(),
        }),
      ),
    }),
    "POST",
    { reportId, page },
  );
}
