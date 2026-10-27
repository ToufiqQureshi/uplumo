import { z } from "zod";
import { apiRequest } from "@/client/lib/uplumoApi";

const statesSchema = z.object({
  states: z.array(
    z.object({
      key: z.string(),
      status: z.enum(["done", "ignored"]),
      updatedAt: z.string(),
    }),
  ),
});
export type ActionStatus = "todo" | "done" | "ignored";

const base = (projectId: string) =>
  `/api/v1/projects/${encodeURIComponent(projectId)}`;

export const nextActionsKey = (projectId: string) =>
  ["next-actions", projectId] as const;

export const insightsApi = {
  actionStates: (projectId: string) =>
    apiRequest(`${base(projectId)}/next-actions/get`, statesSchema, "POST", {}),
  setActionStatus: (projectId: string, key: string, status: ActionStatus) =>
    apiRequest(
      `${base(projectId)}/next-actions/set`,
      z.object({ saved: z.boolean() }),
      "POST",
      { key, status },
    ),
  /** Paid: classifies the run's answers that name the brand. */
  classifySentiment: (projectId: string, runId: string) =>
    apiRequest(
      `${base(projectId)}/ai-search/tracking/runs/${encodeURIComponent(runId)}/sentiment`,
      z.object({
        classified: z.number(),
        notMentioned: z.number(),
        remaining: z.number(),
      }),
      "POST",
      {},
    ),
};
