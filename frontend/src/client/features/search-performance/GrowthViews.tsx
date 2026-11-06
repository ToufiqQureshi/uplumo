import type { ReactNode } from "react";
import type { UseQueryResult } from "@tanstack/react-query";
import { QueryError } from "@/client/components/QueryState";
import type { SearchGrowthView } from "@/types/schemas/search-performance";
import { Ga4OrganicCard } from "./Ga4OrganicCard";
import { DecayingPages, LosingPages, QueryChanges } from "./FixInsights";
import { Cannibalization, CtrGaps } from "./GrowInsights";
import type { InsightsResult, SearchInsights } from "./insightsApi";
import { DoFirst, SegmentsCard } from "./OverviewInsights";
import type { Report } from "./SearchPerformanceColumns";
import { SearchPerformanceLoadingState } from "./SearchPerformanceLoadingState";
import { TotalsCards } from "./SearchPerformanceTotals";
import { SearchPerformanceTrend } from "./SearchPerformanceTrend";
import { QuickWins, SearchVerdict } from "./SearchVerdictCards";

type InsightsQuery = UseQueryResult<InsightsResult>;

function Note({ children }: { children: ReactNode }) {
  return (
    <p className="rounded-xl border bg-card p-5 text-sm text-muted-foreground">
      {children}
    </p>
  );
}

/**
 * The screens built on the analysis share how it loads, fails and refreshes:
 * a spinner the first time, the old numbers dimmed while new ones load, and a
 * retry when Google cannot be read.
 */
function InsightsGate({
  query,
  children,
}: {
  query: InsightsQuery;
  children: (insights: SearchInsights) => ReactNode;
}) {
  if (query.isPending) return <SearchPerformanceLoadingState />;
  const data = query.data;
  if (!data) {
    return (
      <QueryError
        error={query.error}
        fallback="Failed to analyse your Search Console data"
        onRetry={() => void query.refetch()}
        isRetrying={query.isFetching}
      />
    );
  }
  if (!data.connected) return null;
  return (
    <div
      className={query.isPlaceholderData ? "space-y-4 opacity-60" : "space-y-4"}
      aria-busy={query.isPlaceholderData}
    >
      {children(data)}
    </div>
  );
}

export function OverviewView({
  projectId,
  report,
  query,
  brandText,
  onSaveBrands,
  onOpen,
}: {
  projectId: string;
  report: Report;
  query: InsightsQuery;
  brandText: string;
  onSaveBrands: (text: string) => void;
  onOpen: (view: SearchGrowthView) => void;
}) {
  const insights = query.data?.connected ? query.data : null;
  return (
    <>
      <SearchVerdict
        projectId={projectId}
        report={report}
        onOpenFix={() => onOpen("fix")}
      />
      {insights ? <DoFirst actions={insights.actions} onOpen={onOpen} /> : null}
      <TotalsCards report={report} />
      <Ga4OrganicCard projectId={projectId} />
      <SearchPerformanceTrend report={report} />
      <InsightsGate query={query}>
        {(data) => (
          <SegmentsCard
            insights={data}
            brandText={brandText}
            onSaveBrands={onSaveBrands}
          />
        )}
      </InsightsGate>
    </>
  );
}

export function FixView({
  projectId,
  query,
}: {
  projectId: string;
  query: InsightsQuery;
}) {
  return (
    <InsightsGate query={query}>
      {(data) => {
        if (!data.coverage.comparisonAvailable) {
          return (
            <Note>
              This date range reaches back past the 16 months Search Console
              keeps, so there is no earlier period to compare with. Choose a
              shorter range.
            </Note>
          );
        }
        const nothingLost =
          data.losingPages.length === 0 &&
          data.queryLosses.length === 0 &&
          data.queryGains.length === 0 &&
          data.decaying.length === 0;
        if (nothingLost) {
          return (
            <Note>
              No page or search lost clicks against the previous period. Check
              Grow for ways to win more.
            </Note>
          );
        }
        return (
          <>
            <LosingPages projectId={projectId} pages={data.losingPages} />
            <QueryChanges projectId={projectId} insights={data} />
            <DecayingPages insights={data} />
          </>
        );
      }}
    </InsightsGate>
  );
}

export function GrowView({
  projectId,
  report,
  query,
}: {
  projectId: string;
  report: Report;
  query: InsightsQuery;
}) {
  return (
    <>
      <QuickWins projectId={projectId} report={report} />
      <InsightsGate query={query}>
        {(data) =>
          data.ctrGaps.length === 0 && data.cannibalization.length === 0 ? (
            <Note>
              No searches found where people skip you or where your pages
              compete. Every search closest to page one is in the Explore table.
            </Note>
          ) : (
            <>
              <CtrGaps projectId={projectId} insights={data} />
              <Cannibalization insights={data} />
            </>
          )
        }
      </InsightsGate>
    </>
  );
}
