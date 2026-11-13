import { useEffect, useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useBlocker } from "@tanstack/react-router";
import {
  ConfirmDialog,
  type useConfirm,
} from "@/client/components/ConfirmDialog";
import {
  contentEditorQueryKeys,
  getContentDraft,
  type ContentDraft,
} from "./contentEditorApi";

export function useLinkedDraft({
  projectId,
  selectedDraftId,
  currentDraftId,
  dirty,
  keyword,
  onRestore,
}: {
  projectId: string;
  selectedDraftId?: string;
  currentDraftId?: string;
  dirty: boolean;
  /** Fills the opened draft's target keyword when it has none. */
  keyword?: string;
  onRestore: (
    saved?: ContentDraft,
    sent?: undefined,
    keyword?: string,
  ) => void;
}) {
  const query = useQuery({
    queryKey: contentEditorQueryKeys.draft(projectId, selectedDraftId ?? ""),
    queryFn: () =>
      getContentDraft({ data: { projectId, draftId: selectedDraftId! } }),
    enabled: Boolean(selectedDraftId) && selectedDraftId !== currentDraftId,
  });
  useEffect(() => {
    const saved = query.data;
    if (saved && saved.id === selectedDraftId && saved.id !== currentDraftId)
      onRestore(saved, undefined, keyword);
  }, [query.data, selectedDraftId, currentDraftId, keyword, onRestore]);
  const previous = useRef(selectedDraftId);
  useEffect(() => {
    if (previous.current === selectedDraftId) return;
    previous.current = selectedDraftId;
    if (!selectedDraftId) onRestore();
  }, [selectedDraftId, onRestore]);
  // Leaving with unsaved edits waits for the in-app dialog; closing the tab
  // still gets the browser's own prompt (enableBeforeUnload).
  const blocker = useBlocker({
    disabled: !dirty,
    enableBeforeUnload: dirty,
    withResolver: true,
    shouldBlockFn: ({ current, next }) => {
      const nextDraft = "draft" in next.search ? next.search.draft : undefined;
      return !(
        current.pathname === next.pathname && nextDraft === currentDraftId
      );
    },
  });
  const leaveDialog =
    blocker.status === "blocked" ? (
      <ConfirmDialog
        title="Leave this draft?"
        confirmLabel="Discard changes"
        destructive
        onClose={blocker.reset}
        onConfirm={blocker.proceed}
      >
        Your unsaved changes will be lost.
      </ConfirmDialog>
    ) : null;
  return { query, leaveDialog };
}

/** Runs `action` at once, or after the writer agrees to drop unsaved edits. */
export function askToDiscard(
  ask: ReturnType<typeof useConfirm>["ask"],
  dirty: boolean,
  title: string,
  action: () => void,
) {
  if (!dirty) return action();
  ask({
    title,
    body: "Your unsaved changes will be lost.",
    confirmLabel: "Discard changes",
    destructive: true,
    onConfirm: action,
  });
}
