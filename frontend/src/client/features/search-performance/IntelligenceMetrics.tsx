import { ArrowDownRight, ArrowUpRight, Search, Activity } from "lucide-react";
import { intelligenceMetrics } from "./intelligenceMetrics";
import { growthCount, growthRate } from "./growthFormat";
import type { GrowthReport } from "./growthApi";

export function IntelligenceMetrics({ report }: { report: GrowthReport }) {
  const current = intelligenceMetrics(report);
  const previous = intelligenceMetrics(report, true);
  const metrics = [
    {
      label: "Google clicks",
      key: "clicks",
      value: growthCount(current.clicks),
      source: "GSC",
    },
    {
      label: "Search impressions",
      key: "impressions",
      value: growthCount(current.impressions),
      source: "GSC",
    },
    {
      label: "Average search position",
      key: "position",
      value:
        current.position === null ? "Unavailable" : current.position.toFixed(1),
      source: "GSC",
    },
    {
      label: "Organic sessions",
      key: "sessions",
      value: growthCount(current.sessions),
      source: "GA4",
    },
    {
      label: "Engagement rate",
      key: "engagement",
      value: growthRate(current.engagement),
      source: "GA4",
    },
    {
      label: "All key events",
      key: "events",
      value: growthCount(current.events),
      source: "GA4",
    },
    {
      label: report.goalEvent
        ? `${report.goalEvent} session rate`
        : "Session key-event rate",
      key: "outcomeRate",
      value: growthRate(current.outcomeRate),
      source: "GA4",
    },
    {
      label: "Recorded purchase revenue",
      key: "revenue",
      value:
        current.revenue === null
          ? "—"
          : `${current.revenue.toLocaleString(undefined, { maximumFractionDigits: 2 })} ${report.currency}`,
      source: "GA4",
    },
  ] as const;
  return (
    <section
      aria-label="Combined search and business metrics"
      className="space-y-3"
    >
      <div className="grid grid-cols-2 gap-3 xl:grid-cols-4">
        {metrics.map((metric) => {
          const value = current[metric.key];
          const before = previous[metric.key];
          const comparable =
            metric.source === "GSC" || current.comparableAnalytics;
          const delta =
            comparable && before !== null && before > 0 && value !== null
              ? ((value - before) / before) * 100
              : null;
          return (
            <div
              key={metric.key}
              className="min-w-0 rounded-xl border border-border bg-card p-4"
            >
              <div className="flex items-center justify-between gap-2 text-xs text-muted-foreground">
                <span className="break-words">{metric.label}</span>
                {metric.source === "GSC" ? (
                  <Search className="size-3.5 shrink-0 text-violet-500" />
                ) : (
                  <Activity className="size-3.5 shrink-0 text-blue-500" />
                )}
              </div>
              <p className="mt-3 break-all text-2xl font-semibold tabular-nums">
                {metric.value}
              </p>
              <p className="mt-2 flex items-center gap-1 text-xs text-muted-foreground">
                {delta === null ? (
                  "Comparison unavailable"
                ) : (
                  <>
                    {delta >= 0 ? (
                      <ArrowUpRight className="size-3.5" />
                    ) : (
                      <ArrowDownRight className="size-3.5" />
                    )}
                    {delta > 0 ? "+" : ""}
                    {delta.toFixed(1)}% vs prior window
                  </>
                )}
              </p>
            </div>
          );
        })}
      </div>
      <p className="text-xs text-muted-foreground">
        Returned-page coverage: {report.pages.length} GSC pages;{" "}
        {current.joinedPages} matched GA4 landing pages. These are not
        whole-site totals. Rates are session-weighted; key events are not
        verified leads.{" "}
        {report.limited
          ? "Google marked the report limited; business metrics are withheld."
          : null}
      </p>
    </section>
  );
}
