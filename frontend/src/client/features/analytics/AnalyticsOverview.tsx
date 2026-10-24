import { ArrowDown, ArrowRight, Activity } from "lucide-react";
import { Button } from "@/client/components/ui/button";
import { formatCount } from "@/client/features/search-performance/SearchPerformanceColumns";
import type { AnalyticsSummary } from "./analyticsApi";
import { channelRows, displayPath } from "./presentation";

export function AnalyticsOverview({
  summary,
  range,
  onOpenTracking,
}: {
  summary: AnalyticsSummary;
  range: { from: string; to: string };
  onOpenTracking: () => void;
}) {
  const leadingChannel = channelRows(summary.channels)[0];
  let leadingPage = summary.topPages[0];
  for (const page of summary.topPages) {
    if (!leadingPage || page.pageviews > leadingPage.pageviews)
      leadingPage = page;
  }
  const eventCount = summary.events.reduce(
    (total, event) => total + event.count,
    0,
  );
  const missingSessions = summary.pageviews > 0 && summary.sessions === 0;
  const eventNames = summary.events.map((event) => event.name.toLowerCase());
  const aiVisits = summary.aiSources.reduce(
    (total, source) => total + source.visitors,
    0,
  );
  const inconsistentNames = new Set(eventNames).size < eventNames.length;

  return (
    <section
      aria-label="Traffic overview"
      className="overflow-hidden rounded-xl border border-border bg-card"
    >
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-border px-5 py-3 text-xs text-muted-foreground">
        <span className="flex items-center gap-2">
          <Activity className="size-4" /> First-party website activity
        </span>
        <span>
          {range.from} – {range.to} · UTC · Today is partial
        </span>
      </div>
      <div className="space-y-5 p-5 md:p-6">
        <div>
          <h2 className="text-xl font-semibold">Your traffic at a glance</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            {formatCount(summary.pageviews)} pageviews and{" "}
            {formatCount(eventCount)} recorded custom events in this period.
            Events are not verified leads or sales.
          </p>
        </div>
        <div className="grid gap-4 md:grid-cols-3">
          <div className="min-w-0 rounded-lg bg-muted/50 p-4">
            <p className="text-xs font-medium text-muted-foreground">
              Leading traffic source
            </p>
            <p className="mt-1 font-semibold">
              {leadingChannel?.label ?? "Source unavailable"}
            </p>
            <p className="mt-1 text-xs text-muted-foreground">
              {leadingChannel
                ? `${formatCount(leadingChannel.visitors)} visitor-days attributed to this source`
                : "No source attribution recorded for this period."}
            </p>
          </div>
          <div className="min-w-0 rounded-lg bg-muted/50 p-4">
            <p className="text-xs font-medium text-muted-foreground">
              Most viewed page
            </p>
            <p className="mt-1 break-all font-semibold">
              {leadingPage ? displayPath(leadingPage.path) : "Page unavailable"}
            </p>
            <p className="mt-1 text-xs text-muted-foreground">
              {leadingPage
                ? `${formatCount(leadingPage.pageviews)} pageviews · Review the actions visitors take on this page below.`
                : "No page breakdown recorded for this period."}
            </p>
          </div>
          <div className="min-w-0 rounded-lg bg-muted/50 p-4">
            <p className="text-xs font-medium text-muted-foreground">
              AI referrals
            </p>
            <p className="mt-1 font-semibold">
              {formatCount(aiVisits)} visitor-days
            </p>
            <p className="mt-1 text-xs text-muted-foreground">
              Identified assistant referrals. This does not measure how often AI
              mentions your site.
            </p>
          </div>
        </div>
        {missingSessions || inconsistentNames ? (
          <div className="rounded-lg border border-border p-4" role="status">
            <p className="font-medium">Check your measurement</p>
            {missingSessions ? (
              <p className="mt-1 text-sm text-muted-foreground">
                Pageviews were received, but no sessions were recorded.
                Session-based engagement and attribution are unavailable for
                these visits. Check the installed snippet and test a visit.
              </p>
            ) : null}
            {inconsistentNames ? (
              <p className="mt-1 text-sm text-muted-foreground">
                Some event names differ only by letter case. Check your button
                tags so the same action uses one consistent name. Counts below
                remain separate.
              </p>
            ) : null}
            <Button
              className="mt-3"
              size="sm"
              variant="outline"
              onClick={onOpenTracking}
            >
              Review tracking setup <ArrowRight />
            </Button>
          </div>
        ) : (
          <a
            href="#visitor-actions"
            className="inline-flex items-center gap-2 text-sm font-medium underline underline-offset-4"
          >
            {eventCount > 0
              ? "Explore recorded visitor actions"
              : "See how to start measuring visitor actions"}
            <ArrowDown className="size-4" />
          </a>
        )}
      </div>
    </section>
  );
}
