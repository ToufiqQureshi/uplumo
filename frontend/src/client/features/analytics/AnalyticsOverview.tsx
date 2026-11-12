import { Button } from "@/client/components/ui/button";
import { formatCount } from "@/client/features/search-performance/SearchPerformanceColumns";
import type { AnalyticsSummary } from "./analyticsApi";
import {
  aiSourceLabel,
  channelLabel,
  countOf,
  displayPath,
  formatShare,
} from "./presentation";
import { mergeEvents, trackingProblem } from "./visitSummary";

/**
 * Broken tracking: page views arrive but no visits are stitched together, so
 * sources, leads and rates would be wrong. Nothing else is shown until the
 * site runs the current snippet.
 */
export function TrackingProblemCard({
  summary,
  onOpenTracking,
}: {
  summary: AnalyticsSummary;
  onOpenTracking: () => void;
}) {
  if (!trackingProblem(summary)) return null;
  return (
    <section
      role="status"
      aria-labelledby="fix-tracking"
      className="space-y-3 rounded-2xl border border-destructive/40 bg-destructive/5 p-5"
    >
      <p className="label-mono text-destructive">Tracking is half working</p>
      <h2 id="fix-tracking" className="text-lg font-semibold">
        We see {formatCount(summary.pageviews)} page views but no visits, so
        sources, leads and rates stay hidden until this is fixed.
      </h2>
      <p className="text-sm text-muted-foreground">
        Your site runs an old tracking code. Replace it with the current one,
        then open your site once.
      </p>
      <Button onClick={onOpenTracking}>Get the current code</Button>
    </section>
  );
}

/**
 * "+25% (was 40)", "−40% (was 10)", "no change", or null when the previous
 * period has nothing to compare with (often: tracking had not started yet).
 * The old number is shown so a jump from a tiny base reads for what it is.
 */
export function changeLabel(now: number, before: number): string | null {
  if (before === 0) return null;
  const change = (now - before) / before;
  if (Math.abs(change) < 0.005) return "no change";
  return `${change > 0 ? "+" : "−"}${formatShare(Math.abs(change))} (was ${formatCount(before)})`;
}

/**
 * The one-line read of the period, only from facts the data supports: the
 * source that brought the most leads, and the busiest page that converts
 * worse than the site as a whole.
 */
export function leadInsight(summary: AnalyticsSummary): string | null {
  const { leads, visitors } = summary;
  if (!leads.configured || visitors === 0) return null;
  if (leads.visitors === 0) {
    const page = leads.pages[0];
    return page
      ? `No leads yet. ${displayPath(page.path)} gets the most visitors (${formatCount(page.visitors)}); make your lead action easy to find there.`
      : null;
  }
  const parts: string[] = [];
  const most = Math.max(0, ...summary.sourceLeads.map((r) => r.leadVisitors));
  const best = summary.sourceLeads.filter(
    (row) => most > 0 && row.leadVisitors === most,
  );
  if (best.length === 1)
    parts.push(`Most leads came from ${channelLabel(best[0].channel)}.`);
  else if (best.length > 1)
    parts.push(
      `${best.map((row) => channelLabel(row.channel)).join(" and ")} brought the most leads (${formatCount(most)} each).`,
    );
  const siteRate = leads.visitors / visitors;
  const leaky = leads.pages.find(
    (page) =>
      page.visitors >= 2 &&
      page.visitors / visitors >= 0.2 &&
      page.leadVisitors / page.visitors < siteRate / 2,
  );
  if (leaky)
    parts.push(
      `${displayPath(leaky.path)} gets ${formatShare(leaky.visitors / visitors)} of visitors but converts at ${formatShare(leaky.leadVisitors / leaky.visitors)}: start there.`,
    );
  return parts.length ? parts.join(" ") : null;
}

/**
 * The top of Analytics: leads, visitors and the lead rate against the
 * previous period, with one plain sentence on where to act.
 */
