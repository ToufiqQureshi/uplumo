import { sortBy } from "remeda";
import type { WriterAction } from "@/client/features/content-editor/contentEditorApi";
import type { TrackingAnswer } from "@/client/features/ai-search/trackingApi";

/**
 * Next Actions: one ranked to-do list built only from data Uplumo already
 * stored (latest site audit, Search Console rows, latest AI tracker check).
 * Nothing here calls a paid provider. Every action names its source and the
 * numbers behind it, and an estimate is labelled as one.
 */

export type ActionType = "Technical" | "Content" | "AI";
type Impact = "High" | "Medium" | "Low";

type NextAction = {
  /** Stable id, matched by the saved done/ignored decisions. */
  key: string;
  title: string;
  type: ActionType;
  impact: Impact;
  effort: string;
  /** Estimated extra clicks a month; null when there is no basis for one. */
  gainClicks: number | null;
  source: string;
  why: string;
  ai?: {
    action: WriterAction;
    label: string;
    brief: string;
    content?: string;
    targetKeyword?: string;
  };
};

type AuditIssue = {
  issueType: string;
  pageUrl: string;
  severity: string;
};

type QueryRow = {
  key: string;
  clicks: number;
  impressions: number;
  ctr: number;
  position: number;
};

// Share of clicks a result at each organic position typically gets. Used only
// to size an estimate, and always shown as "≈".
const CTR_BY_POSITION = [
  0.28, 0.15, 0.1, 0.07, 0.05, 0.04, 0.03, 0.025, 0.02, 0.018,
];
function expectedCtr(position: number) {
  const i = Math.max(1, Math.round(position)) - 1;
  return i < CTR_BY_POSITION.length ? CTR_BY_POSITION[i] : 0.01;
}

/** A search on page 2 with enough impressions to be worth pushing to #5. */
function strikingQueries(rows: QueryRow[]) {
  const gains = rows
    .filter((r) => r.position >= 8 && r.position <= 20 && r.impressions >= 50)
    .map((r) => ({
      ...r,
      gain: Math.max(0, Math.round(r.impressions * (expectedCtr(5) - r.ctr))),
    }))
    .filter((r) => r.gain > 0);
  return sortBy(gains, [(r) => r.gain, "desc"]);
}

type IssueRule = {
  title: (pages: number) => string;
  type: ActionType;
  impact: Impact;
  effort: string;
  why: string;
  ai?: { action: WriterAction; label: string };
};

// Only issues with a clear payoff become actions; the full list stays in
// Site Audit. Keys are the audit's issue types.
const ISSUE_RULES: Record<string, IssueRule> = {
  "missing-title": {
    title: (n) => `Add a title to ${n} page${n === 1 ? "" : "s"}`,
    type: "Technical",
    impact: "High",
    effort: "15 min",
    why: "Google shows the title as the blue link. Without one it makes one up, and fewer people click.",
    ai: { action: "title_meta", label: "Write titles" },
  },
  "title-too-long": {
    title: (n) =>
      `Shorten ${n} page title${n === 1 ? "" : "s"} Google cuts off`,
    type: "Technical",
    impact: "Medium",
    effort: "15 min",
    why: "Titles over about 60 characters end in “…” in Google, which hides the end of the promise.",
    ai: { action: "title_meta", label: "Rewrite titles" },
  },
  "missing-meta-description": {
    title: (n) => `Add a meta description to ${n} page${n === 1 ? "" : "s"}`,
    type: "Technical",
    impact: "Medium",
    effort: "15 min",
    why: "Without one, Google picks random text from the page for the snippet.",
    ai: { action: "title_meta", label: "Write descriptions" },
  },
  "missing-h1": {
    title: (n) => `Add a main heading (H1) to ${n} page${n === 1 ? "" : "s"}`,
    type: "Technical",
    impact: "Medium",
    effort: "15 min",
    why: "The H1 tells Google and AI assistants what the page is about.",
  },
  "noindex-page": {
    title: (n) =>
      `${n} page${n === 1 ? " is" : "s are"} hidden from Google (noindex)`,
    type: "Technical",
    impact: "High",
    effort: "15 min",
    why: "A noindex tag tells Google not to show the page at all. Remove it if the page should be found.",
  },
  "broken-page": {
    title: (n) => `Fix ${n} broken page${n === 1 ? "" : "s"}`,
    type: "Technical",
    impact: "High",
    effort: "30 min",
    why: "Visitors and crawlers land on an error. Fix the page or redirect it.",
  },
  "broken-internal-link": {
    title: (n) => `Fix ${n} broken internal link${n === 1 ? "" : "s"}`,
    type: "Technical",
    impact: "Low",
    effort: "15 min",
    why: "Links that lead nowhere waste visits and crawl time.",
  },
  "thin-content": {
    title: (n) => `Add substance to ${n} thin page${n === 1 ? "" : "s"}`,
    type: "Content",
    impact: "Medium",
    effort: "1–2 hr",
    why: "Pages with very little text rarely rank and AI assistants have nothing to quote.",
    ai: { action: "outline", label: "Outline what to add" },
  },
  "no-question-answers": {
    title: (n) => `Answer real questions on ${n} page${n === 1 ? "" : "s"}`,
    type: "AI",
    impact: "Medium",
    effort: "30 min",
    why: "AI assistants quote pages that answer a question in the first lines.",
    ai: { action: "faq", label: "Write an FAQ" },
  },
  "missing-business-schema": {
    title: () => "Add business details (schema) to your site",
    type: "AI",
    impact: "Medium",
    effort: "30 min",
    why: "Structured business details help Google and AI read your name, address and hours correctly.",
  },
};

