import type { TrackingAnswer } from "@/client/features/ai-search/trackingApi";

export type Mood = "positive" | "neutral" | "negative";
export const MOODS: Mood[] = ["positive", "neutral", "negative"];

/**
 * Counts of classified answers that talk about the brand. Unclassified,
 * failed and "not mentioned" answers are left out, so the shares describe
 * only what AI actually said about the brand.
 */
export function sentimentSummary(answers: TrackingAnswer[]) {
  const counts: Record<Mood, number> = { positive: 0, neutral: 0, negative: 0 };
  const byEngine = new Map<string, Record<Mood, number>>();
  let pending = 0;
  for (const a of answers) {
    if (a.status !== "completed" || !a.answer.trim()) continue;
    if (a.sentiment === "") {
      pending++;
      continue;
    }
    if (a.sentiment === "not_mentioned") continue;
    counts[a.sentiment]++;
    const e = byEngine.get(a.engine) ?? {
      positive: 0,
      neutral: 0,
      negative: 0,
    };
    e[a.sentiment]++;
    byEngine.set(a.engine, e);
  }
  const total = counts.positive + counts.neutral + counts.negative;
  const share = (m: Mood) =>
    total ? Math.round((100 * counts[m]) / total) : 0;
  return { counts, total, pending, share, byEngine };
}

/** The most common words in negative quotes, as a hint of the complaint. */
export function topComplaint(answers: TrackingAnswer[]) {
  const quotes = answers
    .filter((a) => a.sentiment === "negative" && a.sentimentQuote)
    .map((a) => a.sentimentQuote);
  return quotes[0] ?? "";
}
