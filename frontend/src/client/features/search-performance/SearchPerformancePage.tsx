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
import { SectionHeader } from "@/client/components/PageHeader";
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
import { SearchBreakdownTable } from "@/client/features/search-performance/SearchBreakdownTable";
import { SearchPerformanceLoadingState } from "@/client/features/search-performance/SearchPerformanceLoadingState";
import {
  exportDimensionRows,
  type ExportTarget,
} from "@/client/features/search-performance/SearchPerformanceParts";
import { SearchPerformanceSelects } from "@/client/features/search-performance/SearchPerformanceSelects";
import {
  SearchPerformanceTextFilters,
  type TextFilters,
} from "@/client/features/search-performance/SearchPerformanceTextFilters";
import { SearchPerformanceTotals } from "@/client/features/search-performance/SearchPerformanceTotals";
import { SearchPerformanceTrend } from "@/client/features/search-performance/SearchPerformanceTrend";
import { SearchRowsTable } from "@/client/features/search-performance/SearchRowsTable";
import {
  exportSearchPerformanceTable,
  getSearchPerformanceReport,
} from "@/client/features/search-performance/searchPerformanceApi";
import {
  tableQueryOptions,
  type FilterInput,
} from "@/client/features/search-performance/searchPerformanceQueries";
import { buildSearchInsights } from "@/client/features/search-performance/searchInsights";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import {
  GSC_DEVICES,
  SEARCH_PERFORMANCE_DEFAULT_PAGE_SIZE,
  SEARCH_PERFORMANCE_TABS,
  type SearchPerformanceSearch,
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
  const filterInput = { ...buildFilterInput(search), ...textFilters };

  const reportQuery = useQuery({
    queryKey: ["searchPerformance", projectId, filterInput],
    queryFn: () =>
      getSearchPerformanceReport({ data: { projectId, ...filterInput } }),
    placeholderData: keepPreviousData,
  });
  const report = reportQuery.data;

  const dimension = tab === "pages" ? "page" : "query";
  const tableQuery = useQuery({
    ...tableQueryOptions(projectId, dimension, page, pageSize, filterInput),
    enabled:
      report?.connected === true && (tab === "queries" || tab === "pages"),
    // Hold rows while paging or filtering, never across Keywords/Pages.
    placeholderData: (previous, previousQuery) =>
      previousQuery?.queryKey[1] === projectId &&
      previousQuery.queryKey[2] === dimension
        ? previous
        : undefined,
  });

  // Keywords is the first tab: read its first page as soon as Google is
  // connected so it opens without a spinner. Free Search Console read.
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

  const insights = report?.connected
    ? buildSearchInsights(report, { countryFiltered: Boolean(search.country) })
    : [];

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
          reportError
        ) : !report.connected ? (
          <div className="max-w-2xl">
            <GoogleConnectionCard provider="gsc" projectId={projectId} />
          </div>
        ) : (
          <>
            {reportQuery.isError ? reportError : null}
            <div className="flex flex-col gap-2 rounded-xl border bg-card px-3 py-2 sm:flex-row sm:items-start">
              <Button
                type="button"
                variant={showFilters ? "secondary" : "ghost"}
                size="sm"
                aria-expanded={showFilters}
                onClick={() => setShowFilters((current) => !current)}
                title="Filter by page or keyword, including regular expressions"
              >
                <SlidersHorizontal data-icon="inline-start" />
                Filters
                {filterCount > 0 ? (
                  <Badge size="sm">{filterCount}</Badge>
                ) : null}
              </Button>
              {reportQuery.isFetching ? (
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
              <div className="overflow-hidden rounded-xl border">
                {filtersPanel}
              </div>
            ) : null}

            {reportQuery.isPlaceholderData ? (
              <SearchPerformanceLoadingState />
            ) : (
              <>
                <SearchPerformanceTotals report={report} />
                <SearchPerformanceTrend report={report} />

                {insights.length > 0 ? (
                  <section className="space-y-2 rounded-xl border bg-card p-4">
                    <SectionHeader title="What changed" />
                    <ul className="space-y-1.5">
                      {insights.map((insight) => (
                        <li key={insight.text}>
                          <button
                            type="button"
                            className="text-left text-sm underline-offset-4 hover:underline"
                            onClick={() =>
                              onSearchChange({
                                page: undefined,
                                ...insight.open,
                              })
                            }
                          >
                            {insight.text}
                          </button>
                        </li>
                      ))}
                    </ul>
                  </section>
                ) : null}

                <Tabs
                  value={tab}
                  onValueChange={(value) => {
                    const next = SEARCH_PERFORMANCE_TABS.find(
                      (item) => item === value,
                    );
                    if (next) onSearchChange({ tab: next, page: undefined });
                  }}
                  className="space-y-3"
                >
                  <TabsList>
                    <TabsTrigger value="queries">Keywords</TabsTrigger>
                    <TabsTrigger value="pages">Pages</TabsTrigger>
                    <TabsTrigger value="countries">Countries</TabsTrigger>
                    <TabsTrigger value="devices">Devices</TabsTrigger>
                  </TabsList>
                  {tab === "queries" || tab === "pages" ? (
                    <TabsContent value={tab}>
                      <SearchRowsTable
                        dimension={dimension}
                        tableQuery={tableQuery}
                        page={page}
                        pageSize={pageSize}
                        isFiltered={filterCount > 0}
                        isExporting={isExporting}
                        onPageChange={(next) => onSearchChange({ page: next })}
                        onPageSizeChange={(size) =>
                          onSearchChange({ page: undefined, size })
                        }
                        onClearFilters={resetFilters}
                        onExport={(target) => void handleExport(target)}
                      />
                    </TabsContent>
                  ) : null}
                  <TabsContent value="countries">
                    <SearchBreakdownTable
                      breakdown="country"
                      rows={report.countries}
                      // Country rows ignore the country filter; the totals
                      // follow it, so a chosen country shares of its rows.
                      totalClicks={
                        search.country
                          ? report.countries.reduce(
                              (sum, r) => sum + r.clicks,
                              0,
                            )
                          : report.totals.clicks
                      }
                      onSelect={(country) =>
                        onSearchChange({ page: undefined, country })
                      }
                    />
                  </TabsContent>
                  <TabsContent value="devices">
                    <SearchBreakdownTable
                      breakdown="device"
                      rows={report.devices}
                      totalClicks={report.devices.reduce(
                        (sum, r) => sum + r.clicks,
                        0,
                      )}
                      onSelect={(key) => {
                        const device = GSC_DEVICES.find(
                          (item) => item === key.toUpperCase(),
                        );
                        if (device) onSearchChange({ page: undefined, device });
                      }}
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