export function AnalyticsOverview({
  summary,
  days,
  onPickLeads,
}: {
  summary: AnalyticsSummary;
  days: number;
  onPickLeads: () => void;
}) {
  const { leads } = summary;
  const rate = summary.visitors ? leads.visitors / summary.visitors : 0;
  const insight = leadInsight(summary);
  const merged = mergeEvents(summary.events).filter((row) => row.spellings > 1);

  return (
    <section
      aria-label="This period"
      className="space-y-5 rounded-2xl border bg-card p-5 md:p-6"
    >
      <div className="grid gap-5 sm:grid-cols-3">
        {leads.configured ? (
          <Figure
            label="Leads"
            value={formatCount(leads.visitors)}
            change={changeLabel(leads.visitors, leads.previousVisitors)}
            days={days}
            hint="Visitors who did a lead action, once per day"
          />
        ) : (
          <div className="space-y-2">
            <p className="label-mono text-muted-foreground">Leads</p>
            <p className="text-sm">
              Tell us which action is a lead on your site, and this shows how
              many visitors became one.
            </p>
            <Button size="sm" onClick={onPickLeads}>
              Choose lead actions
            </Button>
          </div>
        )}
        <Figure
          label="Visitors"
          value={formatCount(summary.visitors)}
          change={changeLabel(summary.visitors, summary.previousVisitors)}
          days={days}
          hint="Visitor-days: someone who comes on 3 days counts 3 times"
        />
        {leads.configured ? (
          <Figure
            label="Became leads"
            value={formatShare(rate)}
            hint={`${formatCount(leads.visitors)} of ${formatCount(summary.visitors)} visitors`}
          />
        ) : null}
      </div>
      {insight ? (
        <p className="border-t pt-4 text-sm md:text-base">{insight}</p>
      ) : null}
      {merged.length > 0 ? (
        <p className="text-sm text-muted-foreground" role="note">
          We counted {merged.map((row) => `"${row.name}"`).join(", ")} as one
          action: the same button was tagged with different capital letters.
        </p>
      ) : null}
    </section>
  );
}

function Figure({
  label,
  value,
  change,
  days,
  hint,
}: {
  label: string;
  value: string;
  change?: string | null;
  days?: number;
  hint: string;
}) {
  return (
    <div className="space-y-1">
      <p className="label-mono text-muted-foreground">{label}</p>
      <p className="font-serif text-5xl leading-none tabular-nums">{value}</p>
      {change && days ? (
        <p className="text-sm">
          <span
            className={
              change.startsWith("+") || change === "new"
                ? "font-medium text-success"
                : change.startsWith("−")
                  ? "font-medium text-destructive"
                  : "text-muted-foreground"
            }
          >
            {change}
          </span>{" "}
          <span className="text-muted-foreground">vs previous {days} days</span>
        </p>
      ) : null}
      <p className="text-xs text-muted-foreground">{hint}</p>
    </div>
  );
}

/** Each source that sent visitors, with its leads and lead rate. */
export function LeadSourcesCard({ summary }: { summary: AnalyticsSummary }) {
  const rows = summary.sourceLeads.filter((row) => row.visitors > 0);
  if (rows.length === 0) return null;
  const { configured } = summary.leads;
  const untied = summary.leads.untiedVisitors;
  return (
    <section
      aria-labelledby="lead-sources"
      className="rounded-2xl border bg-card p-5"
    >
      <h2 id="lead-sources" className="font-serif text-2xl">
        {configured ? "Sources that bring leads" : "Where visitors come from"}
      </h2>
      <table className="mt-3 w-full text-sm">
        <thead>
          <tr className="text-left text-xs text-muted-foreground">
            <th className="py-2 font-normal">Source</th>
            <th className="py-2 text-right font-normal">Visitors</th>
            {configured ? (
              <>
                <th className="py-2 text-right font-normal">Leads</th>
                <th className="py-2 text-right font-normal">Lead rate</th>
              </>
            ) : null}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.channel} className="border-t">
              <td className="py-2">{channelLabel(row.channel)}</td>
              <td className="py-2 text-right tabular-nums">
                {formatCount(row.visitors)}
              </td>
              {configured ? (
                <>
                  <td className="py-2 text-right tabular-nums">
                    {formatCount(row.leadVisitors)}
                  </td>
                  <td className="py-2 text-right tabular-nums">
                    {formatShare(row.leadVisitors / row.visitors)}
                  </td>
                </>
              ) : null}
            </tr>
          ))}
          {configured && untied > 0 ? (
            <tr className="border-t text-muted-foreground">
              <td className="py-2">Source not known</td>
              <td className="py-2 text-right">—</td>
              <td className="py-2 text-right tabular-nums">
                {formatCount(untied)}
              </td>
              <td className="py-2 text-right">—</td>
            </tr>
          ) : null}
        </tbody>
      </table>
      <p className="mt-3 text-xs text-muted-foreground">
        A visit belongs to the source of its first page.
        {configured && untied > 0
          ? " Leads sent by an old tracking code carry no visit, so their source is not known."
          : ""}
      </p>
    </section>
  );
}

