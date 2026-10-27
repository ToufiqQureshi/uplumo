import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { useState } from "react";
import { PageHeader } from "@/client/components/PageHeader";
import { Button } from "@/client/components/ui/button";
import { Checkbox } from "@/client/components/ui/checkbox";
import { Input } from "@/client/components/ui/input";
import { Label } from "@/client/components/ui/label";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import {
  trackingApi,
  trackingKey,
} from "@/client/features/ai-search/trackingApi";
import { AiAction, InsightCard } from "./AiAction";
import { addPrompts, parsePromptIdeas, type PromptIdea } from "./promptIdeas";
import { useLatestAnswers } from "./useLatestAnswers";

const TRACKER_LIMIT = 30;

/**
 * AI writes the questions customers ask assistants about a keyword; the
 * chosen ones join the AI Visibility Tracker. Saving them is free; each
 * tracked question is checked (and charged) on the tracker's own schedule.
 */
export function PromptIdeasPage({ projectId }: { projectId: string }) {
  const client = useQueryClient();
  const ai = useLatestAnswers(projectId);
  const [keyword, setKeyword] = useState("");
  const [ideas, setIdeas] = useState<PromptIdea[]>([]);
  const [picked, setPicked] = useState<Set<string>>(new Set());
  const config = ai.configuration;
  const track = useMutation({
    mutationFn: () => {
      const next = addPrompts(config!.prompts, [...picked], TRACKER_LIMIT);
      return trackingApi
        .save(projectId, { ...config!, prompts: next.prompts })
        .then(() => next.added);
    },
    onSuccess: () => {
      setPicked(new Set());
      return client.invalidateQueries({ queryKey: trackingKey(projectId) });
    },
  });
  const room = config ? TRACKER_LIMIT - config.prompts.length : 0;
  const toggle = (q: string) =>
    setPicked((prev) => {
      const next = new Set(prev);
      if (next.has(q)) next.delete(q);
      else next.add(q);
      return next;
    });

  return (
    <div className="mx-auto flex w-full max-w-4xl flex-col gap-6 p-4 sm:p-6">
      <PageHeader
        title="AI Prompt Ideas"
        description="The questions your customers type into ChatGPT and Gemini. Track the ones that matter."
      />
      <InsightCard
        stamp={
          config
            ? `Tracker: ${config.prompts.length} of ${TRACKER_LIMIT} questions used`
            : "AI Visibility Tracker not set up yet"
        }
        headline={
          ideas.length ? (
            <>
              Pick the questions a buyer would ask. Ones with{" "}
              <span className="text-primary">“best”, “price” or “near”</span>{" "}
              are closest to a purchase; track those first.
            </>
          ) : (
            "Type what you sell. AI suggests the questions people ask assistants about it."
          )
        }
        evidence={
          ai.run
            ? [
                {
                  source: "Your latest check",
                  text: `Named in ${ai.run.mentioned} of ${ai.run.answered} answers`,
                },
              ]
            : []
        }
      >
        <form
          className="flex flex-col gap-2"
          onSubmit={(e) => e.preventDefault()}
        >
          <Label htmlFor="prompt-keyword">Keyword</Label>
          <Input
            id="prompt-keyword"
            value={keyword}
            maxLength={200}
            placeholder="e.g. cafe in bhopal"
            onChange={(e) => setKeyword(e.target.value)}
          />
        </form>
        {keyword.trim() ? (
          <AiAction
            key={keyword.trim()}
            projectId={projectId}
            label="Generate question ideas"
            showResult={false}
            input={{ action: "prompt_ideas", targetKeyword: keyword.trim() }}
            onResult={(text) => {
              setIdeas(parsePromptIdeas(text));
              setPicked(new Set());
            }}
          />
        ) : null}
      </InsightCard>

      {ideas.length ? (
        <section className="flex flex-col rounded-xl border bg-card">
          <ul className="flex flex-col divide-y">
            {ideas.map((idea) => {
              const id = `idea-${idea.question}`;
              return (
                <li key={idea.question} className="flex items-center gap-3 p-4">
                  <Checkbox
                    id={id}
                    checked={picked.has(idea.question)}
                    onCheckedChange={() => toggle(idea.question)}
                  />
                  <Label htmlFor={id} className="flex-1 font-normal">
                    {idea.question}
                  </Label>
                  <span className="rounded-full border px-2 py-0.5 text-xs text-muted-foreground">
                    {idea.intent}
                  </span>
                </li>
              );
            })}
          </ul>
          <div className="sticky bottom-3 flex flex-wrap items-center justify-between gap-3 border-t bg-card p-4">
            <span className="text-sm">
              <b>{picked.size}</b> selected
              {config ? ` · room for ${room} more in the tracker` : ""}
            </span>
            {config ? (
              <Button
                disabled={picked.size === 0 || room <= 0 || track.isPending}
                onClick={() => track.mutate()}
              >
                Add {picked.size} to the tracker
              </Button>
            ) : (
              <Button
                nativeButton={false}
                render={
                  <Link
                    to="/p/$projectId/prompt-tracking"
                    params={{ projectId }}
                  />
                }
              >
                Set up the tracker first
              </Button>
            )}
          </div>
          {track.isError ? (
            <p role="alert" className="px-4 pb-4 text-sm text-destructive">
              {getStandardErrorMessage(track.error)}
            </p>
          ) : null}
          {track.isSuccess ? (
            <p className="px-4 pb-4 text-sm text-success">
              Added {track.data} question{track.data === 1 ? "" : "s"}. They are
              checked on the tracker&rsquo;s next run.
            </p>
          ) : null}
        </section>
      ) : null}
    </div>
  );
}
