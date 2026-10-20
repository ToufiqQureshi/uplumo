import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@/client/components/ui/button";
import { QueryError } from "@/client/components/QueryState";
import { getGrowthMonitor, saveGrowthMonitor } from "./growthApi";

export function GrowthMonitor({
  projectId,
  goalEvent,
}: {
  projectId: string;
  goalEvent: string;
}) {
  const [opened, setOpened] = useState(false);
  const client = useQueryClient();
  const key = ["growth-monitor", projectId];
  const monitor = useQuery({
    queryKey: key,
    queryFn: () => getGrowthMonitor(projectId),
    enabled: opened,
    retry: false,
    refetchOnWindowFocus: false,
  });
  const save = useMutation({
    mutationFn: (enabled: boolean) =>
      saveGrowthMonitor(projectId, { enabled, goalEvent }),
    retry: false,
    onSuccess: (state) => client.setQueryData(key, state),
  });
  const state = monitor.data;
  return (
    <details
      className="rounded-lg border border-border p-3"
      onToggle={(e) => setOpened(e.currentTarget.open)}
    >
      <summary className="cursor-pointer text-sm font-medium">
        Weekly actions and important drop alerts
      </summary>
      <div className="mt-3 space-y-3 text-sm">
        <p className="text-muted-foreground">
          Get three evidence-based next actions and an in-app alert when search
          clicks and your selected outcome rate both fall with enough data.
          Checks use Google data only; no AI credits or email delivery.
        </p>
        {monitor.isError ? (
          <QueryError
            error={monitor.error}
            fallback="Could not load monitoring."
          />
        ) : monitor.isPending ? (
          <p>Loading weekly checks…</p>
        ) : null}
        {state ? (
          <>
            <p>
              {state.enabled
                ? `Weekly checks enabled for ${state.goalEvent}.`
                : "Weekly checks are off."}{" "}
              {state.nextCheckAt && state.enabled
                ? `Next check due ${new Date(state.nextCheckAt).toLocaleString()}.`
                : ""}
            </p>
            <Button
              size="sm"
              variant="outline"
              disabled={save.isPending || (!state.enabled && !goalEvent)}
              onClick={() => save.mutate(!state.enabled)}
            >
              {save.isPending
                ? "Saving…"
                : state.enabled
                  ? "Turn off weekly checks"
                  : "Enable weekly checks for selected goal"}
            </Button>
            {state.enabled && goalEvent && goalEvent !== state.goalEvent ? (
              <Button
                size="sm"
                variant="ghost"
                disabled={save.isPending}
                onClick={() => save.mutate(true)}
              >
                Use {goalEvent} for future checks
              </Button>
            ) : null}
            {!state.enabled && !goalEvent ? (
              <p className="text-xs text-muted-foreground">
                Load and select your successful GA4 outcome above before
                enabling.
              </p>
            ) : null}
            {state.lastError ? <p role="status">{state.lastError}</p> : null}
            {state.digest ? (
              <div className="space-y-3">
                <p className="text-xs text-muted-foreground">
                  Last successful check:{" "}
                  {new Date(state.digest.generatedAt).toLocaleString()} ·{" "}
                  {state.digest.range.startDate} – {state.digest.range.endDate}{" "}
                  · Goal: {state.digest.goalEvent}.
                </p>
                {state.digest.limited || state.digest.truncated ? (
                  <p className="text-xs text-muted-foreground">
                    Coverage is limited. These actions are not an exhaustive
                    review.
                  </p>
                ) : null}
                {state.digest.alerts.map((alert, i) => (
                  <div
                    key={`${alert.kind}:${i}`}
                    className="rounded border border-border p-3"
                    role="status"
                  >
                    <p className="font-medium">{alert.title}</p>
                    <p className="break-words text-xs">{alert.evidence}</p>
                    <p className="mt-1">{alert.action}</p>
                  </div>
                ))}
                {state.digest.pages.length ? (
                  state.digest.pages.map((page) => (
                    <article
                      key={page.page}
                      className="rounded border border-border p-3"
                    >
                      <p className="break-all font-medium">{page.page}</p>
                      <p className="mt-1">{page.findings[0]?.title}</p>
                      <p className="text-xs text-muted-foreground">
                        {page.findings[0]?.evidence}
                      </p>
                      <p className="mt-1">{page.findings[0]?.action}</p>
                    </article>
                  ))
                ) : (
                  <p>
                    No findings met the evidence thresholds in the last check.
                    This does not establish that the site has no issues.
                  </p>
                )}
                <p className="text-xs text-muted-foreground">
                  Run a fresh analysis above to investigate or save an action.
                  Weekly evidence compares two complete 28-day periods; it does
                  not establish the cause of a change.
                </p>
              </div>
            ) : state.enabled ? (
              <p className="text-xs text-muted-foreground">
                Your first check is queued. Open this panel again after it
                completes.
              </p>
            ) : null}
          </>
        ) : null}
        {save.isError ? (
          <QueryError
            error={save.error}
            fallback="Could not update weekly checks."
          />
        ) : null}
      </div>
    </details>
  );
}
