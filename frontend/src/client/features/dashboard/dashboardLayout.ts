import type { DashboardLayoutCard } from "@/client/features/dashboard/dashboardApi";

/**
 * Every card the dashboard can show, in default order. AI visibility leads:
 * whether AI cites the site, and who it cites instead, is Uplumo's core
 * question (FOUNDER.md).
 */
export const DASHBOARD_CARDS = [
  { id: "ai_visibility", label: "AI visibility" },
  { id: "gsc", label: "Search performance (Google)" },
  { id: "analytics", label: "Visitors and AI traffic" },
  { id: "rank_tracking", label: "Rank tracking" },
  { id: "audit", label: "Site audit" },
  { id: "backlinks", label: "Backlink pulse" },
  { id: "ga4", label: "Google Analytics" },
] as const;

export type DashboardCardId = (typeof DASHBOARD_CARDS)[number]["id"];
export type ResolvedCard = { id: DashboardCardId; hidden: boolean };

function isCardId(id: string): id is DashboardCardId {
  return DASHBOARD_CARDS.some((card) => card.id === id);
}

/**
 * The layout to show: the saved order and visibility first, then any card
 * the member has never seen (added after they saved) in default order and
 * visible. Ids the client no longer knows are dropped.
 */
export function resolveLayout(
  saved: DashboardLayoutCard[] | null,
): ResolvedCard[] {
  const out: ResolvedCard[] = [];
  const seen = new Set<string>();
  for (const card of saved ?? []) {
    const id = card.id;
    if (!isCardId(id) || seen.has(id)) continue;
    seen.add(id);
    out.push({ id, hidden: card.hidden });
  }
  for (const card of DASHBOARD_CARDS) {
    if (!seen.has(card.id)) out.push({ id: card.id, hidden: false });
  }
  return out;
}

/** Moves the card at `index` by `step` places, staying inside the list. */
export function moveCard(
  cards: ResolvedCard[],
  index: number,
  step: -1 | 1,
): ResolvedCard[] {
  const target = index + step;
  if (
    index < 0 ||
    index >= cards.length ||
    target < 0 ||
    target >= cards.length
  )
    return cards;
  const next = [...cards];
  [next[index], next[target]] = [next[target], next[index]];
  return next;
}

export function cardLabel(id: DashboardCardId): string {
  return DASHBOARD_CARDS.find((card) => card.id === id)?.label ?? id;
}
