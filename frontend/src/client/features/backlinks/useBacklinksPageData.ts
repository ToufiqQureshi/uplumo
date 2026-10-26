import { useMemo } from "react";
import { useQuery, type QueryKey } from "@tanstack/react-query";
import type {
  BacklinksPageProps,
  BacklinksSearchState,
} from "./backlinksPageTypes";
import {
  getErrorCode,
  getStandardErrorMessage,
} from "@/client/lib/error-messages";
import {
  getBacklinksOverview,
  getBacklinksReferringDomains,
  getBacklinksRows,
  getBacklinksTopPages,
} from "./backlinksApi";
import {
  BACKLINKS_DEFAULT_SORT,
  backlinksRowsSortFieldSchema,
  referringDomainsSortFieldSchema,
  topPagesSortFieldSchema,
  type BacklinksSortOrder,
} from "@/types/schemas/backlinks";
import {
  toBacklinksFiltersPayload,
  backlinksFilterBudgetError,
  toReferringDomainsFiltersPayload,
  toTopPagesFiltersPayload,
} from "./backlinksFilterTypes";
import type { BacklinksFiltersState } from "./useBacklinksFilters";
import { toScopeSearchParam } from "@/shared/researchScope";
import { allowPurchase, purchaseAllowed } from "@/client/lib/paidIntent";

type UseBacklinksPageDataArgs = {
  projectId: string;
  searchState: BacklinksSearchState;
  filters: BacklinksFiltersState;
};

// Five-minute client staleness on top of the server's 6h R2 cache, so window
// refocus doesn't re-run the server functions for bytes that can't change.
const BACKLINKS_QUERY_STALE_TIME_MS = 5 * 60 * 1000;

function getBacklinksErrorMessage(
  error: unknown,
  fallback: string,
): string | null {
  if (!error) return null;
  if (getErrorCode(error) === "VALIDATION_ERROR") {
    return "Enter a valid domain or page URL.";
  }

  return getStandardErrorMessage(error, fallback);
}

/**
 * Maps the URL's sort/order params to a request's sortField/sortOrder pair.
 * The sort param is checked against the tab's allowed sort fields; anything
 * unexpected falls back to the tab's default sort.
 */
function toSort<T extends string>(
  sortParam: string | undefined,
  orderParam: BacklinksSortOrder | undefined,
  allowedFields: readonly T[],
  fallback: { field: T; order: BacklinksSortOrder },
): { field: T; order: BacklinksSortOrder } {
  const field = sortParam
    ? allowedFields.find((candidate) => candidate === sortParam)
    : undefined;
  if (!field) return fallback;
  return { field, order: orderParam ?? "desc" };
}

