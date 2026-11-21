import { useMutation } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { Button } from "@/client/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/client/components/ui/dialog";
import {
  saveReport,
  shareReport,
  type SiteReportPreview,
} from "@/client/features/reports/reportApi";
import {
  indexReportDraft,
  renderReportDraft,
} from "@/client/features/reports/siteReportDraft";
import { REPORT_IFRAME_SANDBOX, reportCsp } from "@/shared/report-sandbox";

type SavedReport = {
  reportId: string;
  shareToken: string | null;
  shareFailed: boolean;
};

const previewCsp = reportCsp()
  .split("; ")
  .filter((directive) => !/^(sandbox|frame-ancestors)\b/.test(directive))
  .join("; ");

export function SiteReportPreviewDialog({
  projectId,
  title,
  draft,
  onBack,
  onClose,
  onSaved,
}: {
  projectId: string;
  title: string;
  draft: SiteReportPreview;
  onBack: () => void;
  onClose: () => void;
  onSaved: (report: SavedReport) => void;
}) {
  const indexed = useMemo(() => indexReportDraft(draft.html), [draft.html]);
  const [excluded, setExcluded] = useState<ReadonlySet<string>>(
    () => new Set(),
  );
  const [share, setShare] = useState(false);
  const html = useMemo(
    () => renderReportDraft(indexed.html, excluded),
    [indexed.html, excluded],
  );
  const preview = html.replace(
    "<head>",
    `<head><meta http-equiv="Content-Security-Policy" content="${previewCsp}">`,
  );
  const save = useMutation({
    retry: false,
    meta: { errorToast: false },
    mutationFn: async () => {
      const { reportId } = await saveReport({
        data: { projectId, title, summary: title, html, skill: "Site report" },
      });
      if (share) {
        try {
          const { shareToken } = await shareReport({
            data: { projectId, reportId },
          });
          return { reportId, shareToken, shareFailed: !shareToken };
        } catch {
          return { reportId, shareToken: null, shareFailed: true };
        }
      }
      return { reportId, shareToken: null, shareFailed: false };
    },
    onSuccess: onSaved,
  });
  const toggle = (id: string, include: boolean) => {
    setExcluded((previous) => {
      const next = new Set(previous);
      if (include) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !save.isPending) onClose();
      }}
    >
      <DialogContent className="max-h-[94vh] overflow-y-auto sm:max-w-[min(95vw,1350px)]">
        <DialogHeader>
          <DialogTitle>Preview client report</DialogTitle>
          <DialogDescription>{title}</DialogDescription>
        </DialogHeader>
        {draft.skipped.length > 0 && (
          <details className="text-sm text-muted-foreground">
            <summary>{draft.skipped.length} sections unavailable</summary>
            <ul className="list-disc pl-5">
              {draft.skipped.map(({ section, reason }) => (
                <li key={section}>{reason}</li>
              ))}
            </ul>
          </details>
        )}
        <div className="grid min-w-0 gap-4 lg:grid-cols-[18rem_minmax(0,1fr)]">
          <fieldset
            disabled={save.isPending}
            className="max-h-[32vh] min-w-0 space-y-1 overflow-y-auto rounded-lg border p-3 lg:max-h-[66vh]"
          >
            <legend className="px-1 text-sm font-medium">
              Included content
            </legend>
            {indexed.groups.map((group) => {
              const { views } = group;
              const mode =
                views &&
                (excluded.has(views.table)
                  ? "chart"
                  : excluded.has(views.chart)
                    ? "table"
                    : "both");
              return (
                <details key={group.id} className="rounded border p-2 text-sm">
                  <summary className="cursor-pointer">
                    {group.section} · {group.label}
                  </summary>
                  {views && (
                    <label className="mt-2 block text-sm">
                      Display
                      <select
                        aria-label="Display"
                        className="ml-2 rounded border bg-background p-1"
                        value={mode}
                        onChange={(event) => {
                          const next = event.target.value;
                          toggle(views.chart, next !== "table");
                          toggle(views.table, next !== "chart");
                        }}
                      >
                        <option value="both">Chart and table</option>
                        <option value="chart">Chart only</option>
                        <option value="table">Table only</option>
                      </select>
                    </label>
                  )}
                  <div className="mt-2 space-y-1">
                    {group.items.map(({ id, label }) => (
                      <label
                        key={id}
                        className="flex items-start gap-2 break-words"
                      >
                        <input
                          type="checkbox"
                          checked={!excluded.has(id)}
                          onChange={(event) => toggle(id, event.target.checked)}
                          className="mt-1"
                        />
                        <span>{label}</span>
                      </label>
                    ))}
                  </div>
                </details>
              );
            })}
          </fieldset>
          <iframe
            className="h-[58vh] w-full rounded-lg border bg-background lg:h-[66vh]"
            title="Client report preview"
            sandbox={REPORT_IFRAME_SANDBOX}
            srcDoc={preview}
          />
        </div>
        {save.isError && (
          <p role="alert" className="text-sm text-destructive">
            {save.error.message}
          </p>
        )}
        <DialogFooter>
          <label className="mr-auto flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={share}
              disabled={save.isPending}
              onChange={(event) => setShare(event.target.checked)}
            />
            Create a link anyone can view
          </label>
          <Button
            type="button"
            variant="ghost"
            disabled={save.isPending}
            onClick={onBack}
          >
            Back to settings
          </Button>
          <Button
            type="button"
            disabled={save.isPending}
            onClick={() => save.mutate()}
          >
            {save.isPending
              ? "Saving…"
              : share
                ? "Save and create link"
                : "Save privately"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
