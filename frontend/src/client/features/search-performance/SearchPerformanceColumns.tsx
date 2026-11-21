import { createColumnHelper, type ColumnDef } from "@tanstack/react-table";
import type { ReactNode } from "react";
import type { AppColumnMeta } from "@/client/components/table/DataTable";
import { SortableHeader } from "@/client/components/table/SortableHeader";
import type {
  getSearchPerformanceReport,
  getSearchPerformanceTable,
} from "@/client/features/search-performance/searchPerformanceApi";

export type Report = Extract<
  Awaited<ReturnType<typeof getSearchPerformanceReport>>,
  { connected: true }
>;
export type SearchPerformanceTableRow = Extract<
  Awaited<ReturnType<typeof getSearchPerformanceTable>>,
  { connected: true }
>["rows"][number];
type DimensionRow = SearchPerformanceTableRow;

const numberFormat = new Intl.NumberFormat("en-US");

export function formatCount(value: number): string {
  return numberFormat.format(Math.round(value));
}

export function formatCtr(value: number): string {
  return `${(value * 100).toFixed(1)}%`;
}

export function formatPosition(value: number): string {
  return value.toFixed(1);
}

// Typed with DataTable's column meta. Keep this import: DataTable.tsx also
// augments TanStack's ColumnMeta, and TypeScript only accepts typed accessor
// columns as ColumnDef everywhere when that augmentation loads early.
const rightAligned: AppColumnMeta<DimensionRow> = {
  headerClassName: "text-right",
  cellClassName: "text-right tabular-nums",
};

const dimensionHelper = createColumnHelper<DimensionRow>();

export function buildDimensionColumns(
  keyLabel: string,
  /** An extra last column per row, such as "Edit" on a page. */
  rowAction?: (key: string) => ReactNode,
): ColumnDef<DimensionRow>[] {
  return [
    dimensionHelper.accessor("key", {
      enableSorting: false,
      header: () => keyLabel,
      cell: ({ getValue }) => (
        <span className="block max-w-xl truncate" title={getValue()}>
          {getValue()}
        </span>
      ),
    }),
    dimensionHelper.accessor("clicks", {
      header: ({ column }) => (
        <SortableHeader column={column} label="Clicks" align="right" />
      ),
      cell: ({ getValue }) => formatCount(getValue()),
      meta: rightAligned,
    }),
    dimensionHelper.accessor("impressions", {
      header: ({ column }) => (
        <SortableHeader column={column} label="Impressions" align="right" />
      ),
      cell: ({ getValue }) => formatCount(getValue()),
      meta: rightAligned,
    }),
    dimensionHelper.accessor("ctr", {
      header: ({ column }) => (
        <SortableHeader column={column} label="CTR" align="right" />
      ),
      cell: ({ getValue }) => formatCtr(getValue()),
      meta: rightAligned,
    }),
    dimensionHelper.accessor("position", {
      header: ({ column }) => (
        <SortableHeader column={column} label="Position" align="right" />
      ),
      cell: ({ getValue }) => formatPosition(getValue()),
      meta: rightAligned,
    }),
    ...(rowAction
      ? [
          dimensionHelper.display({
            id: "action",
            header: () => null,
            cell: ({ row }) => rowAction(row.original.key),
            meta: { cellClassName: "text-right" },
          }),
        ]
      : []),
  ];
}
