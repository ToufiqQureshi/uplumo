import { useMutation, useQuery } from "@tanstack/react-query";
import { LoaderCircle, Sparkles } from "lucide-react";
import { useRef, useState } from "react";
import { Button } from "@/client/components/ui/button";
import { Textarea } from "@/client/components/ui/textarea";
import {
  generateWritingSuggestion,
  getContentWriterStatus,
} from "./contentEditorApi";
import { taskInstructions } from "./editor-actions";
import type { ScoreCheck } from "./scoring";
import type { ContentDraftInput, EditorMode } from "./scoring";
import { useEditorSession } from "./editor-session";
import { writerContext } from "./writer-context";
import { useFreeSeoEditor } from "./FreeSeoEditorContext";
import { TitleMetaChoices } from "./TitleMetaChoices";
import { PRIMARY_ACTIONS, writerActions } from "./writer-actions";
import {
  applyWriterTarget,
  captureWriterTarget,
  writerTargetMode,
  type WriterAction,
  type WriterTarget,
} from "./writer-target";

export function WriterAssistant({
  projectId,
  draft,
  mode,
  brief,
  disabled,
  scope,
  task,
  onClearTask,
}: {
  projectId: string;
  draft: ContentDraftInput;
  mode: EditorMode;
  brief: string;
  disabled: boolean;
  scope: string;
  task?: Pick<ScoreCheck, "label" | "detail">;
  onClearTask?: () => void;
}) {
  const { editor } = useEditorSession();
  const { setDraftField } = useFreeSeoEditor();
  const [instructions, setInstructions] = useState("");
  const [language, setLanguage] = useState("");
  const [result, setResult] = useState("");
  const [error, setError] = useState("");
  const [target, setTarget] = useState<WriterTarget | null>(null);
  const requestPending = useRef(false);
  const availability = useQuery({
    queryKey: ["contentWriterStatus"],
    queryFn: getContentWriterStatus,
    staleTime: 60_000,
  });
  const writerDisabled =
    availability.isPending ||
    availability.isError ||
    !availability.data?.configured;
  const mutation = useMutation({
    mutationFn: ({
      action,
      idempotencyKey,
      target: requestedTarget,
    }: {
      action: WriterAction;
      idempotencyKey: string;
      target: WriterTarget;
    }) => {
      return generateWritingSuggestion({
        data: {
          projectId,
          action,
          mode,
          title: draft.title,
          targetKeyword: draft.targetKeyword,
          content: writerContext(
            requestedTarget.document,
            requestedTarget.selected,
            requestedTarget.currentBlock,
          ),
          language,
          instructions: taskInstructions(task, instructions),
          brief: brief.slice(0, 2_000),
          idempotencyKey,
        },
      });
    },
    onSuccess: (suggestion, variables) => {
      setResult(suggestion.content);
      setTarget(variables.target);
      setError("");
    },
    onError: (cause) =>
      setError(
        cause instanceof Error
          ? cause.message
          : "Could not generate a suggestion.",
      ),
  });
  const runAction = (action: WriterAction) => {
    if (requestPending.current) return;
    if (!editor) {
      setError("Open the visual article editor before using AI help.");
      return;
    }
    const requestedTarget = captureWriterTarget(editor, action, scope);
    if (!requestedTarget) {
      setError(
        "Select text within one paragraph (up to 5,000 characters, no images), then try again.",
      );
      return;
    }
    setResult("");
    setTarget(null);
    setError("");
    requestPending.current = true;
    mutation.mutate(
      { action, idempotencyKey: crypto.randomUUID(), target: requestedTarget },
      {
        onSettled: () => {
          requestPending.current = false;
        },
      },
    );
  };

  const actionButton = ({
    id,
    label,
    hint,
  }: {
    id: WriterAction;
    label: string;
    hint: string;
  }) => (
    <Button
      key={id}
      type="button"
      size="sm"
      variant="outline"
      className="h-auto min-h-12 justify-start py-2 text-left whitespace-normal"
      disabled={
        disabled ||
        writerDisabled ||
        mutation.isPending ||
        (id !== "outline" && !draft.content.trim())
      }
      onClick={() => runAction(id)}
    >
      {mutation.isPending && mutation.variables?.action === id ? (
        <LoaderCircle aria-hidden className="animate-spin" />
      ) : null}
      <span className="flex flex-col items-start gap-0.5">
        <span>{label}</span>
        <span className="text-xs font-normal text-muted-foreground">
          {hint}
        </span>
      </span>
    </Button>
  );

  return (
    <section className="space-y-3 rounded-xl border bg-card p-4">
      <div className="flex items-center gap-2">
        <Sparkles aria-hidden className="size-4 text-primary" />
        <h2 className="font-medium">AI writing assistant</h2>
      </div>
      <p className="text-xs leading-5 text-muted-foreground">
        AI only suggests; nothing changes in your article until you press Insert
        or Replace. Check every fact before you publish.
      </p>
      {task ? (
        <div className="space-y-2 rounded-lg border bg-muted p-3 text-sm">
          <h3 className="font-medium">Focused task: {task.label}</h3>
          <p className="text-xs leading-5 text-muted-foreground">
            {task.detail}
          </p>
          <p className="text-xs">
            Select the relevant passage, then choose an action below. Preparing
            this task has not generated or changed any text.
          </p>
          <Button variant="ghost" size="sm" onClick={onClearTask}>
            Clear focused task
          </Button>
        </div>
      ) : null}
      {writerDisabled ? (
        <p
          role="status"
          className="rounded-md bg-muted px-3 py-2 text-sm text-muted-foreground"
        >
          {availability.isError
            ? "Could not check AI writing availability. Try again shortly."
            : availability.data?.message || "Checking AI writing availability…"}
        </p>
      ) : null}
      <label className="block space-y-1 text-sm">
        <span>
          Output language{" "}
          <span className="text-muted-foreground">(optional)</span>
        </span>
        <input
          id="content-assistant-language"
          name="content-assistant-language"
          value={language}
          onChange={(event) => setLanguage(event.target.value)}
          maxLength={80}
          placeholder="Use the language of my draft"
          className="h-9 w-full rounded-md border bg-background px-3 text-sm"
          disabled={disabled || mutation.isPending}
        />
      </label>
      <label className="block space-y-1 text-sm">
        <span>
          Instructions{" "}
          <span className="text-muted-foreground">
            {task ? "(optional, up to 1,100 characters)" : "(optional)"}
          </span>
        </span>
        <Textarea
          id="content-assistant-instructions"
          name="content-assistant-instructions"
          value={instructions}
          onChange={(event) => setInstructions(event.target.value)}
          maxLength={task ? 1_100 : 2_000}
          rows={3}
          placeholder="Audience, tone, or facts to emphasize"
          disabled={disabled || mutation.isPending}
        />
      </label>
      <div className="grid grid-cols-2 gap-2">
        {writerActions.slice(0, PRIMARY_ACTIONS).map(actionButton)}
      </div>
      <details className="text-sm">
        <summary className="cursor-pointer text-muted-foreground">
          More AI actions
        </summary>
        <div className="mt-2 grid grid-cols-2 gap-2">
          {writerActions.slice(PRIMARY_ACTIONS).map(actionButton)}
        </div>
      </details>
      {availability.data ? (
        <p className="text-xs text-muted-foreground">
          Each action usually uses 1–5 credits, never more than{" "}
          {availability.data.maxCreditsPerAction}. You pay only for what the AI
          actually wrote.
        </p>
      ) : null}
      {error ? (
        <p role="alert" className="text-sm text-destructive">
          {error}
        </p>
      ) : null}
      {result ? (
        <div className="space-y-2 border-t pt-3">
          <label
            htmlFor="content-ai-suggestion"
            className="text-sm font-medium"
          >
            AI suggestion · review before using
          </label>
          <Textarea
            id="content-ai-suggestion"
            value={result}
            onChange={(event) => setResult(event.target.value)}
            className="min-h-48 font-mono text-xs leading-5"
            maxLength={30_000}
          />
          <Button
            type="button"
            size="sm"
            disabled={
              disabled ||
              !result.trim() ||
              !target ||
              writerTargetMode(target.action) === "review"
            }
            onClick={() => {
              if (
                !editor ||
                !target ||
                !applyWriterTarget(editor, target, scope, result)
              ) {
                setError(
                  "The article changed since this suggestion was generated. Select the passage and try again; nothing was overwritten.",
                );
                return;
              }
              setResult("");
              setTarget(null);
            }}
          >
            {target && writerTargetMode(target.action) === "replace"
              ? "Replace selected passage"
              : "Insert at saved cursor"}
          </Button>
          {target?.action === "title_meta" ? (
            <TitleMetaChoices suggestion={result} onUse={setDraftField} />
          ) : null}
          {target && writerTargetMode(target.action) === "review" ? (
            <p className="text-xs text-muted-foreground">
              {target.action === "fact_check"
                ? "These are notes to check; your article is not changed."
                : "Review and copy these notes manually; they do not replace your article."}
            </p>
          ) : null}
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => {
              setResult("");
              setTarget(null);
              mutation.reset();
              setError("");
            }}
          >
            Discard
          </Button>
          {mutation.variables ? (
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={disabled || mutation.isPending}
              onClick={() => {
                const action = mutation.variables?.action;
                if (action) runAction(action);
              }}
            >
              Try again
            </Button>
          ) : null}
        </div>
      ) : null}
    </section>
  );
}
