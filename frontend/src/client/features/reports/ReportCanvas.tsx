import type { RefObject } from "react";
import { LayoutTemplate, Settings2, Type, Upload } from "lucide-react";
import { Button } from "@/client/components/ui/button";
import { Input } from "@/client/components/ui/input";
import { Textarea } from "@/client/components/ui/textarea";
import type { ReportDefinition, ReportVisual } from "./reportDefinition";
import { VisualContent } from "./reportDocument";
import { visualTypes } from "./ReportBuilderControls";

export function ReportCanvas({
  report,
  preview,
  selectedId,
  setSelectedId,
  addVisual,
  change,
  fileInput,
}: {
  report: ReportDefinition;
  preview: boolean;
  selectedId: string;
  setSelectedId: (id: string) => void;
  addVisual: (type: ReportVisual["type"]) => void;
  change: (update: Partial<ReportDefinition>) => void;
  fileInput: RefObject<HTMLInputElement | null>;
}) {
  return (
    <main aria-label="Report canvas" className="min-w-0 space-y-4 p-4 md:p-6">
      {!preview && (
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
              Report canvas
            </h2>
            <p className="mt-1 text-xs text-muted-foreground">
              {report.visuals.length}/24 visuals · Select a visual to configure
              it
            </p>
          </div>
          <div
            className="flex flex-wrap gap-1"
            role="group"
            aria-label="Add visual"
          >
            {visualTypes.map((item) => (
              <Button
                key={item.type}
                variant="outline"
                size="sm"
                disabled={report.visuals.length >= 24}
                onClick={() => addVisual(item.type)}
              >
                <item.icon />
                {item.label}
              </Button>
            ))}
          </div>
        </div>
      )}
      <div className="mx-auto min-h-[660px] max-w-5xl rounded-lg border border-border bg-white p-5 text-[#192c21] shadow-sm md:p-8">
        <div className="mb-6 border-b border-[#e0e7e1] pb-6">
          <p className="mb-3 text-[10px] font-medium tracking-[0.18em] text-[#52685b]">
            PERFORMANCE REPORT
          </p>
          {preview ? (
            <>
              <h2 className="break-words text-3xl font-semibold tracking-tight">
                {report.title}
              </h2>
              <p className="mt-3 whitespace-pre-wrap text-sm">
                {report.summary}
              </p>
            </>
          ) : (
            <>
              <Input
                aria-label="Report title"
                maxLength={120}
                value={report.title}
                onChange={(event) => change({ title: event.target.value })}
                className="h-auto border-0 bg-transparent p-0 text-2xl font-semibold tracking-tight shadow-none md:text-3xl"
              />
              <Textarea
                aria-label="Report summary"
                maxLength={2500}
                value={report.summary}
                onChange={(event) => change({ summary: event.target.value })}
                placeholder="What changed, why it matters, and what happens next…"
                rows={2}
                className="mt-3 min-h-14 border-0 bg-transparent p-0 text-sm shadow-none"
              />
            </>
          )}
          <div className="mt-4 flex flex-wrap items-center gap-2 text-xs text-[#52685b]">
            <span>Report dates</span>
            {preview ? (
              <span>
                {report.from || "All dates"} — {report.to || "All dates"}
              </span>
            ) : (
              <>
                <Input
                  aria-label="Report start date"
                  type="date"
                  value={report.from}
                  onChange={(event) => change({ from: event.target.value })}
                  className="h-8 w-36"
                />
                <span>to</span>
                <Input
                  aria-label="Report end date"
                  type="date"
                  value={report.to}
                  onChange={(event) => change({ to: event.target.value })}
                  className="h-8 w-36"
                />
              </>
            )}
          </div>
          <p className="mt-2 text-[11px] text-[#64746b]">
            Date filters apply to sources with a date field. Other sources keep
            their imported period.
          </p>
        </div>
        {!report.visuals.length ? (
          <div className="flex min-h-80 flex-col items-center justify-center text-center">
            <div className="mb-4 rounded-lg border border-dashed border-[#cddbd1] p-5">
              <LayoutTemplate className="size-9 text-[#789480]" />
            </div>
            <h3 className="text-lg font-medium">Your report starts here</h3>
            <p className="mt-2 max-w-xs text-sm leading-relaxed text-[#64746b]">
              Add project data or upload a CSV, then choose a visual. Add notes
              to explain the results.
            </p>
            <div className="mt-5 flex flex-wrap justify-center gap-2">
              <Button
                variant="outline"
                onClick={() => fileInput.current?.click()}
              >
                <Upload />
                Upload CSV
              </Button>
              <Button variant="outline" onClick={() => addVisual("text")}>
                <Type />
                Add notes
              </Button>
            </div>
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            {report.visuals.map((visual, index) => (
              <section
                key={visual.id}
                aria-label={`Visual ${index + 1}: ${visual.title}`}
                className={`min-w-0 rounded-md border p-4 ${visual.wide ? "md:col-span-2" : ""} ${!preview && selectedId === visual.id ? "border-[#367855] ring-1 ring-[#367855]" : "border-[#e0e7e1]"}`}
              >
                <div className="mb-4 flex items-start justify-between gap-3">
                  <h3 className="min-w-0 break-words text-sm font-medium">
                    {visual.title || "Untitled visual"}
                  </h3>
                  {!preview && (
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      aria-label={`Configure ${visual.title}`}
                      aria-pressed={selectedId === visual.id}
                      onClick={() => setSelectedId(visual.id)}
                    >
                      <Settings2 className="size-3.5" />
                    </Button>
                  )}
                </div>
                <VisualContent report={report} visual={visual} />
              </section>
            ))}
          </div>
        )}
      </div>
      <p className="mx-auto max-w-5xl text-xs text-muted-foreground">
        Sharing is managed in the report viewer. Saving also updates existing
        share links. Only displayed results are shared; source rows stay
        private.
      </p>
    </main>
  );
}
