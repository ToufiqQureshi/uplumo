export type PromptIdea = { intent: string; question: string };

const INTENTS: Record<string, string> = {
  best: "Best of",
  compare: "Compare",
  price: "Price",
  local: "Local",
  howto: "How-to",
};

/**
 * Reads the writer's "intent|question" lines. Lines in any other shape are
 * dropped, duplicates are removed, and each question is cut to the tracker's
 * 300-character limit.
 */
export function parsePromptIdeas(text: string): PromptIdea[] {
  const seen = new Set<string>();
  const out: PromptIdea[] = [];
  for (const line of text.split("\n")) {
    const [rawIntent, ...rest] = line.replace(/^[-*\d.\s]+/, "").split("|");
    const question = rest.join("|").trim().slice(0, 300);
    const intent = INTENTS[rawIntent?.trim().toLowerCase() ?? ""];
    if (!intent || question.length < 8) continue;
    const key = question.toLowerCase();
    if (seen.has(key)) continue;
    seen.add(key);
    out.push({ intent, question });
  }
  return out;
}

/** Adds questions to the tracker's list without duplicates, up to its limit. */
export function addPrompts(
  existing: { text: string; topic: string; paused: boolean }[],
  questions: string[],
  limit: number,
) {
  const have = new Set(existing.map((p) => p.text.trim().toLowerCase()));
  const added = questions
    .filter((q) => !have.has(q.trim().toLowerCase()))
    .slice(0, Math.max(0, limit - existing.length))
    .map((text) => ({ text, topic: "Prompt ideas", paused: false }));
  return { prompts: [...existing, ...added], added: added.length };
}
