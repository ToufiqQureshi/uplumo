import { ArrowDown, ArrowRight } from "lucide-react";
import { safeHttpUrl } from "@/shared/safe-url";
import { Button } from "@/client/components/ui/button";
import { buildContentRefreshQueue } from "./contentRefreshQueue";
import {
  formatCount,
  formatCtr,
  type Report,
} from "./SearchPerformanceColumns";

export function SearchPerformanceOverview({
  report,
  onReviewPages,
}: {
  report: Report;
  onReviewPages: () => void;
}) {
  const pages = buildContentRefreshQueue(report.strikingDistance);
  const change = report.totals.clicks - report.prevTotals.clicks;
  const comparable = report.range.comparisonAvailable;
  const headline = !comparable
    ? "Your search performance"
    : change < 0
      ? `${formatCount(-change)} fewer clicks from Google`
      : change > 0
        ? `${formatCount(change)} more clicks from Google`
        : "Google clicks are unchanged";

  return (
    <section
      aria-label="Search overview"
      className="overflow-hidden rounded-xl border border-border bg-card"
    >
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-border px-5 py-3 text-xs text-muted-foreground">
        <span>Google Search Console · Current report filters</span>
        <span>
          {report.range.startDate} – {report.range.endDate}
        </span>
      </div>
      <div className="space-y-5 p-5 md:p-6">
        <div>
          <h2 className="text-xl font-semibold">{headline}</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            {comparable
              ? `Compared with ${report.range.prevStartDate} – ${report.range.prevEndDate}. `
              : "A complete previous-period comparison is unavailable. "}
            Review the evidence before changing your content; these totals do
            not explain the cause.
          </p>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h3 className="font-semibold">Pages worth reviewing</h3>
            <p className="text-xs text-muted-foreground">
              Ranked by observed impressions, not predicted revenue. Choose
              pages relevant to your business.
            </p>
          </div>
          {pages.length > 0 ? (
            <Button size="sm" variant="outline" onClick={onReviewPages}>
              Review all {formatCount(pages.length)} pages <ArrowRight />
            </Button>
          ) : null}
        </div>
        {pages.length > 0 ? (
          <div className="grid gap-3 lg:grid-cols-3">
            {pages.slice(0, 3).map((page) => (
              <article
                key={page.page}
                className="min-w-0 rounded-lg border border-border p-4"
              >
                <p className="text-xs text-muted-foreground">
                  Average position {page.averagePosition.toFixed(1)}
                </p>
                <h4 className="mt-2 break-words font-medium">
                  {page.queries[0]?.query}
                </h4>
                <p className="mt-2 break-all text-xs text-muted-foreground">
                  {page.page}
                </p>
                <p className="mt-3 text-sm tabular-nums">
                  {formatCount(page.impressions)} impressions ·{" "}
                  {formatCount(page.clicks)} clicks ·{" "}
                  {formatCtr(page.clicks / page.impressions)} CTR
                </p>
                <p className="mt-3 text-xs text-muted-foreground">
                  Check whether this page answers these searches, then review
                  its title and internal links.
                </p>
                {safeHttpUrl(page.page) ? (
                  <a
                    className="mt-3 inline-flex text-sm font-medium underline underline-offset-4"
                    href={safeHttpUrl(page.page) ?? undefined}
                    target="_blank"
                    rel="noreferrer"
                  >
                    Open page to review
                  </a>
                ) : null}
              </article>
            ))}
          </div>
        ) : (
          <p className="rounded-lg bg-muted/50 p-4 text-sm text-muted-foreground">
            No candidate pages in the returned rows. Try a wider range or
            explore the full Queries and Pages reports below.
          </p>
        )}
        <a
          href="#search-details"
          className="inline-flex items-center gap-2 text-sm font-medium underline underline-offset-4"
        >
          Explore queries and pages <ArrowDown className="size-4" />
        </a>
        <p className="text-xs text-muted-foreground">
          Candidates group returned queries at positions 5–20 with at least 50
          impressions per page. Counts cover those queries only; Google can omit
          rows.
        </p>
      </div>
    </section>
  );
}
