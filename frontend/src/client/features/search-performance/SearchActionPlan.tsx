import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";
import { ArrowUpRight } from "lucide-react";
import { Button } from "@/client/components/ui/button";
import { safeHttpUrl } from "@/shared/safe-url";
import { formatUrlForDisplay } from "@/client/components/table/url";
import { saveSearchContext } from "@/client/features/content-editor/searchContext";
import {
  buildContentRefreshQueue,
  type RefreshQuery,
} from "./contentRefreshQueue";
import { formatCount } from "./SearchPerformanceColumns";

export function SearchActionPlan({
  projectId,
  rows,
  range,
}: {
  projectId: string;
  rows: RefreshQuery[];
  range: { startDate: string; endDate: string };
}) {
  const navigate = useNavigate();
  const pages = buildContentRefreshQueue(rows)
    .filter((page) => safeHttpUrl(page.page))
    .slice(0, 3);

  function reviewPage(item: (typeof pages)[number]) {
    try {
      saveSearchContext(projectId, {
        page: item.page,
        queries: item.queries.map((query) => query.query.slice(0, 200)),
        reasons: [
          "GSC review candidate, ordered by returned impressions, not revenue. Confirm relevance to your business before editing.",
          ...item.queries.map(
            (query) =>
              `${query.query.slice(0, 200)}: ${query.impressions} impressions, ${query.clicks} clicks, average position ${query.position.toFixed(1)}. Review the existing title, opening answer and query coverage; no cause or improvement is established.`,
          ),
        ],
        ...range,
      });
    } catch {
      toast.error(
        "Could not preserve this page's evidence. Allow tab storage and try again.",
      );
      return;
    }
    void navigate({
      to: "/p/$projectId/content-editor",
      params: { projectId },
      search: { keyword: item.queries[0]?.query.slice(0, 200), growth: "1" },
    });
  }

  return (
    <section
      aria-labelledby="search-actions-heading"
      className="overflow-hidden rounded-xl border border-border bg-card"
    >
      <div className="border-b border-border px-4 py-4 sm:px-5">
        <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
          Start here · Google Search Console
        </p>
        <h2 id="search-actions-heading" className="mt-1 text-lg font-semibold">
          Your next search actions
        </h2>
        <p className="mt-1 max-w-2xl text-sm text-muted-foreground">
          Review an existing page people already find in search. Pick one that
          matters to your business before changing it.
        </p>
        <p className="mt-2 text-xs text-muted-foreground">
          {range.startDate} – {range.endDate} · Current report filters apply ·
          No AI credits used
        </p>
      </div>
      {pages.length ? (
        <div className="divide-y divide-border">
          {pages.map((item, index) => {
            const query = item.queries[0];
            return (
              <article
                key={item.page}
                className="flex min-w-0 gap-3 px-4 py-5 sm:gap-4 sm:px-5"
              >
                <span
                  aria-hidden="true"
                  className="flex size-7 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-medium"
                >
                  {index + 1}
                </span>
                <div className="min-w-0 flex-1 space-y-3">
                  <div>
                    <h3 className="break-words font-medium">
                      Review how this page answers “{query.query}”
                    </h3>
                    <a
                      href={safeHttpUrl(item.page) ?? undefined}
                      target="_blank"
                      rel="noreferrer noopener"
                      className="mt-1 inline-flex max-w-full items-start gap-1 text-xs text-muted-foreground underline underline-offset-4"
                    >
                      <span className="break-all">
                        {formatUrlForDisplay(item.page)}
                      </span>
                      <ArrowUpRight
                        className="size-3 shrink-0"
                        aria-hidden="true"
                      />
                    </a>
                  </div>
                  <p className="text-sm">
                    <strong>{formatCount(query.impressions)}</strong>{" "}
                    impressions → <strong>{formatCount(query.clicks)}</strong>{" "}
                    clicks · average position{" "}
                    <strong>{query.position.toFixed(1)}</strong> for this query.
                  </p>
                  <ol className="list-decimal space-y-1 pl-4 text-sm text-muted-foreground">
                    <li>
                      Open the page and check its title and opening answer
                      against “{query.query}”.
                    </li>
                    <li>
                      Check for a missing answer or outdated fact. Keep useful
                      content; edit only a gap you can verify.
                    </li>
                  </ol>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => reviewPage(item)}
                  >
                    Review this page in editor
                  </Button>
                </div>
              </article>
            );
          })}
        </div>
      ) : (
        <div className="space-y-2 px-4 py-5 sm:px-5">
          <h3 className="font-medium">
            Not enough evidence for a page task yet
          </h3>
          <p className="text-sm text-muted-foreground">
            Try a wider date range or clear filters in the detailed report. If
            this site is new, let Search Console collect more data.
          </p>
        </div>
      )}
      <p className="border-t border-border px-4 py-3 text-xs text-muted-foreground sm:px-5">
        Up to 3 review candidates, ordered by returned impressions, not revenue.
        Based on query rows at positions 5–20, with at least 50 impressions per
        page. Google can omit rows. These signals do not prove a content problem
        or predict enquiries.
      </p>
    </section>
  );
}
