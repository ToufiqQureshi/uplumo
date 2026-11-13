import { Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { ArrowRight } from "lucide-react";
import { VisibilityAlerts } from "./VisibilityAlerts";
import { Button } from "@/client/components/ui/button";
import { useLatestAnswers } from "@/client/features/insights/useLatestAnswers";
import { trackingApi, trackingKey } from "./trackingApi";
import {
  citedSources,
  engineBreakdown,
  shareOfVoice,
  visibilityVerdict,
} from "./aiVisibilityMetrics";
import { EngineName } from "./EngineName";

const pct = (n: number, d: number) => (d ? Math.round((100 * n) / d) : 0);

/**
 * The first thing a customer sees: one sentence on how AI treats the brand,
 * the numbers behind it, and a link to the proof. Reads saved answers only.
 */
export function AiOverviewPage({ projectId }: { projectId: string }) {
  const ai = useLatestAnswers(projectId);
  const state = useQuery({
    queryKey: trackingKey(projectId),
    queryFn: () => trackingApi.state(projectId),
    staleTime: 60_000,
  });

  if (ai.isPending) {
    return (
      <p role="status" className="text-sm text-muted-foreground">
        Reading saved answers…
      </p>
    );
  }
  if (ai.isError) {
    return (
      <p role="alert" className="text-sm text-destructive">
        Could not load AI visibility. Refresh the page to try again.
      </p>
    );
  }
  if (!ai.run || !ai.configuration) {
    return <FirstRun projectId={projectId} configured={!!ai.configuration} />;
  }

  const verdict = visibilityVerdict(ai.answers);
  const shares = shareOfVoice(ai.answers);
  const engines = engineBreakdown(ai.answers);
  const sources = citedSources(ai.answers, ai.configuration.domain).slice(0, 6);
  const answered = engines.reduce((s, e) => s + e.answered, 0);
  const mentioned = engines.reduce((s, e) => s + e.mentioned, 0);
  const cited = engines.reduce((s, e) => s + e.cited, 0);
  const latest = ai.run;
  const previous = state.data?.runs.find(
    (r) =>
      r.id !== latest.id &&
      r.scopeKey === latest.scopeKey &&
      (r.status === "completed" || r.status === "partial") &&
      r.createdAt < latest.createdAt &&
      r.answered > 0,
  );
  const change = previous
    ? pct(latest.mentioned, latest.answered) -
      pct(previous.mentioned, previous.answered)
    : null;
  const ownShare = shares.find((s) => s.own);
  const rank = ownShare ? shares.indexOf(ownShare) + 1 : null;

  return (
    <div className="flex flex-col gap-5">
      <section
        aria-label="Verdict"
        className="flex flex-col gap-3 rounded-xl bg-foreground p-5 text-background sm:p-6"
      >
        <span className="font-mono text-xs uppercase tracking-wider opacity-70">
          Latest check · {new Date(latest.createdAt).toLocaleDateString()} ·{" "}
          {answered} answers read
        </span>
        <p className="text-xl font-semibold leading-snug sm:text-2xl">
          {verdict?.headline ?? "No answers could be read in the latest check."}
          {verdict?.weakest
            ? ` The weakest engine is ${verdict.weakest.label}: named in ${verdict.weakest.mentioned} of ${verdict.weakest.answered} answers.`
            : ""}
        </p>
        <div className="flex flex-wrap gap-2">
          {verdict && verdict.gaps > 0 ? (
            <Button
              variant="secondary"
              nativeButton={false}
              render={
                <Link
                  to="/p/$projectId/ai-visibility/prompts"
                  params={{ projectId }}
                  search={{ show: "gaps" }}
                />
              }
            >
              See the {verdict.gaps} question{verdict.gaps === 1 ? "" : "s"}{" "}
              where AI leaves you out
              <ArrowRight data-icon="inline-end" />
            </Button>
          ) : null}
          <Button
            variant="outline"
            className="border-background/30 bg-transparent text-background hover:bg-background/10 hover:text-background"
            nativeButton={false}
            render={
              <Link
                to="/p/$projectId/ai-visibility/prompts"
                params={{ projectId }}
              />
            }
          >
            Read the answers
          </Button>
        </div>
      </section>

      {previous ? (
        <VisibilityAlerts
          projectId={projectId}
          previousRunId={previous.id}
          previousDate={previous.createdAt}
          latest={ai.answers}
        />
      ) : null}

      <dl className="grid gap-3 sm:grid-cols-3">
        <Stat
          label="Answers that name you"
          value={`${mentioned} of ${answered}`}
          note={
            change === null
              ? "First comparable check"
              : change === 0
                ? "Same as the previous check"
                : `${change > 0 ? "▲" : "▼"} ${Math.abs(change)} points vs previous check`
          }
          tone={
            change === null || change === 0
              ? "plain"
              : change > 0
                ? "up"
                : "down"
          }
        />
        <Stat
          label="Share of voice"
          // With no competitor tracked, "100%" says nothing.
          value={ownShare && shares.length > 1 ? `${ownShare.share}%` : "—"}
          note={
            rank && shares.length > 1
              ? `#${rank} of ${shares.length} tracked brands`
              : "Add competitors in Setup to compare"
          }
        />
        <Stat
          label="Answers that link your site"
          value={`${cited} of ${answered}`}
          note={`${pct(cited, answered)}% cite ${ai.configuration.domain}`}
        />
      </dl>

      <div className="grid gap-4 lg:grid-cols-2">
        {shares.length > 1 ? (
          <Panel
            title="Share of voice"
            hint="How often AI names each tracked brand in the same answers."
          >
            <ul className="flex flex-col gap-3">
              {shares.map((s) => (
                <Bar
                  key={s.domain}
                  label={s.own ? `${s.name} (you)` : s.name}
                  value={`${s.share}%`}
                  width={s.share}
                  strong={s.own}
                  tone={s.own ? "primary" : "muted"}
                />
              ))}
            </ul>
          </Panel>
        ) : null}
        <Panel
          title="By AI engine"
          hint="Share of each engine's answers that name you."
        >
          <ul className="flex flex-col gap-3">
            {engines.map((e) => (
              <Bar
                key={e.engine}
                label={<EngineName engine={e.engine} />}
                value={`${e.mentioned}/${e.answered}`}
                width={pct(e.mentioned, e.answered)}
                tone="accent"
              />
            ))}
          </ul>
        </Panel>
        {sources.length ? (
          <Panel
            title="Sites AI cites most"
            hint="Being listed or linked on these makes AI more likely to name you."
          >
            <ol className="flex flex-col divide-y text-sm">
              {sources.map((s) => (
                <li
                  key={s.domain}
                  className="flex items-center justify-between gap-3 py-2"
                >
                  <span className="min-w-0 truncate font-medium">
                    {s.domain}
                    {s.own ? (
                      <span className="ml-2 text-xs text-success">
                        your site
                      </span>
                    ) : null}
                  </span>
                  <span className="shrink-0 tabular-nums text-muted-foreground">
                    {s.answers} answer{s.answers === 1 ? "" : "s"}
                  </span>
                </li>
              ))}
            </ol>
          </Panel>
        ) : null}
      </div>
    </div>
  );
}

function FirstRun({
  projectId,
  configured,
}: {
  projectId: string;
  configured: boolean;
}) {
  return (
    <section className="flex flex-col items-start gap-3 rounded-xl border bg-card p-6">
      <h2 className="text-lg font-semibold">
        {configured
          ? "Run your first check to see what AI says about you"
          : "Pick the questions your customers ask AI"}
      </h2>
      <p className="max-w-2xl text-sm text-muted-foreground">
        Uplumo asks ChatGPT, Gemini, Google AI Overview and others those
        questions, saves every answer, and shows who gets recommended. You see
        the cost before anything runs.
      </p>
      <div className="flex flex-wrap gap-2">
        <Button
          nativeButton={false}
          render={
            <Link
              to="/p/$projectId/ai-visibility/setup"
              params={{ projectId }}
            />
          }
        >
          {configured ? "Review and run" : "Set up tracking"}
        </Button>
        <Button
          variant="outline"
          nativeButton={false}
          render={
            <Link
              to="/p/$projectId/ai-visibility/test"
              params={{ projectId }}
            />
          }
        >
          Test one question first
        </Button>
      </div>
    </section>
  );
}

function Stat({
  label,
  value,
  note,
  tone = "plain",
}: {
  label: string;
  value: string;
  note: string;
  tone?: "plain" | "up" | "down";
}) {
  return (
    <div className="rounded-xl border bg-card p-4">
      <dt className="text-sm text-muted-foreground">{label}</dt>
      <dd className="mt-1 text-3xl font-semibold tabular-nums">{value}</dd>
      <dd
        className={
          tone === "up"
            ? "mt-1 text-sm font-medium text-success"
            : tone === "down"
              ? "mt-1 text-sm font-medium text-destructive"
              : "mt-1 text-sm text-muted-foreground"
        }
      >
        {note}
      </dd>
    </div>
  );
}

function Panel({
  title,
  hint,
  children,
}: {
  title: string;
  hint: string;
  children: React.ReactNode;
}) {
  return (
    <section className="flex flex-col gap-4 rounded-xl border bg-card p-5">
      <div>
        <h2 className="text-base font-semibold">{title}</h2>
        <p className="text-sm text-muted-foreground">{hint}</p>
      </div>
      {children}
    </section>
  );
}

const BAR_TONE = {
  primary: "bg-primary",
  muted: "bg-muted-foreground/40",
  accent: "bg-chart-2",
} as const;

function Bar({
  label,
  value,
  width,
  tone,
  strong = false,
}: {
  label: React.ReactNode;
  value: string;
  width: number;
  tone: keyof typeof BAR_TONE;
  strong?: boolean;
}) {
  return (
    <li className="grid grid-cols-[minmax(0,10rem)_minmax(0,1fr)_3.5rem] items-center gap-3 text-sm">
      <span className={strong ? "truncate font-semibold" : "truncate"}>
        {label}
      </span>
      <span className="h-2.5 rounded-full bg-muted" aria-hidden>
        <span
          className={`block h-2.5 rounded-full ${BAR_TONE[tone]}`}
          style={{ width: `${Math.max(width, width > 0 ? 2 : 0)}%` }}
        />
      </span>
      <span className="text-right font-medium tabular-nums">{value}</span>
    </li>
  );
}