/** The busiest pages and the visitors who became leads on them. */
export function LeadPagesCard({ summary }: { summary: AnalyticsSummary }) {
  const { leads } = summary;
  if (!leads.configured || leads.pages.length === 0) return null;
  const most = Math.max(...leads.pages.map((page) => page.visitors));
  return (
    <section
      aria-labelledby="lead-pages"
      className="rounded-2xl border bg-card p-5"
    >
      <h2 id="lead-pages" className="font-serif text-2xl">
        Busy pages and their leads
      </h2>
      <ul className="mt-3 space-y-3">
        {leads.pages.map((page) => (
          <li key={page.path} className="space-y-1.5">
            <div className="flex justify-between gap-3 text-sm">
              <span className="truncate font-mono">
                {displayPath(page.path)}
              </span>
              <span className="shrink-0 text-muted-foreground tabular-nums">
                {countOf(page.visitors, "visitor", "visitors")} ·{" "}
                {countOf(page.leadVisitors, "lead", "leads")}
              </span>
            </div>
            <div
              className="flex h-2 overflow-hidden rounded bg-muted"
              aria-hidden
            >
              <div
                className="bg-muted-foreground/40"
                style={{
                  width: `${((page.visitors - page.leadVisitors) / most) * 100}%`,
                }}
              />
              <div
                className="bg-primary"
                style={{ width: `${(page.leadVisitors / most) * 100}%` }}
              />
            </div>
          </li>
        ))}
      </ul>
      <p className="mt-3 text-xs text-muted-foreground">
        Leads counted on the page where the lead action happened.
      </p>
    </section>
  );
}

/** Visitors sent by AI assistants and where they landed; hidden with none. */
export function AiVisitorsCard({ summary }: { summary: AnalyticsSummary }) {
  const total = summary.aiSources.reduce((sum, row) => sum + row.visitors, 0);
  const configured = summary.leads.configured;
  const leads = summary.aiLandings.reduce(
    (sum, row) => sum + row.leadVisitors,
    0,
  );
  if (total === 0) return null;
  return (
    <section
      aria-labelledby="ai-visitors"
      className="space-y-3 rounded-2xl border bg-card p-5"
    >
      <h2 id="ai-visitors" className="font-serif text-2xl">
        Visitors from AI assistants
      </h2>
      <p className="font-serif text-5xl leading-none tabular-nums">
        {formatCount(total)}
      </p>
      {configured ? (
        <p className="text-sm text-muted-foreground">
          {countOf(leads, "became a lead", "became leads")} in the same visit.
        </p>
      ) : null}
      <ul className="space-y-2 text-sm">
        {summary.aiLandings.map((row) => (
          <li
            key={`${row.source}:${row.path}`}
            className="flex justify-between gap-3 border-t pt-2"
          >
            <span className="min-w-0 truncate">
              {aiSourceLabel(row.source)} →{" "}
              <span className="font-mono">{displayPath(row.path)}</span>
            </span>
            <span className="shrink-0 text-muted-foreground tabular-nums">
              {formatCount(row.visitors)}
              {configured
                ? ` · ${countOf(row.leadVisitors, "lead", "leads")}`
                : ""}
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}
