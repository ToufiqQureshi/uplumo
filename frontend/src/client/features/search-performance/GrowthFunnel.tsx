import { X } from "lucide-react";
import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Button } from "@/client/components/ui/button";
import { Input } from "@/client/components/ui/input";
import { QueryError } from "@/client/components/QueryState";
import { getGrowthFunnel, type GrowthReport } from "./growthApi";
import { GrowthMobileResults } from "./GrowthMobileResults";
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
      ? ["page_view", "begin_checkout", "purchase"]
      : ["page_view", "form_start", report.goalEvent || "generate_lead"],
  );
  const [byDevice, setByDevice] = useState(false);
  const funnel = useMutation({
    mutationFn: () =>
      getGrowthFunnel(projectId, {
        ...report.range,
        steps: steps.map((step) => step.trim()).filter(Boolean),
        byDevice,
      }),
    retry: false,
  });
  return (
    <section className="min-w-0 space-y-4">
      <h4 className="font-medium">Where do visitors leave the journey?</h4>
      <p className="text-sm text-muted-foreground">
        Choose 2–5 events in the order visitors should complete them.
      </p>
      <div className="space-y-3">
        {steps.map((step, index) => (
          <div key={index} className="flex min-w-0 items-end gap-2">
            <label className="grid min-w-0 flex-1 gap-1.5 text-xs font-medium">
              Step {index + 1} event
              <Input
                className="w-full min-w-0 text-sm"
                value={step}
                maxLength={40}
                placeholder="GA4 event name"
                onChange={(e) =>
                  setSteps((current) =>
                    current.map((value, i) =>
                      i === index ? e.target.value : value,
                    ),
                  )
                }
              />
            </label>
            {steps.length > 2 ? (
              <Button
                size="icon-sm"
                variant="ghost"
                className="size-9 shrink-0"
                aria-label={`Remove step ${index + 1}`}
                onClick={() =>
                  setSteps((current) => current.filter((_, i) => i !== index))
                }
              >
                <X className="size-4" aria-hidden />
              </Button>
            ) : null}
          </div>
        ))}
      </div>
      {steps.length < 5 ? (
        <Button
          size="sm"
          variant="ghost"
          onClick={() => setSteps((current) => [...current, ""])}
        >
          Add a step
        </Button>
      ) : null}
      <p className="text-xs leading-relaxed text-muted-foreground">
        Use events your site actually tracks. Missing events can look like
        drop-offs.
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
          <details>
            <summary className="cursor-pointer font-medium">
              How to read this journey
            </summary>
            <p className="mt-2 leading-relaxed">{funnel.data.message}</p>
          </details>
          {funnel.data.limited ? (
            <p>
              Funnel data is limited. Avoid drawing business conclusions from
              these rates.
            </p>
          ) : null}
          {funnel.data.rows.length ? (
            <>
              <GrowthMobileResults
                label="Journey step results"
                rows={funnel.data.rows.map((row, index) => ({
                  key: `${row.funnelStepName}:${row.deviceCategory}:${index}`,
                  title: row.funnelStepName,
                  subtitle: row.deviceCategory
                    ? `Device: ${row.deviceCategory}`
                    : undefined,
                  metrics: [
                    { label: "Users", value: growthCount(row.activeUsers) },
                    {
                      label: "Completed next step",
                      value: growthRate(row.funnelStepCompletionRate),
                    },
                    {
                      label: "Left before next step",
                      value: growthCount(row.funnelStepAbandonments),
                    },
                    {
                      label: "Drop-off rate",
                      value: growthRate(row.funnelStepAbandonmentRate),
                    },
                  ],
                }))}
              />
              <div className="hidden overflow-x-auto sm:block">
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
                      <th className="p-2">Left before next step</th>
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
            </>
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
