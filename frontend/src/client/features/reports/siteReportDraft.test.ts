// @vitest-environment happy-dom
import { describe, expect, it } from "vitest";
import { indexReportDraft, renderReportDraft } from "./siteReportDraft";

const template = `<!doctype html><html><body>
<header><a class="tile" href="#analytics">Traffic highlight</a></header>
<nav class="toc"><a href="#analytics">Analytics</a></nav>
<main><section id="analytics"><h2>Analytics</h2>
<div data-report-pair="Traffic channels">
  <figure class="bars" data-report-view="chart">
    <div data-report-row="Internal"><span class="bar-track"><i data-report-value="900"></i>Internal 900</span></div>
    <div data-report-row="Organic"><span class="bar-track"><i data-report-value="100"></i>Organic 100</span></div>
  </figure>
  <div class="table" data-report-view="table"><table>
    <thead><tr><th>Channel</th><th>Sessions</th><th>Revenue</th></tr></thead>
    <tbody><tr data-report-row="Internal"><td>Internal</td><td>900</td><td>Private total</td></tr>
    <tr data-report-row="Organic"><td>Organic</td><td>100</td><td>Visible total</td></tr></tbody>
  </table></div>
</div></section>
<section id="search"><h2>Search</h2><div class="tabs">
  <input id="sc-c" name="sc" type="radio" checked><label for="sc-c">Clicks</label>
  <input id="sc-i" name="sc" type="radio"><label for="sc-i">Impressions</label>
  <div class="panel" data-report-tab="sc-c" data-title="Clicks per day">Hidden trend</div>
  <div class="panel" data-report-tab="sc-i" data-title="Impressions per day">Visible trend</div>
</div></section></main></body></html>`;

describe("client report draft", () => {
  it("removes excluded rows from both views and saves the displayed document", () => {
    const indexed = indexReportDraft(template);
    const group = indexed.groups.find(
      (item) => item.label === "Traffic channels",
    );
    const hiddenRow = group?.items.find((item) => item.label === "Internal");
    const hiddenMetric = group?.items.find(
      (item) => item.label === "Column: Revenue",
    );
    if (!group?.views || !hiddenRow || !hiddenMetric)
      throw new Error("Missing report controls");
    const html = renderReportDraft(
      indexed.html,
      new Set([hiddenRow.id, hiddenMetric.id]),
    );
    expect(html).toContain("Organic 100");
    expect(html).toContain("width: 100%");
    expect(html).not.toMatch(
      /Internal|900|Revenue|Visible total|data-report-edit/,
    );
    const chartOnly = renderReportDraft(
      indexed.html,
      new Set([group.views.table]),
    );
    expect(chartOnly).not.toContain("<table");
    expect(chartOnly).toContain("Organic 100");
  });

  it("removes section navigation and selects the first remaining trend tab", () => {
    const indexed = indexReportDraft(template);
    const section = indexed.groups.find(
      (group) => group.section === "Analytics" && group.label === "Section",
    )?.items[0];
    const tab = indexed.groups
      .flatMap((group) => group.items)
      .find((item) => item.label === "Clicks per day");
    if (!section || !tab) throw new Error("Missing section or tab");
    const html = renderReportDraft(indexed.html, new Set([section.id, tab.id]));
    const doc = new DOMParser().parseFromString(html, "text/html");
    expect(
      doc.querySelectorAll(
        "#analytics,a[href='#analytics'],#sc-c,label[for='sc-c']",
      ),
    ).toHaveLength(0);
    expect(doc.querySelector("#sc-i[checked]")).not.toBeNull();
    expect(html).not.toContain("Hidden trend");
  });
});
