import { useCallback } from "react";
import type { UseQueryResult } from "@tanstack/react-query";
import { EditPageButton } from "@/client/features/content-editor/EditPageButton";
import { ExportMenu } from "@/client/components/ExportMenu";
import { QueryError } from "@/client/components/QueryState";
import { TablePagination } from "@/client/components/table/TablePagination";
import { SearchPerformanceLoadingState } from "@/client/features/search-performance/SearchPerformanceLoadingState";
import {
  DimensionTable,
  type ExportTarget,
} from "@/client/features/search-performance/SearchPerformanceParts";
import type { getSearchPerformanceTable } from "@/client/features/search-performance/searchPerformanceApi";
import {
  SEARCH_PERFORMANCE_PAGE_SIZES,
  type SearchPerformanceTableDimension,
} from "@/types/schemas/search-performance";

type TableResult = Awaited<ReturnType<typeof getSearchPerformanceTable>>;

/** The paginated Keywords or Pages table, read one page at a time from
 *  Search Console (free), with a full export. */
export function SearchRowsTable({
  projectId,
  dimension,
  tableQuery,
  page,
  pageSize,
  isFiltered,
  isExporting,
  onPageChange,
  onPageSizeChange,
  onClearFilters,
  onExport,
}: {
  projectId: string;
  dimension: SearchPerformanceTableDimension;
  tableQuery: UseQueryResult<TableResult>;
  page: number;
  pageSize: number;
  isFiltered: boolean;
  isExporting: boolean;
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: number) => void;
  onClearFilters: () => void;
  onExport: (target: ExportTarget) => void;
}) {
  // One click opens the page's own content in the editor (free: it copies
  // the project's own page).
  const editPage = useCallback(
    (url: string) => (
      <EditPageButton
        projectId={projectId}
        url={url}
        variant="ghost"
        label="Edit"
      />
    ),
    [projectId],
  );

  if (tableQuery.isPending) return <SearchPerformanceLoadingState />;
  if (tableQuery.isError) {
    return (
      <QueryError
        error={tableQuery.error}
        fallback="Failed to load the table"
        onRetry={() => void tableQuery.refetch()}
        isRetrying={tableQuery.isFetching}
      />
    );
  }
  const data = tableQuery.data;
  if (!data.connected) return null;

  return (
    <div className="space-y-3">
      <div className="flex justify-end">
        <ExportMenu
          actions={["sheets", "csv"]}
          onExport={onExport}
          busy={isExporting}
          disabled={data.rows.length === 0}
        />
      </div>
      <DimensionTable
        rows={data.rows}
        keyLabel={dimension === "page" ? "Page" : "Keyword"}
        isFiltered={isFiltered}
        isPastEnd={page > 1 && data.rows.length === 0}
        onClearFilters={onClearFilters}
        onFirstPage={() => onPageChange(1)}
        rowAction={dimension === "page" ? editPage : undefined}
      />
      <TablePagination
        page={page}
        pageSize={pageSize}
        pageSizes={SEARCH_PERFORMANCE_PAGE_SIZES}
        totalCount={data.totalCount}
        hasNextPage={data.hasNextPage}
        isLoading={tableQuery.isFetching}
        onPageChange={onPageChange}
        onPageSizeChange={onPageSizeChange}
      />
    </div>
  );
}
