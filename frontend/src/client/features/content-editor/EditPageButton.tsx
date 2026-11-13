import { useMutation } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { PenLine } from "lucide-react";
import { Button } from "@/client/components/ui/button";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import { importPageDraft } from "./contentEditorApi";

/**
 * One click from any page list to that page's content in the editor. The
 * first click copies the page into a draft; later clicks reopen the draft.
 */
export function EditPageButton({
  projectId,
  url,
  growth = false,
  keyword,
  size = "sm",
  variant = "outline",
  label = "Edit in Content Editor",
  onBeforeOpen,
}: {
  projectId: string;
  url: string;
  /** Opens the editor with the Search Growth context panel. */
  growth?: boolean;
  /** The target keyword the editor's research starts from. */
  keyword?: string;
  size?: "sm" | "default";
  variant?: "outline" | "ghost" | "default";
  label?: string;
  /** Runs first; returning false cancels the click. */
  onBeforeOpen?: () => boolean;
}) {
  const navigate = useNavigate();
  const open = useMutation({
    mutationFn: () => importPageDraft(projectId, url),
    onSuccess: (draft) =>
      navigate({
        to: "/p/$projectId/content-editor",
        params: { projectId },
        search: {
          draft: draft.id,
          ...(growth ? { growth: "1" as const } : {}),
          ...(keyword ? { keyword } : {}),
        },
      }),
  });
  return (
    <span className="inline-flex flex-col items-start gap-1">
      <Button
        size={size}
        variant={variant}
        disabled={open.isPending}
        aria-label={`${label}: ${url}`}
        onClick={() => {
          if (onBeforeOpen && !onBeforeOpen()) return;
          open.mutate();
        }}
      >
        <PenLine data-icon="inline-start" aria-hidden />
        {open.isPending ? "Opening…" : label}
      </Button>
      {open.isError ? (
        <span role="alert" className="text-xs text-destructive">
          {getStandardErrorMessage(open.error)}
        </span>
      ) : null}
    </span>
  );
}
