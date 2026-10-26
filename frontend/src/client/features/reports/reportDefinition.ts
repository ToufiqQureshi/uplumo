import Papa from "papaparse";
import { z } from "zod";

const column = z.string().min(1).max(120);
const datasetSchema = z
  .object({
    id: z.string().min(1).max(80),
    name: z.string().min(1).max(120),
    columns: z.array(column).min(1).max(30),
    rows: z
      .array(
        z
          .array(
            z.union([
              z.string().max(2000),
              z
                .number()
                .finite()
                .min(-Number.MAX_SAFE_INTEGER)
                .max(Number.MAX_SAFE_INTEGER),
              z.null(),
            ]),
          )
          .max(30),
      )
      .max(2000),
    dateColumn: z.string().max(120).default(""),
  })
  .superRefine((data, ctx) => {
    if (
      new Set(data.columns).size !== data.columns.length ||
      data.rows.some((row) => row.length !== data.columns.length) ||
      (data.dateColumn && !data.columns.includes(data.dateColumn))
    ) {
      ctx.addIssue({
        code: "custom",
        message: "Data columns must be unique and match every row.",
      });
    }
  });

export const visualSchema = z.object({
  id: z.string().min(1).max(80),
  type: z.enum(["kpi", "line", "bar", "table", "text"]),
  title: z.string().max(120),
  datasetId: z.string().max(80),
  dimension: z.string().max(120),
  metric: z.string().max(120),
  aggregation: z.enum(["sum", "average", "count", "min", "max"]),
  filterColumn: z.string().max(120),
  filterValue: z.string().max(2000),
  limit: z.number().int().min(1).max(50),
  wide: z.boolean(),
  text: z.string().max(5000),
});

export const reportDefinitionSchema = z
  .object({
    version: z.literal(1),
    title: z.string().max(120),
    summary: z.string().max(2500),
    from: z.union([z.iso.date(), z.literal("")]),
    to: z.union([z.iso.date(), z.literal("")]),
    datasets: z.array(datasetSchema).max(6),
    visuals: z.array(visualSchema).max(24),
  })
  .superRefine((report, ctx) => {
    const ids = report.datasets.map((data) => data.id);
    if (
      new Set(ids).size !== ids.length ||
      new Set(report.visuals.map((v) => v.id)).size !== report.visuals.length ||
      (report.from && report.to && report.from > report.to)
    ) {
      ctx.addIssue({
        code: "custom",
        message: "Report IDs must be unique and dates must be in order.",
      });
    }
    for (const visual of report.visuals) {
      if (visual.type === "text") continue;
      const dataset = report.datasets.find(
        (data) => data.id === visual.datasetId,
      );
      if (
        !dataset ||
        (visual.aggregation !== "count" &&
          !dataset.columns.includes(visual.metric)) ||
        (visual.type !== "kpi" &&
          !dataset.columns.includes(visual.dimension)) ||
        (visual.filterColumn && !dataset.columns.includes(visual.filterColumn))
      ) {
        ctx.addIssue({
          code: "custom",
          message: "Choose valid data fields for every visual.",
        });
      }
    }
  });

export type ReportDefinition = z.infer<typeof reportDefinitionSchema>;
export type ReportDataset = z.infer<typeof datasetSchema>;
export type ReportVisual = z.infer<typeof visualSchema>;

export function numericColumns(data: ReportDataset) {
  return data.columns.filter((_, index) => {
    const values = data.rows
      .map((row) => row[index])
      .filter((value) => value !== null);
    return (
      values.length > 0 && values.every((value) => typeof value === "number")
    );
  });
}

