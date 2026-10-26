import type { RefObject, Dispatch, SetStateAction } from "react";
import { Database, FileSpreadsheet, Plus, Trash2, Upload } from "lucide-react";
import { Button } from "@/client/components/ui/button";
import { Input } from "@/client/components/ui/input";
import {
  parseReportCsv,
  type ReportDefinition,
  type ReportDataset,
} from "./reportDefinition";
import { REPORT_SOURCES } from "./reportSources";
import { FieldSelect } from "./ReportBuilderControls";

export function ReportDataPanel({
  report,
  source,
  setSource,
  importRange,
  setImportRange,
  importing,
  onImport,
  fileInput,
  addDataset,
  setError,
  activeData,
  setActiveDataId,
  change,
}: {
  report: ReportDefinition;
  source: string;
  setSource: (source: string) => void;
  importRange: { from: string; to: string };
  setImportRange: Dispatch<SetStateAction<{ from: string; to: string }>>;
  importing: boolean;
  onImport: () => void;
  fileInput: RefObject<HTMLInputElement | null>;
  addDataset: (data: ReportDataset) => void;
  setError: (error: string) => void;
  activeData: ReportDataset | undefined;
  setActiveDataId: (id: string) => void;
  change: (update: Partial<ReportDefinition>) => void;
}) {
  return (
    <aside
      aria-label="Report data"
      className="space-y-5 border-b border-border bg-background p-4 xl:border-r xl:border-b-0"
    >
      <div>
        <h2 className="flex items-center gap-2 text-sm font-semibold">
          <Database className="size-4" />
          Data sources
        </h2>
        <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
          Connect your numbers, then build your story.
        </p>
      </div>
      <div className="space-y-3">
        <FieldSelect
          label="Import from"
          value={source}
          options={REPORT_SOURCES.map((item) => ({
            value: item.id,
            label: item.name,
          }))}
          onChange={setSource}
        />
        <p className="text-xs text-muted-foreground">
          {REPORT_SOURCES.find((item) => item.id === source)?.description}
        </p>
        <div className="grid grid-cols-2 gap-2">
          <label className="space-y-1 text-xs">
            From
            <Input
              type="date"
              aria-label="Import start date"
              value={importRange.from}
              onChange={(event) =>
                setImportRange({
                  ...importRange,
                  from: event.target.value,
                })
              }
            />
          </label>
          <label className="space-y-1 text-xs">
            To
            <Input
              type="date"
              aria-label="Import end date"
              value={importRange.to}
              onChange={(event) =>
                setImportRange({
                  ...importRange,
                  to: event.target.value,
                })
              }
            />
          </label>
        </div>
        <Button
          className="w-full"
          variant="outline"
          disabled={importing || report.datasets.length >= 6}
          onClick={() => {
            onImport();
          }}
        >
          <Plus />
          {importing ? "Loading data…" : "Add project data"}
        </Button>
        <div className="relative py-1 text-center text-xs text-muted-foreground">
          or bring your own data
        </div>
        <input
          ref={fileInput}
          type="file"
          accept=".csv,text/csv"
          className="sr-only"
          aria-label="Upload CSV file"
          onChange={async (event) => {
            const file = event.target.files?.[0];
            event.target.value = "";
            if (!file) return;
            try {
              if (file.size > 200_000)
                throw new Error("Use a CSV smaller than 200 KB.");
              addDataset({
                id: crypto.randomUUID(),
                name: file.name.slice(0, 120),
                ...parseReportCsv(await file.text()),
              });
            } catch (reason) {
              setError(
                reason instanceof Error
                  ? reason.message
                  : "Could not read CSV.",
              );
            }
          }}
        />
        <Button
          className="w-full"
          variant="outline"
          disabled={report.datasets.length >= 6}
          onClick={() => fileInput.current?.click()}
        >
          <Upload />
          Upload CSV
        </Button>
        <p className="text-[11px] leading-relaxed text-muted-foreground">
          Up to 2,000 rows · 30 columns · 200 KB. Google sources need an
          existing connection. Imports are snapshots; charts use loaded data.
        </p>
      </div>
      {report.datasets.length > 0 && (
        <div className="space-y-2">
          <h3 className="text-xs font-medium text-muted-foreground">
            IN THIS REPORT · {report.datasets.length}/6
          </h3>
          {report.datasets.map((data) => (
            <div
              key={data.id}
              className={`rounded-md border ${activeData?.id === data.id ? "border-primary bg-primary/5" : "border-border"}`}
            >
              <button
                className="w-full p-3 text-left focus-visible:outline-2 focus-visible:outline-primary"
                onClick={() => setActiveDataId(data.id)}
                aria-pressed={activeData?.id === data.id}
              >
                <span className="flex items-start gap-2 text-xs font-medium">
                  <FileSpreadsheet className="mt-0.5 size-4 shrink-0" />
                  <span className="min-w-0 break-words">{data.name}</span>
                </span>
                <span className="mt-1 block text-[11px] text-muted-foreground">
                  {data.rows.length} rows · {data.columns.length} fields
                </span>
              </button>
              {activeData?.id === data.id && (
                <div className="space-y-3 border-t border-border px-3 py-2">
                  <div className="flex flex-wrap gap-1">
                    {data.columns.map((column) => (
                      <span
                        key={column}
                        className="rounded bg-muted px-1.5 py-1 text-[10px]"
                      >
                        {column}
                      </span>
                    ))}
                  </div>
                  <FieldSelect
                    label="Date field (for report filters)"
                    value={data.dateColumn}
                    options={[
                      { value: "", label: "None · fixed period" },
                      ...data.columns.map((column) => ({
                        value: column,
                        label: column,
                      })),
                    ]}
                    onChange={(dateColumn) =>
                      change({
                        datasets: report.datasets.map((item) =>
                          item.id === data.id ? { ...item, dateColumn } : item,
                        ),
                      })
                    }
                  />
                  <Button
                    variant="ghost"
                    size="sm"
                    className="w-full"
                    disabled={report.visuals.some(
                      (visual) => visual.datasetId === data.id,
                    )}
                    onClick={() =>
                      change({
                        datasets: report.datasets.filter(
                          (item) => item.id !== data.id,
                        ),
                      })
                    }
                  >
                    <Trash2 />
                    Remove unused data
                  </Button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </aside>
  );
}
