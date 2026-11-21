type Item = { id: string; label: string };
type Group = {
  id: string;
  label: string;
  section: string;
  items: Item[];
  views?: { chart: string; table: string };
};

const mark = "data-report-edit";
const parse = (html: string) =>
  new DOMParser().parseFromString(html, "text/html");
const documentHTML = (document: Document) =>
  `<!doctype html>\n${document.documentElement.outerHTML}`;
const textOf = (node: Element | null | undefined) =>
  node?.textContent?.trim().replace(/\s+/g, " ") ?? "";

function groupLabel(node: Element) {
  const pair = node.closest("[data-report-pair]");
  if (pair) return pair.getAttribute("data-report-pair");
  const panel = node.closest(".panel[data-title]");
  if (panel) return panel.getAttribute("data-title");
  const details = node.closest("details");
  if (details) return textOf(details.querySelector("summary"));
  for (
    let prior = node.previousElementSibling;
    prior;
    prior = prior.previousElementSibling
  ) {
    if (prior.tagName === "H3") return textOf(prior);
  }
  return null;
}

export function indexReportDraft(html: string) {
  const doc = parse(html);
  const groups = new Map<string, Group>();
  let id = 0;
  const add = (node: Element, label: string, group?: Group, key?: string) => {
    const item = { id: key ?? String(++id), label };
    const keys = node.getAttribute(mark);
    node.setAttribute(mark, keys ? `${keys} ${item.id}` : item.id);
    if (group && !group.items.some((existing) => existing.id === item.id)) {
      group.items.push(item);
    }
    return item.id;
  };
  const groupFor = (node: Element, label: string) => {
    const section = node.closest("main > section");
    const name = section
      ? textOf(section.querySelector("h2"))
      : "Report header";
    const key = `${section?.id ?? "header"}:${label}`;
    let group = groups.get(key);
    if (!group) {
      group = { id: key, section: name, label, items: [] };
      groups.set(key, group);
    }
    return group;
  };
  for (const pair of doc.querySelectorAll("[data-report-pair]")) {
    const group = groupFor(pair, groupLabel(pair) ?? "Chart and table");
    const chart = pair.querySelector('[data-report-view="chart"]');
    const table = pair.querySelector('[data-report-view="table"]');
    if (chart && table) {
      group.views = { chart: add(chart, "Chart"), table: add(table, "Table") };
    }
    const keys = new Map<string, string>();
    for (const row of pair.querySelectorAll("[data-report-row]")) {
      const label = row.getAttribute("data-report-row") ?? "";
      keys.set(label, add(row, label || "Unnamed row", group, keys.get(label)));
    }
  }
  for (const section of doc.querySelectorAll("main > section")) {
    add(
      section,
      textOf(section.querySelector("h2")),
      groupFor(section, "Section"),
    );
  }
  for (const table of doc.querySelectorAll<HTMLTableElement>("main table")) {
    const group = groupFor(
      table,
      groupLabel(table.parentElement ?? table) ?? "Table",
    );
    for (const row of table.querySelectorAll<HTMLTableRowElement>(
      "tbody tr:not([data-report-row])",
    )) {
      add(row, textOf(row.firstElementChild) || "Unnamed row", group);
    }
    const header = table.querySelector("thead > tr");
    if (!header) continue;
    for (const [column, cell] of Array.from(header.children).entries()) {
      if (column === 0) continue;
      const key = add(cell, `Column: ${textOf(cell)}`, group);
      for (const row of table.querySelectorAll<HTMLTableRowElement>(
        "tbody tr",
      )) {
        const value = row.children[column];
        if (value) add(value, "", undefined, key);
      }
      const pair = table.closest("[data-report-pair]");
      const metric = cell.getAttribute("data-report-metric");
      if (!pair || !metric) continue;
      const chartMetrics = pair.querySelectorAll(
        '[data-report-view="chart"] [data-report-metric]',
      );
      for (const bar of chartMetrics) {
        if (bar.getAttribute("data-report-metric") === metric)
          add(bar, "", undefined, key);
      }
    }
  }
  for (const node of doc.querySelectorAll(
    ".tile,.kpi,.card,.sev > div,.rings figure,.cwv span,.note p,.steps p,.work li,details,details li,.evidence > p,.rank-grid,.dist,svg.chart:not(.panel svg),.tabs > .panel:not(.p-rf)",
  )) {
    const label =
      node.getAttribute("data-title") ??
      node.getAttribute("aria-label") ??
      (textOf(node.querySelector(".k,summary,figcaption,b")) || textOf(node));
    add(
      node,
      label.slice(0, 180),
      groupFor(node, groupLabel(node) ?? "Metrics and evidence"),
    );
  }
  return { html: documentHTML(doc), groups: [...groups.values()] };
}

export function renderReportDraft(html: string, excluded: ReadonlySet<string>) {
  const doc = parse(html);
  for (const node of doc.querySelectorAll(`[${mark}]`)) {
    const keys = node.getAttribute(mark)?.split(" ") ?? [];
    if (keys.some((key) => excluded.has(key))) {
      if (node.matches(".panel")) {
        const tab = node.getAttribute("data-report-tab");
        if (tab) {
          doc.getElementById(tab)?.remove();
          doc.querySelector(`label[for="${CSS.escape(tab)}"]`)?.remove();
        }
      }
      node.remove();
    } else {
      node.removeAttribute(mark);
    }
  }
  for (const tabs of doc.querySelectorAll(".tabs")) {
    const first = tabs.querySelector('input[type="radio"]');
    if (first && !tabs.querySelector("input[checked]"))
      first.setAttribute("checked", "");
  }
  for (const anchor of doc.querySelectorAll<HTMLAnchorElement>(
    'a[href^="#"]',
  )) {
    if (doc.getElementById(anchor.getAttribute("href")?.slice(1) ?? ""))
      continue;
    if (anchor.matches(".toc a,.tile,.card")) anchor.remove();
    else anchor.removeAttribute("href");
  }
  for (const node of doc.querySelectorAll("div.table,figure.bars")) {
    if (!node.querySelector("tbody tr,.bar-track")) node.remove();
  }
  for (const chart of doc.querySelectorAll(".bars")) {
    const bars = Array.from(
      chart.querySelectorAll<HTMLElement>("[data-report-value]"),
      (node) => ({ node, value: Number(node.dataset.reportValue) || 0 }),
    );
    const peak = Math.max(1, ...bars.map(({ value }) => value));
    for (const { node, value } of bars) {
      node.style.width = `${Math.max(0, (value / peak) * 100)}%`;
    }
  }
  for (const node of doc.querySelectorAll(
    "[data-report-row],[data-report-view],[data-report-pair],[data-report-tab],[data-report-value],[data-report-metric]",
  )) {
    for (const attribute of [
      "data-report-row",
      "data-report-view",
      "data-report-pair",
      "data-report-tab",
      "data-report-value",
      "data-report-metric",
    ])
      node.removeAttribute(attribute);
  }
  return documentHTML(doc);
}
