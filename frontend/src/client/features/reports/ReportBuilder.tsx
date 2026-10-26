import { useRef, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useBlocker, useNavigate } from "@tanstack/react-router";
import { ArrowLeft, Check, Eye, Save } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/client/components/ui/button";
import { ConfirmDialog } from "@/client/components/ConfirmDialog";
import { QueryState } from "@/client/components/QueryState";
import { rangeForDays } from "@/client/features/analytics/analyticsApi";
import { captureClientEvent } from "@/client/lib/posthog";
import { getReportDefinition, saveReport } from "./reportApi";
import {
  createReportVisual,
  updateReportVisual,
  reportDefinitionSchema,
  type ReportDefinition,
  type ReportDataset,
  type ReportVisual,
} from "./reportDefinition";
import { buildVisualReportHtml } from "./reportDocument";
import { loadReportSource } from "./reportSources";
import { reportQueryKey, reportsQueryKey } from "./shared";
import { ReportCanvas } from "./ReportCanvas";
import { ReportDataPanel } from "./ReportDataPanel";
import { ReportVisualSettings } from "./ReportVisualSettings";

export function ReportBuilder({
  projectId,
  reportId,
}: {
  projectId: string;
  reportId?: string;
}) {
  const query = useQuery({
    queryKey: ["report-definition", projectId, reportId],
    queryFn: () => getReportDefinition(projectId, reportId!),
    enabled: Boolean(reportId),
    retry: false,
    staleTime: 0,
  });
  if (!reportId) return <BuilderEditor key={projectId} projectId={projectId} />;
  return (
    <div className="p-4">
      <QueryState
        query={query}
        errorFallback="Could not open this report in the builder"
      >
        {(definition) => (
          <BuilderEditor
            key={`${projectId}-${reportId}`}
            projectId={projectId}
            reportId={reportId}
            initial={definition}
          />
        )}
      </QueryState>
    </div>
  );
}

