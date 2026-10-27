import { Link } from "@tanstack/react-router";
import { useState } from "react";
import { PageHeader } from "@/client/components/PageHeader";
import { Button } from "@/client/components/ui/button";
import { WhyNotCitedCard } from "@/client/features/ai-search/components/WhyNotCitedCard";
import { whyNotCited } from "@/client/features/ai-search/whyNotCited";
import { AiAction, InsightCard } from "./AiAction";
import { promptGaps } from "./opportunities";
import { useLatestAnswers } from "./useLatestAnswers";

/**
 * Prompts where AI names someone else instead of the customer, from answers
 * the tracker already saved. Reading is free; the AI buttons spend credits.
 */
export function AiOpportunitiesPage({ projectId }: { projectId: string }) {
  const ai = useLatestAnswers(projectId);
  const [selected, setSelected] = useState<string | null>(null);
  const domain = ai.configuration?.domain ?? "";
  const gaps = promptGaps(ai.answers, domain);
  const why = whyNotCited(ai.answers, domain);
  const current = gaps.find((g) => g.prompt === selected) ?? gaps[0];

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col gap-6 p-4 sm:p-6">
      <PageHeader
        title="AI Opportunities"
        description="Questions where AI recommends someone else instead of you, and what to do about each."
      />
      {ai.isPending ? (
        <p className="text-sm text-muted-foreground">Reading saved answers…</p>
      ) : !ai.run ? (
        <InsightCard
          headline="Run your first AI Visibility check to see where AI names competitors instead of you."
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
        <>
          <InsightCard
            stamp={`Latest check · ${new Date(ai.run.createdAt).toLocaleDateString("en-IN")}`}
            headline={
              gaps.length === 0 ? (
                "AI names you in every answer it gave. Track more questions to find new gaps."
              ) : (
                <>
                  AI leaves you out of{" "}
                  <span className="text-destructive">
                    {gaps.length} question{gaps.length === 1 ? "" : "s"}
                  </span>
                  . {why.gaps[0] ? why.gaps[0].title + "." : null}
                </>
              )
            }
            evidence={[
              {
                source: "Answers checked",
                text: `${why.answered} answers, your site linked in ${why.cited}`,
              },
              ...(gaps[0]?.rivals[0]
                ? [
                    {
                      source: "Named instead most",
                      text: gaps[0].rivals[0],
                    },
                  ]
                : []),
              ...(why.gaps[0]
                ? [
                    {
                      source: "Sources AI trusted",
                      text: why.gaps[0].evidence.slice(0, 3).join(", "),
                    },
                  ]
                : []),
            ]}
          >
            {gaps.length ? (
              <AiAction
                projectId={projectId}
                label="Make my action plan"
                input={{
                  action: "action_plan",
                  brief: [
                    `Brand: ${ai.configuration?.brand ?? ""} (${domain}).`,
                    `Answers checked: ${why.answered}; site linked in ${why.cited}.`,
                    ...why.gaps.map(
                      (g) =>
                        `${g.title}: ${g.detail} Evidence: ${g.evidence.join(", ")}.`,
                    ),
                    `Questions where others are named: ${gaps
                      .slice(0, 10)
                      .map((g) => `"${g.prompt}" (${g.rivals.join(", ")})`)
                      .join("; ")}`,
                  ].join("\n"),
                }}
              />
            ) : null}
          </InsightCard>

          {gaps.length ? (
            <div className="flex flex-col gap-4 lg:flex-row lg:items-start">
              <ol className="flex min-w-0 flex-[2] flex-col divide-y rounded-xl border bg-card">
                {gaps.map((g) => (
                  <li key={g.prompt}>
                    <button
                      type="button"
                      aria-pressed={current?.prompt === g.prompt}
                      onClick={() => setSelected(g.prompt)}
                      className="flex w-full flex-col gap-2 p-4 text-left hover:bg-muted/50 aria-pressed:bg-muted"
                    >
                      <span className="font-medium">“{g.prompt}”</span>
                      <span className="flex flex-wrap gap-2 text-xs">
                        {g.rivals.map((r) => (
                          <span
                            key={r}
                            className="rounded-full border border-destructive/40 px-2 py-0.5 text-destructive"
                          >
                            {r}
                          </span>
                        ))}
                        <span className="font-mono text-muted-foreground">
                          {g.engines.join(" · ")}
                        </span>
                      </span>
                    </button>
                  </li>
                ))}
              </ol>
              {current ? (
                <aside className="flex min-w-0 flex-1 flex-col gap-3 rounded-xl border bg-card p-5">
                  <span className="font-mono text-xs text-muted-foreground">
                    SELECTED QUESTION
                  </span>
                  <h2 className="font-serif text-2xl">“{current.prompt}”</h2>
                  <blockquote className="rounded-lg bg-muted p-3 text-sm">
                    {current.excerpt || "No answer text saved."}
                  </blockquote>
                  {current.sources.length ? (
                    <p className="font-mono text-xs text-muted-foreground">
                      Sources AI cited: {current.sources.join(", ")}
                    </p>
                  ) : null}
                  <AiAction
                    key={current.prompt}
                    projectId={projectId}
                    label="Draft the page that wins this question"
                    input={{
                      action: "outline",
                      targetKeyword: current.prompt.slice(0, 200),
                      title: current.prompt.slice(0, 200),
                      brief: `AI named ${current.rivals.join(", ")} for this question and cited ${current.sources.join(", ") || "no pages"}. Our business: ${ai.configuration?.brand ?? ""} (${domain}). Start with a direct 40-word answer AI can quote.`,
                    }}
                  >
                    {() => (
                      <Button
                        size="xs"
                        variant="outline"
                        nativeButton={false}
                        render={
                          <Link
                            to="/p/$projectId/content-editor"
                            params={{ projectId }}
                          />
                        }
                      >
                        Open Content editor
                      </Button>
                    )}
                  </AiAction>
                  <Button
                    variant="outline"
                    nativeButton={false}
                    render={
                      <Link
                        to="/p/$projectId/prompt-explorer"
                        params={{ projectId }}
                      />
                    }
                  >
                    Re-check in Prompt Explorer
                  </Button>
                </aside>
              ) : null}
            </div>
          ) : null}

          <WhyNotCitedCard
            projectId={projectId}
            answers={ai.answers}
            ownDomain={domain}
          />
        </>
      )}
    </div>
  );
}
