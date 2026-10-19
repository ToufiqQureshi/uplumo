import { z } from "zod";
import { apiRequest } from "@/client/lib/uplumoApi";

const prompt = z.object({
  text: z.string(),
  topic: z.string(),
  paused: z.boolean(),
});
const brand = z.object({ name: z.string(), domain: z.string() });
export const trackingConfigSchema = z.object({
  brand: z.string(),
  domain: z.string(),
  competitors: z.array(brand),
  prompts: z.array(prompt),
  engines: z.array(z.string()),
  locationCode: z.number(),
  languageCode: z.string(),
  interval: z.enum(["daily", "weekly", "monthly"]),
  enabled: z.boolean(),
  maxCreditsPerRun: z.number().int().nonnegative(),
});
export type TrackingConfig = z.infer<typeof trackingConfigSchema>;
const run = z.object({
  id: z.string(),
  configuration: trackingConfigSchema,
  scopeKey: z.string(),
  trigger: z.string(),
  status: z.string(),
  errorCode: z.string(),
  createdAt: z.string(),
  finishedAt: z.string().nullable(),
  completed: z.number(),
  failed: z.number(),
  answered: z.number(),
  mentioned: z.number(),
  cited: z.number(),
});
const trackingStateSchema = z.object({
  configuration: trackingConfigSchema.nullable(),
  nextCheckAt: z.string().nullable(),
  runs: z.array(run),
  providerConfigured: z.boolean(),
});
export const trackingEstimateSchema = z.object({
  answers: z.number(),
  credits: z.number(),
  monthlyCredits: z.number(),
  ceiling: z.boolean(),
});
const trackingAnswersSchema = z.array(
  z.object({
    id: z.string(),
    prompt: z.string(),
    topic: z.string(),
    engine: z.string(),
    status: z.string(),
    answer: z.string(),
    errorCode: z.string(),
    collectedAt: z.string().nullable(),
    citations: z.array(
      z.object({
        url: z.url().regex(/^https?:\/\//),
        domain: z.string().nullable(),
        title: z.string().nullable(),
        matchedBrand: z.boolean(),
        source: z.string().nullable().optional(),
        publishedAt: z.string().nullable().optional(),
      }),
    ),
    brands: z.array(
      brand.extend({ mentioned: z.boolean(), cited: z.boolean() }),
    ),
    // Businesses the AI named, in its order. Missing for answers saved
    // before they were kept.
    recommended: z
      .array(
        z.object({
          name: z.string(),
          category: z.string().optional(),
          position: z.number().int(),
        }),
      )
      .default([]),
  }),
);
export type TrackingAnswer = z.infer<typeof trackingAnswersSchema>[number];
export type TrackingEstimate = z.infer<typeof trackingEstimateSchema>;
export const trackingKey = (projectId: string) =>
  ["ai-tracking", projectId] as const;
const path = (id: string) =>
  `/api/v1/projects/${encodeURIComponent(id)}/ai-search/tracking`;
export const trackingApi = {
  state: (id: string) => apiRequest(path(id), trackingStateSchema),
  save: (id: string, config: TrackingConfig) =>
    apiRequest(`${path(id)}/save`, trackingStateSchema, "POST", config),
  estimate: (id: string, config: TrackingConfig) =>
    apiRequest(`${path(id)}/estimate`, trackingEstimateSchema, "POST", config),
  run: (id: string, maxCredits: number) =>
    apiRequest(`${path(id)}/run`, z.object({ runId: z.string() }), "POST", {
      maxCredits,
    }),
  answers: (id: string, runId: string) =>
    apiRequest(
      `${path(id)}/runs/${encodeURIComponent(runId)}`,
      trackingAnswersSchema,
    ),
};

/** Only substantive, successfully collected answers enter mention/citation rates. */
export function summarizeTracking(answers: TrackingAnswer[]) {
  const summaries = new Map<
    string,
    {
      name: string;
      domain: string;
      answers: number;
      mentions: number;
      citations: number;
    }
  >();
  for (const a of answers) {
    if (a.status !== "completed" || !a.answer.trim()) continue;
    for (const b of a.brands) {
      const row = summaries.get(b.domain) ?? {
        name: b.name,
        domain: b.domain,
        answers: 0,
        mentions: 0,
        citations: 0,
      };
      row.answers++;
      row.mentions += Number(b.mentioned);
      row.citations += Number(b.cited);
      summaries.set(b.domain, row);
    }
  }
  return [...summaries.values()];
}
