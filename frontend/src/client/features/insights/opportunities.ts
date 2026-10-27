import { sortBy } from "remeda";
import type { TrackingAnswer } from "@/client/features/ai-search/trackingApi";

const ENGINE: Record<string, string> = {
  chat_gpt: "ChatGPT",
  gemini: "Gemini",
  google_ai_overview: "AI Overviews",
  claude: "Claude",
  perplexity: "Perplexity",
};

type PromptGap = {
  prompt: string;
  /** Businesses AI named instead, most often first. */
  rivals: string[];
  engines: string[];
  /** Domains AI cited in those answers. */
  sources: string[];
  excerpt: string;
};

const bare = (d: string) => d.toLowerCase().replace(/^www\./, "");

/**
 * Questions where at least one completed answer neither names nor links the
 * customer but names another business. Sorted by how many answers left them
 * out. Pure: it reads saved answers only.
 */
export function promptGaps(
  answers: TrackingAnswer[],
  ownDomain: string,
): PromptGap[] {
  const own = bare(ownDomain);
  const byPrompt = new Map<
    string,
    {
      misses: number;
      rivals: Map<string, number>;
      engines: Set<string>;
      sources: Set<string>;
      excerpt: string;
    }
  >();
  for (const a of answers) {
    if (a.status !== "completed" || !a.answer.trim()) continue;
    const you = a.brands[0];
    const linked = a.citations.some(
      (c) =>
        c.matchedBrand ||
        (own !== "" &&
          c.domain !== null &&
          (bare(c.domain) === own || bare(c.domain).endsWith(`.${own}`))),
    );
    if (you?.mentioned || linked || a.recommended.length === 0) continue;
    const row = byPrompt.get(a.prompt) ?? {
      misses: 0,
      rivals: new Map<string, number>(),
      engines: new Set<string>(),
      sources: new Set<string>(),
      excerpt: "",
    };
    row.misses++;
    row.engines.add(ENGINE[a.engine] ?? a.engine);
    for (const r of a.recommended)
      row.rivals.set(r.name, (row.rivals.get(r.name) ?? 0) + 1);
    for (const c of a.citations) if (c.domain) row.sources.add(bare(c.domain));
    if (!row.excerpt) {
      const first = a.recommended[0]?.name ?? "";
      const at = Math.max(0, a.answer.indexOf(first) - 80);
      row.excerpt = a.answer.slice(at, at + 320).trim();
    }
    byPrompt.set(a.prompt, row);
  }
  return sortBy([...byPrompt], [([, r]) => r.misses, "desc"]).map(
    ([prompt, r]) => ({
      prompt,
      rivals: sortBy([...r.rivals], [([, c]) => c, "desc"])
        .slice(0, 3)
        .map(([n]) => n),
      engines: [...r.engines],
      sources: [...r.sources].slice(0, 5),
      excerpt: r.excerpt,
    }),
  );
}
