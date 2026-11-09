import type { KeywordIntent, KeywordResearchRow } from "@/types/keywords";

/** The same spelling rules the server uses to compare keywords. */
function normalizeKeyword(value: string): string {
  return value.trim().toLowerCase().replace(/\s+/g, " ");
}

/** The researched row of the keyword the customer typed, if the data has one. */
export function findSeedRow(
  rows: readonly KeywordResearchRow[],
  seed: string,
): KeywordResearchRow | null {
  const target = normalizeKeyword(seed);
  return rows.find((row) => normalizeKeyword(row.keyword) === target) ?? null;
}

export type IdeaFilters = {
  /** Words every shown keyword must contain. */
  text: string;
  /** Intents to keep; empty keeps all. */
  intents: readonly KeywordIntent[];
  maxDifficulty: number | null;
  minVolume: number | null;
};

export const NO_FILTERS: IdeaFilters = {
  text: "",
  intents: [],
  maxDifficulty: null,
  minVolume: null,
};

export function hasFilters(filters: IdeaFilters): boolean {
  return (
    filters.text.trim() !== "" ||
    filters.intents.length > 0 ||
    filters.maxDifficulty !== null ||
    filters.minVolume !== null
  );
}

/**
 * The ideas that pass every filter. A keyword without a number is hidden by
 * a filter on that number: the customer asked for keywords known to pass.
 */
export function filterIdeas(
  rows: readonly KeywordResearchRow[],
  filters: IdeaFilters,
): KeywordResearchRow[] {
  const words = normalizeKeyword(filters.text).split(" ").filter(Boolean);
  return rows.filter((row) => {
    const keyword = normalizeKeyword(row.keyword);
    if (!words.every((word) => keyword.includes(word))) return false;
    if (filters.intents.length > 0 && !filters.intents.includes(row.intent))
      return false;
    if (
      filters.maxDifficulty !== null &&
      (row.keywordDifficulty === null ||
        row.keywordDifficulty > filters.maxDifficulty)
    )
      return false;
    if (
      filters.minVolume !== null &&
      (row.searchVolume === null || row.searchVolume < filters.minVolume)
    )
      return false;
    return true;
  });
}

// Names that plain title case gets wrong. Every other result type DataForSEO
// returns is still shown, in title case.
const SERP_FEATURE_NAMES: Record<string, string> = {
  ai_overview: "AI Overview",
  local_pack: "Map pack",
  people_also_ask: "People also ask",
  people_also_search: "People also search",
  featured_snippet: "Featured snippet",
  knowledge_graph: "Knowledge panel",
  short_videos: "Short videos",
  paid: "Ads",
};

/** A DataForSEO result type (`people_also_ask`) as people read it. */
export function serpFeatureName(type: string): string {
  const known = SERP_FEATURE_NAMES[type];
  if (known) return known;
  const words = type.replace(/_/g, " ").trim();
  return words.charAt(0).toUpperCase() + words.slice(1);
}

const compact = new Intl.NumberFormat("en", {
  notation: "compact",
  maximumFractionDigits: 1,
});
const whole = new Intl.NumberFormat("en");

export function formatCount(value: number | null | undefined): string {
  if (value == null) return "—";
  return value >= 10_000 ? compact.format(value) : whole.format(value);
}

export function formatUsd(value: number | null | undefined): string {
  return value == null ? "—" : `$${value.toFixed(2)}`;
}

/** A percent change with its sign, or a dash when unknown. */
export function formatChange(value: number | null | undefined): string {
  if (value == null) return "—";
  return `${value > 0 ? "+" : ""}${value}%`;
}

const monthName = new Intl.DateTimeFormat("en", {
  month: "short",
  year: "numeric",
  timeZone: "UTC",
});

/** The trend oldest first, as DataForSEO lists it newest first. */
export function trendOldestFirst(
  row: Pick<KeywordResearchRow, "trend">,
): Array<{ label: string; volume: number }> {
  const months = [...row.trend];
  months.sort((a, b) => a.year - b.year || a.month - b.month);
  return months.map((m) => ({
    label: monthName.format(Date.UTC(m.year, m.month - 1, 1)),
    volume: m.searchVolume,
  }));
}

/** When the provider last refreshed this keyword's numbers, as a month. */
export function formatUpdated(value: string | null | undefined): string {
  if (!value) return "—";
  const date = new Date(value.replace(" +00:00", "Z").replace(" ", "T"));
  return Number.isNaN(date.getTime()) ? "—" : monthName.format(date);
}
