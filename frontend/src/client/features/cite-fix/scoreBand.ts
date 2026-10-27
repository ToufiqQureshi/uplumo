/**
 * The words beside a Cite Fix score. They describe how easy the pages are for
 * an assistant to quote, never a promise that it will.
 */
export function scoreBand(score: number): {
  label: string;
  tone: "destructive" | "warning" | "success";
} {
  if (score >= 80) return { label: "Easy for AI to quote", tone: "success" };
  if (score >= 50) return { label: "Partly quotable", tone: "warning" };
  return { label: "Hard for AI to quote", tone: "destructive" };
}
