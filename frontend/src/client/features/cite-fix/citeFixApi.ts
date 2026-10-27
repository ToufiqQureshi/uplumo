import { z } from "zod";
import { apiRequest } from "@/client/lib/uplumoApi";

const pageRefSchema = z.object({ url: z.string(), title: z.string() });

const reportSchema = z.object({
  state: z.enum(["ready", "no_scan", "needs_rescan"]),
  running: z.object({ auditId: z.string(), status: z.string() }).nullable(),
  scan: z
    .object({
      auditId: z.string(),
      startUrl: z.string(),
      completedAt: z.string(),
    })
    .nullable(),
  score: z.number().int().min(0).max(100).nullable(),
  categories: z.array(
    z.object({
      id: z.string(),
      label: z.string(),
      weight: z.number(),
      score: z.number(),
    }),
  ),
  fixes: z.array(
    z.object({
      id: z.string(),
      category: z.string(),
      title: z.string(),
      why: z.string(),
      how: z.string(),
      impact: z.enum(["high", "medium", "low"]),
      gain: z.number(),
      pagesAffected: z.number(),
      pages: z.array(pageRefSchema),
    }),
  ),
  pages: z.array(
    z.object({
      url: z.string(),
      title: z.string(),
      score: z.number(),
      failed: z.array(z.string()),
    }),
  ),
  pagesChecked: z.number(),
  pagesTotal: z.number(),
});

export type CiteFixReport = z.infer<typeof reportSchema>;
export type CiteFix = CiteFixReport["fixes"][number];

export const citeFixKey = (projectId: string) =>
  ["cite-fix", projectId] as const;

/** Free: scores the latest completed site audit, no provider call. */
export function getCiteFixReport(projectId: string) {
  return apiRequest(
    `/api/v1/projects/${encodeURIComponent(projectId)}/cite-fix/report`,
    reportSchema,
    "POST",
    {},
  );
}
