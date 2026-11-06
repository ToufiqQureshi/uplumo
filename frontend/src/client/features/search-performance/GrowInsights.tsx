import { Link } from "@tanstack/react-router";
import { Button } from "@/client/components/ui/button";
import type { SearchInsights } from "./insightsApi";
import {
  InsightSection,
  InsightTable,
  PageLink,
  ShowMore,
} from "./InsightParts";
import {
  formatCount,
  formatCtr,
  formatPosition,
} from "./SearchPerformanceColumns";

export function CtrGaps({
  projectId,
  insights,
}: {
  projectId: string;
  insights: SearchInsights;
}) {
  const { ctrGaps, totals } = insights;
  if (ctrGaps.length === 0) return null;
  const capped = totals.ctrGaps > ctrGaps.length;
  return (
    <InsightSection
      id="ctr-gaps"
      title={`Searches people see but skip (${formatCount(totals.ctrGaps)})`}
      note={`You rank well here, yet fewer people click than on your other pages at the same rank. A clearer title and description often fixes it. "More clicks possible" is an estimate from your own pages, not a promise.${capped ? ` The ${formatCount(ctrGaps.length)} with the most to gain are shown.` : ""}`}
    >
      <ShowMore items={ctrGaps}>
        {(visible) => (
          <InsightTable
            columns={[
              { label: "Search" },
              { label: "Page" },
              { label: "Rank", align: "right" },
              { label: "Your CTR", align: "right" },
              { label: "Usual CTR here", align: "right" },
              { label: "More clicks possible", align: "right" },
              { label: "" },
            ]}
          >
            {visible.map((gap) => (
              <tr key={gap.query} className="border-t">
                <td className="max-w-xs break-words py-2 pr-3">{gap.query}</td>
                <td className="max-w-[12rem] truncate py-2 pr-3">
                  <PageLink url={gap.page} />
                </td>
                <td className="py-2 pr-3 text-right tabular-nums">
                  {formatPosition(gap.position)}
                </td>
                <td className="py-2 pr-3 text-right tabular-nums">
                  {formatCtr(gap.ctr)}
                </td>
                <td className="py-2 pr-3 text-right tabular-nums">
                  {formatCtr(gap.expectedCtr)}
                </td>
                <td className="py-2 pr-3 text-right font-medium tabular-nums">
                  ~{formatCount(gap.possibleClicks)}
                </td>
                <td className="py-2">
                  <Button
                    size="sm"
                    variant="outline"
                    nativeButton={false}
                    render={
                      <Link
                        to="/p/$projectId/content-editor"
                        params={{ projectId }}
                        search={{ keyword: gap.query.slice(0, 200) }}
                      />
                    }
                  >
                    Rewrite title
                  </Button>
                </td>
              </tr>
            ))}
          </InsightTable>
        )}
      </ShowMore>
    </InsightSection>
  );
}

export function Cannibalization({ insights }: { insights: SearchInsights }) {
  const { cannibalization, totals } = insights;
  if (cannibalization.length === 0) return null;
  return (
    <InsightSection
      id="cannibalization"
      title={`Pages competing with each other (${formatCount(totals.cannibalization)})`}
      note="Two or more of your pages show for the same search, so neither ranks as well as one strong page could. Keep the leader, then merge or redirect the rest, or aim each page at a different search."
    >
      <ShowMore items={cannibalization}>
        {(visible) => (
          <ul className="space-y-3">
            {visible.map((row) => (
              <li key={row.query} className="rounded-lg border p-4">
                <p className="font-medium break-words">
                  &ldquo;{row.query}&rdquo;
                  <span className="ml-2 text-xs font-normal text-muted-foreground">
                    {formatCount(row.clicks)} clicks ·{" "}
                    {formatCount(row.impressions)} views
                  </span>
                </p>
                <InsightTable
                  columns={[
                    { label: "Page" },
                    { label: "Share of views", align: "right" },
                    { label: "Clicks", align: "right" },
                    { label: "Rank", align: "right" },
                  ]}
                >
                  {row.pages.map((page) => (
                    <tr key={page.page} className="border-t">
                      <td className="max-w-xs truncate py-2 pr-3">
                        <PageLink url={page.page} />
                        {page.page === row.leader ? (
                          <span className="ml-2 rounded-md bg-muted px-2 py-0.5 text-xs">
                            Keep
                          </span>
                        ) : null}
                      </td>
                      <td className="py-2 pr-3 text-right tabular-nums">
                        {Math.round(page.share * 100)}%
                      </td>
                      <td className="py-2 pr-3 text-right tabular-nums">
                        {formatCount(page.clicks)}
                      </td>
                      <td className="py-2 pr-3 text-right tabular-nums">
                        {formatPosition(page.position)}
                      </td>
                    </tr>
                  ))}
                </InsightTable>
              </li>
            ))}
          </ul>
        )}
      </ShowMore>
    </InsightSection>
  );
}
