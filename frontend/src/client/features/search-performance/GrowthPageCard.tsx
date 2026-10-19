import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { toast } from "sonner";
import { Button } from "@/client/components/ui/button";
import { Input } from "@/client/components/ui/input";
import { Badge } from "@/client/components/ui/badge";
import { QueryError } from "@/client/components/QueryState";
import { safeHttpUrl } from "@/shared/safe-url";
import { saveSearchContext } from "@/client/features/content-editor/searchContext";
import {
  saveGrowthAction,
  type GrowthPage,
  type GrowthReport,
} from "./growthApi";
import { GrowthClickDiagnosis } from "./GrowthClickDiagnosis";
import { growthCount as count, growthRate as rate } from "./growthFormat";

export function GrowthPageCard({
  projectId,
  report,
  page,
  queryKind,
  onAnalyst,
}: {
  projectId: string;
  report: GrowthReport;
  page: GrowthPage;
  queryKind: string;
  onAnalyst: () => void;
}) {
  const [note, setNote] = useState("");
  const [controlPage, setControlPage] = useState("");
  const navigate = useNavigate();
  function openEditor() {
    try {
      saveSearchContext(projectId, {
        page: page.page,
        queries: page.queries.map((q) => q.query.slice(0, 200)),
        reasons: page.findings.map(
          (f) => `${f.title}: ${f.evidence} ${f.action}`,
        ),
        startDate: report.range.startDate,
        endDate: report.range.endDate,
      });
    } catch {
      toast.error(
        "Could not prepare this page's context. Allow tab storage and try again.",
      );
      return;
    }
    void navigate({
      to: "/p/$projectId/content-editor",
      params: { projectId },
      search: { keyword: page.queries[0]?.query.slice(0, 200), growth: "1" },
    });
  }
  const client = useQueryClient();
  const save = useMutation({
    mutationFn: () =>
      saveGrowthAction(projectId, report.id, page.page, note, controlPage),
    retry: false,
    onSuccess: () => {
      toast.success("Action and baseline saved");
      void client.invalidateQueries({
        queryKey: ["growth-actions", projectId],
      });
    },
  });
  const url = safeHttpUrl(page.page);
  const queries = page.queries.filter(
    (q) =>
      queryKind === "all" || (queryKind === "branded" ? q.branded : !q.branded),
  );
  return (
    <article className="space-y-3 rounded-xl border p-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className="break-all text-sm font-medium">
          {url ? (
            <a
              href={url}
              target="_blank"
              rel="noreferrer noopener"
              className="underline underline-offset-4"
            >
              {page.page}
            </a>
          ) : (
            page.page
          )}
        </h3>
        <Badge variant="secondary">Review candidate</Badge>
      </div>
      <div className="grid grid-cols-2 gap-2 text-sm md:grid-cols-4">
        <span>
          <strong>{count(page.current.clicks)}</strong> Google clicks
        </span>
        <span>
          <strong>{count(page.current.sessions)}</strong> GA4 sessions
        </span>
        <span>
          <strong>{count(page.current.keyEvents)}</strong> all key events
        </span>
        <span>
          <strong>
            {rate(
              report.goalEvent
                ? (page.current.goalRate ?? null)
                : page.current.keyEventRate,
            )}
          </strong>{" "}
          {report.goalEvent
            ? `${report.goalEvent} session rate`
            : "session key-event rate"}
        </span>
      </div>
      {page.current.revenue !== null &&
      (page.current.revenue > 0 || (page.previous?.revenue ?? 0) > 0) ? (
        <p className="text-xs text-muted-foreground">
          Recorded purchase revenue:{" "}
          {page.current.revenue.toLocaleString(undefined, {
            maximumFractionDigits: 2,
          })}{" "}
          {report.currency}. Only purchases reported to GA4 are included.
        </p>
      ) : null}
      {page.findings.slice(0, 1).map((finding) => (
        <div key={finding.kind} className="text-sm">
          <h4 className="font-medium">{finding.title}</h4>
          <p className="mt-1 text-muted-foreground">{finding.evidence}</p>
          <p className="mt-1">{finding.action}</p>
        </div>
      ))}
      <div className="flex flex-wrap gap-2">
        <Button size="sm" variant="outline" onClick={onAnalyst}>
          Create AI improvement brief
        </Button>
        <Button size="sm" variant="ghost" onClick={openEditor}>
          Improve this page in editor
        </Button>
      </div>
      <GrowthClickDiagnosis projectId={projectId} report={report} page={page} />
      <details className="text-sm">
        <summary className="cursor-pointer text-muted-foreground">
          Query evidence and save action
        </summary>
        <div className="mt-3 space-y-3">
          {page.findings.slice(1).map((finding) => (
            <p key={finding.kind} className="text-xs text-muted-foreground">
              <strong>{finding.title}:</strong> {finding.evidence}{" "}
              {finding.action}
            </p>
          ))}
          {queries.length ? (
            <ul className="space-y-1 text-xs">
              {queries.map((q) => (
                <li key={q.query}>
                  {q.query} · {count(q.impressions)} impressions · position{" "}
                  {q.position.toFixed(1)}
                  {q.branded ? " · Branded" : ""}
                  {q.competingPages.length
                    ? ` · Also appears on ${q.competingPageCount ?? q.competingPages.length} other returned pages`
                    : ""}
                </li>
              ))}
            </ul>
          ) : (
            <p className="text-xs text-muted-foreground">
              No returned queries match this filter. Query data can be omitted
              by Google.
            </p>
          )}
          <label className="grid gap-1">
            Optional unchanged comparison page
            <select
              className="max-w-full rounded border bg-background p-2"
              value={controlPage}
              onChange={(e) => setControlPage(e.target.value)}
            >
              <option value="">No comparison page</option>
              {report.pages
                .filter((p) => p.page !== page.page)
                .slice(0, 100)
                .map((p) => (
                  <option key={p.page} value={p.page}>
                    {p.page}
                  </option>
                ))}
            </select>
          </label>
          <p className="text-xs text-muted-foreground">
            Choose a similar page you will leave unchanged. This helps
            contextualize demand changes, but does not establish causation.
          </p>
          <label className="grid gap-1">
            What will you change?
            <Input
              maxLength={1000}
              placeholder="Review mobile enquiry CTA and clarify the offer"
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
          </label>
          <Button
            size="sm"
            disabled={!note.trim() || save.isPending || save.isSuccess}
            onClick={() => save.mutate()}
          >
            {save.isSuccess ? "Saved" : "Save action and baseline"}
          </Button>
          {save.isError ? (
            <QueryError
              error={save.error}
              fallback="Could not save this action."
            />
          ) : null}
        </div>
      </details>
    </article>
  );
}
