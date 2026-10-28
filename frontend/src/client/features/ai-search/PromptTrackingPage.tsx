import { EngineLogo } from "@/client/components/BrandLogo";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@/client/components/ui/button";
import { PageHeader } from "@/client/components/PageHeader";
import { TrackingForm } from "./TrackingForm";
import { AiVisibilityGuide } from "./AiVisibilityGuide";
import { MarkdownAnswer } from "./components/MarkdownAnswer";
import { WhyNotCitedCard } from "./components/WhyNotCitedCard";

const ENGINE_LABELS: Record<string, string> = {
  chat_gpt: "ChatGPT",
  gemini: "Gemini",
  google_ai_overview: "Google AI Overviews",
  claude: "Claude",
  perplexity: "Perplexity",
};
const engineLabel = (engine: string) =>
  ENGINE_LABELS[engine] ?? engine.replaceAll("_", " ");
import { trackingApi, trackingKey, summarizeTracking } from "./trackingApi";

const pct = (n: number, d: number) =>
  d ? `${Math.round((100 * n) / d)}%` : "Not measured";
export function PromptTrackingPage({ projectId }: { projectId: string }) {
  const [selected, setSelected] = useState<string | null>(null);
  const state = useQuery({
    queryKey: trackingKey(projectId),
    queryFn: () => trackingApi.state(projectId),
    staleTime: 30000,
    refetchInterval: (q) =>
      q.state.data?.runs.some(
        (r) => r.status === "queued" || r.status === "running",
      )
        ? 15000
        : false,
  });
  const runId = selected ?? state.data?.runs[0]?.id;
  const answers = useQuery({
    queryKey: [...trackingKey(projectId), "answers", runId],
    queryFn: () => trackingApi.answers(projectId, runId!),
    enabled: Boolean(runId),
    staleTime: 30000,
    refetchInterval: state.data?.runs.some(
      (r) =>
        r.id === runId && (r.status === "queued" || r.status === "running"),
    )
      ? 15000
      : false,
  });
  const summary = summarizeTracking(answers.data ?? []);
  const selectedRun = state.data?.runs.find((r) => r.id === runId);
  const finished =
    state.data?.runs.filter((r) => r.status === "completed") ?? [];
  const [latest, previous] = finished;
  const comparable =
    latest &&
    previous &&
    latest.scopeKey === previous.scopeKey &&
    latest.answered >= 10 &&
    previous.answered >= 10;
  const drop = comparable
    ? (previous.mentioned / previous.answered -
        latest.mentioned / latest.answered) *
      100
    : 0;
  function download() {
    const blob = new Blob(
      [
        JSON.stringify(
          {
            run: state.data?.runs.find((r) => r.id === runId),
            answers: answers.data,
          },
          null,
          2,
        ),
      ],
      { type: "application/json" },
    );
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "ai-visibility-results.json";
    link.click();
    URL.revokeObjectURL(url);
  }
  if (state.isPending)
    return (
      <p role="status" className="p-6">
        Loading tracking…
      </p>
    );
  if (!state.data)
    return (
      <div className="p-6">
        <p role="alert">{state.error?.message ?? "Could not load tracking."}</p>
        <Button onClick={() => void state.refetch()}>Retry</Button>
      </div>
    );
  return (
    <div className="mx-auto max-w-7xl space-y-5 p-4 md:p-6">
      <PageHeader title="AI Visibility Tracker" />
      <AiVisibilityGuide current="prompt-tracking" projectId={projectId} />
      <p className="text-sm text-muted-foreground">
        Save the questions your customers ask. Track fresh brand mentions,
        competitor appearances and website citations over time.
      </p>
      {drop >= 20 ? (
        <p
          role="status"
          className="rounded-lg border border-amber-500 p-3 text-sm"
        >
          Your mention rate fell {Math.round(drop)} percentage points across the
          last two complete checks with the same questions, engines and market.
          Review the captured answers; this sample does not measure all AI
          conversations.
        </p>
      ) : null}
      <div id="tracker-setup" />
      <TrackingForm
        key={JSON.stringify(state.data.configuration)}
        projectId={projectId}
        saved={state.data.configuration}
        providerConfigured={state.data.providerConfigured}
      />
      {state.data.nextCheckAt ? (
        <p className="text-sm">
          Next automatic check:{" "}
          {new Date(state.data.nextCheckAt).toLocaleString()}
        </p>
      ) : null}
      <section className="space-y-3 rounded-xl border p-5">
        <h2 className="text-lg font-semibold">Check history</h2>
        <p className="text-xs text-muted-foreground">
          Compare checks with the same questions, engines and market. Failed
          checks and empty answers do not count as a measured absence.
          Collection errors can reduce coverage.
        </p>
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr>
                <th>Date</th>
                <th>Status</th>
                <th>Answered</th>
                <th>Brand mentioned</th>
                <th>Website cited</th>
                <th>Details</th>
              </tr>
            </thead>
            <tbody>
              {state.data.runs.map((r) => (
                <tr key={r.id} className="border-t">
                  <td className="py-3">
                    {new Date(r.createdAt).toLocaleString()}
                  </td>
                  <td>
                    {r.status}
                    {r.failed ? ` · ${r.failed} failed` : ""}
                    {r.errorCode
                      ? ` · ${r.errorCode.replaceAll("_", " ")}`
                      : ""}
                  </td>
                  <td>{r.answered}</td>
                  <td>{pct(r.mentioned, r.answered)}</td>
                  <td>{pct(r.cited, r.answered)}</td>
                  <td>
                    <Button variant="ghost" onClick={() => setSelected(r.id)}>
                      View answers
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!state.data.runs.length ? (
          <p className="text-sm text-muted-foreground">
            No checks yet. Save questions, review the cost, then run your first
            check.
          </p>
        ) : null}
      </section>
      {answers.data && selectedRun ? (
        <WhyNotCitedCard
          projectId={projectId}
          answers={answers.data}
          ownDomain={selectedRun.configuration.domain}
        />
      ) : null}
      {runId ? (
        <section className="space-y-4 rounded-xl border p-5">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold">
              Captured answers & citations
            </h2>
            <Button
              variant="outline"
              disabled={!answers.data}
              onClick={download}
            >
              Export saved results
            </Button>
          </div>
          {answers.error ? <p role="alert">{answers.error.message}</p> : null}
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr>
                  <th>Brand</th>
                  <th>Answers</th>
                  <th>Mention rate</th>
                  <th>Citation rate</th>
                </tr>
              </thead>
              <tbody>
                {summary.map((b) => (
                  <tr key={b.domain} className="border-t">
                    <td className="py-2">{b.name}</td>
                    <td>{b.answers}</td>
                    <td>{pct(b.mentions, b.answers)}</td>
                    <td>{pct(b.citations, b.answers)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {answers.data?.map((a) => (
            <details key={a.id} className="rounded-lg border p-3">
              <summary className="cursor-pointer text-sm font-medium">
                <EngineLogo engine={a.engine} /> {a.topic} · {a.prompt} ·{" "}
                {engineLabel(a.engine)} · {a.status}
                {a.errorCode ? ` (${a.errorCode.replaceAll("_", " ")})` : ""}
              </summary>
              {a.recommended.length > 0 ? (
                <div className="mt-3 text-sm">
                  <p className="font-medium">Businesses the AI recommended</p>
                  <ol className="mt-1 list-decimal pl-5">
                    {a.recommended.map((r) => (
                      <li key={r.position}>
                        {r.name}
                        {a.brands.some(
                          (b) => b.name.toLowerCase() === r.name.toLowerCase(),
                        )
                          ? " (tracked)"
                          : ""}
                      </li>
                    ))}
                  </ol>
                </div>
              ) : null}
              <div className="mt-3 text-sm">
                {a.answer ? (
                  <MarkdownAnswer text={a.answer} />
                ) : a.status === "completed" ? (
                  "No substantive answer / AI Overview was returned."
                ) : (
                  "No answer captured yet."
                )}
              </div>
              {a.citations.length > 0 ? (
                <p className="mt-3 text-sm font-medium">Sources cited</p>
              ) : null}
              <ul className="mt-1 space-y-1 text-sm">
                {a.citations.map((c) => (
                  <li key={c.url}>
                    <a
                      className="text-primary underline"
                      href={c.url}
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      {c.title ?? c.url}
                    </a>
                    {c.source || c.domain ? (
                      <span className="text-muted-foreground">
                        {" "}
                        · {c.source ?? c.domain}
                        {c.publishedAt
                          ? ` · ${c.publishedAt.slice(0, 10)}`
                          : ""}
                      </span>
                    ) : null}
                    {c.text ? (
                      <p className="mt-1 text-xs text-muted-foreground">
                        {c.text}
                      </p>
                    ) : null}
                  </li>
                ))}
              </ul>
            </details>
          ))}
        </section>
      ) : null}
    </div>
  );
}
