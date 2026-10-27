import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Check } from "lucide-react";
import { useState } from "react";
import { PageHeader } from "@/client/components/PageHeader";
import { Button } from "@/client/components/ui/button";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import {
  getAuditHistory,
  getAuditResults,
} from "@/client/features/audit/auditApi";
import { getSearchPerformanceTable } from "@/client/features/search-performance/searchPerformanceApi";
import { AiAction, InsightCard } from "./AiAction";
import { insightsApi, nextActionsKey, type ActionStatus } from "./insightsApi";
import { buildNextActions, type ActionType } from "./nextActions";
import { useLatestAnswers } from "./useLatestAnswers";

const TABS: { id: ActionStatus; label: string }[] = [
  { id: "todo", label: "To do" },
  { id: "done", label: "Done" },
  { id: "ignored", label: "Ignored" },
];
const TYPES: ("All" | ActionType)[] = ["All", "Technical", "Content", "AI"];

/** Everything here reads saved data; only the AI buttons spend credits. */
export function NextActionsPage({ projectId }: { projectId: string }) {
  const client = useQueryClient();
  const [tab, setTab] = useState<ActionStatus>("todo");
  const [type, setType] = useState<"All" | ActionType>("All");
  const [open, setOpen] = useState<string | null>(null);

  const audit = useQuery({
    queryKey: ["next-actions", projectId, "audit"],
    staleTime: 5 * 60_000,
    queryFn: async () => {
      const history = await getAuditHistory({ data: { projectId } });
      const latest = history.find((a) => a.status === "completed");
      if (!latest) return null;
      return getAuditResults({ data: { projectId, auditId: latest.id } });
    },
  });
  const gsc = useQuery({
    queryKey: ["next-actions", projectId, "gsc"],
    staleTime: 10 * 60_000,
    retry: false,
    queryFn: () =>
      getSearchPerformanceTable({
        data: { projectId, dimension: "query", pageSize: 100 },
      }),
  });
  const ai = useLatestAnswers(projectId);
  const states = useQuery({
    queryKey: nextActionsKey(projectId),
    queryFn: () => insightsApi.actionStates(projectId),
  });
  const setStatus = useMutation({
    mutationFn: ({ key, status }: { key: string; status: ActionStatus }) =>
      insightsApi.setActionStatus(projectId, key, status),
    onSettled: () =>
      client.invalidateQueries({ queryKey: nextActionsKey(projectId) }),
  });

  const gscRows = gsc.data?.connected ? gsc.data.rows : [];
  const actions = buildNextActions({
    issues: audit.data?.issues ?? [],
    queries: gscRows,
    answers: ai.answers,
    brand: ai.configuration?.brand,
  });
  const statusOf = (key: string): ActionStatus =>
    states.data?.states.find((s) => s.key === key)?.status ?? "todo";
  const count = (s: ActionStatus) =>
    actions.filter((a) => statusOf(a.key) === s).length;
  const rows = actions.filter(
    (a) => statusOf(a.key) === tab && (type === "All" || a.type === type),
  );
  const todo = actions.filter((a) => statusOf(a.key) === "todo");
  const gain = todo
    .slice(0, 3)
    .reduce((sum, a) => sum + (a.gainClicks ?? 0), 0);
  const done = count("done");
  const loading = audit.isPending || ai.isPending || states.isPending;

  const sources = [
    audit.data ? "Site audit" : null,
    gsc.data?.connected ? "Search Console" : null,
    ai.run ? "AI Visibility" : null,
  ].filter(Boolean);
  const missing = [
    !audit.data && !audit.isPending ? "run a site audit" : null,
    gsc.data && !gsc.data.connected ? "connect Search Console" : null,
    !ai.run && !ai.isPending ? "run an AI Visibility check" : null,
  ].filter(Boolean);

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col gap-6 p-4 sm:p-6">
      <PageHeader
        title="Next Actions"
        description="Your to-do list, ranked by payoff. Built from data you already have."
        actions={
          actions.length ? (
            <span className="font-mono text-sm text-muted-foreground">
              {done}/{actions.length} done
            </span>
          ) : null
        }
      />

      {loading ? (
        <p className="text-sm text-muted-foreground">Reading your data…</p>
      ) : (
        <InsightCard
          stamp={`From ${sources.join(", ") || "no data yet"}`}
          headline={
            todo.length === 0 ? (
              actions.length ? (
                "Everything on the list is handled. New items appear after your next audit or check."
              ) : (
                `Not enough data yet. To build your list, ${missing.join(", ") || "come back after your first audit"}.`
              )
            ) : gain > 0 ? (
              <>
                Do the top {Math.min(3, todo.length)} this week for an estimated{" "}
                <span className="text-primary">
                  +{gain.toLocaleString("en-IN")} clicks a month
                </span>
                .
              </>
            ) : (
              <>
                You have <span className="text-primary">{todo.length}</span>{" "}
                open item{todo.length === 1 ? "" : "s"}. Start with the first:
                it has the highest impact.
              </>
            )
          }
          evidence={[
            ...(gscRows.length
              ? [
                  {
                    source: "Search Console · 28 days",
                    text: `${gscRows.filter((r) => r.position >= 8 && r.position <= 20).length} searches sit on page 2`,
                  },
                ]
              : []),
            ...(audit.data
              ? [
                  {
                    source: "Site audit",
                    text: `${audit.data.issues.length} issues on ${audit.data.audit.pagesCrawled} pages`,
                  },
                ]
              : []),
            ...(ai.run
              ? [
                  {
                    source: "AI Visibility",
                    text: `Named in ${ai.run.mentioned} of ${ai.run.answered} answers`,
                  },
                ]
              : []),
          ]}
        >
          {missing.length && actions.length ? (
            <p className="text-sm text-muted-foreground">
              For a fuller list, {missing.join(" and ")}.
            </p>
          ) : null}
        </InsightCard>
      )}

      <div className="flex flex-wrap justify-between gap-2">
        <div role="tablist" aria-label="Status" className="flex gap-2">
          {TABS.map((t) => (
            <Button
              key={t.id}
              role="tab"
              aria-selected={tab === t.id}
              variant={tab === t.id ? "default" : "outline"}
              onClick={() => setTab(t.id)}
            >
              {t.label}{" "}
              <span className="font-mono opacity-70">{count(t.id)}</span>
            </Button>
          ))}
        </div>
        <div className="flex flex-wrap gap-2">
          {TYPES.map((t) => (
            <Button
              key={t}
              size="sm"
              aria-pressed={type === t}
              variant={type === t ? "secondary" : "ghost"}
              onClick={() => setType(t)}
            >
              {t}
            </Button>
          ))}
        </div>
      </div>
      {setStatus.isError ? (
        <p role="alert" className="text-sm text-destructive">
          {getStandardErrorMessage(setStatus.error)}
        </p>
      ) : null}

      <ol className="flex flex-col divide-y rounded-xl border bg-card">
        {rows.length === 0 ? (
          <li className="p-10 text-center text-sm text-muted-foreground">
            Nothing here.
          </li>
        ) : null}
        {rows.map((a, i) => {
          const isDone = statusOf(a.key) === "done";
          const isOpen = open === a.key;
          return (
            <li key={a.key} className="flex gap-4 p-4">
              <Button
                variant={isDone ? "default" : "outline"}
                size="icon"
                className="shrink-0"
                aria-label={isDone ? "Mark as not done" : "Mark as done"}
                onClick={() =>
                  setStatus.mutate({
                    key: a.key,
                    status: isDone ? "todo" : "done",
                  })
                }
              >
                {isDone ? <Check aria-hidden /> : null}
              </Button>
              <div className="flex min-w-0 flex-1 flex-col gap-2">
                <div className="flex flex-wrap items-start justify-between gap-2">
                  <button
                    type="button"
                    className="text-left font-medium"
                    aria-expanded={isOpen}
                    onClick={() => setOpen(isOpen ? null : a.key)}
                  >
                    {i + 1}. {a.title}
                  </button>
                  {a.gainClicks ? (
                    <span className="rounded-full border border-primary/40 px-2.5 py-0.5 font-mono text-xs text-primary">
                      ≈ +{a.gainClicks.toLocaleString("en-IN")} clicks/mo
                    </span>
                  ) : null}
                </div>
                <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                  <span className="rounded-full border px-2 py-0.5">
                    {a.impact} impact
                  </span>
                  <span className="rounded-full border px-2 py-0.5">
                    {a.effort}
                  </span>
                  <span className="rounded-full border px-2 py-0.5">
                    {a.type}
                  </span>
                  <span className="font-mono">Source: {a.source}</span>
                </div>
                <p className="text-sm text-muted-foreground">{a.why}</p>
                {isOpen && a.ai ? (
                  <AiAction
                    projectId={projectId}
                    label={a.ai.label}
                    input={{
                      action: a.ai.action,
                      brief: a.ai.brief,
                      content: a.ai.content,
                      targetKeyword: a.ai.targetKeyword,
                    }}
                  />
                ) : null}
                <div className="flex flex-wrap gap-2">
                  {a.ai && !isOpen ? (
                    <Button size="sm" onClick={() => setOpen(a.key)}>
                      {a.ai.label} with AI
                    </Button>
                  ) : null}
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() =>
                      setStatus.mutate({
                        key: a.key,
                        status:
                          statusOf(a.key) === "ignored" ? "todo" : "ignored",
                      })
                    }
                  >
                    {statusOf(a.key) === "ignored" ? "Restore" : "Ignore"}
                  </Button>
                </div>
              </div>
            </li>
          );
        })}
      </ol>
    </div>
  );
}
