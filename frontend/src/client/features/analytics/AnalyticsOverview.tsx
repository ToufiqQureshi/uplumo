import { ArrowDown } from "lucide-react";
import { Button } from "@/client/components/ui/button";
import { formatCount } from "@/client/features/search-performance/SearchPerformanceColumns";
import type { AnalyticsSummary } from "./analyticsApi";
import { channelRows, displayPath } from "./presentation";
import { enquiries, mergeEvents, trackingProblem } from "./visitSummary";

const SETUP_STEPS = [
  "Copy the latest tracking code",
  "Paste it in your site header (WordPress: one click in Settings)",
  "Open your site once; this page confirms it",
];

/**
 * The top of Analytics: fix tracking first if it is broken, then who visited,
 * who tried to contact the business, and where they came from.
 */
export function AnalyticsOverview({
  summary,
  range,
  onOpenTracking,
}: {
  summary: AnalyticsSummary;
  range: { from: string; to: string };
  onOpenTracking: () => void;
}) {
  const problem = trackingProblem(summary);
  const leadingChannel = channelRows(summary.channels)[0];
  let leadingPage = summary.topPages[0];
  for (const page of summary.topPages) {
    if (!leadingPage || page.pageviews > leadingPage.pageviews)
      leadingPage = page;
  }
  const leads = enquiries(summary.events);
  const merged = mergeEvents(summary.events).filter((row) => row.spellings > 1);
  const aiVisits = summary.aiSources.reduce(
    (total, source) => total + source.visitors,
    0,
  );
  const rate = summary.visitors
    ? Math.round((100 * leads.total) / summary.visitors)
    : 0;

  return (
    <div className="space-y-4">
      {problem ? (
        <section
          role="status"
          aria-labelledby="fix-tracking"
          className="space-y-3 rounded-2xl border border-amber-300 bg-amber-50 p-5 text-amber-950 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-100"
        >
          <p className="label-mono">Fix this first</p>
          <h2 id="fix-tracking" className="text-lg font-semibold">
            Your tracking is only half working: {formatCount(summary.pageviews)}{" "}
            page views but 0 visits recorded. The numbers below cannot be
            trusted until this is fixed.
          </h2>
          <ol className="grid gap-2 md:grid-cols-3">
            {SETUP_STEPS.map((step, index) => (
              <li
                key={step}
                className="flex gap-3 rounded-lg bg-background p-3 text-sm text-foreground"
              >
                <span className="flex size-6 shrink-0 items-center justify-center rounded-full bg-foreground text-xs text-background">
                  {index + 1}
                </span>
                {step}
              </li>
            ))}
          </ol>
          <Button onClick={onOpenTracking}>Fix tracking in 2 minutes</Button>
        </section>
      ) : null}

      <section
        aria-labelledby="visits-verdict"
        className="space-y-2 rounded-2xl bg-foreground p-6 text-background"
      >
        <p className="label-mono text-background/70">
          {range.from} – {range.to}
        </p>
        <h2
          id="visits-verdict"
          className="text-xl leading-snug font-semibold md:text-2xl"
        >
          {formatCount(summary.visitors)} people visited.{" "}
          {leads.total > 0
            ? `${formatCount(leads.total)} tried to contact you (${leads.rows
                .slice(0, 2)
                .map((row) => row.name)
                .join(", ")}).`
            : "Nobody tapped WhatsApp, call or a form yet."}
          {leadingChannel ? ` Most came from ${leadingChannel.label}.` : ""}
        </h2>
        <p className="text-sm text-background/80 md:text-base">
          {leads.total === 0
            ? "Tag your WhatsApp, call and enquiry buttons so every enquiry is counted."
            : leadingPage
              ? `Your most viewed page is ${displayPath(leadingPage.path)}. Make sure its WhatsApp or call button is visible without scrolling.`
              : "Keep your contact buttons visible on every page."}
          {aiVisits > 0
            ? ` AI assistants sent ${formatCount(aiVisits)} visitors.`
            : ""}
        </p>
      </section>

      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <Tile
          label="Visitors"
          value={formatCount(summary.visitors)}
          hint="People who opened your site"
        />
        <Tile
          label="Enquiries"
          value={formatCount(leads.total)}
          hint="WhatsApp, call and form taps"
        />
        <Tile
          label="Enquiry rate"
          value={`${rate}%`}
          hint="Visitors who tried to contact you"
        />
        <Tile
          label="From AI assistants"
          value={formatCount(aiVisits)}
          hint="ChatGPT, Gemini, Perplexity, Claude"
        />
      </div>

      {merged.length > 0 ? (
        <p className="rounded-lg border p-3 text-sm" role="note">
          We counted {merged.map((row) => `"${row.name}"`).join(", ")} as one
          action: the same button was tagged with different capital letters.
        </p>
      ) : null}

      <a
        href="#visitor-actions"
        className="inline-flex items-center gap-2 text-sm font-medium underline underline-offset-4"
      >
        See every page, source and action <ArrowDown className="size-4" />
      </a>
    </div>
  );
}

function Tile({
  label,
  value,
  hint,
}: {
  label: string;
  value: string;
  hint: string;
}) {
  return (
    <div className="rounded-xl border bg-card p-4">
      <p className="text-xs text-muted-foreground">{label}</p>
      <p className="mt-1 text-2xl font-semibold tabular-nums">{value}</p>
      <p className="mt-1 text-xs text-muted-foreground">{hint}</p>
    </div>
  );
}
