import { QueryError } from "@/client/components/QueryState";
import { Monitor, Plus, Save, Send, Trash2, X } from "lucide-react";
import { useLayoutEffect, useRef, useState, type ReactNode } from "react";
import { Button } from "@/client/components/ui/button";
import type { ContentDraftSummary } from "./contentEditorApi";
import type { ContentDraftInput } from "./scoring";
import {
  articleWordCount,
  titleForDraft,
  searchPreviewTitle,
  searchPreviewUrl,
} from "./editor-display";

const reminderStorageKey = (projectId: string) =>
  `uplumo:content-editor:writer-reminder:${projectId}`;

export function WriterReminder({ projectId }: { projectId: string }) {
  const [dismissed, setDismissed] = useState(() => {
    if (typeof window === "undefined") return false;
    try {
      return (
        window.sessionStorage.getItem(reminderStorageKey(projectId)) ===
        "dismissed"
      );
    } catch {
      return false;
    }
  });

  if (dismissed) return null;

  return (
    <aside className="flex items-start justify-between gap-3 rounded-lg border border-primary/20 bg-primary/5 px-3 py-2.5">
      <p className="text-sm leading-5 text-muted-foreground">
        <span className="font-medium text-foreground">
          Use your own expertise.
        </span>{" "}
        Let AI help with the draft, add your real examples, and verify every
        claim.
      </p>
      <Button
        type="button"
        variant="ghost"
        size="icon-sm"
        aria-label="Dismiss writing reminder"
        onClick={() => {
          setDismissed(true);
          try {
            window.sessionStorage.setItem(
              reminderStorageKey(projectId),
              "dismissed",
            );
          } catch {
            // Dismissal still works for this render if storage is unavailable.
          }
        }}
      >
        <X aria-hidden />
      </Button>
    </aside>
  );
}

export function SearchPreview({
  draft,
  domain,
}: {
  draft: ContentDraftInput;
  domain: string;
}) {
  const title = searchPreviewTitle(draft);
  const url = searchPreviewUrl(domain, draft.slug);
  return (
    <div>
      <div className="rounded-lg border bg-background p-4">
        <p className="truncate text-xs text-muted-foreground">{url}</p>
        <p className="mt-1 line-clamp-2 text-lg leading-6 text-sky-700 dark:text-sky-400">
          {title || "Your page title will appear here"}
        </p>
        <p className="mt-1 line-clamp-2 text-sm leading-5 text-muted-foreground">
          {draft.metaDescription ||
            "Add a search description to preview how your page may appear in search results."}
        </p>
      </div>
      <p className="mt-2 text-xs text-muted-foreground">
        Search engines may choose a different title or snippet.
      </p>
    </div>
  );
}

export function DraftsPanel({
  drafts,
  isLoading,
  hasError,
  currentDraftId,
  isBusy,
  onNew,
  onSelect,
  onDelete,
}: {
  drafts: ContentDraftSummary[];
  isLoading: boolean;
  hasError: boolean;
  currentDraftId?: string;
  isBusy: boolean;
  onNew: () => void;
  onSelect: (draft: ContentDraftSummary) => void;
  onDelete: (draft: ContentDraftSummary) => void;
}) {
  return (
    <section className="rounded-xl border bg-card p-4">
      {/* The surrounding <summary> already names this list. */}
      <div className="mb-3 flex justify-end">
        <Button variant="outline" size="sm" disabled={isBusy} onClick={onNew}>
          <Plus aria-hidden />
          New
        </Button>
      </div>
      {isLoading ? (
        <p className="text-sm text-muted-foreground">Loading drafts…</p>
      ) : null}
      {hasError ? (
        <p className="text-sm text-destructive">Could not load saved drafts.</p>
      ) : null}
      {drafts.length === 0 && !isLoading ? (
        <p className="text-sm text-muted-foreground">
          Saved drafts will appear here.
        </p>
      ) : null}
      <ul className="space-y-1">
        {drafts.map((saved) => (
          <li
            key={saved.id}
            className={`flex items-center gap-2 rounded-lg px-2 py-2 ${saved.id === currentDraftId ? "bg-muted" : ""}`}
          >
            <button
              type="button"
              disabled={isBusy}
              onClick={() => onSelect(saved)}
              className="min-w-0 flex-1 text-left disabled:opacity-60"
            >
              <span className="block truncate text-sm">
                {saved.title || "Untitled draft"}
              </span>
              <span className="block text-xs text-muted-foreground">
                {new Date(saved.updatedAt).toLocaleDateString()}
              </span>
            </button>
            <Button
              variant="ghost"
              size="icon-sm"
              aria-label={`Delete ${saved.title || "untitled draft"}`}
              disabled={isBusy}
              onClick={() => onDelete(saved)}
            >
              <Trash2 aria-hidden />
            </Button>
          </li>
        ))}
      </ul>
    </section>
  );
}

