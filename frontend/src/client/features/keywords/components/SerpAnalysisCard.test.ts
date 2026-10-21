// @vitest-environment happy-dom
import { act, createElement, type ReactNode } from "react";
import { createRoot, type Root } from "react-dom/client";
import {
  flexRender,
  getCoreRowModel,
  useReactTable,
  type Table,
  type TableOptions,
} from "@tanstack/react-table";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import type { SerpResultItem } from "@/types/keywords";
import { exportRows } from "@/client/lib/exportRows";
import { SerpAnalysisCard } from "./SerpAnalysisCard";

vi.mock("@/client/lib/exportRows", () => ({ exportRows: vi.fn() }));
vi.mock("@/client/components/ExportMenu", () => ({
  ExportMenu: ({ onExport }: { onExport: (format: "csv") => void }) =>
    createElement("button", { onClick: () => onExport("csv") }, "Export CSV"),
}));
vi.mock("@/client/components/ErrorState", () => ({ ErrorState: () => null }));
vi.mock("@/client/components/ui/button", () => ({ Button: () => null }));
vi.mock("@/client/components/table/TablePagination", () => ({
  TablePagination: () => null,
}));
vi.mock("@/client/components/table/DataTableToolbar", () => ({
  DataTableToolbar: ({
    children,
    actions,
  }: {
    children: ReactNode;
    actions: ReactNode;
  }) => createElement("div", null, children, actions),
}));
vi.mock("@/client/components/table/DataTable", () => ({
  useDataTable: (options: TableOptions<SerpResultItem>) =>
    useReactTable({ ...options, getCoreRowModel: getCoreRowModel() }),
  DataTable: ({
    table,
    toolbar,
  }: {
    table: Table<SerpResultItem>;
    toolbar: ReactNode;
  }) =>
    createElement(
      "div",
      null,
      toolbar,
      ...table
        .getRowModel()
        .rows.flatMap((row) =>
          row
            .getVisibleCells()
            .map((cell) =>
              createElement(
                "div",
                { key: cell.id },
                flexRender(cell.column.columnDef.cell, cell.getContext()),
              ),
            ),
        ),
    ),
}));
Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });

let root: Root;
let host: HTMLDivElement;
beforeEach(() => {
  host = document.createElement("div");
  document.body.append(host);
  root = createRoot(host);
});
afterEach(() => {
  act(() => root.unmount());
  host.remove();
  vi.clearAllMocks();
});

const item: SerpResultItem = {
  rank: 1,
  title: "Lonavala hotel",
  url: "https://example.com/hotel",
  domain: "example.com",
  description: "Lake view hotel with family rooms.",
  etv: 0,
  estimatedPaidTrafficCost: 0,
  referringDomains: null,
  backlinks: 12,
  isNew: false,
  rankChange: null,
};
function renderCard() {
  act(() =>
    root.render(
      createElement(SerpAnalysisCard, {
        items: [item],
        keyword: "hotel",
        loading: false,
        loadingMore: false,
        canLoadMore: false,
        retrying: false,
        deepFetchFailed: false,
        page: 0,
        pageSize: 10,
        onPageChange: vi.fn(),
      }),
    ),
  );
}

it("shows the paid organic result description", () => {
  renderCard();
  expect(host.textContent).toContain(item.description);
});

it("keeps the paid description in the SERP export", () => {
  renderCard();
  act(() => host.querySelector("button")?.click());
  expect(exportRows).toHaveBeenCalledWith(
    expect.objectContaining({
      headers: ["Rank", "Title", "URL", "Domain", "Description"],
      rows: [[1, item.title, item.url, item.domain, item.description]],
    }),
  );
});
