import { useMemo, useState } from "react";
import {
  createColumnHelper,
  type ColumnDef,
  type PaginationState,
  type RowSelectionState,
  type SortingState,
} from "@tanstack/react-table";
import {
  DataTable,
  makeSelectionColumn,
  useDataTable,
  useSelectionAnchor,
} from "@/client/components/table/DataTable";
import { SortableHeader } from "@/client/components/table/SortableHeader";
import { TablePagination } from "@/client/components/table/TablePagination";
import { Badge } from "@/client/components/ui/badge";
import { Button } from "@/client/components/ui/button";
import { Input } from "@/client/components/ui/input";
import { Label } from "@/client/components/ui/label";
import { Sparkline } from "@/client/features/keywords/agent/KeywordCharts";
import { IntentBadge } from "@/client/features/keywords/components/IntentBadge";
import type { KeywordIntent, KeywordResearchRow } from "@/types/keywords";
import {
  filterIdeas,
  formatCount,
  formatUsd,
  hasFilters,
  NO_FILTERS,
  serpFeatureName,
  trendOldestFirst,
  type IdeaFilters,
} from "./researchModel";

const column = createColumnHelper<KeywordResearchRow>();
const PAGE_SIZE = 50;
const NUMBER_COLUMN = {
  headerClassName: "text-right",
  cellClassName: "text-right font-mono tabular-nums",
};
const INTENTS: Array<[KeywordIntent, string]> = [
  ["informational", "Learn"],
  ["commercial", "Compare"],
  ["transactional", "Buy"],
  ["navigational", "Brand"],
];

function numberOrNull(value: string): number | null {
  if (value.trim() === "") return null;
  const n = Number(value);
  return Number.isFinite(n) && n >= 0 ? n : null;
}

/**
 * Every keyword the research bought, with every column the data has. Sorting,
 * filtering and paging happen in the browser on rows already paid for, so
 * none of them costs credits.
 */
