import { Link } from "@tanstack/react-router";
import { Button } from "@/client/components/ui/button";
import type { SearchInsights } from "./insightsApi";
import {
  InsightSection,
  InsightTable,
  PageLink,
  ShowMore,
} from "./InsightParts";
import { formatCount, formatPosition } from "./SearchPerformanceColumns";
import { SeverityBadge } from "./SearchVerdictCards";
import { lossText, PAGE_REASON } from "./searchVerdict";

const COMPARED = "Compared with the previous period, from Search Console.";

export function LosingPages({
  projectId,
  pages,
}: {
  projectId: string;
  pages: SearchInsights["losingPages"];
}) {
  if (pages.length === 0) return null;
  return (
    <InsightSection
      id="losing-pages"
      title={`Pages that lost clicks (${pages.length})`}
      note={`Biggest loss first. ${COMPARED}`}
    >
      <ShowMore items={pages}>
        {(visible) => (
          <InsightTable
            columns={[
              { label: "Page" },
              { label: "Clicks" },
              { label: "How bad" },
              { label: "Why" },
              { label: "Do this" },
            ]}
          >
            {visible.map((change) => {
              const reason = PAGE_REASON[change.reason];
              return (
                <tr key={change.page} className="border-t">
                  <td className="max-w-xs truncate py-3 pr-3">
                    <PageLink url={change.page} />
                  </td>
                  <td className="py-3 pr-3 tabular-nums">
                    {formatCount(change.prevClicks)} →{" "}
                    {formatCount(change.clicks)}
                    <span className="block text-xs text-muted-foreground">
                      {lossText(change.prevClicks, change.clicks)}
                    </span>
                  </td>
                  <td className="py-3 pr-3">
                    <SeverityBadge severity={change.severity} />
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
                        <Link
                          to={
                            change.reason === "fewer_searches"
                              ? "/p/$projectId/keywords"
                              : "/p/$projectId/content-editor"
                          }
                          params={{ projectId }}
                        />
                      }
                    >
                      {reason.action}
                    </Button>
                  </td>
                </tr>
              );
            })}
          </InsightTable>
        )}
      </ShowMore>
    </InsightSection>
  );
}

export function QueryChanges({
  projectId,
  insights,
}: {
  projectId: string;
  insights: SearchInsights;
}) {
  const { queryLosses, queryGains } = insights;
  if (queryLosses.length === 0 && queryGains.length === 0) return null;
  return (
    <InsightSection
      id="query-changes"
      title="Searches you lost and won"
      note={`${COMPARED} Named searches only: Google hides rare ones, which are still in your totals.`}
    >
      {queryLosses.length > 0 ? (
        <div className="space-y-2">
          <h3 className="text-sm font-medium">
            Lost ({formatCount(queryLosses.length)})
          </h3>
          <ShowMore items={queryLosses}>
            {(visible) => (
              <ul className="divide-y">
                {visible.map((row) => (
                  <li
                    key={row.query}
                    className="flex flex-wrap items-center gap-x-4 gap-y-2 py-3"
                  >
                    <div className="min-w-0 flex-1 basis-56">
                      <p className="font-medium break-words">
                        &ldquo;{row.query}&rdquo;
                      </p>
                      <p className="text-xs text-muted-foreground tabular-nums">
                        {formatCount(row.prevClicks)} →{" "}
                        {formatCount(row.clicks)} clicks ·{" "}
                        {lossText(row.prevClicks, row.clicks)} · rank{" "}
                        {formatPosition(row.prevPosition)} →{" "}
                        {row.position > 0
                          ? formatPosition(row.position)
                          : "gone"}
                      </p>
                    </div>
                    <SeverityBadge severity={row.severity} />
                    <span className="rounded-md bg-muted px-2 py-1 text-xs">
                      {PAGE_REASON[row.reason].label}
                    </span>
                    <Button
                      size="sm"
                      variant="outline"
                      nativeButton={false}
                      render={
                        <Link
                          to="/p/$projectId/content-editor"
                          params={{ projectId }}
                          search={{ keyword: row.query.slice(0, 200) }}
                        />
                      }
                    >
                      Win it back
                    </Button>
                  </li>
                ))}
              </ul>
            )}
          </ShowMore>
        </div>
      ) : null}
      {queryGains.length > 0 ? (
        <div className="space-y-2">
          <h3 className="text-sm font-medium">
            Growing searches ({formatCount(queryGains.length)})
          </h3>
          <ShowMore items={queryGains}>
            {(visible) => (
              <ul className="grid gap-2 md:grid-cols-3">
                {visible.map((row) => (
                  <li key={row.query} className="rounded-lg border p-3 text-sm">
                    <p className="font-medium break-words">
                      &ldquo;{row.query}&rdquo;
                    </p>
                    <p className="text-xs text-muted-foreground tabular-nums">
                      {row.prevClicks > 0
                        ? `${formatCount(row.prevClicks)} → ${formatCount(row.clicks)} clicks`
                        : `new: ${formatCount(row.clicks)} clicks`}{" "}
                      · rank {formatPosition(row.position)}
                    </p>
                  </li>
                ))}
              </ul>
            )}
          </ShowMore>
        </div>
      ) : null}
    </InsightSection>
  );
}

export function DecayingPages({ insights }: { insights: SearchInsights }) {
  const { decaying, totals, coverage } = insights;
  if (!coverage.decayAvailable) {
    return (
      <InsightSection id="decaying" title="Pages fading for a while">
        <p className="text-sm text-muted-foreground">
          Search Console keeps 16 months. Choose a shorter date range to compare
          three periods in a row.
        </p>
      </InsightSection>
    );
  }
  if (decaying.length === 0) return null;
  return (
    <InsightSection
      id="decaying"
      title={`Pages fading for a while (${totals.decaying})`}
      note="Clicks fell in two periods in a row. These need a refresh before they fall further."
    >
      <ShowMore items={decaying}>
        {(visible) => (
          <InsightTable
            columns={[
              { label: "Page" },
              { label: "Oldest", align: "right" },
              { label: "Middle", align: "right" },
              { label: "Latest", align: "right" },
              { label: "Lost", align: "right" },
            ]}
          >
            {visible.map((row) => (
              <tr key={row.page} className="border-t">
                <td className="max-w-xs truncate py-2 pr-3">
                  <PageLink url={row.page} />
                </td>
                {row.clicks.map((value, index) => (
                  <td key={index} className="py-2 pr-3 text-right tabular-nums">
                    {formatCount(value)}
                  </td>
                ))}
                <td className="py-2 pr-3 text-right tabular-nums">
                  −{formatCount(row.lostClicks)}
                </td>
              </tr>
            ))}
          </InsightTable>
        )}
      </ShowMore>
    </InsightSection>
  );
}
