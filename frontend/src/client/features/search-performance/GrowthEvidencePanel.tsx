import { useId } from "react";
import { GrowthDeviceEvidence } from "./GrowthDeviceEvidence";
import { GrowthInvestigation } from "./GrowthInvestigation";
import { useMutation } from "@tanstack/react-query";
import { Button } from "@/client/components/ui/button";
import { QueryError } from "@/client/components/QueryState";

import {
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
  const group = useId();
  const tracking = useMutation({
    mutationFn: () =>
      getGrowthTracking(projectId, {
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
    <details className="min-w-0 rounded-xl border border-border bg-muted/20 p-3 text-sm sm:p-4">
      <summary className="cursor-pointer font-medium">
        Investigate tracking, devices and AI citations
      </summary>
      <div className="mt-4 min-w-0 space-y-3">
        <GrowthInvestigation
          name={group}
          title="Measurement setup"
          description="Check that your chosen outcome is configured."
        >
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
                Configured key events: {health.data.keyEvents.length}.
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
              <details>
                <summary className="cursor-pointer font-medium">
                  Verify event accuracy
                </summary>{" "}
                <p>
                  Validate your chosen successful outcome in GA4 DebugView and
                  confirm it fires only after success. Form clicks and scrolls
                  are not automatically leads.
                </p>
              </details>
            </div>
          ) : null}
        </GrowthInvestigation>
        <GrowthInvestigation
          name={group}
          title="Attribution and tracking"
          description="Investigate referral sources and tracking gaps."
        >
          <Button
            size="sm"
            variant="outline"
            disabled={tracking.isPending}
            onClick={() => tracking.mutate()}
          >
            Check attribution
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
        </GrowthInvestigation>
        <GrowthInvestigation
          name={group}
          title="Visitor journey"
          description="See which tracked steps users complete."
        >
          <GrowthFunnel projectId={projectId} report={report} />
        </GrowthInvestigation>
        <GrowthInvestigation
          name={group}
          title="Sitemap health"
          description="Review Google’s submitted sitemap status."
        >
          <GrowthSitemaps projectId={projectId} />
        </GrowthInvestigation>
        <GrowthInvestigation
          name={group}
          title="Performance by device"
          description="Compare the selected outcome on mobile and desktop."
        >
          <GrowthDeviceEvidence projectId={projectId} report={report} />
        </GrowthInvestigation>
        <GrowthInvestigation
          name={group}
          title="AI visibility evidence"
          description="Review saved citations and Google’s native reports."
        >
          <GrowthCitationEvidence projectId={projectId} report={report} />
        </GrowthInvestigation>
      </div>
    </details>
  );
}
