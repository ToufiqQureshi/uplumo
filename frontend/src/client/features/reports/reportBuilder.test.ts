import { describe, expect, it } from "vitest";
import {
  aggregateVisual,
  createReportVisual,
  parseReportCsv,
  reportDefinitionSchema,
  updateReportVisual,
} from "./reportDefinition";

describe("report builder", () => {
  it("updates generated titles with fields and preserves custom titles", () => {
    const visual = createReportVisual("bar", {
      id: "d",
      name: "CSV",
      columns: ["Date", "Clicks", "Channel"],
      dateColumn: "",
      rows: [["2026-10-01", 5, "Search"]],
    });
    expect(updateReportVisual(visual, { dimension: "Channel" }).title).toBe(
      "Clicks by Channel",
    );
    expect(
      updateReportVisual(
        { ...visual, title: "My results" },
        { metric: "Revenue" },
      ).title,
    ).toBe("My results");
  });
  it("parses quoted CSV without losing zero, blanks or negative values", () => {
    const data = parseReportCsv(
      'Channel,Revenue,Note\n"Search, organic",0,\nPaid,-20,"two\nlines"',
    );
    expect(data.rows).toEqual([
      ["Search, organic", 0, null],
      ["Paid", -20, "two\nlines"],
    ]);
    expect(data.columns).toEqual(["Channel", "Revenue", "Note"]);
    for (const csv of ["A,A\n1,2", "A,B\n1", "A\n", "A,B\n1,2,3"]) {
      expect(() => parseReportCsv(csv)).toThrow();
    }
  });

  it("filters before grouping, preserves zero and computes averages rather than summing rates", () => {
    const definition = reportDefinitionSchema.parse({
      version: 1,
      title: "Report",
      summary: "",
      from: "2026-10-01",
      to: "2026-10-31",
      datasets: [
        {
          id: "d",
          name: "CSV",
          columns: ["Date", "Channel", "Rate"],
          dateColumn: "Date",
          rows: [
            ["2026-10-01", "Search", 0],
            ["2026-10-02", "Search", 10],
            ["2026-09-30", "Search", 99],
            ["2026-10-02", "Paid", 100],
          ],
        },
      ],
      visuals: [
        {
          id: "v",
          type: "bar",
          title: "Rate",
          datasetId: "d",
          dimension: "Channel",
          metric: "Rate",
          aggregation: "average",
          filterColumn: "Channel",
          filterValue: "Search",
          limit: 10,
          wide: false,
          text: "",
        },
      ],
    });
    expect(aggregateVisual(definition, definition.visuals[0])).toEqual([
      { label: "Search", value: 5 },
    ]);
    expect(
      aggregateVisual(definition, {
        ...definition.visuals[0],
        filterValue: "missing",
      }),
    ).toEqual([]);
    expect(
      aggregateVisual(definition, { ...definition.visuals[0], type: "kpi" }),
    ).toEqual([{ label: "Total", value: 5 }]);
  });
});
