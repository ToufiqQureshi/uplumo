import { useMutation } from "@tanstack/react-query";
import { Sparkles } from "lucide-react";
import { useState, type ReactNode } from "react";
import { Button } from "@/client/components/ui/button";
import { CopyButton } from "@/client/components/CopyButton";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import { Markdown } from "@/client/components/Markdown";
import {
  generateWritingSuggestion,
  type WriterAction,
} from "@/client/features/content-editor/contentEditorApi";

/** The most one AI action can charge (the writer's per-action ceiling). */
export const MAX_AI_ACTION_CREDITS = 15;

type AiActionInput = {
  action: WriterAction;
  /** Evidence the model works from: numbers, sites, reviews. */
  brief?: string;
  content?: string;
  title?: string;
  targetKeyword?: string;
  instructions?: string;
};

/**
 * One paid AI action: the button shows the most it can cost before the click,
 * each click (and each Regenerate) is a new idempotent request, and a failed
 * request is never retried by the browser, because it may already be billed.
 */
export function AiAction({
  projectId,
  label,
  input,
  onResult,
  showResult = true,
  children,
}: {
  projectId: string;
  label: string;
  input: AiActionInput;
  onResult?: (text: string) => void;
  /** False when the page renders the result itself (parsed lists). */
  showResult?: boolean;
  /** Extra buttons under the result, such as "Open in Content editor". */
  children?: (text: string) => ReactNode;
}) {
  const [result, setResult] = useState("");
  const mutation = useMutation({
    retry: false,
    mutationFn: () =>
      generateWritingSuggestion({
        data: {
          projectId,
          action: input.action,
          mode: "aeo",
          title: input.title ?? "",
          targetKeyword: input.targetKeyword ?? "",
          content: input.content ?? "",
          language: "English",
          instructions: input.instructions ?? "",
          brief: (input.brief ?? "").slice(0, 8_000),
          idempotencyKey: crypto.randomUUID(),
        },
      }),
    onSuccess: (out) => {
      setResult(out.content);
      onResult?.(out.content);
    },
  });
  return (
    <div className="flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-2">
        <Button onClick={() => mutation.mutate()} disabled={mutation.isPending}>
          <Sparkles aria-hidden />
          {mutation.isPending ? "Working…" : result ? "Regenerate" : label}
        </Button>
        <span className="font-mono text-xs text-muted-foreground">
          Up to {MAX_AI_ACTION_CREDITS} credits, charged at actual cost
        </span>
      </div>
      {mutation.isError ? (
        <p role="alert" className="text-sm text-destructive">
          {getStandardErrorMessage(mutation.error)}
        </p>
      ) : null}
      {result && showResult ? (
        <div className="flex flex-col gap-3 rounded-lg border border-dashed border-primary/40 bg-muted/40 p-4">
          <span className="font-mono text-xs text-primary">AI result</span>
          <Markdown className="text-sm">{result}</Markdown>
          <div className="flex flex-wrap gap-2">
            <CopyButton value={result} successMessage="Copied" />
            {children?.(result)}
          </div>
        </div>
      ) : null}
    </div>
  );
}

/** A number the advice rests on, with where it came from. */
type Evidence = { source: string; text: ReactNode };

/**
 * The top of every insight page: one sentence on what is happening and what
 * it is worth, the numbers behind it, then what to do (children).
 */
export function InsightCard({
  headline,
  evidence,
  stamp,
  children,
}: {
  headline: ReactNode;
  evidence: Evidence[];
  stamp?: string;
  children?: ReactNode;
}) {
  return (
    <section
      aria-label="AI insight"
      className="flex flex-col gap-4 rounded-xl border border-primary/30 bg-card p-5"
    >
      <div className="flex flex-wrap items-center gap-2">
        <span className="rounded-full bg-highlight px-2.5 py-0.5 font-mono text-xs font-medium text-highlight-foreground">
          Insight
        </span>
        {stamp ? (
          <span className="font-mono text-xs text-muted-foreground">
            {stamp}
          </span>
        ) : null}
      </div>
      <p className="font-serif text-2xl leading-snug sm:text-3xl">{headline}</p>
      {evidence.length ? (
        <dl className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
          {evidence.map((e) => (
            <div
              key={e.source}
              className="flex flex-col gap-0.5 rounded-lg bg-muted px-3 py-2.5"
            >
              <dt className="font-mono text-[11px] uppercase text-muted-foreground">
                {e.source}
              </dt>
              <dd className="text-sm">{e.text}</dd>
            </div>
          ))}
        </dl>
      ) : null}
      {children}
    </section>
  );
}
