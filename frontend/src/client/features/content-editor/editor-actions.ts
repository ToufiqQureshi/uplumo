import { sortBy } from "remeda";
import type { ScoreCheck } from "./scoring";
/** Only applicable unfinished checks can become a paid writing task. */
export function nextEdits(checks: ScoreCheck[]) {
  return sortBy(
    checks.filter((c) => c.weight > 0 && !c.passed && c.earned < c.weight),
    [(c) => c.weight - c.earned, "desc"],
    [(c) => c.id, "asc"],
  ).slice(0, 3);
}
export function taskInstructions(
  task: Pick<ScoreCheck, "label" | "detail"> | undefined,
  instructions: string,
) {
  if (!task) return instructions;
  return [
    instructions.slice(0, 1100),
    `Selected editorial issue: ${task.label.slice(0, 120)}. ${task.detail.slice(0, 550)}`,
    "Address this issue in the selected passage. Preserve the writer's meaning. Do not invent statistics, sources, testimonials or experience; mark facts that require verification.",
  ]
    .filter(Boolean)
    .join("\n")
    .slice(0, 2000);
}
