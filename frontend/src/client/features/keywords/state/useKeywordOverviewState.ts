import { useMemo } from "react";
import type { KeywordResearchRow } from "@/types/keywords";

export function useKeywordOverviewState({
  rows,
  searchedKeyword,
  selectedKeyword,
  hasSearched,
  isLoading,
  lastSearchError,
}: {
  rows: KeywordResearchRow[];
  searchedKeyword: string;
  selectedKeyword: KeywordResearchRow | null;
  hasSearched: boolean;
  isLoading: boolean;
  lastSearchError: boolean;
}) {
  const hasExactMatchInResults = useMemo(() => {
    const normalizedSeed = searchedKeyword.trim().toLowerCase();
    if (!normalizedSeed || rows.length === 0) return false;
    return rows.some(
      (row) => row.keyword.trim().toLowerCase() === normalizedSeed,
    );
  }, [rows, searchedKeyword]);

  const showApproximateMatchNotice =
    hasSearched &&
    !isLoading &&
    !lastSearchError &&
    rows.length > 0 &&
    searchedKeyword.trim() !== "" &&
    !hasExactMatchInResults;

  const overviewKeyword: KeywordResearchRow | null = useMemo(() => {
    if (selectedKeyword) return selectedKeyword;
    if (searchedKeyword && rows.length > 0) {
      const seed = rows.find(
        (row) =>
          row.keyword.trim().toLowerCase() ===
          searchedKeyword.trim().toLowerCase(),
      );
      if (seed) return seed;
    }
    return rows.length > 0 ? rows[0] : null;
  }, [selectedKeyword, searchedKeyword, rows]);

  return { showApproximateMatchNotice, overviewKeyword };
}
