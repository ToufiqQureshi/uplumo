import { sortBy } from "remeda";
import type { TrackingAnswer } from "@/client/features/ai-search/trackingApi";

type Run = {
  id: string;
  status: string;
  answered: number;
  mentioned: number;
  cited: number;
};

/**
 * The newest check with answers, and the one before it. A partial check (one
 * engine failed) still holds real answers, so it counts.
 */
export function latestChecks<T extends Run>(runs: T[]): [T?, T?] {
  const usable = runs.filter(
    (run) =>
      (run.status === "completed" || run.status === "partial") &&
      run.answered > 0,
  );
  return [usable[0], usable[1]];
}

/** The domains AI cited instead of you, most cited first. */
export function citedInstead(answers: TrackingAnswer[], limit = 3) {
  const counts = new Map<string, number>();
  for (const answer of answers) {
    if (answer.status !== "completed") continue;
    const domains = new Set(
      answer.citations
        .filter((c) => !c.matchedBrand && c.domain)
        .map((c) => c.domain!.replace(/^www\./, "")),
    );
    for (const domain of domains)
      counts.set(domain, (counts.get(domain) ?? 0) + 1);
  }
  return sortBy(
    [...counts].map(([domain, answersCiting]) => ({ domain, answersCiting })),
    [(row) => row.answersCiting, "desc"],
    (row) => row.domain,
  ).slice(0, limit);
}

/** What to do next, and where in Uplumo to do it. */
export function aiNextStep(mentionRate: number, citationRate: number) {
  if (mentionRate === 0)
    return {
      text: "AI did not name you in any answer. Write a page that answers these questions directly.",
      label: "Write the page",
      to: "content-editor",
    } as const;
  if (citationRate === 0)
    return {
      text: "AI names you but links to other sites. Add a clear answer and FAQ on your own page so AI can cite it.",
      label: "Improve a page",
      to: "content-editor",
    } as const;
  return {
    text: "AI names and cites you. Keep these pages fresh and track more questions.",
    label: "Track more questions",
    to: "prompt-tracking",
  } as const;
}

/** "↑ from 20%" style change against the previous check, or "". */
export function shareChange(now: number, before: number | undefined) {
  if (
    before === undefined ||
    Math.round(now * 100) === Math.round(before * 100)
  )
    return "";
  return `${now > before ? "↑" : "↓"} from ${Math.round(before * 100)}%`;
}
