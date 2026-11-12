import { firstBy, sortBy } from "remeda";
import type { TrackingAnswer } from "./trackingApi";
import { engineLabel } from "./platformLabels";

/**
 * Everything on the AI visibility Overview and Prompts tabs is read from
 * answers the tracker already saved: no provider call, no credits.
 *
 * Only completed answers with text count as "asked"; a failed or empty
 * answer is not a measured absence. The first tracked brand is the
 * customer's own (the server keeps it first).
 */
const measured = (a: TrackingAnswer) =>
  a.status === "completed" && a.answer.trim() !== "";

type BrandShare = {
  name: string;
  domain: string;
  own: boolean;
  /** Answers that name the brand. */
  mentions: number;
  /** Share of all tracked-brand mentions, 0–100. */
  share: number;
};

/** How often AI names each tracked brand, own brand first when tied. */
export function shareOfVoice(answers: TrackingAnswer[]): BrandShare[] {
  const rows = new Map<string, BrandShare>();
  for (const a of answers) {
    if (!measured(a)) continue;
    a.brands.forEach((b, i) => {
      const row = rows.get(b.domain) ?? {
        name: b.name,
        domain: b.domain,
        own: i === 0,
        mentions: 0,
        share: 0,
      };
      row.mentions += Number(b.mentioned);
      rows.set(b.domain, row);
    });
  }
  const list = [...rows.values()];
  const total = list.reduce((sum, r) => sum + r.mentions, 0);
  for (const r of list) {
    r.share = total ? Math.round((100 * r.mentions) / total) : 0;
  }
  return sortBy(
    list,
    [(r) => r.mentions, "desc"],
    [(r) => Number(r.own), "desc"],
  );
}

type EngineRow = {
  engine: string;
  label: string;
  answered: number;
  mentioned: number;
  cited: number;
};

/** Per engine: answers measured, and how many name or link the customer. */
export function engineBreakdown(answers: TrackingAnswer[]): EngineRow[] {
  const rows = new Map<string, EngineRow>();
  for (const a of answers) {
    if (!measured(a)) continue;
    const row = rows.get(a.engine) ?? {
      engine: a.engine,
      label: engineLabel(a.engine),
      answered: 0,
      mentioned: 0,
      cited: 0,
    };
    row.answered++;
    row.mentioned += Number(a.brands[0]?.mentioned ?? false);
    row.cited += Number(a.brands[0]?.cited ?? false);
    rows.set(a.engine, row);
  }
  return sortBy([...rows.values()], (r) => r.label);
}

export type CellState = "named" | "cited" | "absent" | "failed" | "pending";

export type PromptRow = {
  prompt: string;
  topic: string;
  cells: Record<string, { state: CellState; answer: TrackingAnswer }>;
  /** Brands AI named instead of the customer, most often first. */
  namedInstead: string[];
  /** Measured answers that never name the customer. */
  gap: boolean;
};

function cellState(a: TrackingAnswer): CellState {
  if (a.status === "failed") return "failed";
  if (a.status !== "completed") return "pending";
  if (!a.answer.trim()) return "failed";
  const own = a.brands[0];
  if (own?.mentioned) return "named";
  if (own?.cited) return "cited";
  return "absent";
}

/** One row per tracked question, one cell per engine. */
export function promptMatrix(answers: TrackingAnswer[]): PromptRow[] {
  const rows = new Map<
    string,
    PromptRow & { rivalCounts: Map<string, number> }
  >();
  for (const a of answers) {
    const row = rows.get(a.prompt) ?? {
      prompt: a.prompt,
      topic: a.topic,
      cells: {},
      namedInstead: [],
      gap: false,
      rivalCounts: new Map<string, number>(),
    };
    row.cells[a.engine] = { state: cellState(a), answer: a };
    if (measured(a) && !a.brands[0]?.mentioned) {
      const tracked = new Set(a.brands.map((b) => b.name.toLowerCase()));
      const rivals = [
        ...a.brands
          .slice(1)
          .filter((b) => b.mentioned)
          .map((b) => b.name),
        ...a.recommended
          .map((r) => r.name)
          .filter((n) => !tracked.has(n.toLowerCase())),
      ];
      for (const name of rivals) {
        row.rivalCounts.set(name, (row.rivalCounts.get(name) ?? 0) + 1);
      }
    }
    rows.set(a.prompt, row);
  }
  return [...rows.values()].map(({ rivalCounts, ...row }) => {
    const states = Object.values(row.cells).map((c) => c.state);
    return {
      ...row,
      namedInstead: sortBy([...rivalCounts.entries()], [([, n]) => n, "desc"])
        .map(([name]) => name)
        .slice(0, 3),
      gap:
        states.some((s) => s === "absent" || s === "cited") &&
        !states.includes("named"),
    };
  });
}

