import { Link } from "@tanstack/react-router";
import { Sparkles } from "lucide-react";
import { Button } from "@/client/components/ui/button";
import { VisibilityInsights } from "../ai-search/components/VisibilityInsights";
import type { CiteFixProof } from "./citeFixApi";

/**
 * What the project's latest saved AI Visibility check shows: who AI answers
 * cite for the customer's own questions. It reads saved answers only.
 */
export function ProofCard({
  projectId,
  proof,
}: {
  projectId: string;
  proof: CiteFixProof | null;
}) {
  const tracker = (
    <Button
      size="sm"
      variant="outline"
      nativeButton={false}
      render={
        <Link to="/p/$projectId/prompt-tracking" params={{ projectId }} />
      }
    >
      <Sparkles aria-hidden />
      AI Visibility Tracker
    </Button>
  );
  if (!proof || proof.answers === 0) {
    return (
      <section
        aria-label="AI citation proof"
        className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-dashed p-4 text-sm"
      >
        <p className="text-muted-foreground">
          See which sites ChatGPT, Perplexity and Google AI actually cite for
          your customers' questions: run a check in the AI Visibility Tracker.
        </p>
        {tracker}
      </section>
    );
  }
  return (
    <section
      aria-labelledby="cite-fix-proof"
      className="flex flex-col gap-3 rounded-xl border bg-card p-5"
    >
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="space-y-1">
          <h2 id="cite-fix-proof" className="font-serif text-2xl">
            Who AI cites today
          </h2>
          <p className="text-sm text-muted-foreground">
            From your last AI Visibility check,{" "}
            {new Date(proof.checkedAt).toLocaleDateString("en-IN", {
              day: "numeric",
              month: "short",
            })}
            .
          </p>
        </div>
        {tracker}
      </div>
      <p className="text-lg">
        <span className="font-mono">{proof.citingYou}</span> of{" "}
        <span className="font-mono">{proof.answers}</span> AI answers cite{" "}
        <span className="font-medium">{proof.domain}</span>.
      </p>
      {proof.visibility ? (
        <VisibilityInsights metrics={proof.visibility} />
      ) : null}
      {proof.rivals.length ? (
        <div className="flex flex-col gap-1">
          <p className="label-mono text-muted-foreground">Cited instead</p>
          <ul className="flex flex-wrap gap-2">
            {proof.rivals.map((rival) => (
              <li
                key={rival.domain}
                className="rounded-full border px-2.5 py-0.5 text-xs"
              >
                {rival.domain}{" "}
                <span className="font-mono text-muted-foreground">
                  {rival.count}
                </span>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
      {proof.misses.length ? (
        <div className="flex flex-col gap-1">
          <p className="label-mono text-muted-foreground">
            Questions where you were not cited
          </p>
          <ul className="flex flex-col gap-1 text-sm">
            {proof.misses.map((miss, index) => (
              <li
                key={`${miss.prompt}:${miss.engine}`}
                className="flex flex-wrap items-center gap-x-2 gap-y-1"
              >
                “{miss.prompt}”{" "}
                <span className="text-muted-foreground">
                  ({miss.engine}) cited {miss.cited.join(", ")}
                </span>
                <a
                  className="text-primary underline"
                  href={`#cite-fix-rival-${index}`}
                >
                  Compare with rival
                </a>
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </section>
  );
}
