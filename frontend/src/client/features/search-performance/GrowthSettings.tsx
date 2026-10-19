import { useMutation } from "@tanstack/react-query";
import { Button } from "@/client/components/ui/button";
import { Input } from "@/client/components/ui/input";
import { QueryError } from "@/client/components/QueryState";
import { getGrowthGoals } from "./growthApi";
export type GrowthSettingsValue = {
  brands: string;
  goalEvent: string;
  days: number;
  comparison: string;
  expanded: boolean;
};
export function GrowthSettings({
  projectId,
  settings,
  onChange,
}: {
  projectId: string;
  settings: GrowthSettingsValue;
  onChange: (value: GrowthSettingsValue) => void;
}) {
  const { brands, goalEvent, days, comparison, expanded } = settings;
  const goals = useMutation({
    mutationFn: () => getGrowthGoals(projectId),
    retry: false,
  });
  return (
    <>
      <div className="mt-4 flex flex-wrap items-end gap-3">
        <label className="grid gap-1 text-sm">
          What matters to your business?
          <select
            className="min-w-52 rounded-md border bg-background p-2"
            value={goalEvent}
            onChange={(e) =>
              onChange({ ...settings, goalEvent: e.target.value })
            }
          >
            <option value="">Explore all key events</option>
            {goals.data?.map((event) => (
              <option key={event} value={event}>
                {event === "purchase"
                  ? "Completed purchases"
                  : event === "generate_lead"
                    ? "Successful leads"
                    : event}
              </option>
            ))}
          </select>
        </label>
        <Button
          size="sm"
          variant="outline"
          disabled={goals.isPending}
          onClick={() => goals.mutate()}
        >
          {goals.isPending ? "Loading goals…" : "Load my GA4 goals"}
        </Button>
      </div>
      {goals.isError ? (
        <QueryError
          error={goals.error}
          fallback="Could not load your registered goals."
        />
      ) : null}
      {goals.isSuccess && goals.data.length === 0 ? (
        <p className="mt-2 text-xs text-muted-foreground">
          No goal-specific metrics found. Register your successful outcome as a
          GA4 key event, then reload goals.
        </p>
      ) : null}
      <p className="mt-2 text-xs text-muted-foreground">
        Choose an event that fires after a real success. Configuration alone
        does not verify a lead or sale. Run analysis after changing settings.
      </p>
      <details className="mt-3 text-sm">
        <summary className="cursor-pointer text-muted-foreground">
          Analysis settings
        </summary>
        <div className="mt-3 flex flex-wrap items-end gap-3">
          <label className="grid gap-1">
            Brand names, separated by commas
            <Input
              value={brands}
              maxLength={1000}
              placeholder="Your brand, alternate spelling"
              onChange={(e) =>
                onChange({ ...settings, brands: e.target.value })
              }
            />
          </label>
          <label className="grid gap-1">
            Comparison window
            <select
              className="rounded-md border bg-background p-2"
              value={days}
              onChange={(e) =>
                onChange({ ...settings, days: Number(e.target.value) })
              }
            >
              <option value={14}>14 days</option>
              <option value={28}>28 days</option>
            </select>
          </label>
          <label className="grid gap-1">
            Compare with
            <select
              className="rounded-md border bg-background p-2"
              value={comparison}
              onChange={(e) =>
                onChange({ ...settings, comparison: e.target.value })
              }
            >
              <option value="previous">Previous period (same weekdays)</option>
              <option value="year">
                Same weekdays last year (364 days earlier)
              </option>
            </select>
          </label>
          <label className="flex items-center gap-2">
            <input
              type="checkbox"
              checked={expanded}
              onChange={(e) =>
                onChange({ ...settings, expanded: e.target.checked })
              }
            />
            Expanded coverage (up to 5,000 rows/source)
          </label>
          <p className="text-xs text-muted-foreground">
            Google organic traffic only. Independent of the report filters
            below. No AI credits used.
          </p>
        </div>
      </details>
    </>
  );
}
