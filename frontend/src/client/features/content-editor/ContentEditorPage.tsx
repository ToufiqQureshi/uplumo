import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { sortBy } from "remeda";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { projectsQueryOptions } from "@/client/features/projects/projectQueries";
import {
  contentEditorQueryKeys,
  deleteContentDraft,
  getContentDraft,
  listContentDrafts,
  saveContentDraft,
  type ContentDraft,
  type ContentDraftSummary,
  type SearchBrief,
} from "./contentEditorApi";
import { briefMatchesTopic, toScoringBrief } from "./brief-view";
import {
  scoreContent,
  type ContentDraftInput,
  type EditorMode,
} from "./scoring";
import {
  DraftLoadingState,
  DraftToolbar,
  WriterReminder,
} from "./EditorPanels";
import { useLinkedDraft } from "./useLinkedDraft";
import { ArticleComposer } from "./ArticleComposer";
import { FreeSeoEditorProvider } from "./FreeSeoEditorContext";
import {
  ContentEditorSidePanel,
  type EditorTab,
} from "./ContentEditorSidePanel";
import { isArticleContentWithinLimit } from "./content-limits";
import { draftAfterSave, sameDraft } from "./draft-save";
import { primaryTargetTopic } from "./target-topics";

const blankDraft: ContentDraftInput = {
  title: "",
  targetKeyword: "",
  slug: "",
  metaDescription: "",
  content: "",
};

const asInput = (draft: ContentDraft): ContentDraftInput => ({
  title: draft.title,
  targetKeyword: draft.targetKeyword,
  slug: draft.slug,
  metaDescription: draft.metaDescription,
  content: draft.content,
  authorId: draft.authorId,
  lastReviewedAt: draft.lastReviewedAt?.slice(0, 10),
});
const slugFromTitle = (title: string) =>
  title
    .normalize("NFKD")
    .replace(/[\u0300-\u036f]/gu, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/gu, "-")
    .replace(/^-|-$/gu, "")
    .slice(0, 300);

const blankDraftForKeyword = (keyword?: string): ContentDraftInput => ({
  ...blankDraft,
  targetKeyword: keyword ?? "",
});

