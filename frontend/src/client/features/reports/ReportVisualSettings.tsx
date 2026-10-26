import { ArrowDown, ArrowUp, Settings2, Trash2 } from "lucide-react";
import { Button } from "@/client/components/ui/button";
import { Input } from "@/client/components/ui/input";
import { Textarea } from "@/client/components/ui/textarea";
import {
  numericColumns,
  visualSchema,
  type ReportDataset,
  type ReportVisual,
} from "./reportDefinition";
import { FieldSelect, visualTypes } from "./ReportBuilderControls";

export function ReportVisualSettings({
  selected,
  selectedData,
  activeData,
  datasets,
  updateVisual,
  moveVisual,
  onRemove,
  canMoveUp,
  canMoveDown,
}: {
  selected: ReportVisual | undefined;
  selectedData: ReportDataset | undefined;
  activeData: ReportDataset | undefined;
  datasets: ReportDataset[];
  updateVisual: (update: Partial<ReportVisual>) => void;
  moveVisual: (direction: number) => void;
  onRemove: () => void;
  canMoveUp: boolean;
  canMoveDown: boolean;
}) {
  return (
    <aside
      aria-label="Visual settings"
      className="space-y-5 border-t border-border bg-background p-4 xl:border-t-0 xl:border-l"
    >
      <h2 className="flex items-center gap-2 text-sm font-semibold">
        <Settings2 className="size-4" />
        Visual settings
      </h2>
      {!selected ? (
        <p className="text-xs leading-relaxed text-muted-foreground">
          Add a visual or use its settings button on the canvas to choose
          fields, filters and layout.
        </p>
      ) : (
        <>
          <div
            className="grid grid-cols-5 gap-1"
            role="group"
            aria-label="Visual type"
          >
            {visualTypes.map((item) => (
              <Button
                key={item.type}
                variant={selected.type === item.type ? "secondary" : "ghost"}
                size="icon-sm"
                title={item.label}
                aria-label={`Change to ${item.label}`}
                aria-pressed={selected.type === item.type}
                disabled={item.type !== "text" && !activeData}
                onClick={() =>
                  updateVisual({
                    type: item.type,
                    ...(selected.type === "text" && activeData
                      ? {
                          datasetId: activeData.id,
                          dimension: activeData.columns[0],
                          metric: numericColumns(activeData)[0] ?? "",
                          aggregation: numericColumns(activeData).length
                            ? "sum"
                            : "count",
                        }
                      : {}),
                  })
                }
              >
                <item.icon />
              </Button>
            ))}
          </div>
          <label className="block space-y-1.5 text-xs font-medium">
            Visual title
            <Input
              value={selected.title}
              maxLength={120}
              onChange={(event) => updateVisual({ title: event.target.value })}
            />
          </label>
          {selected.type === "text" ? (
            <label className="block space-y-1.5 text-xs font-medium">
              Findings and next steps
              <Textarea
                rows={10}
                maxLength={5000}
                value={selected.text}
                onChange={(event) => updateVisual({ text: event.target.value })}
                placeholder="Explain what changed and the recommended action…"
              />
            </label>
          ) : (
            <>
              <FieldSelect
                label="Data source"
                value={selected.datasetId}
                options={datasets.map((data) => ({
                  value: data.id,
                  label: data.name,
                }))}
                onChange={(datasetId) => {
                  const data = datasets.find((item) => item.id === datasetId)!;
                  const metric = numericColumns(data)[0] ?? "";
                  updateVisual({
                    datasetId,
                    dimension: data.columns[0],
                    metric,
                    aggregation: metric ? "sum" : "count",
                    filterColumn: "",
                    filterValue: "",
                  });
                }}
              />
              {selected.type !== "kpi" && (
                <FieldSelect
                  label="Break down by"
                  value={selected.dimension}
                  options={(selectedData?.columns ?? []).map((column) => ({
                    value: column,
                    label: column,
                  }))}
                  onChange={(dimension) => updateVisual({ dimension })}
                />
              )}
              <FieldSelect
                label="Metric"
                value={selected.metric}
                options={[
                  { value: "", label: "Row count" },
                  ...(selectedData ? numericColumns(selectedData) : []).map(
                    (column) => ({ value: column, label: column }),
                  ),
                ]}
                onChange={(metric) =>
                  updateVisual({
                    metric,
                    aggregation: metric ? "sum" : "count",
                  })
                }
              />
              {selected.metric && (
                <FieldSelect
                  label="Calculate"
                  value={selected.aggregation}
                  options={[
                    { value: "sum", label: "Sum" },
                    { value: "average", label: "Average" },
                    { value: "count", label: "Count rows" },
                    { value: "min", label: "Minimum" },
                    { value: "max", label: "Maximum" },
                  ]}
                  onChange={(aggregation) =>
                    updateVisual({
                      aggregation:
                        visualSchema.shape.aggregation.parse(aggregation),
                    })
                  }
                />
              )}
              <p className="text-[11px] leading-relaxed text-muted-foreground">
                Use Average for rates or positions. Daily visitors are
                visitor-days when summed, not unique people across the period.
              </p>
              <div className="space-y-3 border-t border-border pt-4">
                <h3 className="text-xs font-medium">Filter this visual</h3>
                <FieldSelect
                  label="Filter field"
                  value={selected.filterColumn}
                  options={[
                    { value: "", label: "No filter" },
                    ...(selectedData?.columns ?? []).map((column) => ({
                      value: column,
                      label: column,
                    })),
                  ]}
                  onChange={(filterColumn) =>
                    updateVisual({ filterColumn, filterValue: "" })
                  }
                />
                {selected.filterColumn && (
                  <label className="block space-y-1.5 text-xs">
                    Contains
                    <Input
                      value={selected.filterValue}
                      maxLength={2000}
                      onChange={(event) =>
                        updateVisual({
                          filterValue: event.target.value,
                        })
                      }
                      placeholder="Enter a value…"
                    />
                  </label>
                )}
              </div>
              {selected.type !== "kpi" && (
                <FieldSelect
                  label={
                    selected.type === "line"
                      ? "First groups by label"
                      : "Top groups by value"
                  }
                  value={String(selected.limit)}
                  options={[5, 10, 20, 50].map((value) => ({
                    value: String(value),
                    label: String(value),
                  }))}
                  onChange={(limit) => updateVisual({ limit: Number(limit) })}
                />
              )}
            </>
          )}
          <div className="space-y-3 border-t border-border pt-4">
            <h3 className="text-xs font-medium">Layout</h3>
            <FieldSelect
              label="Width"
              value={selected.wide ? "full" : "half"}
              options={[
                { value: "half", label: "Half page" },
                { value: "full", label: "Full page" },
              ]}
              onChange={(width) => updateVisual({ wide: width === "full" })}
            />
            <div className="flex gap-2">
              <Button
                size="sm"
                variant="outline"
                disabled={!canMoveUp}
                onClick={() => moveVisual(-1)}
              >
                <ArrowUp />
                Move up
              </Button>
              <Button
                size="sm"
                variant="outline"
                disabled={!canMoveDown}
                onClick={() => moveVisual(1)}
              >
                <ArrowDown />
                Down
              </Button>
            </div>
            <Button
              variant="ghost"
              size="sm"
              className="w-full text-destructive"
              onClick={onRemove}
            >
              <Trash2 />
              Remove visual
            </Button>
          </div>
        </>
      )}
    </aside>
  );
}
