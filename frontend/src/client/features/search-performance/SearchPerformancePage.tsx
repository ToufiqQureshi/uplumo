import { SearchGrowthPanel } from "./SearchGrowthPanel";
import { ExploreView } from "./ExploreView";
import { FixView, GrowView, OverviewView } from "./GrowthViews";
import { useBrandNames } from "./brandNames";
import { insightsQueryOptions } from "./insightsApi";
import { useEffect, useMemo, useState } from "react";
import { Link } from "@tanstack/react-router";
import {
  keepPreviousData,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { SlidersHorizontal } from "lucide-react";
import { toast } from "sonner";
import { QueryError } from "@/client/components/QueryState";
import { ExportMenu } from "@/client/components/ExportMenu";
import { Badge } from "@/client/components/ui/badge";
import { Button } from "@/client/components/ui/button";
import { Spinner } from "@/client/components/ui/spinner";
import { ViewTabs } from "./ViewTabs";
import { GoogleConnectionCard } from "@/client/features/integrations/GoogleConnectionCard";
import { IndexNowCard } from "@/client/features/indexnow/IndexNowCard";
import { SectionHeader } from "@/client/components/PageHeader";
import { SearchPerformanceSelects } from "@/client/features/search-performance/SearchPerformanceSelects";
import { SearchPerformanceLoadingState } from "@/client/features/search-performance/SearchPerformanceLoadingState";
import {
  tableQueryOptions,
  type FilterInput,
} from "@/client/features/search-performance/searchPerformanceQueries";
import {
  SearchPerformanceTextFilters,
  type TextFilters,
} from "@/client/features/search-performance/SearchPerformanceTextFilters";
import {
  exportContentRefreshQueue,
  exportDimensionRows,
  exportStriking,
  type ExportTarget,
} from "@/client/features/search-performance/SearchPerformanceParts";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import { SearchPerformanceIndexing } from "@/client/features/search-performance/SearchPerformanceIndexing";
import { buildContentRefreshQueue } from "@/client/features/search-performance/contentRefreshQueue";
import {
  exportSearchPerformanceTable,
  getSearchPerformanceReport,
} from "@/client/features/search-performance/searchPerformanceApi";
import {
  SEARCH_PERFORMANCE_DEFAULT_PAGE_SIZE,
  type SearchGrowthView,
  type SearchPerformanceSearch,
  type SearchPerformanceTab,
  type SearchPerformanceTableDimension,
} from "@/types/schemas/search-performance";

function tabDimension(
  tab: SearchPerformanceTab,
): SearchPerformanceTableDimension {
  return tab === "pages" ? "page" : "query";
}

function activeFilterCount(
  search: SearchPerformanceSearch,
  filters: TextFilters,
): number {
  const { country, device, searchType, range, startDate, endDate } = search;
  return (
    Number(Boolean(filters.pageFilter)) +
    Number(Boolean(filters.queryFilter)) +
    Number(Boolean(country)) +
    Number(Boolean(device)) +
    Number(Boolean(searchType && searchType !== "web")) +
    Number(Boolean(startDate && endDate) || (range && range !== "last_28_days"))
  );
}

function buildFilterInput(search: SearchPerformanceSearch): FilterInput {
  return {
    dateRange: search.range ?? "last_28_days",
    searchType: search.searchType ?? "web",
    ...(search.startDate && search.endDate
      ? { startDate: search.startDate, endDate: search.endDate }
      : {}),
    ...(search.device ? { device: search.device } : {}),
    ...(search.country ? { country: search.country } : {}),
  };
}

export function SearchPerformancePage({
  projectId,
  search,
  onSearchChange,
}: {
  projectId: string;
  search: SearchPerformanceSearch;
  onSearchChange: (update: Partial<SearchPerformanceSearch>) => void;
}) {
  const queryClient = useQueryClient();
  const textFilters = useMemo<TextFilters>(
    () => ({
      pageFilter: search.pageText
        ? {
            operator: search.pageMatch ?? "contains",
            expression: search.pageText,
          }
        : undefined,
      queryFilter: search.queryText
        ? {
            operator: search.queryMatch ?? "contains",
            expression: search.queryText,
          }
        : undefined,
    }),
    [search.pageText, search.pageMatch, search.queryText, search.queryMatch],
  );
  const [showFilters, setShowFilters] = useState(false);
  const [isExporting, setIsExporting] = useState(false);
  const filterCount = activeFilterCount(search, textFilters);
  const tab = search.tab ?? "striking";
  const page = search.page ?? 1;
  const pageSize = search.size ?? SEARCH_PERFORMANCE_DEFAULT_PAGE_SIZE;
  const view = search.view ?? "overview";
  const brands = useBrandNames(projectId);

  const filterInput = {
    ...buildFilterInput(search),
    ...textFilters,
  };

  const reportQuery = useQuery({
    queryKey: ["searchPerformance", projectId, filterInput],
    queryFn: () =>
      getSearchPerformanceReport({ data: { projectId, ...filterInput } }),
    placeholderData: keepPreviousData,
  });
  const report = reportQuery.data;
  // Only the screens that use the analysis read for it, and the analysis is
  // kept for five minutes, so moving between screens never re-reads Google.
  const insightsQuery = useQuery({
    ...insightsQueryOptions(projectId, filterInput, brands.names),
    enabled:
      report?.connected === true &&
      !reportQuery.isPlaceholderData &&
      (view === "overview" || view === "fix" || view === "grow"),
    placeholderData: keepPreviousData,
  });
  const isTableTab = tab === "queries" || tab === "pages";
  const dimension = tabDimension(tab);
  const tableQuery = useQuery({
    ...tableQueryOptions(projectId, dimension, page, pageSize, filterInput),
    enabled: report?.connected === true && isTableTab,
    // Hold the current rows while paging or filtering, but not across a tab
    // or project change: Query rows must not render under the Pages header.
    placeholderData: (previous, previousQuery) =>
      previousQuery?.queryKey[1] === projectId &&
      previousQuery.queryKey[2] === dimension
        ? previous
        : undefined,
  });

  // Warm the Queries tab (first page) as soon as the report connects so the tab
  // opens instantly instead of showing a spinner. Free first-party GSC data.
  useEffect(() => {
    if (report?.connected !== true) return;
    void queryClient.prefetchQuery(
      tableQueryOptions(
        projectId,
        "query",
        1,
        SEARCH_PERFORMANCE_DEFAULT_PAGE_SIZE,
        { ...buildFilterInput(search), ...textFilters },
      ),
    );
  }, [report?.connected, projectId, search, textFilters, queryClient]);

  const handleExport = async (target: ExportTarget) => {
    if (!report?.connected || reportQuery.isPlaceholderData || isExporting)
      return;
    setIsExporting(true);
    try {
      if (tab === "striking") {
        await exportStriking(report, target);
        return;
      }
      if (tab === "refresh") {
        await exportContentRefreshQueue(
          buildContentRefreshQueue(report.strikingDistance),
          report.range,
          target,
        );
        return;
      }
      const data = await exportSearchPerformanceTable({
        data: { projectId, dimension, ...filterInput },
      });
      await exportDimensionRows(dimension, data.rows, report.range, target);
    } catch (error) {
      toast.error(getStandardErrorMessage(error, "Export failed"));
    } finally {
      setIsExporting(false);
    }
  };

  const resetFilters = () =>
    onSearchChange({
      page: undefined,
      pageText: undefined,
      pageMatch: undefined,
      queryText: undefined,
      queryMatch: undefined,
      country: undefined,
      device: undefined,
      range: undefined,
      startDate: undefined,
      endDate: undefined,
      searchType: undefined,
    });

  const reportError = (
    <QueryError
      error={reportQuery.error}
      fallback="Failed to load Search Console data"
      onRetry={() => void reportQuery.refetch()}
      isRetrying={reportQuery.isFetching}
    />
  );

  const filtersPanel = (
    <SearchPerformanceTextFilters
      // Remount on URL changes (back/forward) so the draft matches.
      key={JSON.stringify(textFilters)}
      value={textFilters}
      activeFilterCount={filterCount}
      onApply={(filters) => {
        onSearchChange({
          page: undefined,
          pageText: filters.pageFilter?.expression,
          pageMatch: filters.pageFilter?.operator,
          queryText: filters.queryFilter?.expression,
          queryMatch: filters.queryFilter?.operator,
        });
      }}
      onReset={resetFilters}
    />
  );

  const openView = (next: SearchGrowthView) =>
    onSearchChange({ view: next, page: undefined });

  const showControls =
    view === "overview" ||
    view === "fix" ||
    view === "grow" ||
    view === "explore";

  const indexNowSection = (
    <section id="indexnow" className="max-w-2xl space-y-3 pt-4">
      <SectionHeader title="Request indexing on Bing and other engines (IndexNow)" />
      <IndexNowCard projectId={projectId} />
    </section>
  );

  return (
    <div className="overflow-auto px-4 py-4 pb-24 md:px-6 md:py-6 md:pb-8">
      <div className="mx-auto max-w-7xl space-y-4">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h1 className="text-2xl font-semibold">Search Growth</h1>
            <p className="text-sm text-muted-foreground">
              What changed in Google search, why, and what to do next.
            </p>
          </div>
          {report?.connected ? (
            <Link
              to="/p/$projectId/settings/integrations"
              params={{ projectId }}
              className="shrink-0 self-start text-sm font-medium text-muted-foreground underline-offset-4 hover:text-foreground hover:underline sm:mt-1"
            >
              Change property
            </Link>
          ) : null}
        </div>

        {reportQuery.isPending ? (
          <SearchPerformanceLoadingState />
        ) : !report ? (
          <>
            {filtersPanel}
            {reportError}
          </>
        ) : !report.connected ? (
          <>
            <div className="max-w-2xl">
              <GoogleConnectionCard provider="gsc" projectId={projectId} />
            </div>
            {/* IndexNow needs no Search Console connection and works on any
                website, so it shows whether or not Google is connected. */}
            {indexNowSection}
          </>
        ) : (
          <>
            {reportQuery.isError ? reportError : null}
            <ViewTabs value={view} onChange={openView} />

            {showControls ? (
              <>
                <div className="flex flex-col gap-2 rounded-xl border bg-card px-3 py-2 sm:flex-row sm:items-start">
                  <Button
                    type="button"
                    variant={showFilters ? "secondary" : "ghost"}
                    size="sm"
                    aria-expanded={showFilters}
                    aria-controls="search-performance-filters"
                    onClick={() => setShowFilters((current) => !current)}
                    title="Filter by page or search, including regular expressions"
                  >
                    <SlidersHorizontal data-icon="inline-start" />
                    Filters
                    {filterCount > 0 ? (
                      <Badge size="sm">{filterCount}</Badge>
                    ) : null}
                  </Button>
                  {reportQuery.isFetching && !reportQuery.isPending ? (
                    <Spinner className="size-4 text-muted-foreground" />
                  ) : null}
                  <SearchPerformanceSelects
                    search={search}
                    countries={report.countries}
                    reportRange={report.range}
                    onSearchChange={onSearchChange}
                  />
                </div>
                {showFilters ? (
                  <div className="overflow-hidden rounded-xl border border-border">
                    {filtersPanel}
                  </div>
                ) : null}
              </>
            ) : null}

            {reportQuery.isPlaceholderData && showControls ? (
              <SearchPerformanceLoadingState />
            ) : view === "overview" ? (
              <OverviewView
                projectId={projectId}
                report={report}
                query={insightsQuery}
                brandText={brands.text}
                onSaveBrands={brands.save}
                onOpen={openView}
              />
            ) : view === "fix" ? (
              <FixView
                projectId={projectId}
                report={report}
                query={insightsQuery}
              />
            ) : view === "grow" ? (
              <GrowView
                projectId={projectId}
                report={report}
                query={insightsQuery}
              />
            ) : view === "explore" ? (
              <ExploreView
                projectId={projectId}
                report={report}
                tab={tab}
                onTabChange={(next) =>
                  onSearchChange({ tab: next, page: undefined })
                }
                exportMenu={
                  <ExportMenu
                    actions={["sheets", "csv"]}
                    onExport={(target) => void handleExport(target)}
                    busy={isExporting}
                    disabled={reportQuery.isPlaceholderData}
                  />
                }
                tableQuery={tableQuery}
                page={page}
                pageSize={pageSize}
                onPageChange={(nextPage) => onSearchChange({ page: nextPage })}
                onPageSizeChange={(size) =>
                  onSearchChange({ page: undefined, size })
                }
                isFiltered={filterCount > 0}
                onClearFilters={resetFilters}
              />
            ) : view === "indexing" ? (
              <>
                <SearchPerformanceIndexing
                  projectId={projectId}
                  strikingDistance={report.strikingDistance}
                />
                {indexNowSection}
              </>
            ) : (
              <SearchGrowthPanel key={projectId} projectId={projectId} />
            )}
          </>
        )}
      </div>
    </div>
  );
}