export function Field({
  id,
  label,
  hint,
  children,
}: {
  id: string;
  label: string;
  hint: string;
  children: ReactNode;
}) {
  return (
    <div className="block space-y-2">
      <label
        htmlFor={id}
        className="flex items-baseline justify-between gap-2 text-sm font-medium"
      >
        {label}
        <span className="text-xs font-normal text-muted-foreground">
          {hint}
        </span>
      </label>
      {children}
    </div>
  );
}

export function DraftToolbar({
  draft,
  saving,
  dirty,
  saved,
  canSave,
  onPreview,
  onSave,
  onPublish,
}: {
  draft: ContentDraftInput;
  saving: boolean;
  dirty: boolean;
  saved: boolean;
  canSave: boolean;
  onPreview: () => void;
  onSave: () => void;
  onPublish: () => void;
}) {
  const words = articleWordCount(draft.content);
  const readMinutes = Math.ceil(words / 200);
  const ref = useRef<HTMLDivElement>(null);
  // The formatting toolbar sticks just below this bar, whose height changes
  // when its buttons wrap on narrow screens.
  useLayoutEffect(() => {
    const bar = ref.current;
    const host = bar?.parentElement;
    if (!bar || !host) return;
    const observer = new ResizeObserver(() =>
      host.style.setProperty("--draft-toolbar-h", `${bar.offsetHeight}px`),
    );
    observer.observe(bar);
    return () => observer.disconnect();
  }, []);
  return (
    // Negative top cancels the page's scroll padding, so the bar sits flush
    // under the header and no content shows above it.
    <div
      ref={ref}
      className="sticky -top-4 z-20 -mx-1 flex flex-wrap items-center gap-3 rounded-xl border bg-background px-4 py-3 md:-top-6"
    >
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{titleForDraft(draft)}</p>
        <p className="text-xs text-muted-foreground">
          {words.toLocaleString()} words ·{" "}
          {words > 0 ? `about ${readMinutes} min read · ` : ""}
          {saving
            ? "Saving…"
            : dirty
              ? "Unsaved changes"
              : saved
                ? "Saved"
                : "New draft"}
        </p>
      </div>
      <Button variant="outline" onClick={onPreview}>
        <Monitor aria-hidden />
        Preview as on website
      </Button>
      <Button variant="outline" onClick={onSave} disabled={!canSave}>
        <Save aria-hidden />
        {saving ? "Saving…" : "Save draft"}
      </Button>
      <Button onClick={onPublish}>
        <Send aria-hidden />
        Publish
      </Button>
    </div>
  );
}

export function DraftLoadingState({
  error,
  onRetry,
  onNew,
}: {
  error: Error | null;
  onRetry: () => void;
  onNew: () => void;
}) {
  if (!error)
    return (
      <p role="status" aria-busy="true">
        Opening saved draft…
      </p>
    );
  return (
    <div className="space-y-3">
      <QueryError
        error={error}
        fallback="Could not open this saved draft."
        onRetry={onRetry}
      />
      <Button variant="outline" onClick={onNew}>
        Start a new draft
      </Button>
    </div>
  );
}
