import type { ComponentProps, ReactNode } from "react";
import { DataTableTabs } from "@/client/components/table/DataTableToolbar";
import { TabsTrigger } from "@/client/components/ui/tabs";
import {
  SEARCH_PERFORMANCE_TABS,
  type SearchPerformanceTab,
} from "@/types/schemas/search-performance";
import { ContentRefreshQueue } from "./ContentRefreshQueuePanel";
import { DimensionSection } from "./SearchPerformanceDimensionSection";
import type { Report } from "./SearchPerformanceColumns";
import { StrikingDistanceTable } from "./SearchPerformanceParts";

/**
 * Every row Search Console returns for the chosen range and filters: the
 * striking-distance searches, all queries and all pages, paged, with export.
 */
export function ExploreView({
  projectId,
  report,
  tab,
  onTabChange,
  exportMenu,
  tableQuery,
  page,
  pageSize,
  onPageChange,
  onPageSizeChange,
  isFiltered,
  onClearFilters,
}: {
  projectId: string;
  report: Extract<Report, { connected: true }>;
  tab: SearchPerformanceTab;
  onTabChange: (tab: SearchPerformanceTab) => void;
  exportMenu: ReactNode;
  tableQuery: ComponentProps<typeof DimensionSection>["tableQuery"];
  page: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: number) => void;
  isFiltered: boolean;
  onClearFilters: () => void;
}) {
  const paging = {
    page,
    pageSize,
    onPageChange,
    onPageSizeChange,
    isFiltered,
    onClearFilters,
  };
  return (
    <div className="overflow-hidden rounded-xl border border-border bg-card">
      <DataTableTabs
        value={tab}
        onValueChange={(value) => {
          const next = SEARCH_PERFORMANCE_TABS.find((item) => item === value);
          if (next) onTabChange(next);
        }}
        actions={exportMenu}
      >
        <TabsTrigger value="striking">
          {`Striking distance (${report.strikingDistance.length})`}
        </TabsTrigger>
        <TabsTrigger value="queries">Queries</TabsTrigger>
        <TabsTrigger value="pages">Pages</TabsTrigger>
        <TabsTrigger value="refresh">Refresh queue</TabsTrigger>
      </DataTableTabs>
      {tab === "striking" ? (
        <StrikingDistanceTable
          projectId={projectId}
          rows={report.strikingDistance}
          {...paging}
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
          {...paging}
        />
      )}
    </div>
  );
}