type Verdict = {
  /** The one-line answer to "how is AI treating us". */
  headline: string;
  /** The engine with the lowest mention rate, when one stands out. */
  weakest: EngineRow | null;
  gaps: number;
};

const rate = (r: EngineRow) => (r.answered ? r.mentioned / r.answered : 0);

/**
 * The sentence at the top of the Overview. Built only from saved answers, so
 * it never claims more than the sample shows.
 */
export function visibilityVerdict(answers: TrackingAnswer[]): Verdict | null {
  const engines = engineBreakdown(answers);
  const answered = engines.reduce((s, e) => s + e.answered, 0);
  if (!answered) return null;
  const mentioned = engines.reduce((s, e) => s + e.mentioned, 0);
  const shares = shareOfVoice(answers);
  const own = shares.find((s) => s.own);
  const leader = shares.find((s) => !s.own && s.mentions > 0);
  const weakest = engines.length > 1 ? (firstBy(engines, rate) ?? null) : null;
  const gaps = promptMatrix(answers).filter((r) => r.gap).length;
  const ownName = own?.name ?? "You";

  let headline = `${ownName} is named in ${mentioned} of ${answered} AI answers.`;
  if (leader && own && leader.mentions > own.mentions) {
    const times = own.mentions ? leader.mentions / own.mentions : 0;
    headline =
      times >= 1.5
        ? `AI names ${leader.name} ${times.toFixed(1)}× as often as ${ownName}.`
        : own.mentions === 0
          ? `AI names ${leader.name} in ${leader.mentions} answer${leader.mentions === 1 ? "" : "s"} and ${ownName} in none.`
          : `AI names ${leader.name} more often than ${ownName} (${leader.mentions} vs ${own.mentions} answers).`;
  } else if (leader && own && own.mentions >= leader.mentions) {
    headline = `${ownName} is named more often than any tracked competitor: ${own.mentions} of ${answered} answers.`;
  }
  return {
    headline,
    weakest: weakest && rate(weakest) < mentioned / answered ? weakest : null,
    gaps,
  };
}

type CitedSource = {
  domain: string;
  /** Answers that cite this domain. */
  answers: number;
  /** True when it is the customer's own site. */
  own: boolean;
};

/** Domains AI cites across the answers, most cited first. */
export function citedSources(
  answers: TrackingAnswer[],
  ownDomain: string,
): CitedSource[] {
  const own = ownDomain.toLowerCase().replace(/^www\./, "");
  const counts = new Map<string, number>();
  for (const a of answers) {
    if (!measured(a)) continue;
    const seen = new Set<string>();
    for (const c of a.citations) {
      const domain = (c.domain ?? safeHost(c.url)).replace(/^www\./, "");
      if (!domain || seen.has(domain)) continue;
      seen.add(domain);
      counts.set(domain, (counts.get(domain) ?? 0) + 1);
    }
  }
  return sortBy(
    [...counts.entries()].map(([domain, n]) => ({
      domain,
      answers: n,
      own: domain === own || domain.endsWith(`.${own}`),
    })),
    [(x) => x.answers, "desc"],
    (x) => x.domain,
  );
}

function safeHost(url: string): string {
  try {
    return new URL(url).hostname.toLowerCase();
  } catch {
    return "";
  }
}
