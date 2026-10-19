import { useMutation } from "@tanstack/react-query";
import { Button } from "@/client/components/ui/button";
import { QueryError } from "@/client/components/QueryState";
import { inspectSearchConsoleUrls } from "./searchPerformanceApi";
import { recommendedNextStep } from "./indexingGuidance";
import {
  diagnoseGrowthClicks,
  type GrowthReport,
  type GrowthPage,
} from "./growthApi";
import { GrowthMobileResults } from "./GrowthMobileResults";
import { growthCount, growthRate } from "./growthFormat";

export function GrowthClickDiagnosis({
  projectId,
  report,
  page,
}: {
  projectId: string;
  report: GrowthReport;
  page: GrowthPage;
}) {
  const diagnosis = useMutation({
    mutationFn: () => diagnoseGrowthClicks(projectId, report.id, page.page),
    retry: false,
  });
  const inspection = useMutation({
    mutationFn: () =>
      inspectSearchConsoleUrls({ projectId, urls: [page.page] }),
    retry: false,
  });
  const index = inspection.data?.results[0]?.result?.indexStatusResult;
  return (
    <details className="text-sm">
      <summary className="cursor-pointer text-muted-foreground">
        Why are clicks low or falling?
      </summary>
      <div className="mt-3 space-y-3">
        <p className="text-xs text-muted-foreground">
          Investigate visibility, ranking and click-through changes for this
          page. These checks do not spend AI credits or edit your site.
        </p>
        <div className="flex flex-wrap gap-2">
          <Button
            className="h-auto min-h-9 max-w-full whitespace-normal py-2 text-left"
            size="sm"
            variant="outline"
            disabled={diagnosis.isPending}
            onClick={() => diagnosis.mutate()}
          >
            {diagnosis.isPending
              ? "Investigating…"
              : "Investigate this page's clicks"}
          </Button>
          <Button
            className="h-auto min-h-9 max-w-full whitespace-normal py-2 text-left"
            size="sm"
            variant="ghost"
            disabled={inspection.isPending}
            onClick={() => inspection.mutate()}
          >
            {inspection.isPending
              ? "Inspecting…"
              : "Check Google's stored index status"}
          </Button>
        </div>
        {diagnosis.isError ? (
          <QueryError
            error={diagnosis.error}
            fallback="Could not investigate clicks. Refresh analysis and try again."
          />
        ) : null}
        {inspection.isError ? (
          <QueryError
            error={inspection.error}
            fallback="Could not inspect index status."
          />
        ) : null}
        {inspection.data?.results[0]?.error ? (
          <p className="text-xs" role="status">
            {inspection.data.results[0].error}
          </p>
        ) : null}
        {inspection.isSuccess ? (
          <div className="rounded border border-border p-2 text-xs">
            {index ? (
              <>
                <p>
                  Stored index coverage: {index.coverageState || "Unknown"} ·{" "}
                  {index.verdict || "Unknown"}. Last crawl:{" "}
                  {index.lastCrawlTime || "Unknown"}.
                </p>
                <p className="mt-1">{recommendedNextStep(index)}</p>
              </>
            ) : (
              <p>No stored index result was returned.</p>
            )}
            <p className="mt-1 text-muted-foreground">
              This is Google's stored result, not a live page test or an
              indexing request.
            </p>
          </div>
        ) : null}
        {diagnosis.data ? (
          <div className="space-y-3">
            <p className="text-xs text-muted-foreground">
              {diagnosis.data.range.startDate} – {diagnosis.data.range.endDate}{" "}
              vs {diagnosis.data.previousRange.startDate} –{" "}
              {diagnosis.data.previousRange.endDate}. {diagnosis.data.message}
            </p>
            {diagnosis.data.truncated ? (
              <p className="text-xs">
                Query evidence reached the 1,000-row bound in at least one
                period.
              </p>
            ) : null}
            {diagnosis.data.findings.map((f) => (
              <div key={f.kind} className="rounded border border-border p-3">
                <p className="font-medium">{f.title}</p>
                <p className="mt-1 text-xs text-muted-foreground">
                  {f.evidence}
                </p>
                <p className="mt-1">{f.action}</p>
                <p className="mt-1 text-xs text-muted-foreground">
                  {f.confidence}
                </p>
              </div>
            ))}
            {diagnosis.data.queries.length ? (
              <>
                <GrowthMobileResults
                  label="Query comparison results"
                  rows={diagnosis.data.queries.map((q) => ({
                    key: q.query,
                    title: q.query,
                    subtitle: "Previous → current",
                    metrics: [
                      {
                        label: "Clicks",
                        value: `${growthCount(q.previous?.clicks ?? null)} → ${growthCount(q.current?.clicks ?? null)}`,
                      },
                      {
                        label: "Impressions",
                        value: `${growthCount(q.previous?.impressions ?? null)} → ${growthCount(q.current?.impressions ?? null)}`,
                      },
                      {
                        label: "Click-through rate",
                        value: `${growthRate(q.previous?.ctr ?? null)} → ${growthRate(q.current?.ctr ?? null)}`,
                      },
                      {
                        label: "Average position",
                        value: `${q.previous?.position.toFixed(1) ?? "Unknown"} → ${q.current?.position.toFixed(1) ?? "Unknown"}`,
                      },
                    ],
                  }))}
                />
                <div className="hidden overflow-x-auto sm:block">
                  <table className="w-full text-left text-xs">
                    <caption className="sr-only">
                      Returned query evidence, previous to current
                    </caption>
                    <thead>
                      <tr>
                        <th className="p-2">Query</th>
                        <th className="p-2">Clicks before → now</th>
                        <th className="p-2">Impressions before → now</th>
                        <th className="p-2">CTR before → now</th>
                        <th className="p-2">Position before → now</th>
                      </tr>
                    </thead>
                    <tbody>
                      {diagnosis.data.queries.map((q) => (
                        <tr key={q.query}>
                          <td className="p-2">{q.query}</td>
                          <td className="p-2">
                            {growthCount(q.previous?.clicks ?? null)} →{" "}
                            {growthCount(q.current?.clicks ?? null)}
                          </td>
                          <td className="p-2">
                            {growthCount(q.previous?.impressions ?? null)} →{" "}
                            {growthCount(q.current?.impressions ?? null)}
                          </td>
                          <td className="p-2">
                            {growthRate(q.previous?.ctr ?? null)} →{" "}
                            {growthRate(q.current?.ctr ?? null)}
                          </td>
                          <td className="p-2">
                            {q.previous?.position.toFixed(1) ?? "Unknown"} →{" "}
                            {q.current?.position.toFixed(1) ?? "Unknown"}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </>
            ) : (
              <p className="text-xs">
                No query rows returned; query-level reasons are unknown.
              </p>
            )}
          </div>
        ) : null}
      </div>
    </details>
  );
}
