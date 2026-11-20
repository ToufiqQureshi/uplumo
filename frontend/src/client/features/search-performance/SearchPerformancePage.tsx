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
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/client/components/ui/tabs";
import { GoogleConnectionCard } from "@/client/features/integrations/GoogleConnectionCard";
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
  exportDimensionRows,
  DimensionTable,
  type ExportTarget,
} from "@/client/features/search-performance/SearchPerformanceParts";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import {
  exportSearchPerformanceTable,
  getSearchPerformanceReport,
} from "@/client/features/search-performance/searchPerformanceApi";
import { buildSearchInsights } from "@/client/features/search-performance/searchInsights";
import { SearchPerformanceTotals } from "@/client/features/search-performance/SearchPerformanceTotals";
import { SearchPerformanceTrend } from "@/client/features/search-performance/SearchPerformanceTrend";
import {
  buildDimensionColumns,
  type Report,
  formatCount,
  formatCtr,
  formatPosition,
} from "@/client/features/search-performance/SearchPerformanceColumns";
import {
  SEARCH_PERFORMANCE_DEFAULT_PAGE_SIZE,
  type SearchPerformanceSearch,
  type SearchPerformanceTableDimension,
} from "@/types/schemas/search-performance";

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

/** Render a table for a dimension (query or page). */
function DimensionTableWrapper({
  dimension,
  report,
  search,
  page,
  pageSize,
  onPageChange,
  onPageSizeChange,
  onSearchChange,
  tableQuery,
  isExporting,
  onExport,
}: {
  dimension: SearchPerformanceTableDimension;
  report: Report;
  search: SearchPerformanceSearch;
  page: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: number) => void;
  onSearchChange: (update: Partial<SearchPerformanceSearch>) => void;
  tableQuery: ReturnType<typeof useQuery>;
  isExporting: boolean;
  onExport: (target: ExportTarget) => void;
}) {
  const isPage = dimension === "page";
  const keyLabel = isPage ? "Page" : "Query";

  if (tableQuery.isPending) {
    return <SearchPerformanceLoadingState />;
  }
  if (tableQuery.isError) {
    return (
      <QueryError
        error={tableQuery.error}
        fallback="Failed to load table"
        onRetry={() => void tableQuery.refetch()}
        isRetrying={tableQuery.isFetching}
      />
    );
  }
  if (!tableQuery.data?.connected || tableQuery.data.connected === false)
    return null;

  const data = tableQuery.data as Extract<typeof tableQuery.data, { connected: true }>;
  const rows = data.rows;
  const hasNextPage = data.hasNextPage;
  const totalCount = data.totalCount;

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="text-sm text-muted-foreground">
          {totalCount != null
            ? `${totalCount} total ${isPage ? "pages" : "queries"}`
            : ""}
        </div>
        <ExportMenu
          actions={["sheets", "csv"]}
          onExport={onExport}
          busy={isExporting}
          disabled={rows.length === 0}
        />
      </div>
      <DimensionTable
        rows={rows}
        keyLabel={keyLabel}
        isFiltered={Boolean(search.queryText || search.pageText)}
        isPastEnd={!hasNextPage && rows.length > 0}
        onClearFilters={() =>
          onSearchChange({
            page: undefined,
            queryText: undefined,
            pageText: undefined,
          })
        }
        onFirstPage={() => onSearchChange({ page: 1 })}
      />
      <div className="flex items-center justify-between">
        <div />
        {(hasNextPage || page > 1) && (
          <div className="flex items-center gap-2">
            <Button
              size="sm"
              variant="outline"
              disabled={page === 1 || tableQuery.isFetching}
              onClick={() => onPageChange(page - 1)}
            >
              Previous
            </Button>
            <span className="text-sm text-muted-foreground">Page {page}</span>
            <Button
              size="sm"
              variant="outline"
              disabled={!hasNextPage || tableQuery.isFetching}
              onClick={() => onPageChange(page + 1)}
            >
              Next
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}

/** Render countries or devices table (non-paginated, from report data). */
function StaticDimensionTable({
  dimension,
  rows,
  search,
  onSearchChange,
}: {
  dimension: "country" | "device";
  rows: Array<{
    key: string;
    clicks: number;
    impressions: number;
    ctr: number;
    position: number;
  }>;
  search: SearchPerformanceSearch;
  onSearchChange: (update: Partial<SearchPerformanceSearch>) => void;
}) {
  const columns = useMemo(
    () => buildDimensionColumns(dimension === "country" ? "Country" : "Device"),
    [dimension],
  );

  return (
    <div className="rounded-lg border border-border overflow-hidden">
      <table className="w-full">
        <thead>
          <tr className="border-b bg-muted/50">
            {columns.map((col) => (
              <th
                key={col.id}
                className={`px-4 py-3 text-sm font-medium text-left ${col.meta?.headerClassName || ""}`}
              >
                {typeof col.header === "string"
                  ? col.header
                  : typeof col.header === "function"
                    ? col.header({ column: col, header: col } as any)
                    : null}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr
              key={row.key}
              className="border-b hover:bg-muted/50 cursor-pointer transition-colors"
              onClick={() => {
                onSearchChange({
                  page: undefined,
                  [dimension]: row.key.toUpperCase(),
                });
              }}
            >
              <td className="px-4 py-3 text-sm font-medium">{row.key}</td>
              <td className="px-4 py-3 text-sm text-right tabular-nums">
                {formatCount(row.clicks)}
              </td>
              <td className="px-4 py-3 text-sm text-right tabular-nums">
                {formatCount(row.impressions)}
              </td>
              <td className="px-4 py-3 text-sm text-right tabular-nums">
                {formatCtr(row.ctr)}
              </td>
              <td className="px-4 py-3 text-sm text-right tabular-nums">
                {formatPosition(row.position)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
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
  const tab = search.tab ?? "queries";
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
  const dimension: SearchPerformanceTableDimension =
    tab === "pages" ? "page" : "query";
  const tableQuery = useQuery({
    ...tableQueryOptions(projectId, dimension, page, pageSize, filterInput),
    enabled: report?.connected === true && isTableTab,
    placeholderData: (previous, previousQuery) =>
      previousQuery?.queryKey[1] === projectId &&
      previousQuery.queryKey[2] === dimension
        ? previous
        : undefined,
  });

  // Warm the Queries tab as soon as the report connects
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

  const insights = report?.connected ? buildSearchInsights(report) : [];

  return (
    <div className="overflow-auto px-4 py-4 pb-24 md:px-6 md:py-6 md:pb-8">
      <div className="mx-auto max-w-7xl space-y-4">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <h1 className="text-2xl font-semibold">Google Search</h1>
            <p className="text-sm text-muted-foreground">
              Clicks, keywords, pages and countries from your Google Search
              Console.
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

            {/* Filter bar */}
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

            {/* KPI row */}
            {reportQuery.isPlaceholderData ? (
              <SearchPerformanceLoadingState />
            ) : (
              <>
                <SearchPerformanceTotals report={report} />

                {/* Trend chart */}
                <SearchPerformanceTrend report={report} />

                {/* What changed card */}
                {insights.length > 0 && (
                  <div className="rounded-lg border bg-card p-4 space-y-3">
                    <SectionHeader
                      title="What changed"
                      hint="Your biggest wins and losses"
                    />
                    <ul className="space-y-2">
                      {insights.map((insight, idx) => (
                        <li key={idx} className="text-sm">
                          {insight.link && (
                            <button
                              onClick={() =>
                                onSearchChange({
                                  page: undefined,
                                  [insight.link!.param]: insight.link!.value,
                                })
                              }
                              className="text-left hover:underline text-blue-600 dark:text-blue-400"
                            >
                              {insight.text}
                            </button>
                          )}
                          {!insight.link && <span>{insight.text}</span>}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}

                {/* Data tabs */}
                <Tabs
                  value={tab}
                  onValueChange={(value) =>
                    onSearchChange({
                      tab: value as typeof tab,
                      page: undefined,
                    })
                  }
                  className="space-y-4"
                >
                  <TabsList>
                    <TabsTrigger value="queries">Keywords</TabsTrigger>
                    <TabsTrigger value="pages">Pages</TabsTrigger>
                    <TabsTrigger value="countries">Countries</TabsTrigger>
                    <TabsTrigger value="devices">Devices</TabsTrigger>
                  </TabsList>

                  <TabsContent value="queries" className="space-y-4">
                    <DimensionTableWrapper
                      dimension="query"
                      report={report}
                      search={search}
                      page={page}
                      pageSize={pageSize}
                      onPageChange={(nextPage) =>
                        onSearchChange({ page: nextPage })
                      }
                      onPageSizeChange={(size) =>
                        onSearchChange({ page: undefined, size })
                      }
                      onSearchChange={onSearchChange}
                      tableQuery={tableQuery}
                      isExporting={isExporting}
                      onExport={(target) => void handleExport(target)}
                    />
                  </TabsContent>

                  <TabsContent value="pages" className="space-y-4">
                    <DimensionTableWrapper
                      dimension="page"
                      report={report}
                      search={search}
                      page={page}
                      pageSize={pageSize}
                      onPageChange={(nextPage) =>
                        onSearchChange({ page: nextPage })
                      }
                      onPageSizeChange={(size) =>
                        onSearchChange({ page: undefined, size })
                      }
                      onSearchChange={onSearchChange}
                      tableQuery={tableQuery}
                      isExporting={isExporting}
                      onExport={(target) => void handleExport(target)}
                    />
                  </TabsContent>

                  <TabsContent value="countries">
                    <StaticDimensionTable
                      dimension="country"
                      rows={report.countries}
                      search={search}
                      onSearchChange={onSearchChange}
                    />
                  </TabsContent>

                  <TabsContent value="devices">
                    <StaticDimensionTable
                      dimension="device"
                      rows={report.devices}
                      search={search}
                      onSearchChange={onSearchChange}
                    />
                  </TabsContent>
                </Tabs>
              </>
            )}
          </>
        )}
      </div>
    </div>
  );
}
