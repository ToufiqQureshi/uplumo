import type { CsvValue } from "@/client/lib/csv";
import { formatLocationLabel } from "@/shared/keyword-locations";

// Column headers and CSV formatting shared by the saved-keywords export.
/** Local exports name the area the volume, CPC, and competition cover. */
function keywordResearchHeaders(locationName?: string) {
  const scope = locationName ? ` (${formatLocationLabel(locationName)})` : "";
  return [
    "Keyword",
    `Volume${scope}`,
    `CPC${scope}`,
    `Competition${scope}`,
    "Score",
    "Intent",
    "Top bid low (USD)",
    "Top bid high (USD)",
    "Ref. domains",
    "3-mo trend (%)",
    "Google shows",
    "Monthly searches",
    "Backlinks",
    "Dofollow backlinks",
    "Referring pages",
    "1-mo trend (%)",
    "12-mo trend (%)",
    "Competition level",
    "Last updated",
    "Detected language",
    "Words count",
    "Secondary intents",
  ];
}

export const KEYWORD_RESEARCH_HEADERS = keywordResearchHeaders();

/**
 * CSV downloads show CPC and competition (columns 3 and 4 of the keyword
 * headers) with two decimals. Sheets keeps raw numbers.
 */
export function formatCentsForCsv(rows: CsvValue[][]): CsvValue[][] {
  return rows.map((row) =>
    row.map((cell, idx) =>
      (idx === 2 || idx === 3) && typeof cell === "number"
        ? cell.toFixed(2)
        : cell,
    ),
  );
}
