import type { GrowthPage, GrowthReport } from "./growthApi";

type Metrics = GrowthPage["current"];
function sum(rows: Metrics[], field: keyof Metrics): number | null {
  if (!rows.length || rows.some((row) => row[field] == null)) return null;
  return rows.reduce((total, row) => total + (row[field] ?? 0), 0);
}
const weighted = (
  field: keyof Metrics,
  rows: Metrics[],
  weight: "sessions" | "impressions",
) => {
  const denominator = sum(rows, weight);
  if (!denominator || rows.some((row) => row[field] == null)) return null;
  return (
    rows.reduce(
      (total, row) => total + (row[field] ?? 0) * (row[weight] ?? 0),
      0,
    ) / denominator
  );
};

export function intelligenceMetrics(report: GrowthReport, previous = false) {
  const metrics = report.pages.map((page) =>
    previous ? page.previous : page.current,
  );
  const complete = metrics.length > 0 && metrics.every((row) => row !== null);
  const search = complete
    ? metrics.filter((row): row is Metrics => row !== null)
    : [];
  const joined = search.filter((row) => row.sessions !== null);
  const clicks = sum(search, "clicks");
  const impressions = sum(search, "impressions");
  const sessions = sum(joined, "sessions");
  const limited = report.limited;
  return {
    clicks,
    impressions,
    ctr: impressions && clicks !== null ? clicks / impressions : null,
    position: weighted("position", search, "impressions"),
    sessions,
    engagement: limited ? null : weighted("engagementRate", joined, "sessions"),
    events: limited ? null : sum(joined, "keyEvents"),
    outcomeRate: limited
      ? null
      : weighted(
          report.goalEvent ? "goalRate" : "keyEventRate",
          joined,
          "sessions",
        ),
    revenue: limited ? null : sum(joined, "revenue"),
    joinedPages: joined.length,
    // Compare the same Google-organic landing-page cohort, not a changed match set.
    comparableAnalytics:
      complete &&
      report.pages.every(
        (page) =>
          (page.current.sessions !== null) ===
          (page.previous?.sessions != null),
      ),
  };
}
