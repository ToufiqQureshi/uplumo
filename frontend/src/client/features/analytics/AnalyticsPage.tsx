import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Code } from "lucide-react";
import {
  AiVisitorsCard,
  AnalyticsOverview,
  LeadPagesCard,
  LeadSourcesCard,
  TrackingProblemCard,
} from "./AnalyticsOverview";
import { LEAD_ACTIONS_ID, LeadActionsCard } from "./LeadActionsCard";
import { trackingProblem } from "./visitSummary";
import { QueryError } from "@/client/components/QueryState";
import { SegmentedToggle } from "@/client/components/SegmentedToggle";
import { SkeletonStatGrid } from "@/client/components/SkeletonPresets";
import { StatTile } from "@/client/components/StatTile";
import { Button } from "@/client/components/ui/button";
import { Card, CardContent } from "@/client/components/ui/card";
import { Skeleton } from "@/client/components/ui/skeleton";
import { ProjectPageHeader } from "@/client/features/projects/ProjectPageHeader";
import { formatCount } from "@/client/features/search-performance/SearchPerformanceColumns";
import {
  analyticsSummaryQueryOptions,
  RANGE_DAYS,
  rangeForDays,
  type AnalyticsSummary,
  type RangeDays,
} from "./analyticsApi";
import {
  CampaignPerformanceCard,
  CountListCard,
  deviceRows,
  TrafficChartCard,
} from "./AnalyticsCards";
import { displayPath, formatDuration, formatShare } from "./presentation";
import { TrackingCodeCard } from "./TrackingCodeCard";
import { CountriesCard } from "./CountriesCard";
import {
  DailyAudienceCard,
  ClickDestinationsCard,
} from "./AudienceJourneyCards";

const RANGE_ITEMS = RANGE_DAYS.map((days) => ({
  value: `${days}` as const,
  icon: null,
  label: `${days} days`,
}));

export function AnalyticsPage({
  projectId,
  days,
  onDaysChange,
}: {
  projectId: string;
  days: RangeDays;
  onDaysChange: (days: RangeDays) => void;
}) {
  // Day strings: the query key stays the same until the date changes.
  const summaryQuery = useQuery(
    analyticsSummaryQueryOptions(projectId, rangeForDays(days, new Date())),
  );
  const [showCode, setShowCode] = useState(false);
  // Kept here so a range change, which shows the skeleton for a moment,
  // does not close "All numbers".
  const [numbersOpen, setNumbersOpen] = useState(false);
  const summary = summaryQuery.data;
  const hasVisits = summary !== undefined && summary.pageviews > 0;

  return (
    <div className="h-full overflow-auto px-4 py-4 pb-24 md:px-6 md:py-6 md:pb-8">
      <div className="mx-auto max-w-7xl space-y-6">
        <ProjectPageHeader
          projectId={projectId}
          title="Analytics"
          actions={
            <>
              <SegmentedToggle
                items={RANGE_ITEMS}
                value={`${days}`}
                onChange={(value) => {
                  const picked = RANGE_DAYS.find((d) => `${d}` === value);
                  if (picked) onDaysChange(picked);
                }}
                showLabels
              />
              {hasVisits ? (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setShowCode((shown) => !shown)}
                  aria-expanded={showCode}
                >
                  <Code />
                  Tracking code
                </Button>
              ) : null}
            </>
          }
        />
        <p className="text-sm text-muted-foreground">
          Understand where visits come from, what gets read, and which actions
          get recorded.
        </p>
        {summaryQuery.isPending ? (
          <DashboardSkeleton />
        ) : summaryQuery.isError && !summary ? (
          <QueryError
            error={summaryQuery.error}
            fallback="We couldn't load your analytics."
            onRetry={() => void summaryQuery.refetch()}
            isRetrying={summaryQuery.isFetching}
          />
        ) : !hasVisits ? (
          <SetupState
            projectId={projectId}
            days={days}
            onCheckAgain={() => void summaryQuery.refetch()}
            isChecking={summaryQuery.isFetching}
          />
        ) : (
          <>
            {summaryQuery.isError ? (
              <QueryError
                error={summaryQuery.error}
                fallback="Showing the last loaded analytics. Refresh failed."
                onRetry={() => void summaryQuery.refetch()}
                isRetrying={summaryQuery.isFetching}
              />
            ) : null}
            <TrackingProblemCard
              summary={summary}
              onOpenTracking={() => setShowCode(true)}
            />
            {showCode ? (
              <TrackingCodeCard key={projectId} projectId={projectId} />
            ) : null}
            {trackingProblem(summary) ? null : (
              <>
                <AnalyticsOverview
                  summary={summary}
                  days={days}
                  onPickLeads={() =>
                    document
                      .getElementById(LEAD_ACTIONS_ID)
                      ?.scrollIntoView({ behavior: "smooth" })
                  }
                />
                <LeadActionsCard
                  projectId={projectId}
                  events={summary.events}
                />
                <div
                  className={
                    summary.aiSources.some((row) => row.visitors > 0)
                      ? "grid gap-6 lg:grid-cols-2"
                      : "grid gap-6"
                  }
                >
                  <LeadSourcesCard summary={summary} />
                  <AiVisitorsCard summary={summary} />
                </div>
                <LeadPagesCard summary={summary} />
                <Dashboard
                  summary={summary}
                  projectId={projectId}
                  range={rangeForDays(days, new Date())}
                  open={numbersOpen}
                  onOpenChange={setNumbersOpen}
                />
              </>
            )}
          </>
        )}
      </div>
    </div>
  );
}

