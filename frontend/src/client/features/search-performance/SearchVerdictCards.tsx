import { Link } from "@tanstack/react-router";
import { sortBy } from "remeda";
import { Button } from "@/client/components/ui/button";
import {
  formatCount,
  formatPosition,
  type Report,
} from "./SearchPerformanceColumns";
import { SEVERITY, searchVerdict, urgentCount } from "./searchVerdict";

/**
 * The top of Search Growth: what happened, why, and the pages to fix
 * first. Built from the report already loaded; no extra request.
 */
export function SearchVerdict({
  projectId,
  report,
  onOpenFix,
}: {
  projectId: string;
  report: Report;
  onOpenFix: () => void;
}) {
  const verdict = searchVerdict(report);
  const urgent = urgentCount(report);
  return (
    <section
      aria-labelledby="search-verdict"
      className="space-y-3 rounded-2xl bg-foreground p-6 text-background"
    >
      <p className="label-mono text-background/70">What happened</p>
      <h2
        id="search-verdict"
        className="text-xl leading-snug font-semibold md:text-2xl"
      >
        {verdict.headline}
      </h2>
      <p className="text-sm text-background/80 md:text-base">
        {verdict.detail}
      </p>
      {urgent > 0 ? (
        <p className="inline-flex rounded-md bg-red-600 px-2.5 py-1 text-sm font-medium text-white">
          {urgent} urgent {urgent === 1 ? "drop needs" : "drops need"} a fix now
        </p>
      ) : null}
      <div className="flex flex-wrap gap-2 pt-1">
        {report.pageChanges.length > 0 || report.queryLosses.length > 0 ? (
          <Button size="lg" variant="secondary" onClick={onOpenFix}>
            See what to fix
          </Button>
        ) : null}
        <Button
          size="lg"
          variant="outline"
          className="border-background/40 bg-transparent text-background hover:bg-background/10 hover:text-background"
          nativeButton={false}
          render={<Link to="/p/$projectId/sam" params={{ projectId }} />}
        >
          Ask SAM why
        </Button>
      </div>
    </section>
  );
}

export const pathOf = (url: string) => {
  try {
    const parsed = new URL(url);
    return parsed.pathname === "/" ? "/ (home page)" : parsed.pathname;
  } catch {
    return url;
  }
};

export function SeverityBadge({
  severity,
}: {
  severity: keyof typeof SEVERITY;
}) {
  const level = SEVERITY[severity];
  return (
    <span
      className={`rounded-md px-2 py-1 text-xs font-medium ${level.className}`}
    >
      {level.label}
    </span>
  );
}

/** The three searches closest to page one, with one fix each. */
export function QuickWins({
  projectId,
  report,
}: {
  projectId: string;
  report: Report;
}) {
  const wins = sortBy(report.strikingDistance, [
    (row) => row.impressions,
    "desc",
  ]).slice(0, 3);
  if (wins.length === 0) return null;
  return (
    <section
      aria-labelledby="quick-wins"
      className="space-y-3 rounded-xl border bg-card p-5"
    >
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 id="quick-wins" className="text-lg font-semibold">
          Quick wins: you are almost on top
        </h2>
        <p className="text-xs text-muted-foreground">
          Searches where you rank 5 to 20. A small improvement moves you up.
        </p>
      </div>
      <div className="grid gap-3 md:grid-cols-3">
        {wins.map((row) => (
          <div
            key={row.query}
            className="flex flex-col gap-2 rounded-lg border p-4"
          >
            <p className="font-medium">&ldquo;{row.query}&rdquo;</p>
            <p className="text-xs text-muted-foreground">
              Rank {formatPosition(row.position)} · shown{" "}
              {formatCount(row.impressions)} times · {formatCount(row.clicks)}{" "}
              clicks
            </p>
            <p className="truncate text-xs" title={row.page}>
              {pathOf(row.page)}
            </p>
            <Button
              size="sm"
              className="mt-auto"
              nativeButton={false}
              render={
                <Link
                  to="/p/$projectId/content-editor"
                  params={{ projectId }}
                  search={{ keyword: row.query.slice(0, 200) }}
                />
              }
            >
              Improve in editor
            </Button>
          </div>
        ))}
      </div>
    </section>
  );
}
