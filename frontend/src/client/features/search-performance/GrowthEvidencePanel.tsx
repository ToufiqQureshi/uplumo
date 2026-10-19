import { useMutation } from "@tanstack/react-query";
import { Button } from "@/client/components/ui/button";
import { QueryError } from "@/client/components/QueryState";

import {
  getGrowthDevices,
  getGrowthHealth,
  getGrowthTracking,
  type GrowthReport,
} from "./growthApi";

import { GrowthCitationEvidence } from "./GrowthCitationEvidence";
import { GrowthSitemaps } from "./GrowthSitemaps";
import { GrowthFunnel } from "./GrowthFunnel";

const healthMessages: Record<string, string> = {
  no_web_stream:
    "No web stream is configured. Check that the selected property measures this website.",
  enhanced_measurement_disabled:
    "Enhanced measurement is disabled. Review which interactions you need to measure.",
  site_search_measurement_disabled:
    "Site-search measurement is disabled. Enable it if your site has internal search.",
  no_key_events_configured:
    "No key events are configured. Set up a successful lead or purchase event before evaluating business outcomes.",
};

export function GrowthEvidencePanel({
  projectId,
  report,
}: {
  projectId: string;
  report: GrowthReport;
}) {
  const tracking = useMutation({
    mutationFn: () =>
      getGrowthTracking(projectId, {
        ...report.range,
        goalEvent: report.goalEvent,
      }),
    retry: false,
  });
  const devices = useMutation({
    mutationFn: () =>
      getGrowthDevices(projectId, {
        ...report.range,
        goalEvent: report.goalEvent,
      }),
    retry: false,
  });
  const health = useMutation({
    mutationFn: () => getGrowthHealth(projectId),
    retry: false,
  });

  return (
    <details className="rounded-lg border p-3 text-sm">
      <summary className="cursor-pointer font-medium">
        Investigate tracking, devices and AI citations
      </summary>
      <div className="mt-3 space-y-4">
        <section className="space-y-2">
          <Button
            variant="outline"
            size="sm"
            disabled={health.isPending}
            onClick={() => health.mutate()}
          >
            Check GA4 measurement setup
          </Button>
          {health.isError ? (
            <QueryError
              error={health.error}
              fallback="Could not inspect measurement setup."
            />
          ) : null}
          {health.data ? (
            <div className="space-y-2 text-xs">
              <p>
                {health.data.keyEvents.length} configured key events.
                Configuration does not prove that events fire correctly or
                represent real sales.
              </p>
              {health.data.keyEvents.length ? (
                <p>
                  Events:{" "}
                  {health.data.keyEvents.map((e) => e.eventName).join(", ")}
                </p>
              ) : null}
              {health.data.issues.map((issue) => (
                <p key={issue}>
                  {healthMessages[issue] ?? issue.replaceAll("_", " ")}
                </p>
              ))}
              <p>
                Validate your chosen successful outcome in GA4 DebugView and
                confirm it fires only after success. Form clicks and scrolls are
                not automatically leads.
              </p>
            </div>
          ) : null}
        </section>
        <section className="space-y-2">
          <Button
            size="sm"
            variant="outline"
            disabled={tracking.isPending}
            onClick={() => tracking.mutate()}
          >
            Investigate attribution and tracking
          </Button>
          {tracking.isError ? (
            <QueryError
              error={tracking.error}
              fallback="Could not inspect tracking evidence."
            />
          ) : null}
          {tracking.data ? (
            <div className="space-y-2 text-xs">
              {tracking.data.limited ? (
                <p>
                  Returned tracking evidence is limited; missing referral rows
                  are unknown.
                </p>
              ) : null}
              {tracking.data.findings.map((finding, i) => (
                <p key={`${finding.kind}:${i}`}>
                  <strong>{finding.title}:</strong> {finding.evidence}{" "}
                  {finding.action}
                </p>
              ))}
              <ul className="list-disc space-y-1 pl-4">
                {tracking.data.checklist.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
              <details>
                <summary className="cursor-pointer">
                  Observed host and referral rows
                </summary>
                <ul className="mt-2 space-y-1">
                  {tracking.data.rows.map((row, i) => (
                    <li
                      key={`${row.hostName}:${row.sessionSource}:${row.sessionMedium}:${i}`}
                    >
                      {row.hostName} · {row.sessionSource} / {row.sessionMedium}{" "}
                      · {row.sessions ?? "Unavailable"} sessions ·{" "}
                      {row.keyEvents ?? "Unavailable"} all key events
                    </li>
                  ))}
                </ul>
              </details>
            </div>
          ) : null}
        </section>
        <GrowthFunnel projectId={projectId} report={report} />
        <GrowthSitemaps projectId={projectId} />
        <section className="space-y-2">
          <Button
            variant="outline"
            size="sm"
            disabled={devices.isPending}
            onClick={() => devices.mutate()}
          >
            Compare Google organic traffic by device
          </Button>
          {devices.isError ? (
            <QueryError
              error={devices.error}
              fallback="Could not inspect device performance."
            />
          ) : null}
          {devices.data ? (
            <div className="space-y-2 text-xs">
              {devices.data.limited ? (
                <p>
                  Analytics data is limited. Avoid drawing business conclusions
                  from these rates.
                </p>
              ) : null}
              {devices.data.rows.length === 0 ? (
                <p>No device rows returned.</p>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-left">
                    <caption className="sr-only">
                      Google organic traffic by device
                    </caption>
                    <thead>
                      <tr>
                        <th className="p-2">Device</th>
                        <th className="p-2">Sessions</th>
                        <th className="p-2">All key events</th>
                        <th className="p-2">
                          {report.goalEvent
                            ? `${report.goalEvent} session rate`
                            : "Session key-event rate"}
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {devices.data.rows.map((row) => (
                        <tr key={row.deviceCategory}>
                          <td className="p-2">{row.deviceCategory}</td>
                          <td className="p-2">
                            {row.sessions ?? "Unavailable"}
                          </td>
                          <td className="p-2">
                            {row.keyEvents ?? "Unavailable"}
                          </td>
                          <td className="p-2">
                            {(report.goalEvent
                              ? row.goalRate
                              : row.sessionKeyEventRate) == null
                              ? "Unavailable"
                              : `${((report.goalEvent ? (row.goalRate ?? 0) : (row.sessionKeyEventRate ?? 0)) * 100).toFixed(1)}%`}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
              <p>
                Site-wide comparison for {report.range.startDate} –{" "}
                {report.range.endDate}. Small samples and different
                audience/page mixes can explain differences. This does not
                measure speed or establish a cause.
              </p>
            </div>
          ) : null}
        </section>
        <GrowthCitationEvidence projectId={projectId} report={report} />
      </div>
    </details>
  );
}
