import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRef, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/client/components/ui/button";
import { Checkbox } from "@/client/components/ui/checkbox";
import { Textarea } from "@/client/components/ui/textarea";
import { downloadFile } from "@/client/lib/download";
import {
  articleBodyHTML,
  articleHTMLFile,
  articleMarkdownFile,
  exportFileName,
} from "./article-export";
import {
  contentEditorQueryKeys,
  getBlogTheme,
  MAX_THEME_CSS_BYTES,
  saveBlogTheme,
  type BlogTheme,
} from "./contentEditorApi";
import type { ContentDraftInput } from "./scoring";

/** The project's saved blog CSS; free, shared by the cards and the preview. */
export function useBlogTheme(projectId: string, enabled = true) {
  return useQuery({
    queryKey: contentEditorQueryKeys.blogTheme(projectId),
    queryFn: () => getBlogTheme(projectId),
    enabled,
    staleTime: Infinity,
  });
}

function sizeLabel(css: string) {
  return `${(new Blob([css]).size / 1024).toFixed(1)} KB`;
}

/** Copy or download the current draft for sites that are not WordPress. */
export function ExportCard({
  projectId,
  draft,
  schemas,
}: {
  projectId: string;
  draft: ContentDraftInput;
  // JSON-LD for the saved draft; undefined while unsaved edits would make
  // it disagree with the exported text.
  schemas: unknown[] | undefined;
}) {
  const theme = useBlogTheme(projectId);
  const [useTheme, setUseTheme] = useState(true);
  const css = theme.data?.css ?? "";
  const empty = !draft.content.trim();

  async function copyHTML() {
    try {
      await navigator.clipboard.writeText(
        await articleBodyHTML(draft.content, draft.title),
      );
      toast.success("HTML copied. Paste it into your site's editor.");
    } catch {
      toast.error("Could not copy. Allow clipboard access and try again.");
    }
  }

  async function downloadHTML() {
    const file = await articleHTMLFile({
      title: draft.title,
      description: draft.metaDescription,
      content: draft.content,
      css: useTheme ? css : "",
      schemas,
    });
    downloadFile(
      file,
      exportFileName(draft.slug, draft.title, "html"),
      "text/html",
    );
  }

  function downloadMarkdown() {
    downloadFile(
      articleMarkdownFile({
        title: draft.title,
        description: draft.metaDescription,
        slug: draft.slug,
        content: draft.content,
      }),
      exportFileName(draft.slug, draft.title, "md"),
      "text/markdown",
    );
  }

  return (
    <section className="space-y-3 rounded-lg border border-border bg-card p-4">
      <h2 className="text-sm font-medium">Export</h2>
      <p className="text-xs text-muted-foreground">
        For sites that are not WordPress. Upload the .html file with your host's
        file manager, or paste the HTML into any site editor.
      </p>
      <div className="flex flex-wrap gap-2">
        <Button type="button" size="sm" disabled={empty} onClick={copyHTML}>
          Copy HTML
        </Button>
        <Button
          type="button"
          size="sm"
          variant="outline"
          disabled={empty}
          onClick={downloadHTML}
        >
          Download .html
        </Button>
        <Button
          type="button"
          size="sm"
          variant="outline"
          disabled={empty}
          onClick={downloadMarkdown}
        >
          Download .md
        </Button>
      </div>
      {css ? (
        <label className="flex items-center gap-2 text-sm">
          <Checkbox
            checked={useTheme}
            onCheckedChange={(value) => setUseTheme(Boolean(value))}
          />
          Put my blog theme inside the .html file
        </label>
      ) : null}
      <p className="text-xs text-muted-foreground">
        {empty
          ? "Write the article to export it."
          : schemas
            ? "The .html file includes the title, meta description and structured data."
            : "Save the draft to include structured data in the .html file."}
      </p>
    </section>
  );
}

/** Paste or upload the CSS of the user's own blog, saved per project. */
export function BlogThemeCard({ projectId }: { projectId: string }) {
  const theme = useBlogTheme(projectId);
  return (
    <section className="space-y-3 rounded-lg border border-border bg-card p-4">
      <h2 className="flex items-center justify-between gap-2 text-sm font-medium">
        My blog theme
        {theme.data?.css ? (
          <span className="text-xs font-normal text-muted-foreground">
            Saved · {sizeLabel(theme.data.css)}
          </span>
        ) : null}
      </h2>
      <p className="text-xs text-muted-foreground">
        Paste your blog's CSS or upload a .css file. It is used in the article
        preview and the .html export. Max 100 KB.
      </p>
      {theme.isError ? (
        <p role="alert" className="text-xs text-destructive">
          Could not load your theme. Reload the page to try again.
        </p>
      ) : theme.data ? (
        <ThemeEditor
          // A fresh editor per saved version, so the box starts from it.
          key={theme.data.updatedAt ?? "none"}
          projectId={projectId}
          saved={theme.data}
        />
      ) : (
        <p className="text-xs text-muted-foreground">Loading…</p>
      )}
    </section>
  );
}

function ThemeEditor({
  projectId,
  saved,
}: {
  projectId: string;
  saved: BlogTheme;
}) {
  const queryClient = useQueryClient();
  const [css, setCSS] = useState(saved.css);
  const fileInput = useRef<HTMLInputElement>(null);
  const save = useMutation({
    mutationFn: (next: string) => saveBlogTheme({ projectId, css: next }),
    retry: false,
    onSuccess: (data) => {
      queryClient.setQueryData(
        contentEditorQueryKeys.blogTheme(projectId),
        data,
      );
      toast.success(
        data.css ? "Theme saved for this project." : "Theme removed.",
      );
    },
  });
  const tooBig = new Blob([css]).size > MAX_THEME_CSS_BYTES;
  const changed = css.trim() !== saved.css;

  async function upload(file: File | undefined) {
    if (!file) return;
    if (file.size > MAX_THEME_CSS_BYTES) {
      toast.error(
        "This file is over 100 KB. Remove rules your posts do not use.",
      );
      return;
    }
    setCSS(await file.text());
  }

  return (
    <div className="space-y-2">
      <Textarea
        aria-label="Your blog CSS"
        spellCheck={false}
        value={css}
        onChange={(event) => setCSS(event.target.value)}
        placeholder="body { font-family: Georgia, serif; } h2 { color: #7a2e0e; }"
        className="max-h-60 min-h-28 font-mono text-xs"
        aria-invalid={tooBig}
      />
      {tooBig ? (
        <p role="alert" className="text-xs text-destructive">
          This theme is over 100 KB. Remove rules your posts do not use.
        </p>
      ) : null}
      <input
        ref={fileInput}
        type="file"
        accept=".css,text/css"
        className="sr-only"
        tabIndex={-1}
        aria-hidden
        onChange={(event) => {
          void upload(event.target.files?.[0]);
          event.target.value = "";
        }}
      />
      <div className="flex flex-wrap gap-2">
        <Button
          type="button"
          size="sm"
          variant="outline"
          disabled={save.isPending}
          onClick={() => fileInput.current?.click()}
        >
          Upload .css
        </Button>
        <Button
          type="button"
          size="sm"
          disabled={!changed || !css.trim() || tooBig || save.isPending}
          onClick={() => save.mutate(css)}
        >
          {save.isPending && save.variables ? "Saving…" : "Save theme"}
        </Button>
        {saved.css ? (
          <Button
            type="button"
            size="sm"
            variant="ghost"
            className="text-destructive"
            disabled={save.isPending}
            onClick={() => save.mutate("")}
          >
            Remove
          </Button>
        ) : null}
      </div>
    </div>
  );
}
