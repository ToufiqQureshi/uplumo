import { ShieldCheck, RefreshCw } from "lucide-react";
import { Button } from "@/client/components/ui/button";
import type { AnalyticsSummary } from "./analyticsApi";

export function TrackingStatusCard({
  summary,
  onCheck,
  checking,
}: {
  summary?: AnalyticsSummary;
  onCheck: () => void;
  checking: boolean;
}) {
  const lastDay = summary?.series
    .filter((day) => day.pageviews > 0)
    .at(-1)?.date;
  return (
    <section
      aria-label="Tracking verification"
      className="rounded-xl border border-border bg-card p-5"
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex items-center gap-2">
          <ShieldCheck className="size-5 text-violet-500" />
          <h2 className="font-semibold">Verify your tracking</h2>
        </div>
        <Button
          variant="outline"
          size="sm"
          disabled={checking}
          onClick={onCheck}
        >
          <RefreshCw />
          {checking ? "Checking…" : "Check received data"}
        </Button>
      </div>
      <p className="mt-2 text-sm text-muted-foreground">
        {lastDay
          ? `Latest day with a pageview in this report: ${lastDay} (UTC).`
          : "No pageview recorded in the selected period."}{" "}
        This checks stored data; it does not prove every page or visitor is
        tracked.
      </p>
      <ol className="mt-4 grid list-inside list-decimal gap-3 text-sm md:grid-cols-3">
        <li>Install one current snippet per page, then visit your website.</li>
        <li>
          Navigate to another page and click a tagged button or an outbound
          link.
        </li>
        <li>
          Check received data and review pageviews, sessions and destinations
          below.
        </li>
      </ol>
      <p className="mt-3 text-xs text-muted-foreground">
        Consent settings, Do Not Track, Global Privacy Control and blockers can
        prevent collection. A click is not a completed booking. For confirmed
        outcomes, emit a named event only after server-confirmed success.
      </p>
    </section>
  );
}
