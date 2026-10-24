import {
  ArrowDownRight,
  ArrowUpRight,
  Search,
  Activity,
  HelpCircle,
  Lightbulb,
  Target,
  TrendingUp,
  AlertTriangle,
} from "lucide-react";
import { intelligenceMetrics } from "./intelligenceMetrics";
import { growthCount, growthRate } from "./growthFormat";
import type { GrowthReport } from "./growthApi";
import { Badge } from "@/client/components/ui/badge";

export function IntelligenceMetrics({ report }: { report: GrowthReport }) {
  const current = intelligenceMetrics(report);
  const previous = intelligenceMetrics(report, true);

  // Compute overall business insights for the non-technical summary
  const clickDelta =
    previous.clicks !== null && previous.clicks > 0 && current.clicks !== null
      ? ((current.clicks - previous.clicks) / previous.clicks) * 100
      : null;

  const sessionDelta =
    current.comparableAnalytics &&
    previous.sessions !== null &&
    previous.sessions > 0 &&
    current.sessions !== null
      ? ((current.sessions - previous.sessions) / previous.sessions) * 100
      : null;

  const pageCountWithFindings = report.pages.filter(
    (p) => p.findings.length > 0,
  ).length;

  const metrics = [
    {
      label: "Google Clicks",
      key: "clicks",
      value: growthCount(current.clicks),
      source: "GSC",
      hint: "Actual website visitors arriving from Google search results.",
      actionAdvice:
        clickDelta !== null && clickDelta < -5
          ? "Traffic is down. Review candidate pages below to recover lost clicks."
          : "Higher clicks mean more search visitors reaching your website.",
    },
    {
      label: "Search Impressions",
      key: "impressions",
      value: growthCount(current.impressions),
      source: "GSC",
      hint: "How many times your site was seen in Google search results.",
      actionAdvice:
        "High impressions with low clicks means you should improve your page title & meta description.",
    },
    {
      label: "Average Search Position",
      key: "position",
      value:
        current.position === null ? "Unavailable" : current.position.toFixed(1),
      source: "GSC",
      hint: "Your average rank on Google. Positions 1–10 are Page 1.",
      actionAdvice:
        current.position !== null &&
        current.position >= 5 &&
        current.position <= 20
          ? "Positions 5–20 are 'Striking Distance'—small content improvements can move you to Page 1!"
          : "Position closer to 1 gets significantly higher click rates.",
    },
    {
      label: "Organic Sessions",
      key: "sessions",
      value: growthCount(current.sessions),
      source: "GA4",
      hint: "Website visits recorded by Analytics coming from Google Search.",
      actionAdvice:
        sessionDelta !== null && sessionDelta < -5
          ? "Organic visits declined. Check if key landing pages lost ranking."
          : "Organic visits represent real people browsing your website.",
    },
    {
      label: "Engagement Rate",
      key: "engagement",
      value: growthRate(current.engagement),
      source: "GA4",
      hint: "Percentage of visitors who stayed and interacted on your site.",
      actionAdvice:
        current.engagement !== null && current.engagement < 0.4
          ? "Low engagement (<40%). Make your page heading and main offer clearer."
          : "Good engagement shows visitors find your content relevant.",
    },
    {
      label: "All Key Events",
      key: "events",
      value: growthCount(current.events),
      source: "GA4",
      hint: "Total valuable actions (form fills, button clicks, purchases) taken by visitors.",
      actionAdvice:
        "Measures whether search visitors are taking meaningful actions on your site.",
    },
    {
      label: report.goalEvent
        ? `${report.goalEvent} Rate`
        : "Session Outcome Rate",
      key: "outcomeRate",
      value: growthRate(current.outcomeRate),
      source: "GA4",
      hint: "Percentage of search visits that completed your selected business goal.",
      actionAdvice:
        "Higher conversion rate means more leads or sales from the same search traffic.",
    },
    {
      label: "Purchase Revenue",
      key: "revenue",
      value:
        current.revenue === null
          ? "—"
          : `${current.revenue.toLocaleString(undefined, { maximumFractionDigits: 2 })} ${report.currency}`,
      source: "GA4",
      hint: "Total sales value generated directly from organic Google visitors.",
      actionAdvice:
        "Tracks direct revenue impact from organic search visitors.",
    },
  ] as const;

  return (
    <section
      aria-label="Combined search and business metrics"
      className="space-y-4"
    >
      {/* Executive Plain-Language Decision Guide Banner */}
      <div className="rounded-xl border border-violet-500/20 bg-gradient-to-r from-violet-500/10 via-purple-500/5 to-transparent p-4 sm:p-5">
        <div className="flex items-center gap-2 font-semibold text-foreground">
          <Lightbulb className="size-5 text-violet-500 shrink-0" />
          <span>Executive Decision Guide: What your numbers mean</span>
        </div>
        <p className="mt-1 text-xs text-muted-foreground">
          Plain-language summary of your Search Console & Analytics performance
          to guide your next decision.
        </p>

        <div className="mt-3 grid gap-3 sm:grid-cols-3">
          <div className="rounded-lg border border-border/60 bg-card/80 p-3 text-xs space-y-1">
            <div className="flex items-center gap-1.5 font-medium text-foreground">
              {clickDelta !== null && clickDelta < 0 ? (
                <AlertTriangle className="size-3.5 text-amber-500 shrink-0" />
              ) : (
                <TrendingUp className="size-3.5 text-emerald-500 shrink-0" />
              )}
              1. Search Traffic & Visibility
            </div>
            <p className="text-muted-foreground leading-relaxed">
              {clickDelta === null
                ? "Your Google search clicks and impressions give a clear picture of user demand."
                : clickDelta < 0
                  ? `Google clicks dropped by ${Math.abs(clickDelta).toFixed(1)}%. Key pages need content or title updates.`
                  : `Google clicks grew by +${clickDelta.toFixed(1)}%. Your search reach is expanding nicely.`}
            </p>
          </div>

          <div className="rounded-lg border border-border/60 bg-card/80 p-3 text-xs space-y-1">
            <div className="flex items-center gap-1.5 font-medium text-foreground">
              <Activity className="size-3.5 text-blue-500 shrink-0" />
              2. Visitor Conversion Quality
            </div>
            <p className="text-muted-foreground leading-relaxed">
              {current.engagement !== null
                ? `Engagement is ${growthRate(current.engagement)} with ${growthCount(current.events)} total key events recorded.`
                : "Tracks whether visitors arriving from Google find what they need and convert."}
            </p>
          </div>

          <div className="rounded-lg border border-border/60 bg-card/80 p-3 text-xs space-y-1">
            <div className="flex items-center gap-1.5 font-medium text-foreground">
              <Target className="size-3.5 text-violet-500 shrink-0" />
              3. Recommended Immediate Action
            </div>
            <p className="text-muted-foreground leading-relaxed">
              {pageCountWithFindings > 0
                ? `Focus on the ${pageCountWithFindings} candidate page${pageCountWithFindings > 1 ? "s" : ""} identified below with evidence-backed recommendations.`
                : "No urgent click losses detected. Review your striking distance queries for quick wins."}
            </p>
          </div>
        </div>
      </div>

      {/* Metrics Grid with Plain-Language Guidance */}
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
              className="flex flex-col justify-between min-w-0 rounded-xl border border-border bg-card p-4 space-y-2"
            >
              <div>
                <div className="flex items-center justify-between gap-2 text-xs text-muted-foreground">
                  <span className="font-medium text-foreground break-words flex items-center gap-1">
                    {metric.label}
                  </span>
                  {metric.source === "GSC" ? (
                    <Badge
                      variant="outline"
                      className="text-[10px] px-1.5 py-0 h-4 border-violet-500/30 text-violet-600 dark:text-violet-400"
                    >
                      <Search className="size-2.5 mr-1 inline" /> GSC
                    </Badge>
                  ) : (
                    <Badge
                      variant="outline"
                      className="text-[10px] px-1.5 py-0 h-4 border-blue-500/30 text-blue-600 dark:text-blue-400"
                    >
                      <Activity className="size-2.5 mr-1 inline" /> GA4
                    </Badge>
                  )}
                </div>

                <p className="mt-2 break-all text-2xl font-semibold tabular-nums">
                  {metric.value}
                </p>

                <div className="mt-1 flex items-center gap-1 text-xs text-muted-foreground">
                  {delta === null ? (
                    <span className="text-muted-foreground/70">
                      Comparison unavailable
                    </span>
                  ) : (
                    <span
                      className={`inline-flex items-center gap-0.5 font-medium ${
                        delta >= 0
                          ? "text-emerald-600 dark:text-emerald-400"
                          : "text-amber-600 dark:text-amber-400"
                      }`}
                    >
                      {delta >= 0 ? (
                        <ArrowUpRight className="size-3.5" />
                      ) : (
                        <ArrowDownRight className="size-3.5" />
                      )}
                      {delta > 0 ? "+" : ""}
                      {delta.toFixed(1)}% vs prior window
                    </span>
                  )}
                </div>
              </div>

              {/* Plain-Language What This Means Callout */}
              <div className="pt-2 border-t border-border/50 text-[11px] text-muted-foreground space-y-1">
                <p className="leading-snug">
                  <HelpCircle className="size-3 inline mr-1 text-muted-foreground/70" />
                  {metric.hint}
                </p>
                <p className="text-foreground/80 font-medium text-[10px] leading-tight">
                  💡 {metric.actionAdvice}
                </p>
              </div>
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
