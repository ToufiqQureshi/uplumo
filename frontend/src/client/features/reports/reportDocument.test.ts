import { describe, expect, it } from "vitest";
import { buildVisualReportHtml } from "./reportDocument";
import { createReportVisual, type ReportDefinition } from "./reportDefinition";

describe("visual report export", () => {
  it("exports only displayed aggregates and escapes labels, titles and notes", () => {
    const data = {
      id: "data",
      name: "CSV",
      columns: ["Channel", "Clicks", "Private"],
      dateColumn: "",
      rows: [
        ["<img src=x onerror=alert(1)>", 5, "hidden-source-value"],
        ["Excluded", 12, "hidden-row"],
      ],
    };
    const visual = {
      ...createReportVisual("bar", data),
      filterColumn: "Channel",
      filterValue: "<img",
    };
    const report: ReportDefinition = {
      version: 1,
      title: "<script>bad()</script>",
      summary: "",
      from: "",
      to: "",
      datasets: [data],
      visuals: [
        visual,
        { ...createReportVisual("text"), text: "<script>notes()</script>" },
      ],
    };
    const html = buildVisualReportHtml(report);
    expect(html).toContain("&lt;img");
    expect(html).toContain("&lt;script&gt;notes()");
    expect(html).not.toContain("<script>");
    for (const hidden of [
      "hidden-source-value",
      "hidden-row",
      "Excluded",
      "Private",
      "datasets",
    ])
      expect(html).not.toContain(hidden);
    expect(html).toContain("<strong>5</strong>");
    expect(html).toContain("<table");
    expect(html.endsWith("</html>")).toBe(true);
  });

  it("refuses oversized reports rather than silently dropping imported rows", () => {
    const data = {
      id: "d",
      name: "CSV",
      columns: ["Text", "Value"],
      dateColumn: "",
      rows: Array.from({ length: 400 }, () => ["a".repeat(1500), 1]),
    };
    expect(() =>
      buildVisualReportHtml({
        version: 1,
        title: "Big",
        summary: "",
        from: "",
        to: "",
        datasets: [data],
        visuals: [createReportVisual("kpi", data)],
      }),
    ).toThrow("500 KB");
  });
});
