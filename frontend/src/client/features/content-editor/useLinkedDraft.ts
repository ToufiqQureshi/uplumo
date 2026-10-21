import { useEffect, useRef } from "react";
import { useQuery } from "@tanstack/react-query";
import { useBlocker } from "@tanstack/react-router";
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
  onRestore,
}: {
  projectId: string;
  selectedDraftId?: string;
  currentDraftId?: string;
  dirty: boolean;
  onRestore: (saved?: ContentDraft) => void;
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
      onRestore(saved);
  }, [query.data, selectedDraftId, currentDraftId, onRestore]);
  const previous = useRef(selectedDraftId);
  useEffect(() => {
    if (previous.current === selectedDraftId) return;
    previous.current = selectedDraftId;
    if (!selectedDraftId) onRestore();
  }, [selectedDraftId, onRestore]);
  useBlocker({
    disabled: !dirty,
    enableBeforeUnload: dirty,
    shouldBlockFn: ({ current, next }) => {
      const nextDraft = "draft" in next.search ? next.search.draft : undefined;
      if (current.pathname === next.pathname && nextDraft === currentDraftId)
        return false;
      return !window.confirm(
        "Discard your unsaved changes and leave this draft?",
      );
    },
  });
  return query;
}
