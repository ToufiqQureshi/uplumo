import { Card } from "@/client/components/ui/card";
import {
  formatCount,
  formatCtr,
  formatPosition,
  type Report,
} from "@/client/features/search-performance/SearchPerformanceColumns";

type Delta = { text: string; improved: boolean } | null;

function percentDelta(current: number, previous: number): Delta {
  if (previous <= 0) return null;
  const change = (current - previous) / previous;
  const pct = (change * 100).toFixed(1);
  return { text: `${change >= 0 ? "+" : ""}${pct}%`, improved: change >= 0 };
}

/** Position falls as rankings improve, so the delta is inverted. */
function positionDelta(current: number, previous: number): Delta {
  if (previous <= 0 || current <= 0) return null;
  const change = previous - current;
  return {
    text: `${change >= 0 ? "+" : ""}${change.toFixed(1)}`,
    improved: change >= 0,
  };
}

export function TotalsCards({ report }: { report: Report }) {
  const { totals, prevTotals, range } = report;
  const deltaTitle = `vs ${range.prevStartDate} to ${range.prevEndDate}`;
  const getDelta = (current: number, previous: number) =>
    range.comparisonAvailable ? percentDelta(current, previous) : null;
  const position = range.comparisonAvailable
    ? positionDelta(totals.position, prevTotals.position)
    : null;

  return (
    <>
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <TotalCard
          label="Clicks"
          value={formatCount(totals.clicks)}
          delta={getDelta(totals.clicks, prevTotals.clicks)}
          deltaTitle={deltaTitle}
        />
        <TotalCard
          label="Impressions"
          value={formatCount(totals.impressions)}
          delta={getDelta(totals.impressions, prevTotals.impressions)}
          deltaTitle={deltaTitle}
        />
        <TotalCard
          label="CTR"
          value={formatCtr(totals.ctr)}
          delta={getDelta(totals.ctr, prevTotals.ctr)}
          deltaTitle={deltaTitle}
        />
        <TotalCard
          label="Avg position"
          value={formatPosition(totals.position)}
          delta={position}
          deltaTitle={deltaTitle}
        />
      </div>
      {!range.comparisonAvailable ? (
        <p className="text-xs text-muted-foreground" role="status">
          Comparison is unavailable beyond Search Console&apos;s 16-month
          history.
        </p>
      ) : null}
    </>
  );
}

function TotalCard({
  label,
  value,
  delta,
  deltaTitle,
}: {
  label: string;
  value: string;
  delta: Delta;
  deltaTitle: string;
}) {
  return (
    <Card className="gap-0 p-4">
      <div className="text-xs uppercase tracking-wide text-muted-foreground">
        {label}
      </div>
      <div className="mt-1 flex items-baseline gap-2">
        <span className="text-2xl font-semibold">{value}</span>
        {delta ? (
          <span
            className={`text-xs ${delta.improved ? "text-success" : "text-destructive"}`}
            title={deltaTitle}
          >
            {delta.text}
          </span>
        ) : null}
      </div>
    </Card>
  );
}
