import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Button } from "@/client/components/ui/button";
import { getGa4DashboardReport } from "@/client/features/dashboard/ga4DashboardApi";
import { formatCount } from "./SearchPerformanceColumns";

/**
 * What Google search visitors did after they arrived, from GA4's organic
 * search report: visits, engagement and key events against the previous
 * period. Shares the dashboard's cached query (free Google read, no credits).
 */
export function Ga4OrganicCard({ projectId }: { projectId: string }) {
  const query = useQuery({
    queryKey: ["dashboardGa4Report", projectId],
    queryFn: () => getGa4DashboardReport(projectId),
    staleTime: 5 * 60_000,
    retry: false,
  });
  if (query.isPending) {
    return (
      <section className="rounded-xl border bg-card p-5 text-sm text-muted-foreground">
        Loading what Google visitors did on your site (GA4)…
      </section>
    );
  }
  if (query.isError) {
    return (
      <section className="rounded-xl border bg-card p-5 text-sm text-muted-foreground">
        Google Analytics did not answer right now.{" "}
        <button
          type="button"
          className="underline underline-offset-4"
          onClick={() => void query.refetch()}
        >
          Try again
        </button>
      </section>
    );
  }
  const report = query.data;
  if (!report.connected) {
    return (
      <section className="flex flex-wrap items-center justify-between gap-3 rounded-xl border bg-card p-5">
        <div>
          <h2 className="text-lg font-semibold">
            What happens after the click
          </h2>
          <p className="text-sm text-muted-foreground">
            Connect Google Analytics to see if Google visitors stay, enquire or
            buy.
          </p>
        </div>
        <Button
          variant="outline"
          nativeButton={false}
          render={
            <Link
              to="/p/$projectId/settings/integrations"
              params={{ projectId }}
            />
          }
        >
          Connect GA4
        </Button>
      </section>
    );
  }
  const { totals, prevTotals } = report;
  const tiles = [
    {
      label: "Visits from Google",
      now: totals.sessions,
      before: prevTotals.sessions,
      format: formatCount,
    },
    {
      label: "Engaged visits",
      now: totals.engagementRate,
      before: prevTotals.engagementRate,
      format: (value: number) => `${Math.round(value * 100)}%`,
    },
    {
      label: "Key events",
      now: totals.keyEvents,
      before: prevTotals.keyEvents,
      format: formatCount,
    },
  ];
  const perHundred =
    totals.sessions && totals.keyEvents !== null
      ? (100 * totals.keyEvents) / totals.sessions
      : null;
  return (
    <section
      aria-labelledby="ga4-organic"
      className="space-y-3 rounded-xl border bg-card p-5"
    >
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 id="ga4-organic" className="text-lg font-semibold">
          What Google visitors did on your site
        </h2>
        <p className="text-xs text-muted-foreground">
          Google Analytics, organic search, last 28 days vs the 28 before
        </p>
      </div>
      <div className="grid gap-3 sm:grid-cols-3">
        {tiles.map((tile) => (
          <div key={tile.label} className="rounded-lg border p-3">
            <p className="text-xs text-muted-foreground">{tile.label}</p>
            <p className="text-2xl font-semibold tabular-nums">
              {tile.now === null ? "—" : tile.format(tile.now)}
            </p>
            <Change now={tile.now} before={tile.before} />
          </div>
        ))}
      </div>
      <p className="text-sm text-muted-foreground">
        {perHundred === null
          ? "Mark your enquiry or sale event as a key event in GA4 to see which Google visits turn into business."
          : `${perHundred.toFixed(1)} key events per 100 Google visits.`}
      </p>
    </section>
  );
}

function Change({
  now,
  before,
}: {
  now: number | null;
  before: number | null;
}) {
  if (now === null || before === null || before === 0) {
    return <p className="text-xs text-muted-foreground">No earlier period</p>;
  }
  const change = (now - before) / before;
  const down = change < 0;
  return (
    <p
      className={`text-xs tabular-nums ${down ? "text-red-700 dark:text-red-400" : "text-emerald-700 dark:text-emerald-400"}`}
    >
      {down ? "▼" : "▲"} {Math.abs(Math.round(change * 100))}% vs before
    </p>
  );
}
