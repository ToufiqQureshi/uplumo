// @vitest-environment happy-dom
import { act, createElement } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { KeywordResearchRow } from "@/types/keywords";
import { useKeywordOverviewState } from "./useKeywordOverviewState";
import { useSaveAndExportActions } from "./keywordControllerActions";
import { exportRows } from "@/client/lib/exportRows";

vi.mock("@/client/lib/exportRows", () => ({ exportRows: vi.fn() }));
vi.mock("@/client/lib/posthog", () => ({ captureClientEvent: vi.fn() }));

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

let root: Root;
let host: HTMLDivElement;
beforeEach(() => {
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(() => {
  act(() => root.unmount());
  host.remove();
  vi.clearAllMocks();
});

const row: KeywordResearchRow = {
  keyword: "hotel in lonavala maharashtra",
  searchVolume: 18100,
  cpc: 0.26,
  competition: 0.98,
  keywordDifficulty: 0,
  intent: "commercial",
  topOfPageBidLow: 0.05,
  topOfPageBidHigh: 0.32,
  referringDomains: 5,
  quarterlyChange: -33,
  trend: [{ year: 2026, month: 9, searchVolume: 18100 }],
};

describe("keyword summary delivery", () => {
  it("labels an approximate result even in Auto mode", () => {
    let result: ReturnType<typeof useKeywordOverviewState> | undefined;
    function Summary() {
      result = useKeywordOverviewState({
        rows: [row],
        searchedKeyword: "hotel in lonavala",
        selectedKeyword: null,
        hasSearched: true,
        isLoading: false,
        lastSearchError: false,
      });
      return null;
    }
    act(() => root.render(createElement(Summary)));
    expect(result?.showApproximateMatchNotice).toBe(true);
    expect(result?.overviewKeyword?.keyword).toBe(row.keyword);
  });

  it("uses the exact normalized seed instead of the first suggestion", () => {
    let result: ReturnType<typeof useKeywordOverviewState> | undefined;
    function Summary() {
      result = useKeywordOverviewState({
        rows: [
          row,
          { ...row, keyword: "hotel in lonavala", searchVolume: 9000 },
        ],
        searchedKeyword: " Hotel In Lonavala ",
        selectedKeyword: null,
        hasSearched: true,
        isLoading: false,
        lastSearchError: false,
      });
      return null;
    }
    act(() => root.render(createElement(Summary)));
    expect(result?.showApproximateMatchNotice).toBe(false);
    expect(result?.overviewKeyword?.searchVolume).toBe(9000);
  });
});

describe("keyword export delivery", () => {
  it.each(["csv", "sheets"] as const)(
    "keeps every table metric and monthly trend in %s",
    (format) => {
      let actions: ReturnType<typeof useSaveAndExportActions> | undefined;
      function Export() {
        actions = useSaveAndExportActions({
          selectedRows: new Set(),
          filteredRows: [{ ...row, parentKeyword: null }],
          input: {
            projectId: "p1",
            keywordInput: "hotel in lonavala",
            locationCode: 2356,
            displayedLocationCode: 2356,
            locationName: undefined,
            resultLimit: 150,
            keywordMode: "auto",
            clickstream: false,
            groupKeywords: false,
            sortField: "searchVolume",
            sortDir: "desc",
            setPreferredLocationCode: vi.fn(),
            onFormSubmit: vi.fn(),
          },
          saveKeywordsMutate: vi.fn(),
          setShowSaveDialog: vi.fn(),
        });
        return null;
      }
      act(() => root.render(createElement(Export)));
      act(() => actions?.exportAll(format));
      expect(exportRows).toHaveBeenCalledWith(
        expect.objectContaining({
          headers: [
            "Keyword",
            "Volume",
            "CPC",
            "Competition",
            "Score",
            "Intent",
            "Top bid low (USD)",
            "Top bid high (USD)",
            "Ref. domains",
            "3-mo trend (%)",
            "Monthly searches",
          ],
          rows: [
            [
              row.keyword,
              18100,
              format === "csv" ? "0.26" : 0.26,
              format === "csv" ? "0.98" : 0.98,
              0,
              "commercial",
              0.05,
              0.32,
              5,
              -33,
              JSON.stringify(row.trend),
            ],
          ],
        }),
      );
    },
  );
});
