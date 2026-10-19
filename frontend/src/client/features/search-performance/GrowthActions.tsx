import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/client/components/ui/button";
import { Badge } from "@/client/components/ui/badge";
import { QueryError } from "@/client/components/QueryState";
import {
  listGrowthActions,
  updateGrowthAction,
  deleteGrowthAction,
  compareGrowthAction,
  type GrowthAction,
} from "./growthApi";
import { growthRate as rate } from "./growthFormat";

export function GrowthActions({ projectId }: { projectId: string }) {
  const [opened, setOpened] = useState(false);
  const actions = useQuery({
    queryKey: ["growth-actions", projectId],
    queryFn: () => listGrowthActions(projectId),
    enabled: opened,
    retry: false,
    refetchOnWindowFocus: false,
  });
  return (
    <details
      onToggle={(e) => setOpened(e.currentTarget.open)}
      className="rounded-lg border p-3"
    >
      <summary className="cursor-pointer text-sm font-medium">
        Saved actions and results
      </summary>
      <div className="mt-3 space-y-3">
        {actions.isPending ? (
          <p className="text-sm">Loading saved actions…</p>
        ) : actions.isError ? (
          <QueryError
            error={actions.error}
            fallback="Could not load actions."
          />
        ) : actions.data.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            Save a page action to keep its baseline. Mark it applied after
            making the actual change, then compare a complete post-change
            window.
          </p>
        ) : (
          actions.data.map((a) => (
            <SavedAction key={a.id} projectId={projectId} action={a} />
          ))
        )}
      </div>
    </details>
  );
}
function SavedAction({
  projectId,
  action,
}: {
  projectId: string;
  action: GrowthAction;
}) {
  const client = useQueryClient();
  const refresh = () =>
    void client.invalidateQueries({ queryKey: ["growth-actions", projectId] });
  const update = useMutation({
    mutationFn: (status: GrowthAction["status"]) =>
      updateGrowthAction(projectId, action.id, status),
    retry: false,
    onSuccess: refresh,
  });
  const remove = useMutation({
    mutationFn: () => deleteGrowthAction(projectId, action.id),
    retry: false,
    onSuccess: refresh,
  });
  const comparison = useMutation({
    mutationFn: (days: number) =>
      compareGrowthAction(projectId, action.id, days),
    retry: false,
  });
  return (
    <article className="space-y-2 rounded border p-3 text-sm">
      <p className="break-all font-medium">{action.page}</p>
      <p>{action.note}</p>
      <Badge variant="secondary">{action.status}</Badge>
      <p className="text-xs text-muted-foreground">
        Baseline: {action.baselineRange.startDate} –{" "}
        {action.baselineRange.endDate}. Key events reflect the connected
        property's setup. Goal: {action.source.goalEvent || "all key events"}.
      </p>
      <div className="flex flex-wrap gap-2">
        {action.status !== "applied" ? (
          <Button
            size="sm"
            variant="outline"
            disabled={update.isPending}
            onClick={() => update.mutate("applied")}
          >
            Mark change applied
          </Button>
        ) : (
          <>
            {[14, 28].map((days) => (
              <Button
                key={days}
                size="sm"
                variant="outline"
                disabled={comparison.isPending}
                onClick={() => comparison.mutate(days)}
              >
                Compare {days} days
              </Button>
            ))}
          </>
        )}
        <Button
          size="sm"
          variant="ghost"
          disabled={update.isPending || action.status === "dismissed"}
          onClick={() => update.mutate("dismissed")}
        >
          Dismiss
        </Button>
        <Button
          size="sm"
          variant="ghost"
          disabled={remove.isPending}
          onClick={() => remove.mutate()}
        >
          Remove saved action
        </Button>
      </div>
      {update.isError || remove.isError || comparison.isError ? (
        <QueryError
          error={update.error ?? remove.error ?? comparison.error}
          fallback="Could not update this action."
        />
      ) : null}
      {comparison.data ? (
        <div role="status" className="space-y-2 rounded bg-muted/40 p-3">
          <p>{comparison.data.message}</p>
          {comparison.data.status === "ready" ? (
            <>
              <p>
                After: {comparison.data.afterRange.startDate} –{" "}
                {comparison.data.afterRange.endDate}
              </p>
              {comparison.data.after ? (
                <p>
                  Clicks/day{" "}
                  {comparison.data.baselineClicksPerDay?.toFixed(1) ??
                    "Unavailable"}{" "}
                  →{" "}
                  {comparison.data.afterClicksPerDay?.toFixed(1) ??
                    "Unavailable"}{" "}
                  · Session outcome rate{" "}
                  {rate(
                    action.source.goalEvent
                      ? (comparison.data.baseline.goalRate ?? null)
                      : comparison.data.baseline.keyEventRate,
                  )}{" "}
                  →{" "}
                  {rate(
                    action.source.goalEvent
                      ? (comparison.data.after.goalRate ?? null)
                      : comparison.data.after.keyEventRate,
                  )}
                </p>
              ) : (
                <p>
                  No matching post-change Search Console row was returned.
                  Results are unknown.
                </p>
              )}
              {comparison.data.controlPage ? (
                <p className="break-all">
                  Unchanged comparison page: {comparison.data.controlPage}.{" "}
                  {comparison.data.controlMessage}{" "}
                  {comparison.data.controlAdjustedClickChange != null
                    ? `Relative click movement: ${(comparison.data.controlAdjustedClickChange * 100).toFixed(1)}% (ratio of before/after changes).`
                    : ""}
                </p>
              ) : null}
              {comparison.data.limited ? (
                <p>
                  Analytics data is limited; avoid drawing business conclusions.
                </p>
              ) : null}
            </>
          ) : null}
        </div>
      ) : null}
    </article>
  );
}
