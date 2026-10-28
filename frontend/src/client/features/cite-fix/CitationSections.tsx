import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { useState } from "react";
import { Button } from "@/client/components/ui/button";
import { ConfirmDialog } from "@/client/components/ConfirmDialog";
import { SafeExternalLink } from "@/client/components/SafeExternalLink";
import type { CiteFixReport, GoogleAIOverviewState } from "./citeFixApi";
import {
  citeFixKey,
  collectCiteFixGoogleAIOverview,
  refreshCiteFixGoogleAIOverview,
} from "./citeFixApi";

export function GoogleAIOverviewSection({
  projectId,
  initial,
}: {
  projectId: string;
  initial: GoogleAIOverviewState | null;
}) {
  const queryClient = useQueryClient();
  const [confirming, setConfirming] = useState(false);
  const [pendingAfterRefresh, setPendingAfterRefresh] = useState(false);
  const queryKey = ["cite-fix", projectId, "google-ai-overview"] as const;
  const hasPending =
    initial?.results.some((result) => result.status === "pending") ?? false;
  const collection = useQuery({
    queryKey,
    queryFn: () => collectCiteFixGoogleAIOverview(projectId),
    enabled: hasPending || pendingAfterRefresh,
    retry: false,
    refetchInterval: (query) =>
      query.state.data?.results.some((result) => result.status === "pending")
        ? 5000
        : false,
  });
  const state = collection.data ?? initial;
  const refresh = useMutation({
    retry: false,
    mutationFn: () => refreshCiteFixGoogleAIOverview(projectId),
    onSuccess: (next) => {
      queryClient.setQueryData(queryKey, next);
      setPendingAfterRefresh(
        next.results.some((result) => result.status === "pending"),
      );
      queryClient.setQueryData<CiteFixReport>(citeFixKey(projectId), (old) =>
        old ? { ...old, googleAiOverview: next } : old,
      );
      setConfirming(false);
    },
  });
  const canRefresh = Boolean(
    state?.available &&
    state.results.some(
      (result) => result.status === "not_checked" || result.status === "stale",
    ),
  );

  if (!state) return null;
  return (
    <section
      aria-labelledby="cite-fix-google-aio"
      className="flex flex-col gap-3"
    >
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 id="cite-fix-google-aio" className="font-serif text-2xl">
            Google AI Overview citations
          </h2>
          <p className="text-sm text-muted-foreground">
            {!state.providerConfigured
              ? "Google AI Overview checks are not available on this server."
              : state.available
                ? `Check up to ${state.maxQuestions} tracked questions. Results are saved for seven days.`
                : "Save tracked questions in AI Visibility to check Google's AI Overview."}
          </p>
        </div>
        {state.available ? (
          <Button
            size="sm"
            disabled={!canRefresh || refresh.isPending}
            pending={refresh.isPending}
            onClick={() => setConfirming(true)}
          >
            {canRefresh ? "Check citations" : "Saved for seven days"}
          </Button>
        ) : state.providerConfigured ? (
          <Button
            size="sm"
            variant="outline"
            nativeButton={false}
            render={
              <Link to="/p/$projectId/prompt-tracking" params={{ projectId }} />
            }
          >
            AI Visibility Tracker
          </Button>
        ) : null}
      </div>
      {state.results.length ? (
        <div className="flex flex-col gap-3">
          {state.results.map((result) => (
            <article
              key={result.prompt}
              className="rounded-xl border bg-card p-4"
            >
              <div className="flex flex-wrap justify-between gap-2">
                <h3 className="font-medium">{result.prompt}</h3>
                <span className="text-xs text-muted-foreground">
                  {statusLabel(result.status)}
                </span>
              </div>
              {result.status === "completed" ? (
                <>
                  <p className="mt-2 text-sm">
                    Your domain{" "}
                    {result.customerCited ? "is cited" : "was not cited"} in the
                    AI Overview.
                  </p>
                  {result.answer ? (
                    <p className="mt-2 whitespace-pre-wrap text-sm text-muted-foreground">
                      {result.answer}
                    </p>
                  ) : null}
                  <ul className="mt-3 flex flex-col gap-2">
                    {result.citations.map((citation) => (
                      <li key={citation.url} className="text-sm">
                        <SafeExternalLink
                          url={citation.url}
                          label={
                            citation.title || citation.domain || citation.url
                          }
                          className="inline-flex items-center gap-1 text-primary underline"
                        />
                        <p className="text-xs text-muted-foreground">
                          {[
                            citation.domain,
                            citation.source,
                            citation.publishedAt,
                          ]
                            .filter(Boolean)
                            .join(" · ") || "AI Overview reference"}
                        </p>
                        {citation.text ? (
                          <p className="mt-1 text-xs text-muted-foreground">
                            {citation.text}
                          </p>
                        ) : null}
                      </li>
                    ))}
                  </ul>
                  {result.requestedAt ? (
                    <p className="mt-2 text-xs text-muted-foreground">
                      Checked {new Date(result.requestedAt).toLocaleString()}
                    </p>
                  ) : null}
                </>
              ) : result.status === "pending" ? (
                <p role="status" className="mt-2 text-sm text-muted-foreground">
                  Google is preparing the AI Overview. Collecting the result
                  uses no additional credits.
                </p>
              ) : result.status === "not_checked" ||
                result.status === "stale" ? (
                <p className="mt-2 text-sm text-muted-foreground">
                  No current check is saved for this question.
                </p>
              ) : (
                <p className="mt-2 text-sm text-muted-foreground">
                  No result was available for this question. It will not be
                  retried automatically.
                </p>
              )}
            </article>
          ))}
        </div>
      ) : null}
      {confirming ? (
        <ConfirmDialog
          title="Check Google AI Overview citations?"
          confirmLabel={`Continue · up to ${state.maxCredits} credits`}
          pending={refresh.isPending}
          onClose={() => setConfirming(false)}
          onConfirm={() => refresh.mutate()}
        >
          This sends up to {state.maxQuestions} tracked questions to Google
          through DataForSEO. The maximum is {state.maxCredits} credits; the
          actual provider cost is charged. Results stay saved for seven days. No
          paid request runs until you confirm here.
        </ConfirmDialog>
      ) : null}
      {refresh.isError ? (
        <p role="alert" className="text-sm text-destructive">
          {refresh.error.message}
        </p>
      ) : null}
      {collection.isError ? (
        <p role="status" className="text-xs text-muted-foreground">
          The saved AI Overview task is still waiting to be collected.
        </p>
      ) : null}
    </section>
  );
}

