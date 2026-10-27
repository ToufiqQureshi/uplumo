import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { Sparkles } from "lucide-react";
import { useState } from "react";
import { PageHeader } from "@/client/components/PageHeader";
import { Button } from "@/client/components/ui/button";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import { trackingKey } from "@/client/features/ai-search/trackingApi";
import { AiAction, InsightCard, MAX_AI_ACTION_CREDITS } from "./AiAction";
import { insightsApi } from "./insightsApi";
import { MOODS, sentimentSummary, topComplaint, type Mood } from "./sentiment";
import { useLatestAnswers } from "./useLatestAnswers";

const ENGINE: Record<string, string> = {
  chat_gpt: "ChatGPT",
  gemini: "Gemini",
  google_ai_overview: "AI Overviews",
  claude: "Claude",
  perplexity: "Perplexity",
};
const MOOD_STYLE: Record<Mood, string> = {
  positive: "text-success",
  neutral: "text-muted-foreground",
  negative: "text-destructive",
};
const BAR: Record<Mood, string> = {
  positive: "bg-success",
  neutral: "bg-muted-foreground/40",
  negative: "bg-destructive",
};

/**
 * How AI answers speak about the brand. Classifying is paid and only runs on
 * a click; reading the saved labels is free.
 */
export function AiSentimentPage({ projectId }: { projectId: string }) {
  const client = useQueryClient();
  const ai = useLatestAnswers(projectId);
  const [mood, setMood] = useState<Mood | null>(null);
  const [open, setOpen] = useState<string | null>(null);
  const classify = useMutation({
    retry: false,
    mutationFn: () => insightsApi.classifySentiment(projectId, ai.run!.id),
    onSettled: () =>
      client.invalidateQueries({ queryKey: trackingKey(projectId) }),
  });
  const s = sentimentSummary(ai.answers);
  const brand = ai.configuration?.brand ?? "your brand";
  const complaint = topComplaint(ai.answers);
  const negatives = ai.answers.filter((a) => a.sentiment === "negative");
  const list = ai.answers.flatMap((a) =>
    a.sentiment !== "" &&
    a.sentiment !== "not_mentioned" &&
    (!mood || a.sentiment === mood)
      ? [{ ...a, mood: a.sentiment }]
      : [],
  );

  if (ai.isPending)
    return <p className="p-6 text-sm text-muted-foreground">Loading…</p>;

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-6 p-4 sm:p-6">
      <PageHeader
        title="AI Sentiment"
        description={`What ChatGPT, Gemini and other assistants say about ${brand}, and where it comes from.`}
      />
      {!ai.run ? (
        <InsightCard
          headline="Run your first AI Visibility check, then come back to see how AI talks about you."
          evidence={[]}
        >
          <Button
            nativeButton={false}
            render={
              <Link to="/p/$projectId/prompt-tracking" params={{ projectId }} />
            }
          >
            Set up the AI Visibility Tracker
          </Button>
        </InsightCard>
      ) : (
        <InsightCard
          stamp={`Latest check · ${s.total} answers that mention you`}
          headline={
            s.total === 0 ? (
              s.pending ? (
                `${s.pending} saved answers are not classified yet. Classify them to see whether AI speaks well of ${brand}.`
              ) : (
                "No saved answer mentions you yet, so there is nothing to judge. Start with AI Opportunities."
              )
            ) : (
              <>
                AI speaks{" "}
                <span className="text-success">
                  well of you in {s.share("positive")}%
                </span>{" "}
                of answers that mention you
                {s.counts.negative ? (
                  <>
                    , and{" "}
                    <span className="text-destructive">
                      negatively in {s.counts.negative}
                    </span>
                    {complaint ? <> (“{complaint}”)</> : null}
                  </>
                ) : null}
                .
              </>
            )
          }
          evidence={MOODS.map((m) => ({
            source: m,
            text: `${s.counts[m]} answers · ${s.share(m)}%`,
          }))}
        >
          <div className="flex flex-wrap items-center gap-2">
            {s.pending ? (
              <Button
                onClick={() => classify.mutate()}
                disabled={classify.isPending}
              >
                <Sparkles aria-hidden />
                {classify.isPending
                  ? "Reading answers…"
                  : `Classify ${Math.min(s.pending, 20)} answer${s.pending === 1 ? "" : "s"}`}
              </Button>
            ) : null}
            {s.pending ? (
              <span className="font-mono text-xs text-muted-foreground">
                Up to {MAX_AI_ACTION_CREDITS} credits per click; answers that
                don&rsquo;t name you are free
              </span>
            ) : null}
          </div>
          {classify.isError ? (
            <p role="alert" className="text-sm text-destructive">
              {getStandardErrorMessage(classify.error)}
            </p>
          ) : null}
          {classify.data?.remaining ? (
            <p className="text-sm text-muted-foreground">
              {classify.data.remaining} answers left; click again to continue.
            </p>
          ) : null}
          {negatives.length ? (
            <div className="flex flex-col gap-3 border-t pt-4">
              <span className="font-mono text-xs text-muted-foreground">
                FIX THE COMPLAINT
              </span>
              <AiAction
                projectId={projectId}
                label="Write a “what we fixed” Google post"
                input={{
                  action: "business_post",
                  brief: `Business: ${brand}. AI answers repeat this complaint: ${negatives
                    .map((a) => a.sentimentQuote || a.sentimentReason)
                    .filter(Boolean)
                    .slice(0, 5)
                    .join(
                      " | ",
                    )}. Write about what changed only as [VERIFY] placeholders.`,
                }}
              />
            </div>
          ) : null}
        </InsightCard>
      )}

      {s.total ? (
        <>
          <div className="grid gap-3 sm:grid-cols-3">
            {MOODS.map((m) => (
              <button
                key={m}
                type="button"
                aria-pressed={mood === m}
                onClick={() => setMood(mood === m ? null : m)}
                className="flex flex-col gap-1 rounded-xl border bg-card p-4 text-left hover:border-foreground/30 aria-pressed:border-foreground"
              >
                <span className="text-sm capitalize text-muted-foreground">
                  {m}
                </span>
                <span className={`font-serif text-4xl ${MOOD_STYLE[m]}`}>
                  {s.share(m)}%
                </span>
                <span className="font-mono text-xs text-muted-foreground">
                  {s.counts[m]} answers · click to filter
                </span>
              </button>
            ))}
          </div>
          <section className="flex flex-col gap-3 rounded-xl border bg-card p-4">
            <h2 className="font-mono text-xs text-muted-foreground">
              BY ENGINE
            </h2>
            {[...s.byEngine].map(([engine, c]) => {
              const n = c.positive + c.neutral + c.negative;
              return (
                <div
                  key={engine}
                  className="grid grid-cols-[100px_1fr_70px] items-center gap-3 text-sm"
                >
                  <span>{ENGINE[engine] ?? engine}</span>
                  <div
                    className="flex h-3 overflow-hidden rounded-full bg-muted"
                    role="img"
                    aria-label={`${ENGINE[engine] ?? engine}: ${c.positive} positive, ${c.neutral} neutral, ${c.negative} negative`}
                  >
                    {MOODS.map((m) => (
                      <div
                        key={m}
                        className={BAR[m]}
                        style={{ width: `${(100 * c[m]) / n}%` }}
                      />
                    ))}
                  </div>
                  <span className="text-right font-mono text-xs text-muted-foreground">
                    {n} answers
                  </span>
                </div>
              );
            })}
          </section>
          <ol className="flex flex-col divide-y rounded-xl border bg-card">
            {list.map((a) => (
              <li key={a.id}>
                <button
                  type="button"
                  aria-expanded={open === a.id}
                  onClick={() => setOpen(open === a.id ? null : a.id)}
                  className="flex w-full flex-col gap-2 p-4 text-left hover:bg-muted/50"
                >
                  <span className="flex flex-wrap items-center gap-2 text-xs">
                    <span
                      className={`rounded-full border px-2 py-0.5 capitalize ${MOOD_STYLE[a.mood]}`}
                    >
                      {a.sentiment}
                    </span>
                    <span className="font-mono text-muted-foreground">
                      {ENGINE[a.engine] ?? a.engine} · “{a.prompt}”
                    </span>
                  </span>
                  <span className="text-sm">
                    {a.sentimentQuote
                      ? `“${a.sentimentQuote}”`
                      : a.sentimentReason}
                  </span>
                  {open === a.id ? (
                    <span className="rounded-lg bg-muted p-3 text-sm text-muted-foreground">
                      <b className="text-foreground">Why:</b>{" "}
                      {a.sentimentReason}
                    </span>
                  ) : null}
                </button>
              </li>
            ))}
          </ol>
        </>
      ) : null}
    </div>
  );
}