export function ContentEditorPage({
  projectId,
  initialKeyword,
  selectedDraftId,
  onDraftIdChange,
}: {
  projectId: string;
  initialKeyword?: string;
  selectedDraftId?: string;
  onDraftIdChange: (id: string | undefined) => void;
}) {
  const queryClient = useQueryClient();
  const draftsQuery = useQuery({
    queryKey: contentEditorQueryKeys.drafts(projectId),
    queryFn: () => listContentDrafts({ data: { projectId } }),
  });
  const projectsQuery = useQuery(projectsQueryOptions());
  const projectDomain =
    projectsQuery.data?.find((p) => p.id === projectId)?.domain ?? "";
  const [mode, setMode] = useState<EditorMode>("seo");
  const [tab, setTab] = useState<EditorTab>("score");
  const [previewOpen, setPreviewOpen] = useState(false);
  const [draft, setDraft] = useState<ContentDraftInput>(() =>
    blankDraftForKeyword(initialKeyword),
  );
  const [draftId, setDraftId] = useState<string>();
  const [savedSnapshot, setSavedSnapshot] = useState<ContentDraftInput>(() =>
    blankDraftForKeyword(initialKeyword),
  );
  const [initialized, setInitialized] = useState(false);
  const [imageUploadBusy, setImageUploadBusy] = useState(false);
  const [scoredDraft, setScoredDraft] = useState(draft);
  const [slugEdited, setSlugEdited] = useState(false);
  const dirty = !sameDraft(draft, savedSnapshot);
  const draftRef = useRef(draft);
  draftRef.current = draft;
  useEffect(() => {
    const timer = window.setTimeout(() => setScoredDraft(draft), 300);
    return () => window.clearTimeout(timer);
  }, [draft]);
  const [brief, setBrief] = useState<SearchBrief | null>(null);
  // A brief scores only the topic it was built for.
  const scoredTopic = primaryTargetTopic(scoredDraft.targetKeyword);
  const activeBrief =
    brief && briefMatchesTopic(brief, scoredTopic) ? brief : null;
  const scoreDraft = useMemo(
    () => ({ ...scoredDraft, targetKeyword: scoredTopic }),
    [scoredDraft, scoredTopic],
  );
  const scoringBrief = useMemo(
    () => (activeBrief ? toScoringBrief(activeBrief) : undefined),
    [activeBrief],
  );
  const seo = useMemo(
    () => scoreContent("seo", scoreDraft, scoringBrief),
    [scoreDraft, scoringBrief],
  );
  const aeo = useMemo(
    () => scoreContent("aeo", scoreDraft, scoringBrief),
    [scoreDraft, scoringBrief],
  );

  useEffect(() => {
    if (draftsQuery.isPending || initialized || !draftsQuery.data) return;
    setInitialized(true);
  }, [draftsQuery.data, draftsQuery.isPending, initialized]);

  const restoreDraft = useCallback((saved?: ContentDraft) => {
    const snapshot = saved ? asInput(saved) : blankDraft;
    setDraftId(saved?.id);
    setDraft(snapshot);
    setSavedSnapshot(snapshot);
    setSlugEdited(snapshot.slug !== slugFromTitle(snapshot.title));
  }, []);

  const saveMutation = useMutation({
    mutationFn: (sent: ContentDraftInput) =>
      saveContentDraft({
        data: { ...sent, projectId, ...(draftId ? { id: draftId } : {}) },
      }),
    onSuccess: async (saved, sent) => {
      // The writer keeps typing while an autosave is in flight: take the
      // server copy only when nothing changed since it was sent.
      const snapshot = asInput(saved);
      if (draftAfterSave(draftRef.current, sent, snapshot) === snapshot) {
        restoreDraft(saved);
      } else {
        setDraftId(saved.id);
        setSavedSnapshot(snapshot);
      }
      await queryClient.cancelQueries({
        queryKey: contentEditorQueryKeys.drafts(projectId),
        exact: true,
      });
      queryClient.setQueryData(
        contentEditorQueryKeys.draft(projectId, saved.id),
        saved,
      );
      const draftListKey = contentEditorQueryKeys.drafts(projectId);
      const currentDrafts =
        queryClient.getQueryData<ContentDraftSummary[]>(draftListKey);
      if (currentDrafts) {
        const summary: ContentDraftSummary = {
          id: saved.id,
          projectId: saved.projectId,
          title: saved.title,
          targetKeyword: saved.targetKeyword,
          updatedAt: saved.updatedAt,
        };
        queryClient.setQueryData(
          draftListKey,
          sortBy(
            [summary, ...currentDrafts.filter((item) => item.id !== saved.id)],
            [(item) => item.updatedAt, "desc"],
          ),
        );
      } else {
        void queryClient.invalidateQueries({
          queryKey: draftListKey,
          exact: true,
        });
      }
      onDraftIdChange(saved.id);
    },
    onError: (error) =>
      toast.error(
        error instanceof Error
          ? error.message
          : "Could not save this draft. Your writing is still here.",
      ),
  });
  const saveDraft = saveMutation.mutate;

  useEffect(() => {
    if (!initialized || !dirty || saveMutation.isPending) return;
    if (saveMutation.isError && saveMutation.variables === draft) return;
    if (
      (!draft.title.trim() && !draft.content.trim()) ||
      !isArticleContentWithinLimit(draft.content)
    )
      return;
    const timer = window.setTimeout(() => saveDraft(draft), 3000);
    return () => window.clearTimeout(timer);
  }, [
    draft,
    dirty,
    initialized,
    saveDraft,
    saveMutation.isPending,
    saveMutation.isError,
    saveMutation.variables,
  ]);

  const deleteMutation = useMutation({
    mutationFn: (id: string) =>
      deleteContentDraft({ data: { projectId, draftId: id } }),
    onSuccess: async (_, id) => {
      await queryClient.cancelQueries({
        queryKey: contentEditorQueryKeys.drafts(projectId),
        exact: true,
      });
      if (id === draftId) {
        restoreDraft();
        onDraftIdChange(undefined);
      }
      queryClient.removeQueries({
        queryKey: contentEditorQueryKeys.draft(projectId, id),
        exact: true,
      });
      const draftListKey = contentEditorQueryKeys.drafts(projectId);
      const currentDrafts =
        queryClient.getQueryData<ContentDraftSummary[]>(draftListKey);
      if (currentDrafts) {
        queryClient.setQueryData(
          draftListKey,
          currentDrafts.filter((item) => item.id !== id),
        );
      } else {
        void queryClient.invalidateQueries({
          queryKey: draftListKey,
          exact: true,
        });
      }
      toast.success("Draft deleted");
    },
    onError: (error) =>
      toast.error(
        error instanceof Error ? error.message : "Could not delete this draft.",
      ),
  });

  const openMutation = useMutation({
    mutationFn: (id: string) =>
      queryClient.fetchQuery({
        queryKey: contentEditorQueryKeys.draft(projectId, id),
        queryFn: () => getContentDraft({ data: { projectId, draftId: id } }),
      }),
    onSuccess: (saved) => {
      restoreDraft(saved);
      onDraftIdChange(saved.id);
    },
    onError: (error) =>
      toast.error(
        error instanceof Error ? error.message : "Could not open this draft.",
      ),
  });
  const linkedDraft = useLinkedDraft({
    projectId,
    selectedDraftId,
    currentDraftId: draftId,
    dirty,
    onRestore: restoreDraft,
  });
  // Saving never locks the writing fields: an autosave runs every few
  // seconds, and locking them dropped focus and the keys typed meanwhile.
  const writingLocked =
    openMutation.isPending || deleteMutation.isPending || imageUploadBusy;
  const editorBusy = saveMutation.isPending || writingLocked;

  const startNewDraft = () => {
    if (
      dirty &&
      !window.confirm("Discard your unsaved changes and start a new draft?")
    )
      return;
    restoreDraft();
    setInitialized(true);
    onDraftIdChange(undefined);
  };

  const selectDraft = (saved: ContentDraftSummary) => {
    if (saved.id === draftId) return;
    if (
      dirty &&
      !window.confirm("Discard your unsaved changes and open this draft?")
    )
      return;
    openMutation.mutate(saved.id);
  };

  const update = (field: keyof ContentDraftInput, value: string) => {
    if (field === "slug") setSlugEdited(true);
    setDraft((current) => {
      if (field === "title" && !slugEdited) {
        const slug = slugFromTitle(value);
        return { ...current, title: value, slug };
      }
      return { ...current, [field]: value };
    });
  };
  if (selectedDraftId && selectedDraftId !== draftId) {
    return (
      <DraftLoadingState
        error={linkedDraft.error}
        onRetry={() => void linkedDraft.refetch()}
        onNew={startNewDraft}
      />
    );
  }

  return (
    <FreeSeoEditorProvider onDraftFieldChange={update}>
      <div className="space-y-5">
        <DraftToolbar
          draft={draft}
          saving={saveMutation.isPending}
          dirty={dirty}
          saved={Boolean(draftId)}
          canSave={
            !editorBusy &&
            isArticleContentWithinLimit(draft.content) &&
            Boolean(draft.title.trim() || draft.content.trim())
          }
          onPreview={() => setPreviewOpen(true)}
          onSave={() =>
            saveMutation.mutate(draft, {
              onSuccess: () => toast.success("Draft saved"),
            })
          }
          onPublish={() => setTab("publish")}
        />

        <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
          <section className="min-w-0 space-y-5">
            <WriterReminder key={projectId} projectId={projectId} />
            <ArticleComposer
              draft={draft}
              projectId={projectId}
              projectDomain={projectDomain}
              onTopicLink={
                primaryTargetTopic(draft.targetKeyword)
                  ? `/p/${projectId}/keywords?q=${encodeURIComponent(primaryTargetTopic(draft.targetKeyword))}`
                  : undefined
              }
              disabled={writingLocked}
              onImageUploadStateChange={setImageUploadBusy}
              onChange={update}
              previewOpen={previewOpen}
              onPreviewOpenChange={setPreviewOpen}
            />
          </section>

          <ContentEditorSidePanel
            projectId={projectId}
            draft={draft}
            draftId={draftId}
            dirty={dirty}
            mode={mode}
            onModeChange={setMode}
            brief={brief}
            onBrief={setBrief}
            tab={tab}
            onTabChange={setTab}
            seo={seo}
            aeo={aeo}
            editorBusy={editorBusy}
            drafts={draftsQuery.data ?? []}
            draftsLoading={draftsQuery.isPending}
            draftsError={draftsQuery.isError}
            onNewDraft={startNewDraft}
            onSelectDraft={selectDraft}
            onDeleteDraft={(saved) => {
              const unsaved =
                saved.id === draftId && dirty
                  ? " Your unsaved edits will also be discarded."
                  : "";
              if (
                window.confirm(
                  `Delete "${saved.title || "Untitled draft"}"? This cannot be undone.${unsaved}`,
                )
              ) {
                deleteMutation.mutate(saved.id);
              }
            }}
          />
        </div>
      </div>
    </FreeSeoEditorProvider>
  );
}
