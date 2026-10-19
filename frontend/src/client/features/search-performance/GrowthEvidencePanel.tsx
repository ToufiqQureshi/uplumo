import { useMutation } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Button } from "@/client/components/ui/button";
import { QueryError } from "@/client/components/QueryState";
import { safeHttpUrl } from "@/shared/safe-url";
import {
  trackingApi,
  type TrackingAnswer,
} from "@/client/features/ai-search/trackingApi";
import {
  getGrowthDevices,
  getGrowthHealth,
  type GrowthReport,
} from "./growthApi";

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
  const devices = useMutation({
    mutationFn: () => getGrowthDevices(projectId, report.range),
    retry: false,
  });
  const health = useMutation({
    mutationFn: () => getGrowthHealth(projectId),
    retry: false,
  });
  const ai = useMutation({
    mutationFn: async () => {
      const state = await trackingApi.state(projectId);
      const run = state.runs.find((r) => r.status === "completed");
      if (!run) return { run: null, answers: [] as TrackingAnswer[] };
      return { run, answers: await trackingApi.answers(projectId, run.id) };
    },
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
                        <th className="p-2">Key events</th>
                        <th className="p-2">Session key-event rate</th>
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
                            {row.sessionKeyEventRate === null
                              ? "Unavailable"
                              : `${(row.sessionKeyEventRate * 100).toFixed(1)}%`}
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
        <section className="space-y-2">
          <Button
            variant="outline"
            size="sm"
            disabled={ai.isPending}
            onClick={() => ai.mutate()}
          >
            Load saved AI citation evidence
          </Button>
          <p className="text-xs text-muted-foreground">
            Reads existing tracked answers only. Does not buy new AI answers or
            enable a schedule.
          </p>
          {ai.isError ? (
            <QueryError
              error={ai.error}
              fallback="Could not load saved AI evidence."
            />
          ) : null}
          {ai.data ? (
            ai.data.run ? (
              <div className="space-y-3">
                <p className="text-xs text-muted-foreground">
                  Saved run: {new Date(ai.data.run.createdAt).toLocaleString()}{" "}
                  · {ai.data.run.completed} completed, {ai.data.run.failed}{" "}
                  failed. These prompt/model samples do not represent every
                  consumer AI answer and use a separate date scope from GSC/GA4.
                </p>
                {ai.data.answers
                  .filter((a) => a.status === "completed" && a.answer.trim())
                  .slice(0, 10)
                  .map((answer) => (
                    <details key={answer.id} className="rounded border p-2">
                      <summary className="cursor-pointer">
                        {answer.prompt} · {answer.engine}
                      </summary>
                      <p className="mt-2 whitespace-pre-wrap text-xs text-muted-foreground">
                        {answer.answer.slice(0, 1600)}
                        {answer.answer.length > 1600 ? "…" : ""}
                      </p>
                      <ul className="mt-2 space-y-1 text-xs">
                        {answer.citations.slice(0, 10).map((citation, i) => {
                          const url = safeHttpUrl(citation.url);
                          return (
                            <li key={`${citation.url}:${i}`}>
                              {url ? (
                                <a
                                  className="break-all underline"
                                  href={url}
                                  target="_blank"
                                  rel="noreferrer noopener"
                                >
                                  {citation.title || citation.url}
                                </a>
                              ) : (
                                "Unsafe source URL omitted"
                              )}
                              {citation.matchedBrand
                                ? " · Tracked brand source"
                                : " · Other source"}
                            </li>
                          );
                        })}
                      </ul>
                      <p className="mt-2 text-xs">
                        Compare what these cited sources answer with your
                        existing page. Absence from this sample is not proof of
                        absence from AI search.
                      </p>
                    </details>
                  ))}
              </div>
            ) : (
              <p className="text-xs">
                No completed AI tracking run is saved yet.
              </p>
            )
          ) : null}
          <Link
            to="/p/$projectId/prompt-tracking"
            params={{ projectId }}
            className="text-xs underline"
          >
            Review full prompt tracking and citation coverage
          </Link>
        </section>
      </div>
    </details>
  );
}
