import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Button } from "@/client/components/ui/button";
import { Input } from "@/client/components/ui/input";
import { QueryError } from "@/client/components/QueryState";
import { getGrowthFunnel, type GrowthReport } from "./growthApi";
import { growthRate, growthCount } from "./growthFormat";

export function GrowthFunnel({
  projectId,
  report,
}: {
  projectId: string;
  report: GrowthReport;
}) {
  const [steps, setSteps] = useState(
    report.goalEvent === "purchase"
      ? "page_view, begin_checkout, purchase"
      : report.goalEvent
        ? `page_view, form_start, ${report.goalEvent}`
        : "page_view, form_start, generate_lead",
  );
  const [byDevice, setByDevice] = useState(false);
  const funnel = useMutation({
    mutationFn: () =>
      getGrowthFunnel(projectId, {
        ...report.range,
        steps: steps
          .split(",")
          .map((x) => x.trim())
          .filter(Boolean),
        byDevice,
      }),
    retry: false,
  });
  return (
    <section className="space-y-2">
      <h4 className="font-medium">Where do visitors leave the journey?</h4>
      <label className="grid gap-1 text-xs">
        Tracked event names in order, separated by commas (2–5)
        <Input
          value={steps}
          maxLength={204}
          onChange={(e) => setSteps(e.target.value)}
        />
      </label>
      <p className="text-xs text-muted-foreground">
        These are editable examples. Use events your site actually sends;
        form_start does not exist on every site. Missing steps do not prove
        users abandoned.
      </p>
      <label className="flex items-center gap-2 text-xs">
        <input
          type="checkbox"
          checked={byDevice}
          onChange={(e) => setByDevice(e.target.checked)}
        />
        Break down by device
      </label>
      <Button
        size="sm"
        variant="outline"
        disabled={funnel.isPending}
        onClick={() => funnel.mutate()}
      >
        {funnel.isPending ? "Reading journey…" : "Inspect tracked journey"}
      </Button>
      {funnel.isError ? (
        <QueryError
          error={funnel.error}
          fallback="Could not load this funnel. Verify events and property support."
        />
      ) : null}
      {funnel.data ? (
        <div className="space-y-2 text-xs">
          <p>{funnel.data.message}</p>
          {funnel.data.limited ? (
            <p>
              Funnel data is limited. Avoid drawing business conclusions from
              these rates.
            </p>
          ) : null}
          {funnel.data.rows.length ? (
            <div className="overflow-x-auto">
              <table className="w-full text-left">
                <caption className="sr-only">
                  Ordered Google organic user funnel
                </caption>
                <thead>
                  <tr>
                    <th className="p-2">Step</th>
                    <th className="p-2">Device</th>
                    <th className="p-2">Users</th>
                    <th className="p-2">Complete next step</th>
                    <th className="p-2">Abandonments</th>
                  </tr>
                </thead>
                <tbody>
                  {funnel.data.rows.map((row, i) => (
                    <tr
                      key={`${row.funnelStepName}:${row.deviceCategory}:${i}`}
                    >
                      <td className="p-2">{row.funnelStepName}</td>
                      <td className="p-2">{row.deviceCategory ?? "All"}</td>
                      <td className="p-2">{growthCount(row.activeUsers)}</td>
                      <td className="p-2">
                        {growthRate(row.funnelStepCompletionRate)}
                      </td>
                      <td className="p-2">
                        {growthCount(row.funnelStepAbandonments)} (
                        {growthRate(row.funnelStepAbandonmentRate)})
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <p>
              No funnel rows returned. Check event instrumentation and allow
              time for processing.
            </p>
          )}
        </div>
      ) : null}
    </section>
  );
}