export function parseReportCsv(
  csv: string,
): Pick<ReportDataset, "columns" | "rows" | "dateColumn"> {
  if (new TextEncoder().encode(csv).length > 200_000)
    throw new Error("Use a CSV smaller than 200 KB.");
  const parsed = Papa.parse<string[]>(csv, { skipEmptyLines: "greedy" });
  if (parsed.errors.length)
    throw new Error("This CSV could not be read. Check quotes and separators.");
  const [header, ...rawRows] = parsed.data;
  const columns = header?.map((value) => value.trim()) ?? [];
  if (!rawRows.length)
    throw new Error("Add a header and at least one data row.");
  const rows = rawRows.map((row) =>
    row.map((raw) => {
      const value = raw.trim();
      if (!value) return null;
      // Only complete numeric strings convert; dates, percentages and IDs keep their meaning.
      return /^-?(?:0|[1-9]\d*)(?:\.\d+)?$/.test(value) &&
        Number.isFinite(Number(value)) &&
        Math.abs(Number(value)) <= Number.MAX_SAFE_INTEGER
        ? Number(value)
        : value;
    }),
  );
  const dateColumn =
    columns.find(
      (_, index) =>
        rows.some((row) => typeof row[index] === "string") &&
        rows.every(
          (row) =>
            row[index] === null || z.iso.date().safeParse(row[index]).success,
        ),
    ) ?? "";
  const data = datasetSchema.safeParse({
    id: "csv",
    name: "CSV",
    columns,
    rows,
    dateColumn,
  });
  if (!data.success)
    throw new Error(
      "Use unique, non-empty headers (up to 30 columns), matching rows (up to 2,000), and cells under 2,000 characters.",
    );
  return { columns, rows, dateColumn };
}

export function aggregateVisual(
  report: ReportDefinition,
  visual: ReportVisual,
) {
  const dataset = report.datasets.find((data) => data.id === visual.datasetId);
  if (!dataset || visual.type === "text") return [];
  const dimension = dataset.columns.indexOf(visual.dimension);
  const metric = dataset.columns.indexOf(visual.metric);
  const filter = dataset.columns.indexOf(visual.filterColumn);
  const date = dataset.columns.indexOf(dataset.dateColumn);
  const groups = new Map<string, number[]>();
  for (const row of dataset.rows) {
    if (
      filter >= 0 &&
      visual.filterValue &&
      !String(row[filter] ?? "")
        .toLowerCase()
        .includes(visual.filterValue.toLowerCase())
    )
      continue;
    if (date >= 0 && (report.from || report.to)) {
      const value = row[date];
      if (
        typeof value !== "string" ||
        !z.iso.date().safeParse(value).success ||
        (report.from && value < report.from) ||
        (report.to && value > report.to)
      )
        continue;
    }
    const value = visual.aggregation === "count" ? 1 : row[metric];
    if (typeof value !== "number" || !Number.isFinite(value)) continue;
    const label =
      visual.type === "kpi" ? "Total" : String(row[dimension] ?? "(empty)");
    const values = groups.get(label) ?? [];
    values.push(value);
    groups.set(label, values);
  }
  const points = [...groups].map(([label, values]) => {
    const sum = values.reduce((total, value) => total + value, 0);
    const value =
      visual.aggregation === "average"
        ? sum / values.length
        : visual.aggregation === "min"
          ? Math.min(...values)
          : visual.aggregation === "max"
            ? Math.max(...values)
            : sum;
    return { label, value };
  });
  if (visual.type === "line")
    points.sort((a, b) => a.label.localeCompare(b.label));
  else
    points.sort((a, b) => b.value - a.value || a.label.localeCompare(b.label));
  return points.slice(0, visual.limit);
}

export function createReportVisual(
  type: ReportVisual["type"],
  dataset?: ReportDataset,
): ReportVisual {
  const metrics = dataset ? numericColumns(dataset) : [];
  const metric = metrics[0] ?? "";
  return {
    id: crypto.randomUUID(),
    type,
    title: defaultVisualTitle({
      type,
      metric,
      dimension: dataset?.columns[0] ?? "",
    }),
    datasetId: dataset?.id ?? "",
    dimension: dataset?.columns[0] ?? "",
    metric,
    aggregation: metric ? "sum" : "count",
    filterColumn: "",
    filterValue: "",
    limit: type === "line" ? 50 : 10,
    wide: type === "line" || type === "text",
    text: "",
  };
}

function defaultVisualTitle(
  visual: Pick<ReportVisual, "type" | "metric" | "dimension">,
) {
  if (visual.type === "text") return "Findings & next steps";
  if (visual.type === "kpi") return visual.metric || "Row count";
  return `${visual.metric || "Rows"} by ${visual.dimension || "field"}`;
}

export function updateReportVisual(
  visual: ReportVisual,
  update: Partial<ReportVisual>,
): ReportVisual {
  const next = { ...visual, ...update };
  if (update.title === undefined && visual.title === defaultVisualTitle(visual))
    next.title = defaultVisualTitle(next);
  return next;
}
