import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { QueryError } from "@/client/components/QueryState";
import { Button } from "@/client/components/ui/button";
import { Checkbox } from "@/client/components/ui/checkbox";
import { Skeleton } from "@/client/components/ui/skeleton";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import {
  analyticsLeadsQueryOptions,
  saveAnalyticsLeads,
  type AnalyticsSummary,
} from "./analyticsApi";
import { actionLabel, countOf } from "./presentation";

export const LEAD_ACTIONS_ID = "lead-actions";

/** The most lead actions the API keeps (backend/internal/analytics/leads.go). */
const MAX_LEAD_ACTIONS = 20;

interface LeadOption {
  name: string;
  count: number;
  visitors: number;
  seen: boolean;
}

/**
 * Every action the owner can pick: the saved ones first (even when nothing
 * happened in this period, so a pick never disappears), then the period's
 * recorded actions, busiest first.
 */
export function leadOptions(
  saved: string[],
  events: AnalyticsSummary["events"],
): LeadOption[] {
  const byName = new Map(events.map((row) => [row.name, row]));
  const options: LeadOption[] = saved.map((name) => {
    const row = byName.get(name);
    return {
      name,
      count: row?.count ?? 0,
      visitors: row?.visitors ?? 0,
      seen: row !== undefined,
    };
  });
  for (const row of events) {
    if (!saved.includes(row.name)) options.push({ ...row, seen: true });
  }
  return options;
}

/**
 * Lets the owner say which recorded actions are leads. Nothing is guessed:
 * until a pick is saved, Analytics shows no lead numbers.
 */
export function LeadActionsCard({
  projectId,
  events,
}: {
  projectId: string;
  events: AnalyticsSummary["events"];
}) {
  const client = useQueryClient();
  const leadsQuery = useQuery(analyticsLeadsQueryOptions(projectId));
  const [draft, setDraft] = useState<string[] | null>(null);
  const [editing, setEditing] = useState(false);
  const save = useMutation({
    mutationFn: (names: string[]) => saveAnalyticsLeads(projectId, names),
    meta: { errorToast: false },
    onSuccess: (names) => {
      client.setQueryData(
        analyticsLeadsQueryOptions(projectId).queryKey,
        names,
      );
      void client.invalidateQueries({
        queryKey: ["goAnalyticsSummary", projectId],
      });
      setDraft(null);
      setEditing(false);
    },
  });

  if (leadsQuery.isPending) return <Skeleton className="h-40" />;
  if (leadsQuery.isError)
    return (
      <QueryError
        error={leadsQuery.error}
        fallback="Could not load your lead actions."
        onRetry={() => void leadsQuery.refetch()}
        isRetrying={leadsQuery.isFetching}
      />
    );

  const saved = leadsQuery.data;
  const chosen = draft ?? saved;
  const options = leadOptions(saved, events);
  const changed =
    chosen.length !== saved.length || chosen.some((n) => !saved.includes(n));
  const open = editing || saved.length === 0;

  return (
    <section
      id={LEAD_ACTIONS_ID}
      aria-labelledby="lead-actions-title"
      className="scroll-mt-4 space-y-4 rounded-2xl border bg-card p-5"
    >
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div className="space-y-1">
          <h2 id="lead-actions-title" className="font-serif text-2xl">
            What counts as a lead?
          </h2>
          <p className="text-sm text-muted-foreground">
            {saved.length > 0
              ? `Leads: ${saved.map(actionLabel).join(", ")}.`
              : "Pick the actions that mean business for you. We never guess."}
          </p>
        </div>
        {open ? null : (
          <Button variant="outline" size="sm" onClick={() => setEditing(true)}>
            Change
          </Button>
        )}
      </div>

      {open ? (
        options.length === 0 ? (
          <TagHelp />
        ) : (
          <>
            <ul className="grid gap-2 sm:grid-cols-2">
              {options.map((option) => {
                const checked = chosen.includes(option.name);
                const full = !checked && chosen.length >= MAX_LEAD_ACTIONS;
                return (
                  <li key={option.name}>
                    <label className="flex min-h-12 cursor-pointer items-center gap-3 rounded-xl border px-3 py-2 has-[[data-checked]]:border-primary">
                      <Checkbox
                        checked={checked}
                        disabled={full || save.isPending}
                        onCheckedChange={(next) =>
                          setDraft(
                            next
                              ? [...chosen, option.name]
                              : chosen.filter((n) => n !== option.name),
                          )
                        }
                      />
                      <span className="min-w-0 text-sm">
                        <span className="block truncate font-medium">
                          {actionLabel(option.name)}
                        </span>
                        <span className="block text-xs text-muted-foreground">
                          {option.seen
                            ? `${countOf(option.count, "time", "times")} · ${countOf(option.visitors, "visitor-day", "visitor-days")}`
                            : "Not recorded in this period"}
                        </span>
                      </span>
                    </label>
                  </li>
                );
              })}
            </ul>
            {save.isError ? (
              <p role="alert" className="text-sm text-destructive">
                {getStandardErrorMessage(
                  save.error,
                  "Could not save. Please try again.",
                )}
              </p>
            ) : null}
            <div className="flex flex-wrap items-center gap-3">
              <Button
                disabled={!changed || save.isPending}
                onClick={() => save.mutate(chosen)}
              >
                {save.isPending ? "Saving…" : "Save lead actions"}
              </Button>
              {saved.length > 0 ? (
                <Button
                  variant="ghost"
                  disabled={save.isPending}
                  onClick={() => {
                    setDraft(null);
                    setEditing(false);
                  }}
                >
                  Cancel
                </Button>
              ) : null}
            </div>
            <TagHelp />
          </>
        )
      ) : null}
    </section>
  );
}

function TagHelp() {
  return (
    <p className="text-xs text-muted-foreground">
      Missing an action? Add{" "}
      <code className="text-foreground">data-uplumo-event="Signup"</code> to any
      button or link, or call{" "}
      <code className="text-foreground">uplumo("Signup")</code> from your code.
      It shows here after the first click.
    </p>
  );
}
