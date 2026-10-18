import type { Editor } from "@tiptap/react";

export type WriterAction =
  | "outline"
  | "write_section"
  | "fill_gap"
  | "answer_first"
  | "fix_top_issue"
  | "title_meta"
  | "faq"
  | "destuff"
  | "fact_check"
  | "tone_language";

export type WriterTarget = {
  action: WriterAction;
  from: number;
  to: number;
  selected: string;
  currentBlock: string;
  document: string;
  scope: string;
};

const replacementActions: ReadonlySet<WriterAction> = new Set([
  "answer_first",
  "fix_top_issue",
  "destuff",
  "tone_language",
]);
const reviewActions: ReadonlySet<WriterAction> = new Set([
  "outline",
  "title_meta",
  "fact_check",
]);

export function captureWriterTarget(
  editor: Editor,
  action: WriterAction,
  scope: string,
): WriterTarget | null {
  const { from, to } = editor.state.selection;
  const selected = editor.state.doc.textBetween(from, to, "\n");
  if (replacementActions.has(action) && !selected.trim()) return null;
  return {
    action,
    from,
    to,
    selected,
    currentBlock: editor.state.selection.$from.parent.textContent,
    document: editor.getMarkdown(),
    scope,
  };
}

export function writerTargetMode(
  action: WriterAction,
): "replace" | "insert" | "review" {
  if (replacementActions.has(action)) return "replace";
  return reviewActions.has(action) ? "review" : "insert";
}

export function applyWriterTarget(
  editor: Editor,
  target: WriterTarget,
  currentScope: string,
  suggestion: string,
): boolean {
  if (
    writerTargetMode(target.action) === "review" ||
    !suggestion.trim() ||
    target.scope !== currentScope ||
    editor.getMarkdown() !== target.document
  )
    return false;
  if (
    target.from > editor.state.doc.content.size ||
    target.to > editor.state.doc.content.size
  )
    return false;
  if (
    editor.state.doc.textBetween(target.from, target.to, "\n") !==
    target.selected
  )
    return false;
  const range =
    writerTargetMode(target.action) === "replace"
      ? { from: target.from, to: target.to }
      : { from: target.to, to: target.to };
  return editor
    .chain()
    .focus()
    .insertContentAt(
      range,
      suggestion,
      writerTargetMode(target.action) === "replace"
        ? {}
        : { contentType: "markdown" },
    )
    .run();
}
