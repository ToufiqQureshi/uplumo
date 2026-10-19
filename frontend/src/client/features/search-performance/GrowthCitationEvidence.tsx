import { useMutation } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Button } from "@/client/components/ui/button";
import { QueryError } from "@/client/components/QueryState";
import { safeHttpUrl } from "@/shared/safe-url";
import {
  trackingApi,
  type TrackingAnswer,
} from "@/client/features/ai-search/trackingApi";
import type { GrowthReport } from "./growthApi";

export function GrowthCitationEvidence({
  projectId,
  report,
}: {
  projectId: string;
  report: GrowthReport;
}) {
  const ai = useMutation({
    mutationFn: async () => {
      const state = await trackingApi.state(projectId);
      const run = state.runs.find((r) => r.status === "completed");
      if (!run) return { run: null, answers: [] as TrackingAnswer[] };
      return { run, answers: await trackingApi.answers(projectId, run.id) };
    },
    retry: false,
  });
  return (
    <section className="space-y-2">
      <h4 className="font-medium">Google AI visibility</h4>
      <p className="text-xs text-muted-foreground">
        Review Google's native generative AI report for AI Overviews and AI Mode
        impressions. It has a different scope from saved prompt samples and GA4
        referrals. Uplumo does not automatically import this report.
      </p>
      <a
        className="text-xs underline"
        target="_blank"
        rel="noreferrer noopener"
        href={`https://search.google.com/search-console?resource_id=${encodeURIComponent(report.siteUrl)}`}
      >
        Open this property in Google Search Console
      </a>
      <p className="text-xs">
        <a
          className="underline"
          target="_blank"
          rel="noreferrer noopener"
          href="https://support.google.com/webmasters/answer/16984139"
        >
          How to use Google's generative AI report
        </a>
      </p>
      <Button
        variant="outline"
        size="sm"
        disabled={ai.isPending}
        onClick={() => ai.mutate()}
      >
        Load saved AI citation evidence
      </Button>
      <p className="text-xs text-muted-foreground">
        Reads existing tracked answers only. Does not buy new AI answers or
        enable a schedule.
      </p>
      {ai.isError ? (
        <QueryError
          error={ai.error}
          fallback="Could not load saved AI evidence."
        />
      ) : null}
      {ai.data ? (
        ai.data.run ? (
          <div className="space-y-3">
            <p className="text-xs text-muted-foreground">
              Saved run: {new Date(ai.data.run.createdAt).toLocaleString()} ·{" "}
              {ai.data.run.completed} completed, {ai.data.run.failed} failed.
              These prompt/model samples do not represent every consumer AI
              answer and use a separate date scope from GSC/GA4.
            </p>
            {ai.data.answers
              .filter((a) => a.status === "completed" && a.answer.trim())
              .slice(0, 10)
              .map((answer) => (
                <details key={answer.id} className="rounded border p-2">
                  <summary className="cursor-pointer">
                    {answer.prompt} · {answer.engine}
                  </summary>
                  <p className="mt-2 whitespace-pre-wrap text-xs text-muted-foreground">
                    {answer.answer.slice(0, 1600)}
                    {answer.answer.length > 1600 ? "…" : ""}
                  </p>
                  <ul className="mt-2 space-y-1 text-xs">
                    {answer.citations.slice(0, 10).map((citation, i) => {
                      const url = safeHttpUrl(citation.url);
                      return (
                        <li key={`${citation.url}:${i}`}>
                          {url ? (
                            <a
                              className="break-all underline"
                              href={url}
                              target="_blank"
                              rel="noreferrer noopener"
                            >
                              {citation.title || citation.url}
                            </a>
                          ) : (
                            "Unsafe source URL omitted"
                          )}
                          {citation.matchedBrand
                            ? " · Tracked brand source"
                            : " · Other source"}
                        </li>
                      );
                    })}
                  </ul>
                  <p className="mt-2 text-xs">
                    Compare what these cited sources answer with your existing
                    page. Absence from this sample is not proof of absence from
                    AI search.
                  </p>
                </details>
              ))}
          </div>
        ) : (
          <p className="text-xs">No completed AI tracking run is saved yet.</p>
        )
      ) : null}
      <Link
        to="/p/$projectId/prompt-tracking"
        params={{ projectId }}
        className="text-xs underline"
      >
        Review full prompt tracking and citation coverage
      </Link>
    </section>
  );
}