export function useBacklinksPageData({
  projectId,
  searchState,
  filters,
}: UseBacklinksPageDataArgs) {
  const searchCardInitialValues = useMemo(
    () => ({
      target: searchState.target,
      scope: searchState.scope,
    }),
    [searchState.scope, searchState.target],
  );

  const { target, scope, tab, page, pageSize, sort, order, view } = searchState;
  const rowsMode = view === "all" ? "as_is" : "one_per_domain";
  const targetReady = Boolean(target);
  const baseQueryKeyParts = [projectId, scope, target] as const;
  // Without a Search or Refresh click in this session, ask for cached data
  // only: an expired report then shows Refresh instead of buying unasked.
  // Read at request time, so a Refresh click applies to the refetch it starts.
  const intentKey = backlinksIntentKey(projectId, target, scope);
  const cachedOnly = () => !purchaseAllowed(intentKey);
  const pageInputBase = { projectId, target, scope, page, pageSize };
  // Keep the current table on screen while paging, sorting or filtering the
  // same target. A new target shows the loading state instead. Each tab has
  // its own query, so switching tabs never shows another tab's rows.
  const keepSameTarget = <T>(
    previous: T | undefined,
    previousQuery: { queryKey: QueryKey } | undefined,
  ) =>
    baseQueryKeyParts.every(
      (part, index) => previousQuery?.queryKey[index + 1] === part,
    )
      ? previous
      : undefined;

  const overviewQuery = useQuery({
    queryKey: ["backlinksOverview", ...baseQueryKeyParts],
    enabled: targetReady,
    // The POST may already have reached DataForSEO when a 5xx is returned.
    retry: false,
    refetchOnReconnect: false,
    refetchOnWindowFocus: false,
    staleTime: BACKLINKS_QUERY_STALE_TIME_MS,
    queryFn: () =>
      getBacklinksOverview({
        projectId,
        target,
        scope,
        cachedOnly: cachedOnly(),
      }),
  });

  // Each table is a paid call. It waits for the overview and is skipped when
  // the overview found no links (a typo or a new site): the table could only
  // come back empty, and the customer would pay for it.
  const overviewLinks = overviewQuery.data?.summary.backlinks ?? 0;
  const tablesReady = targetReady && overviewLinks > 0;

  const rowsSort = toSort(
    sort,
    order,
    backlinksRowsSortFieldSchema.options,
    BACKLINKS_DEFAULT_SORT.backlinks,
  );
  const rowsFilters = useMemo(
    () => toBacklinksFiltersPayload(filters.backlinks.values),
    [filters.backlinks.values],
  );
  const rowsFilterError = backlinksFilterBudgetError(
    filters.backlinks.values,
    scope,
    !searchState.includeSpam,
  );
  const rowsQuery = useQuery({
    queryKey: [
      "backlinksRows",
      ...baseQueryKeyParts,
      page,
      pageSize,
      rowsSort.field,
      rowsSort.order,
      rowsFilters,
      rowsMode,
      searchState.includeSpam ?? false,
    ],
    enabled: tablesReady && tab === "backlinks" && !rowsFilterError,
    retry: false,
    refetchOnReconnect: false,
    refetchOnWindowFocus: false,
    staleTime: BACKLINKS_QUERY_STALE_TIME_MS,
    // Over-budget filters never run the query, so no rows (and no export) may
    // stand in for them.
    placeholderData: rowsFilterError ? undefined : keepSameTarget,
    queryFn: () =>
      getBacklinksRows({
        ...pageInputBase,
        cachedOnly: cachedOnly(),
        sortField: rowsSort.field,
        sortOrder: rowsSort.order,
        filters: rowsFilters,
        mode: rowsMode,
        hideSpam: !searchState.includeSpam,
      }),
  });

  const domainsSort = toSort(
    sort,
    order,
    referringDomainsSortFieldSchema.options,
    BACKLINKS_DEFAULT_SORT.domains,
  );
  const domainsFilters = useMemo(
    () => toReferringDomainsFiltersPayload(filters.domains.values),
    [filters.domains.values],
  );
  const referringDomainsQuery = useQuery({
    queryKey: [
      "backlinksReferringDomains",
      ...baseQueryKeyParts,
      page,
      pageSize,
      domainsSort.field,
      domainsSort.order,
      domainsFilters,
    ],
    enabled: tablesReady && tab === "domains",
    retry: false,
    refetchOnReconnect: false,
    refetchOnWindowFocus: false,
    staleTime: BACKLINKS_QUERY_STALE_TIME_MS,
    placeholderData: keepSameTarget,
    queryFn: () =>
      getBacklinksReferringDomains({
        ...pageInputBase,
        cachedOnly: cachedOnly(),
        sortField: domainsSort.field,
        sortOrder: domainsSort.order,
        filters: domainsFilters,
      }),
  });

  const pagesSort = toSort(
    sort,
    order,
    topPagesSortFieldSchema.options,
    BACKLINKS_DEFAULT_SORT.pages,
  );
  const pagesFilters = useMemo(
    () => toTopPagesFiltersPayload(filters.pages.values),
    [filters.pages.values],
  );
  const topPagesQuery = useQuery({
    queryKey: [
      "backlinksTopPages",
      ...baseQueryKeyParts,
      page,
      pageSize,
      pagesSort.field,
      pagesSort.order,
      pagesFilters,
    ],
    enabled: tablesReady && tab === "pages",
    retry: false,
    refetchOnReconnect: false,
    refetchOnWindowFocus: false,
    staleTime: BACKLINKS_QUERY_STALE_TIME_MS,
    placeholderData: keepSameTarget,
    queryFn: () =>
      getBacklinksTopPages({
        ...pageInputBase,
        cachedOnly: cachedOnly(),
        sortField: pagesSort.field,
        sortOrder: pagesSort.order,
        filters: pagesFilters,
      }),
  });

  const overviewErrorMessage = getBacklinksErrorMessage(
    overviewQuery.error,
    "Could not load backlinks data.",
  );
  const activeTabQuery =
    tab === "backlinks"
      ? rowsQuery
      : tab === "domains"
        ? referringDomainsQuery
        : topPagesQuery;
  const activeTabFilterError = tab === "backlinks" ? rowsFilterError : null;
  const activeTabErrorMessage =
    activeTabFilterError ??
    getBacklinksErrorMessage(activeTabQuery.error, "Could not load this tab.");

  // Refresh or Retry is the customer asking to buy this report again.
  const buyFresh = (query: { refetch: () => unknown }) => {
    allowPurchase(intentKey);
    void query.refetch();
  };

  return {
    activeTabErrorMessage,
    buyFresh,
    activeTabFilterError,
    activeTabQuery,
    overviewErrorMessage,
    overviewQuery,
    referringDomainsQuery,
    rowsQuery,
    searchCardInitialValues,
    topPagesQuery,
  };
}

export type BacklinksPageData = ReturnType<typeof useBacklinksPageData>;

export function navigateToBacklinksSearch(
  navigate: BacklinksPageProps["navigate"],
  values: Pick<BacklinksSearchState, "target" | "scope">,
) {
  navigate({
    search: (prev) => ({
      ...prev,
      target: values.target,
      scope: toScopeSearchParam(values.target, values.scope),
      tab: undefined,
      page: undefined,
      sort: undefined,
      order: undefined,
    }),
    replace: true,
  });
}

/** The paid-intent key of one Backlinks report (see paidIntent). */
export function backlinksIntentKey(
  projectId: string,
  target: string | undefined,
  scope: string | undefined,
) {
  return `backlinks:${projectId}:${scope ?? ""}:${target ?? ""}`;
}