function statusLabel(status: string) {
  switch (status) {
    case "not_checked":
      return "Not checked";
    case "pending":
      return "Waiting for Google";
    case "completed":
      return "Ready";
    case "failed":
      return "No result";
    case "unknown":
      return "Outcome unknown";
    case "stale":
      return "Refresh available";
    default:
      return status;
  }
}

export function SEOIssuesSection({
  projectId,
  report,
}: {
  projectId: string;
  report: CiteFixReport;
}) {
  if (!report.seoIssues.length) return null;
  return (
    <section
      aria-labelledby="cite-fix-seo-issues"
      className="flex flex-col gap-3 rounded-xl border bg-card p-5"
    >
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 id="cite-fix-seo-issues" className="font-serif text-2xl">
            Also fix for Google
          </h2>
          <p className="text-sm text-muted-foreground">
            Critical and warning issues from your latest Site Audit.
          </p>
        </div>
        <Button
          size="sm"
          variant="outline"
          nativeButton={false}
          render={<Link to="/p/$projectId/audit" params={{ projectId }} />}
        >
          Open Site Audit
        </Button>
      </div>
      <ul className="divide-y rounded-lg border">
        {report.seoIssues.map((issue, index) => (
          <li
            key={`${issue.type}:${issue.pageUrl}:${index}`}
            className="flex flex-col gap-1 p-3 text-sm"
          >
            <div className="flex flex-wrap justify-between gap-2">
              <span className="font-medium">{issue.title}</span>
              <span className="text-xs uppercase text-muted-foreground">
                {issue.severity}
              </span>
            </div>
            {issue.pageUrl ? (
              <SafeExternalLink
                url={issue.pageUrl}
                label={issue.pageUrl}
                className="inline-flex items-center gap-1 truncate text-xs text-primary underline"
              />
            ) : null}
            <p className="text-xs text-muted-foreground">{issue.howToFix}</p>
          </li>
        ))}
      </ul>
    </section>
  );
}
