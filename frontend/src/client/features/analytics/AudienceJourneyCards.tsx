import { useState } from "react";
import { reverse } from "remeda";
import { useInfiniteQuery } from "@tanstack/react-query";
import { CardShell } from "@/client/components/CardShell";
import { QueryError } from "@/client/components/QueryState";
import { Button } from "@/client/components/ui/button";
import {
  analyticsClicksQueryOptions,
  type AnalyticsSummary,
} from "./analyticsApi";
import { displayPath } from "./presentation";

export function DailyAudienceCard({
  days,
}: {
  days: AnalyticsSummary["audience"];
}) {
  const [expanded, setExpanded] = useState(false);
  const rows = reverse(days);
  const recognized = days.some(
    (day) => day.newBrowsers + day.returningBrowsers > 0,
  );
  return (
    <CardShell
      title="Daily new and returning"
      stamp="UTC days · Today is partial. Counts describe recognized browsers, not exact people."
    >
      <p className="mb-4 text-sm text-muted-foreground">
        New means first recorded on that day; returning means first recorded on
        an earlier day. Ten visits from one recognized browser count once per
        day.
      </p>
      {!recognized ? (
        <p className="mb-4 rounded-lg border border-border p-3 text-sm">
          No recognized browsers in this period. Enable optional browser
          recognition in Tracking code after connecting your consent setup.
          Older visits cannot be classified.
        </p>
      ) : null}
      <p className="mb-2 text-xs text-muted-foreground sm:hidden">
        Swipe the table to see all columns.
      </p>
      <div
        className="overflow-x-auto"
        role="region"
        aria-label="Daily audience table"
        tabIndex={0}
      >
        <table className="w-full min-w-96 text-sm">
          <thead>
            <tr className="text-left text-xs text-muted-foreground">
              <th scope="col" className="py-2 pr-3">
                Date (UTC)
              </th>
              <th scope="col" className="px-2 text-right">
                New browsers
              </th>
              <th scope="col" className="px-2 text-right">
                Returning browsers
              </th>
              <th scope="col" className="pl-2 text-right">
                Unrecognized pageviews
              </th>
            </tr>
          </thead>
          <tbody>
            {(expanded ? rows : rows.slice(0, 7)).map((day) => (
              <tr
                key={day.date}
                className="border-t border-border tabular-nums"
              >
                <th
                  scope="row"
                  className="py-2 pr-3 text-left font-normal whitespace-nowrap"
                >
                  {day.date}
                </th>
                <td className="px-2 text-right">
                  {day.newBrowsers.toLocaleString()}
                </td>
                <td className="px-2 text-right">
                  {day.returningBrowsers.toLocaleString()}
                </td>
                <td className="pl-2 text-right">
                  {day.unknownPageviews.toLocaleString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {rows.length > 7 ? (
        <Button
          className="mt-3 self-start"
          size="sm"
          variant="outline"
          aria-expanded={expanded}
          onClick={() => setExpanded(!expanded)}
        >
          {expanded ? "Show latest 7 days" : `Show all ${rows.length} days`}
        </Button>
      ) : null}
      <p className="mt-3 text-xs text-muted-foreground">
        Unrecognized pageviews measure coverage, not unique users. Storage
        clearing, expiry, or a different browser/device can create a new
        browser. Daily counts must not be added to claim unique people across a
        date range.
      </p>
    </CardShell>
  );
}

export function ClickDestinationsCard({
  projectId,
  range,
}: {
  projectId: string;
  range: { from: string; to: string };
}) {
  const report = useInfiniteQuery(
    analyticsClicksQueryOptions(projectId, range),
  );
  const rows = report.data?.pages.flatMap((page) => page.rows) ?? [];
  return (
    <CardShell
      title="Button and link destinations"
      stamp="Click intent, not proof of arrival or success. Clicking visitor-days can count the same person on different days."
    >
      <p className="mb-4 text-sm text-muted-foreground">
        Configured button clicks and automatically tracked outbound links.
        Internal arrival means another pageview in the same tab session matched
        the destination within 30 minutes; it does not prove the click caused
        that visit.
      </p>
      {report.isPending ? (
        <p role="status">Loading click destinations…</p>
      ) : null}
      {report.isError ? (
        <QueryError
          error={report.error}
          fallback="Could not load click destinations."
          onRetry={() => void report.refetch()}
          isRetrying={report.isFetching}
        />
      ) : null}
      {!report.isPending && !report.isError && rows.length === 0 ? (
        <p className="text-sm text-muted-foreground">
          No destination clicks in this period. Existing custom events remain
          below. Install the latest snippet, tag important buttons, then test a
          real click.
        </p>
      ) : null}
      {rows.length > 0 ? (
        <p className="mb-2 text-xs text-muted-foreground sm:hidden">
          Swipe the table to see all columns.
        </p>
      ) : null}
      {rows.length > 0 ? (
        <div
          className="overflow-x-auto"
          role="region"
          aria-label="Click destination table"
          tabIndex={0}
        >
          <table className="w-full min-w-[640px] text-sm">
            <thead>
              <tr className="text-left text-xs text-muted-foreground">
                <th scope="col" className="pb-2 pr-3">
                  Page / button
                </th>
                <th scope="col" className="pb-2 pr-3">
                  Destination
                </th>
                <th scope="col" className="pb-2 px-2 text-right">
                  Clicks
                </th>
                <th scope="col" className="pb-2 px-2 text-right">
                  Clicking visitor-days
                </th>
                <th scope="col" className="pb-2 pl-2 text-right">
                  Clicks with observed arrival
                </th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr
                  key={JSON.stringify([
                    row.page,
                    row.name,
                    row.host,
                    row.path,
                    row.type,
                  ])}
                  className="border-t border-border"
                >
                  <td className="max-w-56 py-3 pr-3 break-words">
                    <span className="font-medium">{row.name}</span>
                    <span className="block text-xs text-muted-foreground">
                      {displayPath(row.page)}
                    </span>
                  </td>
                  <td className="max-w-64 py-3 pr-3 break-words">
                    {row.type === "unknown"
                      ? "Unknown destination"
                      : `${row.host}${displayPath(row.path)}`}
                    <span className="block text-xs text-muted-foreground">
                      {row.type === "external"
                        ? "External · further activity unavailable"
                        : row.type === "internal"
                          ? "Internal"
                          : "No supported target was provided"}
                    </span>
                  </td>
                  <td className="px-2 text-right tabular-nums">
                    {row.clicks.toLocaleString()}
                  </td>
                  <td className="px-2 text-right tabular-nums">
                    {row.visitorDays.toLocaleString()}
                  </td>
                  <td className="pl-2 text-right tabular-nums">
                    {row.type === "internal"
                      ? row.arrivals.toLocaleString()
                      : "—"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
      {report.hasNextPage ? (
        <Button
          className="mt-4 self-start"
          variant="outline"
          size="sm"
          disabled={report.isFetchingNextPage}
          onClick={() => void report.fetchNextPage()}
        >
          {report.isFetchingNextPage ? "Loading…" : "Load more destinations"}
        </Button>
      ) : null}
      {report.data?.pages.length === 1000 && report.data.pages[999].hasMore ? (
        <p className="mt-3 text-xs text-muted-foreground">
          This report reached its paging limit. Choose a shorter date range to
          explore more destinations.
        </p>
      ) : null}
    </CardShell>
  );
}
