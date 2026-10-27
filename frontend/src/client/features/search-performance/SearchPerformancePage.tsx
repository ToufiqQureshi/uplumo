import { SearchGrowthPanel } from "./SearchGrowthPanel";
import { Ga4OrganicCard } from "./Ga4OrganicCard";
import {
  PagesToFix,
  QuickWins,
  SearchesChanged,
  SearchVerdict,
} from "./SearchVerdictCards";
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
import { DataTableTabs } from "@/client/components/table/DataTableToolbar";
import { TabsTrigger } from "@/client/components/ui/tabs";
import { GoogleConnectionCard } from "@/client/features/integrations/GoogleConnectionCard";
import { IndexNowCard } from "@/client/features/indexnow/IndexNowCard";
import { SectionHeader } from "@/client/components/PageHeader";
import { DimensionSection } from "@/client/features/search-performance/SearchPerformanceDimensionSection";
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
  StrikingDistanceTable,
  type ExportTarget,
} from "@/client/features/search-performance/SearchPerformanceParts";
import { TotalsCards } from "@/client/features/search-performance/SearchPerformanceTotals";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import { ContentRefreshQueue } from "@/client/features/search-performance/ContentRefreshQueuePanel";
import { SearchPerformanceTrend } from "@/client/features/search-performance/SearchPerformanceTrend";
import { SearchPerformanceIndexing } from "@/client/features/search-performance/SearchPerformanceIndexing";
import { buildContentRefreshQueue } from "@/client/features/search-performance/contentRefreshQueue";
import {
  exportSearchPerformanceTable,
  getSearchPerformanceReport,
} from "@/client/features/search-performance/searchPerformanceApi";
import {
  SEARCH_PERFORMANCE_DEFAULT_PAGE_SIZE,
  SEARCH_PERFORMANCE_TABS,
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

  return (
    <div className="overflow-auto px-4 py-4 pb-24 md:px-6 md:py-6 md:pb-8">
      <div className="mx-auto max-w-7xl space-y-4">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h1 className="text-2xl font-semibold">Search Performance</h1>
            <p className="text-sm text-muted-foreground">
              What changed in Google search, why, and what to fix first.
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
          <div className="max-w-2xl">
            <GoogleConnectionCard provider="gsc" projectId={projectId} />
          </div>
        ) : (
          <>
            {reportQuery.isError ? reportError : null}
            {reportQuery.isPlaceholderData ? (
              <SearchPerformanceLoadingState />
            ) : (
              <>
                <SearchVerdict projectId={projectId} report={report} />
                <TotalsCards report={report} />
                <PagesToFix projectId={projectId} report={report} />
                <SearchesChanged projectId={projectId} report={report} />
                <QuickWins projectId={projectId} report={report} />
                <Ga4OrganicCard projectId={projectId} />
              </>
            )}
            {!reportQuery.isPlaceholderData ? (
              <SearchPerformanceTrend report={report} />
            ) : null}
            <SearchPerformanceIndexing
              projectId={projectId}
              strikingDistance={
                report?.connected ? report.strikingDistance : []
              }
            />
            <div className="overflow-hidden rounded-xl border border-border bg-card">
              <DataTableTabs
                value={tab}
                onValueChange={(value) => {
                  const nextTab = SEARCH_PERFORMANCE_TABS.find(
                    (item) => item === value,
                  );
                  if (nextTab)
                    onSearchChange({ tab: nextTab, page: undefined });
                }}
                actions={
                  <ExportMenu
                    actions={["sheets", "csv"]}
                    onExport={(target) => void handleExport(target)}
                    busy={isExporting}
                    disabled={reportQuery.isPlaceholderData}
                  />
                }
              >
                <TabsTrigger value="striking">
                  {reportQuery.isPlaceholderData
                    ? "Striking distance"
                    : `Striking distance (${report.strikingDistance.length})`}
                </TabsTrigger>
                <TabsTrigger value="queries">Queries</TabsTrigger>
                <TabsTrigger value="pages">Pages</TabsTrigger>
                <TabsTrigger value="refresh">Refresh queue</TabsTrigger>
              </DataTableTabs>
              <div className="flex flex-wrap items-center gap-2 border-b border-border px-4 py-2">
                <Button
                  type="button"
                  variant={showFilters ? "secondary" : "ghost"}
                  size="sm"
                  aria-expanded={showFilters}
                  aria-controls="search-performance-filters"
                  onClick={() => setShowFilters((current) => !current)}
                  title="Toggle table filters"
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

              {showFilters ? filtersPanel : null}
              {reportQuery.isPlaceholderData ? (
                <div className="p-8 text-sm" role="status">
                  Loading matching results…
                </div>
              ) : tab === "striking" ? (
                <StrikingDistanceTable
                  projectId={projectId}
                  rows={report.strikingDistance}
                  page={page}
                  pageSize={pageSize}
                  onPageChange={(nextPage) =>
                    onSearchChange({ page: nextPage })
                  }
                  onPageSizeChange={(size) =>
                    onSearchChange({ page: undefined, size })
                  }
                  isFiltered={filterCount > 0}
                  onClearFilters={resetFilters}
                />
              ) : tab === "refresh" ? (
                <div className="p-4">
                  <ContentRefreshQueue
                    projectId={projectId}
                    rows={report.strikingDistance}
                  />
                </div>
              ) : (
                <DimensionSection
                  projectId={projectId}
                  tableQuery={tableQuery}
                  keyLabel={tab === "queries" ? "Query" : "Page"}
                  page={page}
                  pageSize={pageSize}
                  onPageChange={(nextPage) =>
                    onSearchChange({ page: nextPage })
                  }
                  onPageSizeChange={(size) =>
                    onSearchChange({ page: undefined, size })
                  }
                  isFiltered={filterCount > 0}
                  onClearFilters={resetFilters}
                />
              )}
            </div>
            <SearchGrowthPanel key={projectId} projectId={projectId} />
          </>
        )}

        {/* IndexNow needs no Search Console connection and works on any
            website, so it shows whether or not Google is connected. */}
        {reportQuery.isPending ? null : (
          <section id="indexnow" className="max-w-2xl space-y-3 pt-4">
            <SectionHeader
              title="Request indexing on Bing and other engines (IndexNow)"
              hint="Works on any website, WordPress or not. Google does not use IndexNow: for Google, use URL Inspection in Search Console."
            />
            <IndexNowCard projectId={projectId} />
          </section>
        )}
      </div>
    </div>
  );
}
