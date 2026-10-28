import { z } from "zod";

// The present_keywords tool's output (backend/internal/sam/keywords_mode.go).
// Metrics are joined server-side from the reply's research rows, so every
// number here is one DataForSEO returned; a keyword without research has null.
const metricsSchema = z.object({
  searchVolume: z.number().nullish(),
  keywordDifficulty: z.number().nullish(),
  cpc: z.number().nullish(),
  intent: z.string().nullish(),
  monthlyVolumes: z.array(z.number()).nullish(),
  latestMonth: z.string().nullish(),
  yearlyChange: z.number().nullish(),
  serpFeatures: z.array(z.string()).nullish(),
  topPagesDomainRank: z.number().nullish(),
});

const reportSchema = z.object({
  understood: z
    .array(z.object({ label: z.string(), value: z.string() }))
    .nullish(),
  summary: z.string().nullish(),
  keywords: z.array(
    z.object({
      keyword: z.string(),
      chance: z.enum(["high", "medium", "low"]),
      note: z.string().nullish(),
      metrics: metricsSchema.nullish(),
    }),
  ),
  quickWins: z
    .array(
      z.object({
        page: z.string(),
        keyword: z.string(),
        position: z.number().nullish(),
        target: z.string().nullish(),
        actions: z.array(z.string()),
      }),
    )
    .nullish(),
  contentIdeas: z
    .array(
      z.object({
        type: z.string(),
        title: z.string(),
        keywords: z.array(z.string()),
        outline: z.array(z.string()),
        why: z.string().nullish(),
      }),
    )
    .nullish(),
  followUps: z.array(z.string()).nullish(),
  budget: z
    .object({
      seedsUsed: z.number(),
      seedsMax: z.number(),
      serpUsed: z.number(),
      serpMax: z.number(),
    })
    .nullish(),
});

export type KeywordReport = z.infer<typeof reportSchema>;
export type ReportKeyword = KeywordReport["keywords"][number];

export const PRESENT_PART = "tool-present_keywords";

/** The report a finished present_keywords call carries, or null. */
export function reportFromPart(part: {
  type: string;
  output?: unknown;
}): KeywordReport | null {
  if (part.type !== PRESENT_PART) return null;
  const parsed = reportSchema.safeParse(part.output);
  return parsed.success ? parsed.data : null;
}

// Average click-through rate of position 3 on Google: a conservative share of
// searches a page there gets. Shown as an estimate, never a promise.
const TOP3_CTR = 0.1;

export function reportTotals(keywords: ReportKeyword[]) {
  let volume = 0;
  let kdSum = 0;
  let kdCount = 0;
  for (const k of keywords) {
    volume += k.metrics?.searchVolume ?? 0;
    if (k.metrics?.keywordDifficulty != null) {
      kdSum += k.metrics.keywordDifficulty;
      kdCount++;
    }
  }
  return {
    volume,
    averageDifficulty: kdCount ? Math.round(kdSum / kdCount) : null,
    top3Visits: Math.round(volume * TOP3_CTR),
  };
}

export function difficultyLabel(kd: number): string {
  if (kd < 30) return "easy";
  if (kd < 50) return "medium";
  return "hard";
}
