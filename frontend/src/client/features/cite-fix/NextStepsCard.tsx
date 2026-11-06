import type { ReactNode } from "react";
import type { Step, StepKind } from "./citeFixInsights";

/** The ordered moves, each with its reason, its evidence and its action. */
export function NextStepsCard({
  steps,
  actionFor,
}: {
  steps: Step[];
  actionFor: (kind: StepKind) => ReactNode;
}) {
  if (steps.length === 0) return null;
  return (
    <section aria-labelledby="cite-fix-steps" className="space-y-3">
      <h2 id="cite-fix-steps" className="font-serif text-2xl">
        What to do next
      </h2>
      <ol className="space-y-3">
        {steps.map((step, index) => (
          <li
            key={step.kind}
            className="flex flex-wrap items-start gap-4 rounded-2xl border bg-card p-5"
          >
            <span className="font-mono text-xs text-primary">
              STEP {index + 1}
            </span>
            <div className="min-w-0 flex-[999_1_24rem] space-y-1">
              <p className="text-lg font-semibold">{step.title}</p>
              <p className="text-sm text-muted-foreground">{step.why}</p>
              <p className="text-sm">
                <span className="text-muted-foreground">Evidence: </span>
                {step.proof}
              </p>
            </div>
            <div className="flex flex-[1_1_12rem] justify-end">
              {actionFor(step.kind)}
            </div>
          </li>
        ))}
      </ol>
      <p className="text-xs text-muted-foreground">
        No tool can promise a citation: the AI decides. These are the gaps the
        evidence points to.
      </p>
    </section>
  );
}
