import { renderToStaticMarkup } from "react-dom/server";
import {
  aggregateVisual,
  reportDefinitionSchema,
  type ReportDefinition,
  type ReportVisual,
} from "./reportDefinition";

const number = new Intl.NumberFormat("en", { maximumFractionDigits: 2 });

export function VisualContent({
  report,
  visual,
}: {
  report: ReportDefinition;
  visual: ReportVisual;
}) {
  if (visual.type === "text")
    return (
      <p
        style={{
          whiteSpace: "pre-wrap",
          overflowWrap: "anywhere",
          lineHeight: 1.7,
        }}
      >
        {visual.text ||
          "Add your findings and next steps in the settings panel."}
      </p>
    );
  const points = aggregateVisual(report, visual);
  if (!points.length)
    return (
      <p style={{ padding: "40px 0", color: "#64746b", textAlign: "center" }}>
        No data matches these fields and filters.
      </p>
    );
  if (visual.type === "kpi")
    return (
      <div style={{ padding: "16px 0" }}>
        <strong
          style={{ fontSize: 42, letterSpacing: "-0.04em", fontWeight: 600 }}
        >
          {number.format(points[0].value)}
        </strong>
        <p style={{ color: "#64746b", fontSize: 12 }}>
          {visual.aggregation} of {visual.metric || "rows"}
        </p>
      </div>
    );
  const table = (
    <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 12 }}>
      <thead>
        <tr>
          <th
            style={{
              textAlign: "left",
              padding: "8px 0",
              borderBottom: "1px solid #e3e9e5",
            }}
          >
            {visual.dimension}
          </th>
          <th
            style={{
              textAlign: "right",
              padding: "8px 0",
              borderBottom: "1px solid #e3e9e5",
            }}
          >
            {visual.metric || "Rows"} · {visual.aggregation}
          </th>
        </tr>
      </thead>
      <tbody>
        {points.map((point) => (
          <tr key={point.label}>
            <td
              style={{
                padding: "8px 0",
                borderBottom: "1px solid #edf1ee",
                overflowWrap: "anywhere",
              }}
            >
              {point.label}
            </td>
            <td
              style={{
                padding: "8px 0 8px 12px",
                borderBottom: "1px solid #edf1ee",
                textAlign: "right",
                fontVariantNumeric: "tabular-nums",
              }}
            >
              {number.format(point.value)}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
  if (visual.type === "table") return table;
  const low = Math.min(0, ...points.map((point) => point.value));
  const high = Math.max(0, ...points.map((point) => point.value));
  const span = high - low || 1;
  const chart =
    visual.type === "bar" ? (
      <div style={{ display: "grid", gap: 14, padding: "12px 0" }}>
        {points.map((point) => (
          <div key={point.label}>
            <div
              style={{
                display: "flex",
                justifyContent: "space-between",
                gap: 12,
                fontSize: 12,
                marginBottom: 5,
              }}
            >
              <span style={{ overflowWrap: "anywhere" }}>{point.label}</span>
              <strong>{number.format(point.value)}</strong>
            </div>
            <div
              style={{
                position: "relative",
                height: 9,
                background: "#f0f4f1",
                borderRadius: 2,
              }}
            >
              <div
                style={{
                  position: "absolute",
                  left: `${((Math.min(0, point.value) - low) / span) * 100}%`,
                  width: `${(Math.abs(point.value) / span) * 100}%`,
                  height: "100%",
                  background: point.value < 0 ? "#b44238" : "#367855",
                  borderRadius: 2,
                }}
              />
            </div>
          </div>
        ))}
      </div>
    ) : (
      <div>
        <svg
          viewBox="0 0 600 230"
          role="img"
          aria-label={`${visual.title}: ${points.length} data points. Data table below.`}
          style={{ width: "100%", display: "block" }}
        >
          <title>{visual.title}</title>
          {[0, 1, 2, 3].map((tick) => (
            <g key={tick}>
              <line
                x1="62"
                x2="580"
                y1={20 + tick * 55}
                y2={20 + tick * 55}
                stroke="#e8eee9"
              />
              <text
                x="52"
                y={24 + tick * 55}
                textAnchor="end"
                fontSize="10"
                fill="#64746b"
              >
                {number.format(high - (span * tick) / 3)}
              </text>
            </g>
          ))}
          <polyline
            fill="none"
            stroke="#367855"
            strokeWidth="2.5"
            strokeLinejoin="round"
            points={points
              .map(
                (point, index) =>
                  `${62 + (index / Math.max(1, points.length - 1)) * 518},${20 + ((high - point.value) / span) * 165}`,
              )
              .join(" ")}
          />
          {points.map((point, index) => (
            <circle
              key={point.label}
              cx={62 + (index / Math.max(1, points.length - 1)) * 518}
              cy={20 + ((high - point.value) / span) * 165}
              r="3"
              fill="#367855"
            >
              <title>
                {point.label}: {number.format(point.value)}
              </title>
            </circle>
          ))}
          <text x="62" y="215" fontSize="10" fill="#64746b">
            {points[0].label.slice(0, 32)}
          </text>
          <text x="580" y="215" textAnchor="end" fontSize="10" fill="#64746b">
            {points.at(-1)?.label.slice(0, 32)}
          </text>
        </svg>
      </div>
    );
  return (
    <>
      {chart}
      <details style={{ marginTop: 16, fontSize: 12 }}>
        <summary style={{ cursor: "pointer", color: "#52685b" }}>
          View data · {points.length} groups shown
        </summary>
        {table}
      </details>
    </>
  );
}

export function buildVisualReportHtml(input: ReportDefinition) {
  const parsed = reportDefinitionSchema.safeParse(input);
  if (!parsed.success) throw new Error(parsed.error.issues[0].message);
  const report = parsed.data;
  if (!report.title.trim() || !report.visuals.length)
    throw new Error("Add a title and at least one visual before saving.");
  const html =
    "<!doctype html>" +
    renderToStaticMarkup(
      <html lang="en">
        <head>
          <meta charSet="utf-8" />
          <meta name="viewport" content="width=device-width, initial-scale=1" />
          <title>{report.title}</title>
          <style>{`*{box-sizing:border-box}body{margin:0;background:#f3f5f3;color:#192c21;font:14px/1.5 system-ui,sans-serif}main{max-width:1120px;margin:auto;padding:48px 32px}h1{font-size:32px;line-height:1.2;margin:12px 0}h2{font-size:15px;margin:0 0 16px}header{margin-bottom:32px;border-bottom:1px solid #dce5de;padding-bottom:24px}.report-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:20px}.visual{background:#fff;padding:24px;border:1px solid #e0e7e1;border-radius:8px;break-inside:avoid;overflow-wrap:anywhere}.wide{grid-column:1/-1}p{margin:0 0 12px}small{color:#52685b}@media(max-width:640px){main{padding:24px 16px}.report-grid{grid-template-columns:1fr}.wide{grid-column:auto}}@media print{body{background:#fff}main{padding:0}details{display:block}summary{display:none}.visual{border-color:#dce5de}header{break-after:avoid}}`}</style>
        </head>
        <body>
          <main>
            <header>
              <small>PERFORMANCE REPORT · SAVED SNAPSHOT</small>
              <h1>{report.title}</h1>
              {report.summary && (
                <p style={{ whiteSpace: "pre-wrap" }}>{report.summary}</p>
              )}
              <small>
                {report.from || "All dates"} — {report.to || "All dates"} ·{" "}
                {report.datasets.map((data) => data.name).join(" / ")}
              </small>
            </header>
            <div className="report-grid">
              {report.visuals.map((visual) => (
                <section
                  key={visual.id}
                  className={`visual ${visual.wide ? "wide" : ""}`}
                >
                  <h2>{visual.title || "Untitled visual"}</h2>
                  <VisualContent report={report} visual={visual} />
                </section>
              ))}
            </div>
          </main>
        </body>
      </html>,
    );
  if (
    new TextEncoder().encode(html).length +
      new TextEncoder().encode(JSON.stringify(report)).length >
    500_000
  )
    throw new Error(
      "This report exceeds 500 KB. Remove a data source, reduce CSV rows, or use fewer visuals.",
    );
  return html;
}
