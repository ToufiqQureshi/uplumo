import { z } from "zod";
import { apiRequest } from "@/client/lib/uplumoApi";
const metric = z.object({
  visitors: z.number().nonnegative(),
  goalVisitors: z.number().nonnegative(),
  rate: z.number().min(0).max(100),
});
const change = z.object({
  id: z.string(),
  path: z.string(),
  eventName: z.string(),
  note: z.string(),
  baseline: metric,
  baselineStart: z.string(),
  baselineEnd: z.string(),
  createdAt: z.string(),
});
const report = z.object({
  eventName: z.string(),
  siteConfigured: z.boolean(),
  goalObserved: z.boolean(),
  beforeStart: z.string(),
  currentStart: z.string(),
  until: z.string(),
  pages: z.array(
    z.object({
      path: z.string(),
      before: metric,
      after: metric,
      signal: z.enum([
        "verify_tracking",
        "low_data",
        "traffic_changed",
        "enquiry_drop",
        "no_goal_events",
        "stable",
      ]),
    }),
  ),
  changes: z.array(change),
});
const comparison = z.object({
  change,
  after: metric,
  afterStart: z.string(),
  afterEnd: z.string(),
  ready: z.boolean(),
});
export const enquiryKey = (id: string) => ["enquiry-evidence", id] as const;
const path = (id: string) =>
  `/api/v1/projects/${encodeURIComponent(id)}/analytics/enquiries`;
export const enquiryApi = {
  report: (id: string) => apiRequest(path(id), report),
  saveGoal: (id: string, eventName: string) =>
    apiRequest(`${path(id)}/goal`, z.object({ saved: z.boolean() }), "POST", {
      eventName,
    }),
  saveChange: (id: string, page: string, note: string) =>
    apiRequest(`${path(id)}/changes`, z.object({ id: z.string() }), "POST", {
      path: page,
      note,
    }),
  compare: (id: string, changeId: string) =>
    apiRequest(
      `${path(id)}/changes/${encodeURIComponent(changeId)}`,
      comparison,
    ),
  deleteChange: (id: string, changeId: string) =>
    apiRequest(
      `${path(id)}/changes/${encodeURIComponent(changeId)}/delete`,
      z.object({ saved: z.boolean() }),
      "POST",
    ),
};