const IMPACT_RANK: Record<Impact, number> = { High: 0, Medium: 1, Low: 2 };

/** Short, regex-safe id for free text such as a search query. */
function hashKey(text: string) {
  let h = 2166136261;
  for (let i = 0; i < text.length; i++) {
    h ^= text.charCodeAt(i);
    h = Math.imul(h, 16777619);
  }
  return (h >>> 0).toString(16).padStart(8, "0");
}

export function buildNextActions(input: {
  issues: AuditIssue[];
  queries: QueryRow[];
  answers: TrackingAnswer[];
  brand?: string;
}): NextAction[] {
  const out: NextAction[] = [];

  for (const q of strikingQueries(input.queries).slice(0, 5)) {
    out.push({
      key: `gsc:q:${hashKey(q.key)}`,
      title: `Push “${q.key}” from #${Math.round(q.position)} to page 1`,
      type: "Content",
      impact: q.gain >= 50 ? "High" : "Medium",
      effort: "30 min",
      gainClicks: q.gain,
      source: "Search Console · 28 days",
      why: `${q.impressions.toLocaleString("en-IN")} impressions, ${(q.ctr * 100).toFixed(1)}% clicked. Around #5 a result typically gets ${Math.round(expectedCtr(5) * 100)}% of clicks.`,
      ai: {
        action: "faq",
        label: "Write an FAQ for this search",
        targetKeyword: q.key,
        brief: `Search: ${q.key}. Position ${q.position.toFixed(1)}, ${q.impressions} impressions, CTR ${(q.ctr * 100).toFixed(1)}%.`,
        content: `Questions people searching "${q.key}" want answered.`,
      },
    });
  }

  const byType = new Map<string, Set<string>>();
  for (const issue of input.issues) {
    if (!ISSUE_RULES[issue.issueType]) continue;
    const pages = byType.get(issue.issueType) ?? new Set<string>();
    pages.add(issue.pageUrl);
    byType.set(issue.issueType, pages);
  }
  for (const [type, pages] of byType) {
    const rule = ISSUE_RULES[type];
    const urls = [...pages];
    out.push({
      key: `audit:${type}`,
      title: rule.title(urls.length),
      type: rule.type,
      impact: rule.impact,
      effort: rule.effort,
      gainClicks: null,
      source: "Site audit",
      why: `${rule.why} Pages: ${urls.slice(0, 3).join(", ")}${urls.length > 3 ? ` and ${urls.length - 3} more` : ""}.`,
      ai: rule.ai
        ? {
            ...rule.ai,
            brief: `Pages with this issue: ${urls.slice(0, 15).join(", ")}`,
            content: urls.slice(0, 15).join("\n"),
          }
        : undefined,
    });
  }

  const answered = input.answers.filter(
    (a) => a.status === "completed" && a.answer.trim(),
  );
  const missed = answered.filter((a) => a.brands[0] && !a.brands[0].mentioned);
  if (missed.length) {
    const rivals = new Map<string, number>();
    for (const a of missed) {
      for (const r of a.recommended) {
        rivals.set(r.name, (rivals.get(r.name) ?? 0) + 1);
      }
    }
    const top = sortBy([...rivals], [([, c]) => c, "desc"])[0];
    out.push({
      key: "ai:not_named",
      title: `Get named by AI in ${missed.length} of ${answered.length} answers that skip you`,
      type: "AI",
      impact: missed.length * 2 >= answered.length ? "High" : "Medium",
      effort: "1–2 hr",
      gainClicks: null,
      source: "AI Visibility tracker · latest check",
      why: top
        ? `${top[0]} is named in ${top[1]} of the answers that leave you out.`
        : "These AI answers name other businesses or none.",
      ai: {
        action: "action_plan",
        label: "Make a plan to get named",
        brief: `Brand: ${input.brand ?? ""}. Missing in ${missed.length} of ${answered.length} AI answers. Rivals named instead: ${[
          ...rivals,
        ]
          .slice(0, 8)
          .map(([n, c]) => `${n} (${c})`)
          .join(", ")}. Prompts: ${missed
          .slice(0, 10)
          .map((a) => a.prompt)
          .join(" | ")}`,
      },
    });
  }

  // Impact first; inside one level, the bigger measured click gain first.
  return sortBy(out, (a) => IMPACT_RANK[a.impact], [
    (a) => a.gainClicks ?? -1,
    "desc",
  ]);
}
