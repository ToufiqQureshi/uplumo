import { Link } from "@tanstack/react-router";
import { sortBy } from "remeda";
import { Button } from "@/client/components/ui/button";
import {
  formatCount,
  formatPosition,
  type Report,
} from "./SearchPerformanceColumns";
import { PAGE_REASON, searchVerdict } from "./searchVerdict";

/**
 * The top of Search Performance: what happened, why, and the pages to fix
 * first. Built from the report already loaded; no extra request.
 */
export function SearchVerdict({
  projectId,
  report,
}: {
  projectId: string;
  report: Report;
}) {
  const verdict = searchVerdict(report);
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
      <div className="flex flex-wrap gap-2 pt-1">
        {report.pageChanges.length > 0 ? (
          <Button
            size="lg"
            variant="secondary"
            nativeButton={false}
            render={<a href="#pages-to-fix" />}
          >
            See the pages to fix
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

const pathOf = (url: string) => {
  try {
    const parsed = new URL(url);
    return parsed.pathname === "/" ? "/ (home page)" : parsed.pathname;
  } catch {
    return url;
  }
};

/** Pages that lost the most clicks, each with why and one next step. */
export function PagesToFix({
  projectId,
  report,
}: {
  projectId: string;
  report: Report;
}) {
  if (report.pageChanges.length === 0) return null;
  return (
    <section
      id="pages-to-fix"
      aria-labelledby="pages-to-fix-title"
      className="space-y-3 rounded-xl border bg-card p-5"
    >
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h2 id="pages-to-fix-title" className="text-lg font-semibold">
          Pages that lost the most clicks
        </h2>
        <p className="text-xs text-muted-foreground">
          From your Search Console data, compared with the previous period
        </p>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full min-w-[640px] text-left text-sm">
          <thead className="text-xs text-muted-foreground">
            <tr>
              <th className="py-2 font-medium">Page</th>
              <th className="py-2 font-medium">Clicks</th>
              <th className="py-2 font-medium">Why</th>
              <th className="py-2 font-medium">Do this</th>
            </tr>
          </thead>
          <tbody>
            {report.pageChanges.map((change) => {
              const reason = PAGE_REASON[change.reason];
              return (
                <tr key={change.page} className="border-t">
                  <td className="max-w-xs truncate py-3 pr-3">
                    <a
                      href={change.page}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="underline-offset-4 hover:underline"
                      title={change.page}
                    >
                      {pathOf(change.page)}
                    </a>
                  </td>
                  <td className="py-3 pr-3 tabular-nums">
                    {formatCount(change.prevClicks)} →{" "}
                    {formatCount(change.clicks)}
                  </td>
                  <td className="py-3 pr-3">
                    <span className="rounded-md bg-muted px-2 py-1 text-xs">
                      {reason.label}
                      {change.reason === "rank_fell" && change.position > 0
                        ? ` (${formatPosition(change.prevPosition)} → ${formatPosition(change.position)})`
                        : ""}
                    </span>
                  </td>
                  <td className="py-3">
                    <Button
                      size="sm"
                      variant="outline"
                      nativeButton={false}
                      render={
                        change.reason === "fewer_searches" ? (
                          <Link
                            to="/p/$projectId/keywords"
                            params={{ projectId }}
                          />
                        ) : (
                          <Link
                            to="/p/$projectId/content-editor"
                            params={{ projectId }}
                          />
                        )
                      }
                    >
                      {reason.action}
                    </Button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </section>
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
