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
  proof: z
    .object({
      checkedAt: z.string(),
      domain: z.string(),
      answers: z.number(),
      citingYou: z.number(),
      rivals: z.array(z.object({ domain: z.string(), count: z.number() })),
      yourPages: z.array(z.object({ url: z.string(), count: z.number() })),
      misses: z.array(
        z.object({
          prompt: z.string(),
          engine: z.string(),
          cited: z.array(z.string()),
        }),
      ),
    })
    .nullable(),
  agent: z.object({
    available: z.boolean(),
    planAllowed: z.boolean(),
    quickMaxCredits: z.number().int().positive(),
    deepMaxCredits: z.number().int().positive(),
    maxPages: z.number().int().positive(),
    defaultPages: z.number().int().positive(),
  }),
});

export type CiteFixReport = z.infer<typeof reportSchema>;
export type CiteFix = CiteFixReport["fixes"][number];
export type CiteFixProof = NonNullable<CiteFixReport["proof"]>;
export type AgentOffer = CiteFixReport["agent"];
export type AgentMode = "quick" | "deep";

const agentResultSchema = z.object({
  url: z.string(),
  mode: z.enum(["quick", "deep"]),
  cached: z.boolean(),
  credits: z.number().int().nonnegative(),
  createdAt: z.string(),
  partial: z.boolean(),
  summary: z.string(),
  targetQuestion: z.string(),
  recommendations: z.array(
    z.object({ title: z.string(), detail: z.string(), check: z.string() }),
  ),
  answerFirst: z.string(),
  faq: z.array(z.object({ question: z.string(), answer: z.string() })),
  titleTag: z.string(),
  metaDescription: z.string(),
  schemaJsonLd: z.string(),
  verify: z.array(z.string()),
  draft: z.string(),
});

export type AgentResult = z.infer<typeof agentResultSchema>;

export const citeFixResultsKey = (projectId: string) =>
  ["cite-fix", projectId, "agent-results"] as const;

export const citeFixKey = (projectId: string) =>
  ["cite-fix", projectId] as const;

const base = (projectId: string) =>
  `/api/v1/projects/${encodeURIComponent(projectId)}/cite-fix`;

/** Free: scores the latest completed site audit, no provider call. */
export function getCiteFixReport(projectId: string) {
  return apiRequest(`${base(projectId)}/report`, reportSchema, "POST", {});
}

/** Free: the agent's saved results, newest per page and mode. */
export function getCiteFixAgentResults(projectId: string) {
  return apiRequest(
    `${base(projectId)}/agent/results`,
    z.object({ results: z.array(agentResultSchema) }),
    "POST",
    {},
  );
}

/**
 * Paid: one page. Each click sends a new idempotency key; a failed request is
 * never retried, because the provider may already have billed it.
 */
export function runCiteFixAgent(
  projectId: string,
  input: { url: string; mode: AgentMode; idempotencyKey: string },
) {
  return apiRequest(
    `${base(projectId)}/agent/run`,
    agentResultSchema,
    "POST",
    input,
  );
}