function BuilderEditor({
  projectId,
  reportId,
  initial,
}: {
  projectId: string;
  reportId?: string;
  initial?: ReportDefinition;
}) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [report, setReport] = useState<ReportDefinition>(() => {
    const range = rangeForDays(30, new Date());
    return (
      initial ?? {
        version: 1,
        title: "Untitled report",
        summary: "",
        from: range.from,
        to: range.to,
        datasets: [],
        visuals: [],
      }
    );
  });
  const [selectedId, setSelectedId] = useState(initial?.visuals[0]?.id ?? "");
  const [activeDataId, setActiveDataId] = useState(
    initial?.datasets[0]?.id ?? "",
  );
  const [preview, setPreview] = useState(false);
  const [source, setSource] = useState("traffic");
  const [importRange, setImportRange] = useState({
    from: report.from,
    to: report.to,
  });
  const [dirty, setDirty] = useState(false);
  const saved = useRef(false);
  const fileInput = useRef<HTMLInputElement>(null);
  const [error, setError] = useState("");
  const blocker = useBlocker({
    shouldBlockFn: () => dirty && !saved.current,
    enableBeforeUnload: () => dirty && !saved.current,
    withResolver: true,
  });
  const selected = report.visuals.find((visual) => visual.id === selectedId);
  const activeData =
    report.datasets.find((data) => data.id === activeDataId) ??
    report.datasets[0];
  const selectedData = report.datasets.find(
    (data) => data.id === selected?.datasetId,
  );
  const change = (update: Partial<ReportDefinition>) => {
    setReport((current) => ({ ...current, ...update }));
    setDirty(true);
    setError("");
  };
  const updateVisual = (update: Partial<ReportVisual>) =>
    change({
      visuals: report.visuals.map((visual) =>
        visual.id === selectedId ? updateReportVisual(visual, update) : visual,
      ),
    });
  const addDataset = (dataset: ReportDataset) => {
    if (report.datasets.length >= 6)
      throw new Error(
        "Use up to 6 data sources per report. Remove an unused source first.",
      );
    if (!dataset.rows.length)
      throw new Error(
        "No data for this period. Choose another date range, check the connection, or upload a CSV.",
      );
    const next = reportDefinitionSchema.safeParse({
      ...report,
      datasets: [...report.datasets, dataset],
    });
    if (!next.success) throw new Error(next.error.issues[0].message);
    change({ datasets: next.data.datasets });
    setActiveDataId(dataset.id);
    toast.success("Data added. Choose a visual to start building.");
  };
  const importMutation = useMutation({
    mutationFn: () => {
      if (
        !importRange.from ||
        !importRange.to ||
        importRange.from > importRange.to
      )
        throw new Error("Choose a valid start and end date.");
      const days =
        (Date.parse(importRange.to) - Date.parse(importRange.from)) /
        86_400_000;
      if (!Number.isFinite(days) || days > 365)
        throw new Error("Import up to 366 days at a time.");
      return queryClient.fetchQuery({
        queryKey: [
          "report-source",
          projectId,
          source,
          importRange.from,
          importRange.to,
        ],
        queryFn: () =>
          loadReportSource(projectId, source, importRange.from, importRange.to),
        retry: false,
        staleTime: 60_000,
      });
    },
    retry: false,
    onSuccess: (data) => {
      try {
        addDataset({ ...data, id: crypto.randomUUID() });
      } catch (reason) {
        setError(
          reason instanceof Error ? reason.message : "Could not add data.",
        );
      }
    },
    onError: (reason) => setError(reason.message),
  });
  const saveMutation = useMutation({
    mutationFn: async () => {
      const html = buildVisualReportHtml(report);
      return saveReport({
        data: {
          projectId,
          reportId,
          title: report.title.trim(),
          summary:
            report.summary.trim() ||
            `${report.visuals.length} visuals · ${report.datasets.length} data sources`,
          html,
          skill: "Visual report",
          definition: report,
        },
      });
    },
    retry: false,
    onSuccess: async (result) => {
      saved.current = true;
      setDirty(false);
      captureClientEvent("report:created", {
        project_id: projectId,
        report_id: result.reportId,
        source: "visual-builder",
        updated: Boolean(reportId),
      });
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: reportsQueryKey(projectId) }),
        queryClient.invalidateQueries({
          queryKey: reportQueryKey(projectId, result.reportId),
        }),
        queryClient.invalidateQueries({
          queryKey: ["report-definition", projectId, result.reportId],
        }),
      ]);
      toast.success("Report saved. Share or export it from the report viewer.");
      void navigate({
        to: "/p/$projectId/reports/$reportId",
        params: { projectId, reportId: result.reportId },
      });
    },
    onError: (reason) => setError(reason.message),
  });

  function addVisual(type: ReportVisual["type"]) {
    if (report.visuals.length >= 24) return;
    if (type !== "text" && !activeData) {
      setError("Add a data source first, then choose a chart.");
      return;
    }
    const visual = createReportVisual(type, activeData);
    change({ visuals: [...report.visuals, visual] });
    setSelectedId(visual.id);
  }

  function moveVisual(direction: number) {
    const visuals = [...report.visuals];
    const index = visuals.findIndex((visual) => visual.id === selectedId);
    const next = index + direction;
    if (index < 0 || next < 0 || next >= visuals.length) return;
    [visuals[index], visuals[next]] = [visuals[next], visuals[index]];
    change({ visuals });
  }

  const close = () =>
    void navigate({
      to: "/p/$projectId/reports",
      params: { projectId },
      search: {},
    });
  const saving = saveMutation.isPending;
  return (
    <div className="flex min-h-full flex-col bg-muted/25">
      <header className="flex flex-wrap items-center justify-between gap-3 border-b border-border bg-background px-4 py-3">
        <div className="flex min-w-0 items-center gap-3">
          <Button
            variant="ghost"
            size="icon"
            aria-label="Back to reports"
            onClick={close}
            disabled={saving}
          >
            <ArrowLeft />
          </Button>
          <div>
            <h1 className="text-base font-semibold">Report builder</h1>
            <p className="flex items-center gap-1 text-xs text-muted-foreground">
              {dirty ? (
                "Unsaved changes"
              ) : (
                <>
                  <Check className="size-3" />
                  {reportId ? "Saved report" : "New report"}
                </>
              )}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            disabled={!report.visuals.length}
            onClick={() => setPreview(!preview)}
          >
            <Eye />
            {preview ? "Back to editor" : "Preview"}
          </Button>
          <Button
            onClick={() => saveMutation.mutate()}
            disabled={
              saving ||
              importMutation.isPending ||
              !report.visuals.length ||
              !report.title.trim()
            }
          >
            <Save />
            {saving ? "Saving…" : reportId ? "Save changes" : "Save report"}
          </Button>
        </div>
      </header>
      {error && (
        <div
          role="alert"
          className="border-b border-destructive/20 bg-destructive/5 px-5 py-3 text-sm text-destructive"
        >
          {error}
        </div>
      )}
      <fieldset disabled={saving} className="m-0 min-w-0 flex-1 border-0 p-0">
        <legend className="sr-only">Report editor</legend>
        <div
          className={`grid min-w-0 ${preview ? "" : "xl:grid-cols-[240px_minmax(0,1fr)_260px]"}`}
        >
          {!preview && (
            <ReportDataPanel
              report={report}
              source={source}
              setSource={setSource}
              importRange={importRange}
              setImportRange={setImportRange}
              importing={importMutation.isPending}
              onImport={() => {
                setError("");
                importMutation.mutate();
              }}
              fileInput={fileInput}
              addDataset={addDataset}
              setError={setError}
              activeData={activeData}
              setActiveDataId={setActiveDataId}
              change={change}
            />
          )}
          <ReportCanvas
            report={report}
            preview={preview}
            selectedId={selectedId}
            setSelectedId={setSelectedId}
            addVisual={addVisual}
            change={change}
            fileInput={fileInput}
          />
          {!preview && (
            <ReportVisualSettings
              selected={selected}
              selectedData={selectedData}
              activeData={activeData}
              datasets={report.datasets}
              updateVisual={updateVisual}
              moveVisual={moveVisual}
              canMoveUp={report.visuals[0]?.id !== selectedId}
              canMoveDown={report.visuals.at(-1)?.id !== selectedId}
              onRemove={() => {
                change({
                  visuals: report.visuals.filter(
                    (visual) => visual.id !== selectedId,
                  ),
                });
                setSelectedId("");
              }}
            />
          )}
        </div>
      </fieldset>
      {blocker.status === "blocked" && (
        <ConfirmDialog
          title="Leave this report?"
          confirmLabel="Discard changes"
          destructive
          onConfirm={blocker.proceed}
          onClose={blocker.reset}
        >
          Your unsaved changes will be lost. Save your report to keep its data
          and layout.
        </ConfirmDialog>
      )}
    </div>
  );
}
