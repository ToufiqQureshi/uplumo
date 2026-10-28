import type { ReactNode } from "react";
import type { ReportKeyword } from "./keywordReport";

const MONTHS = [
  "Jan",
  "Feb",
  "Mar",
  "Apr",
  "May",
  "Jun",
  "Jul",
  "Aug",
  "Sep",
  "Oct",
  "Nov",
  "Dec",
];

function formatNumber(value: number | null | undefined): string {
  return value == null ? "—" : value.toLocaleString("en-IN");
}

export function Sparkline({
  values,
  label,
}: {
  values: number[];
  label: string;
}) {
  const max = Math.max(1, ...values);
  if (values.length === 0)
    return <span className="text-muted-foreground">—</span>;
  return (
    <div className="flex h-6 items-end gap-px" role="img" aria-label={label}>
      {values.map((v, i) => (
        <div
          key={i}
          className="w-1 rounded-[1px] bg-muted-foreground/60"
          style={{ height: `${Math.max(8, (v / max) * 100)}%` }}
        />
      ))}
    </div>
  );
}

export function Card({
  title,
  hint,
  children,
}: {
  title: string;
  hint?: string;
  children: ReactNode;
}) {
  return (
    <section className="rounded-xl border bg-card p-4">
      <h3 className="text-sm font-semibold">{title}</h3>
      {hint ? (
        <p className="mt-0.5 text-xs text-muted-foreground">{hint}</p>
      ) : null}
      <div className="mt-3">{children}</div>
    </section>
  );
}

export function Seasonality({ keyword }: { keyword: ReportKeyword }) {
  const values = keyword.metrics?.monthlyVolumes ?? [];
  if (values.length < 6) return null;
  const max = Math.max(1, ...values);
  const peak = values.indexOf(max);
  // monthlyVolumes end at latestMonth ("2026-09"); label backwards from it.
  const latest = Number(keyword.metrics?.latestMonth?.slice(5, 7));
  if (!latest) return null;
  const lastMonth = latest - 1;
  const monthAt = (i: number) =>
    MONTHS[(lastMonth - (values.length - 1 - i) + 24) % 12];
  return (
    <Card
      title={`When people search "${keyword.keyword}"`}
      hint={`Searches peak in ${monthAt(peak)}. Publish or update a few weeks before it.`}
    >
      <div
        className="flex h-32 items-end gap-1 border-b"
        role="img"
        aria-label={`Monthly searches, peak ${max}`}
      >
        {values.map((v, i) => (
          <div key={i} className="flex h-full flex-1 flex-col justify-end">
            <div
              className={
                i === peak
                  ? "rounded-t bg-foreground"
                  : "rounded-t bg-muted-foreground/40"
              }
              style={{ height: `${(v / max) * 100}%` }}
              title={`${v}`}
            />
          </div>
        ))}
      </div>
      <div className="mt-1 flex gap-1 text-center text-[11px] text-muted-foreground">
        {values.map((_, i) => (
          <span key={i} className="flex-1">
            {monthAt(i).charAt(0)}
          </span>
        ))}
      </div>
    </Card>
  );
}

export function DifficultyMatrix({ keywords }: { keywords: ReportKeyword[] }) {
  const points = keywords.filter(
    (k) =>
      k.metrics?.searchVolume != null && k.metrics.keywordDifficulty != null,
  );
  if (points.length < 2) return null;
  const maxVolume = Math.max(
    ...points.map((k) => k.metrics?.searchVolume ?? 0),
    1,
  );
  return (
    <Card
      title="Where to start: searches vs difficulty"
      hint="Top-left is the sweet spot: more searches, easier to rank."
    >
      <div className="relative h-44 border-b border-l">
        <div className="absolute left-0 top-0 h-1/2 w-1/2 bg-lime-100/70 dark:bg-lime-900/30" />
        <span className="absolute left-2 top-1 text-[11px] font-semibold">
          Do first
        </span>
        {points.map((k) => {
          const x = Math.min(
            100,
            ((k.metrics?.keywordDifficulty ?? 0) / 100) * 100,
          );
          const y = 100 - ((k.metrics?.searchVolume ?? 0) / maxVolume) * 90;
          return (
            <div
              key={k.keyword}
              className="absolute flex -translate-x-1/2 -translate-y-1/2 items-center gap-1"
              style={{ left: `${x}%`, top: `${y}%` }}
            >
              <span className="size-2.5 rounded-full border-2 border-background bg-foreground" />
              <span className="max-w-40 truncate rounded bg-background/85 px-1 text-[11px]">
                {k.keyword}
              </span>
            </div>
          );
        })}
      </div>
      <div className="mt-1 flex justify-between text-[11px] text-muted-foreground">
        <span>Easy (0)</span>
        <span>Difficulty</span>
        <span>Hard (100)</span>
      </div>
    </Card>
  );
}