function SetupState({
  projectId,
  days,
  onCheckAgain,
  isChecking,
}: {
  projectId: string;
  days: RangeDays;
  onCheckAgain: () => void;
  isChecking: boolean;
}) {
  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        No visits in the last {days} days. Install the tracking code to see who
        visits your site, which pages they read, and how many arrive from AI
        assistants.
      </p>
      <TrackingCodeCard key={projectId} projectId={projectId} />
      <Button variant="outline" onClick={onCheckAgain} disabled={isChecking}>
        {isChecking ? "Checking…" : "I've installed it, check again"}
      </Button>
    </div>
  );
}

function Dashboard({
  summary,
  projectId,
  range,
  open,
  onOpenChange,
}: {
  summary: AnalyticsSummary;
  projectId: string;
  range: { from: string; to: string };
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  // The reports inside load their own data (clicks, countries); they mount
  // only once the section is opened, so a closed section asks for nothing.
  return (
    <details
      className="rounded-2xl border bg-card p-5"
      open={open}
      onToggle={(event) => onOpenChange(event.currentTarget.open)}
    >
      <summary className="cursor-pointer font-medium">
        All numbers: visits, pages, actions, audience and campaigns
      </summary>
      {open ? (
        <div className="mt-5 space-y-6">
          <Card>
            <CardContent className="grid grid-cols-2 gap-4 md:grid-cols-3 xl:grid-cols-5">
              <StatTile
                label="Visitors"
                value={formatCount(summary.visitors)}
                hint="Anonymous visitor-days, not people"
              />
              <StatTile
                label="Sessions"
                value={formatCount(summary.sessions)}
                hint="Tab-scoped visits"
              />
              <StatTile
                label="Pageviews"
                value={formatCount(summary.pageviews)}
              />
              <StatTile
                label="Avg visible time"
                value={
                  summary.sessions > 0
                    ? formatDuration(summary.avgEngagementMs)
                    : "—"
                }
                hint="Visible page time per session"
              />
              <StatTile
                label="Bounce rate"
                value={
                  summary.sessions > 0 ? formatShare(summary.bounceRate) : "—"
                }
                hint={`${formatCount(summary.engagedSessions)} engaged sessions`}
              />
            </CardContent>
          </Card>
          <TrafficChartCard series={summary.series} />
          <CountListCard
            title="Top pages"
            unit="Pageviews"
            emptyTitle="No pages viewed in this period"
            rows={summary.topPages.map((row) => ({
              key: row.path,
              label: `${displayPath(row.path)} · avg ${formatDuration(row.avgEngagementMs)}`,
              count: row.pageviews,
            }))}
          />
          <section id="visitor-actions" className="space-y-3 scroll-mt-4">
            <h2 className="text-lg font-semibold">Actions visitors take</h2>
            <p className="text-sm text-muted-foreground">
              Recorded clicks and custom events show activity. They do not
              confirm an enquiry, booking or sale.
            </p>
            <ClickDestinationsCard projectId={projectId} range={range} />
            <CountListCard
              title="Custom events"
              unit="Events"
              emptyTitle="No custom events recorded. Tag important buttons in Tracking code to start measuring actions."
              rows={summary.events.map((row) => ({
                key: row.name,
                label: `${row.name} · ${row.visitors} visitor-days`,
                count: row.count,
              }))}
            />
          </section>
          <div className="space-y-6">
            <DailyAudienceCard days={summary.audience} />
            <div className="grid gap-6 lg:grid-cols-2">
              <CampaignPerformanceCard campaigns={summary.campaigns} />
              <CountriesCard projectId={projectId} range={range} />
              <CountListCard
                title="Landing pages"
                unit="Sessions"
                emptyTitle="No session entry pages yet"
                rows={summary.entries.map((row) => ({
                  key: row.path,
                  label: `${displayPath(row.path)} · avg ${formatDuration(row.avgEngagementMs)}`,
                  count: row.sessions,
                }))}
              />
              <CountListCard
                title="Exit pages"
                unit="Sessions"
                emptyTitle="No session exit pages yet"
                rows={summary.exits.map((row) => ({
                  key: row.path,
                  label: `${displayPath(row.path)} · avg ${formatDuration(row.avgEngagementMs)}`,
                  count: row.sessions,
                }))}
              />
              <CountListCard
                title="Referring websites"
                unit="Visitors"
                emptyTitle="No visitors came from other websites"
                rows={summary.referrers.map((row) => ({
                  key: row.host,
                  label: row.host,
                  count: row.visitors,
                }))}
              />
              <CountListCard
                title="Devices"
                unit="Visitors"
                emptyTitle="No device data in this period"
                rows={deviceRows(summary.devices)}
              />
            </div>
          </div>
        </div>
      ) : null}
    </details>
  );
}

function DashboardSkeleton() {
  return (
    <div className="space-y-6" aria-busy>
      <SkeletonStatGrid count={5} className="md:grid-cols-3 xl:grid-cols-5" />
      <div className="grid gap-6 lg:grid-cols-3">
        <Skeleton className="h-80 lg:col-span-2" />
        <Skeleton className="h-80" />
      </div>
      <div className="grid gap-6 lg:grid-cols-2">
        {Array.from({ length: 4 }, (_, index) => (
          <Skeleton key={index} className="h-56" />
        ))}
      </div>
    </div>
  );
}