export function KeywordIdeasTable({
  rows,
  activeKeyword,
  onOpen,
  onSave,
  savedKeywords,
  savingKeywords,
}: {
  rows: KeywordResearchRow[];
  activeKeyword: string | null;
  onOpen: (row: KeywordResearchRow) => void;
  onSave: (rows: KeywordResearchRow[]) => void;
  savedKeywords: ReadonlySet<string>;
  savingKeywords: ReadonlySet<string>;
}) {
  const [filters, setFilters] = useState<IdeaFilters>(NO_FILTERS);
  const [sorting, setSorting] = useState<SortingState>([
    { id: "searchVolume", desc: true },
  ]);
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: PAGE_SIZE,
  });
  const [rowSelection, setRowSelection] = useState<RowSelectionState>({});
  const anchor = useSelectionAnchor();
  const shown = useMemo(() => filterIdeas(rows, filters), [rows, filters]);
  const update = (next: Partial<IdeaFilters>) => {
    setFilters((f) => ({ ...f, ...next }));
    setPagination((p) => ({ ...p, pageIndex: 0 }));
  };

  const columns = useMemo<ColumnDef<KeywordResearchRow>[]>(
    () => [
      makeSelectionColumn<KeywordResearchRow>(anchor),
      column.accessor("keyword", {
        header: () => "Keyword",
        cell: ({ row }) => (
          <span className="flex flex-col">
            <span className="font-medium">{row.original.keyword}</span>
            {row.original.coreKeyword &&
            row.original.coreKeyword !== row.original.keyword ? (
              <span className="text-xs text-muted-foreground">
                shares volume with “{row.original.coreKeyword}”
              </span>
            ) : null}
          </span>
        ),
      }),
      column.accessor("searchVolume", {
        sortUndefined: "last",
        header: ({ column: c }) => (
          <SortableHeader
            column={c}
            label="Searches"
            helpText="Average searches per month in the chosen country."
          />
        ),
        cell: ({ getValue }) => formatCount(getValue()),
        meta: NUMBER_COLUMN,
      }),
      column.display({
        id: "trend",
        header: () => "12 months",
        cell: ({ row }) => (
          <Sparkline
            values={trendOldestFirst(row.original).map((m) => m.volume)}
            label={`Search trend for ${row.original.keyword}`}
          />
        ),
      }),
      column.accessor("yearlyChange", {
        sortUndefined: "last",
        header: ({ column: c }) => (
          <SortableHeader
            column={c}
            label="1 yr"
            helpText="Change in searches over the last year."
          />
        ),
        cell: ({ getValue }) => {
          const v = getValue();
          if (v == null) return "—";
          return (
            <span
              className={
                v > 0 ? "text-success" : v < 0 ? "text-destructive" : ""
              }
            >
              {v > 0 ? "+" : ""}
              {v}%
            </span>
          );
        },
        meta: NUMBER_COLUMN,
      }),
      column.accessor("keywordDifficulty", {
        sortUndefined: "last",
        header: ({ column: c }) => (
          <SortableHeader
            column={c}
            label="KD"
            helpText="Keyword difficulty, 0 to 100. Lower is easier to rank for."
          />
        ),
        cell: ({ getValue }) => getValue() ?? "—",
        meta: NUMBER_COLUMN,
      }),
      column.accessor("cpc", {
        sortUndefined: "last",
        header: ({ column: c }) => (
          <SortableHeader
            column={c}
            label="CPC"
            helpText="What advertisers pay per click, in US dollars."
          />
        ),
        cell: ({ getValue }) => formatUsd(getValue()),
        meta: NUMBER_COLUMN,
      }),
      column.accessor("competition", {
        sortUndefined: "last",
        header: ({ column: c }) => (
          <SortableHeader
            column={c}
            label="Ads comp."
            helpText="How many advertisers bid on it, from 0 to 1."
          />
        ),
        cell: ({ getValue }) => getValue()?.toFixed(2) ?? "—",
        meta: NUMBER_COLUMN,
      }),
      column.accessor("intent", {
        header: () => "Intent",
        cell: ({ getValue }) => <IntentBadge intent={getValue()} />,
      }),
      column.display({
        id: "serp",
        header: () => "On Google",
        cell: ({ row }) => {
          const features = row.original.serpFeatures ?? [];
          if (features.length === 0)
            return <span className="text-muted-foreground">—</span>;
          return (
            <span className="flex flex-wrap gap-1">
              {features.slice(0, 2).map((t) => (
                <Badge key={t} variant="secondary">
                  {serpFeatureName(t)}
                </Badge>
              ))}
              {features.length > 2 ? (
                <Badge variant="outline">+{features.length - 2}</Badge>
              ) : null}
            </span>
          );
        },
      }),
      column.display({
        id: "save",
        header: () => <span className="sr-only">Save</span>,
        cell: ({ row }) => {
          const keyword = row.original.keyword;
          const saved = savedKeywords.has(keyword);
          return (
            <Button
              size="sm"
              variant={saved ? "ghost" : "outline"}
              disabled={saved}
              pending={savingKeywords.has(keyword)}
              onClick={(event) => {
                event.stopPropagation();
                onSave([row.original]);
              }}
            >
              {saved ? "Saved" : "Save"}
            </Button>
          );
        },
      }),
    ],
    [anchor, onSave, savedKeywords, savingKeywords],
  );

  const table = useDataTable({
    data: shown,
    columns,
    getRowId: (row) => row.keyword,
    state: { sorting, pagination, rowSelection },
    onSortingChange: setSorting,
    onPaginationChange: setPagination,
    onRowSelectionChange: setRowSelection,
    enableRowSelection: true,
    withSorting: true,
    withPagination: true,
  });
  const selected = table.getSelectedRowModel().rows.map((r) => r.original);

  const toolbar = (
    <div className="flex flex-col gap-3 border-b border-border p-3">
      <div className="flex flex-wrap items-end gap-3">
        <div className="flex min-w-48 flex-1 flex-col gap-1">
          <Label htmlFor="ideas-text">Contains words</Label>
          <Input
            id="ideas-text"
            value={filters.text}
            placeholder="e.g. free"
            onChange={(e) => update({ text: e.target.value })}
          />
        </div>
        <div className="flex w-28 flex-col gap-1">
          <Label htmlFor="ideas-kd">KD at most</Label>
          <Input
            id="ideas-kd"
            inputMode="numeric"
            value={filters.maxDifficulty ?? ""}
            onChange={(e) =>
              update({ maxDifficulty: numberOrNull(e.target.value) })
            }
          />
        </div>
        <div className="flex w-32 flex-col gap-1">
          <Label htmlFor="ideas-volume">Searches at least</Label>
          <Input
            id="ideas-volume"
            inputMode="numeric"
            value={filters.minVolume ?? ""}
            onChange={(e) =>
              update({ minVolume: numberOrNull(e.target.value) })
            }
          />
        </div>
      </div>
      <div
        role="group"
        aria-label="Filter by what searchers want"
        className="flex flex-wrap items-center gap-2"
      >
        {INTENTS.map(([intent, label]) => {
          const on = filters.intents.includes(intent);
          return (
            <Button
              key={intent}
              size="sm"
              variant={on ? "default" : "outline"}
              aria-pressed={on}
              onClick={() =>
                update({
                  intents: on
                    ? filters.intents.filter((i) => i !== intent)
                    : [...filters.intents, intent],
                })
              }
            >
              {label}
            </Button>
          );
        })}
        <span className="ml-auto text-sm text-muted-foreground">
          {shown.length === rows.length
            ? `${rows.length} keywords`
            : `${shown.length} of ${rows.length} keywords`}
        </span>
        {selected.length > 0 ? (
          <Button
            size="sm"
            onClick={() => {
              onSave(selected.filter((r) => !savedKeywords.has(r.keyword)));
              setRowSelection({});
            }}
          >
            Save {selected.length} selected
          </Button>
        ) : null}
      </div>
    </div>
  );

  return (
    <DataTable
      table={table}
      toolbar={toolbar}
      isFiltered={hasFilters(filters)}
      onClearFilters={() => update(NO_FILTERS)}
      onRowClick={(row) => onOpen(row.original)}
      activeRowId={activeKeyword ?? undefined}
      empty={{
        title: "No keywords came back",
        description: "Try a broader or shorter keyword.",
      }}
      footer={
        shown.length > PAGE_SIZE ? (
          <TablePagination
            page={pagination.pageIndex + 1}
            pageSize={PAGE_SIZE}
            totalCount={shown.length}
            onPageChange={(page) =>
              setPagination((p) => ({ ...p, pageIndex: page - 1 }))
            }
          />
        ) : null
      }
    />
  );
}
