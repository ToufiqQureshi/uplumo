import type { SortDir, SortField } from "@/client/features/keywords/components";
import type {
  KeywordMode,
  ResultLimit,
} from "@/client/features/keywords/keywordResearchTypes";

type KeywordSearchParams = {
  q?: string;
  loc?: number;
  locName?: string;
  kLimit?: ResultLimit;
  mode?: KeywordMode;
  cs?: boolean;
  grp?: boolean;
  sort?: SortField;
  order?: SortDir;
  minVol?: string;
  maxVol?: string;
  minCpc?: string;
  maxCpc?: string;
  minKd?: string;
  maxKd?: string;
  include?: string;
  exclude?: string;
};

export function normalizeLegacyKeywordSearch(search: KeywordSearchParams): {
  normalized: KeywordSearchParams;
  changed: boolean;
} {
  const normalized: KeywordSearchParams = {
    ...search,
    q: search.q === "" ? undefined : search.q,
    loc: search.loc,
    locName: search.locName === "" ? undefined : search.locName,
    kLimit: search.kLimit === 150 ? undefined : search.kLimit,
    mode: search.mode === "auto" ? undefined : search.mode,
    cs: search.cs === true ? true : undefined,
    grp: search.grp === true ? true : undefined,
    sort: search.sort === "searchVolume" ? undefined : search.sort,
    order: search.order === "desc" ? undefined : search.order,
    minVol: undefined,
    maxVol: undefined,
    minCpc: undefined,
    maxCpc: undefined,
    minKd: undefined,
    maxKd: undefined,
    include: undefined,
    exclude: undefined,
  };

  const keys: Array<keyof KeywordSearchParams> = [
    "q",
    "loc",
    "locName",
    "kLimit",
    "mode",
    "cs",
    "grp",
    "sort",
    "order",
    "minVol",
    "maxVol",
    "minCpc",
    "maxCpc",
    "minKd",
    "maxKd",
    "include",
    "exclude",
  ];

  return {
    normalized,
    changed: keys.some((key) => search[key] !== normalized[key]),
  };
}
