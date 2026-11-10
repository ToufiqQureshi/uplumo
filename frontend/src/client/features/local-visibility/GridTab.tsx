import { useState } from "react";
import { cn } from "cn";
import { Button } from "@/client/components/ui/button";
import {
  type GridResult,
  gridResultSchema,
  type Overview,
  runResult,
  useStartRun,
} from "./localVisibilityApi";
import { PaidButton, RunPanel } from "./RunPanel";

const SPACINGS = [0.5, 1, 2, 5];

// Rank bands of the legend, from the top 3 to not in the top 20.
const BANDS = [
  {
    label: "1–3",
    test: (r: number) => r >= 1 && r <= 3,
    className: "bg-success text-white",
  },
  {
    label: "4–10",
    test: (r: number) => r >= 4 && r <= 10,
    className: "bg-warning text-black",
  },
  {
    label: "11–20",
    test: (r: number) => r >= 11,
    className: "bg-destructive/70 text-white",
  },
  {
    label: "not in top 20",
    test: (r: number) => r === 0,
    className: "bg-muted text-muted-foreground",
  },
];

export function GridTab({
  projectId,
  overview,
}: {
  projectId: string;
  overview: Overview;
}) {
  const start = useStartRun(projectId);
  const [size, setSize] = useState<3 | 5>(3);
  const [spacingKm, setSpacingKm] = useState(1);
  const run = overview.runs.grid;
  const result = runResult(run, gridResultSchema);
  const term = overview.business?.searchTerm ?? "";
  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-sm text-muted-foreground">Grid</span>
        {([3, 5] as const).map((n) => (
          <Button
            key={n}
            size="sm"
            variant={size === n ? "secondary" : "ghost"}
            aria-pressed={size === n}
            onClick={() => setSize(n)}
          >
            {n} × {n}
          </Button>
        ))}
        <span className="ml-2 text-sm text-muted-foreground">Apart</span>
        {SPACINGS.map((km) => (
          <Button
            key={km}
            size="sm"
            variant={spacingKm === km ? "secondary" : "ghost"}
            aria-pressed={spacingKm === km}
            onClick={() => setSpacingKm(km)}
          >
            {km} km
          </Button>
        ))}
        <div className="ml-auto">
          <PaidButton
            label={`Scan ${size} × ${size}`}
            credits={size === 3 ? overview.prices.grid3 : overview.prices.grid5}
            pending={start.isPending || run?.status === "running"}
            onClick={() => start.mutate({ kind: "grid", size, spacingKm })}
          />
        </div>
      </div>
      <RunPanel
        projectId={projectId}
        run={run}
        source={`Google Maps searches for “${term}” around your address`}
        error={start.error}
      >
        {result ? <GridView result={result} /> : null}
      </RunPanel>
    </div>
  );
}

function GridView({ result }: { result: GridResult }) {
  const [selected, setSelected] = useState<number | null>(null);
  const point = selected === null ? undefined : result.points[selected];
  return (
    <div className="space-y-4">
      <p className="font-medium">{result.summary}</p>
      <p className="text-xs text-muted-foreground">
        “{result.searchTerm}”, {result.size} × {result.size} points{" "}
        {result.spacingKm} km apart, north at the top, your business in the
        centre.
      </p>
      <div className="flex flex-col gap-4 md:flex-row">
        <div
          className="grid w-fit gap-2"
          style={{ gridTemplateColumns: `repeat(${result.size}, 2.75rem)` }}
        >
          {result.points.map((p, i) => {
            const band = BANDS.find((b) => b.test(p.rank));
            return (
              <button
                key={`${p.row}-${p.col}`}
                type="button"
                aria-label={`Point ${p.row + 1}, ${p.col + 1}: ${p.missing ? "no data" : p.rank > 0 ? `rank ${p.rank}` : "not in top 20"}`}
                aria-pressed={selected === i}
                onClick={() => setSelected(i)}
                className={cn(
                  "flex size-11 items-center justify-center rounded-full text-sm font-semibold ring-offset-2 aria-pressed:ring-2 aria-pressed:ring-ring",
                  p.missing
                    ? "border border-dashed text-muted-foreground"
                    : band?.className,
                )}
              >
                {p.missing ? "?" : p.rank > 0 ? p.rank : "–"}
              </button>
            );
          })}
        </div>
        <div className="min-w-0 flex-1 rounded-lg border p-3 text-sm">
          {point && !point.missing ? (
            <>
              <p className="font-medium">
                {point.rank > 0
                  ? `You rank #${point.rank} here.`
                  : "You are not in the top 20 here."}
              </p>
              <ol className="mt-2 list-decimal pl-5 text-muted-foreground">
                {(point.top ?? []).map((title) => (
                  <li key={title}>{title}</li>
                ))}
              </ol>
            </>
          ) : (
            <p className="text-muted-foreground">
              Pick a point to see the top 3 there.
            </p>
          )}
        </div>
      </div>
      <ul className="flex flex-wrap gap-3 text-xs text-muted-foreground">
        {BANDS.map((b) => (
          <li key={b.label} className="flex items-center gap-1">
            <span
              className={cn("inline-block size-3 rounded-full", b.className)}
            />
            {b.label}
          </li>
        ))}
      </ul>
    </div>
  );
}
