import { SafeExternalLink } from "@/client/components/SafeExternalLink";
import type { VisibilityMetrics } from "../trackingApi";

type Source = VisibilityMetrics["domains"][number];

const date = (value: string) =>
  new Date(value).toLocaleDateString("en-IN", {
    day: "numeric",
    month: "short",
    year: "numeric",
  });
const rank = (value: number | null) =>
  value === null ? "Not cited" : `#${value.toFixed(1)}`;

function Sources({
  label,
  rows,
  pages,
}: {
  label: string;
  rows: Source[];
  pages: boolean;
}) {
  if (!rows.length) return null;
  return (
    <details>
      <summary className="cursor-pointer font-medium">{label}</summary>
      <div className="overflow-x-auto">
        <table className="mt-2 w-full text-left text-sm">
          <thead>
            <tr>
              <th>Source</th>
              <th>First seen</th>
              <th>Last seen</th>
              <th>Average position</th>
              <th>Change</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr key={row.name} className="border-t">
                <td className="break-all py-2">
                  {pages ? (
                    <SafeExternalLink
                      url={row.name}
                      label={row.name}
                      className="hover:underline"
                    />
                  ) : (
                    row.name
                  )}
                </td>
                <td>{date(row.firstSeen)}</td>
                <td>{date(row.lastSeen)}</td>
                <td>{rank(row.averageRank)}</td>
                <td>
                  {row.momentum === "unavailable"
                    ? "Not comparable"
                    : row.momentum}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </details>
  );
}

/** Derived only from saved answers, with no new check or provider call. */
export function VisibilityInsights({
  metrics,
}: {
  metrics: VisibilityMetrics;
}) {
  if (!metrics.runs) return null;
  return (
    <div className="flex flex-col gap-3 rounded-lg border p-4">
      <h3 className="font-medium">Citation history</h3>
      <p className="text-xs text-muted-foreground">
        History from {metrics.runs} saved checks with the same questions,
        engines and market, ending with the latest saved check.
        {metrics.historyLimited
          ? " First and last seen use the latest 30 checks."
          : " First and last seen use these comparable checks."}
        {metrics.sourcesLimited ? " The source list is limited." : null} Source
        positions use the saved source list. Answer positions use only saved
        citation URLs found in the answer text. Changes compare complete checks
        only.
      </p>
      <Sources label="Cited domains" rows={metrics.domains} pages={false} />
      <Sources label="Cited pages" rows={metrics.pages} pages />
      {metrics.prompts.length ? (
        <details>
          <summary className="cursor-pointer font-medium">
            Your citation position by question
          </summary>
          <ul className="mt-2 space-y-2 text-sm">
            {metrics.prompts.map((item) => (
              <li key={`${item.prompt}:${item.engine}`}>
                {item.prompt} ({item.engine}): latest {rank(item.latestRank)},
                average {rank(item.averageRank)} across {item.citedAnswers}{" "}
                cited of {item.answers} answers in the source list. Answer
                position:{" "}
                {item.latestAnswerRank === null
                  ? "unavailable"
                  : rank(item.latestAnswerRank)}
                , average{" "}
                {item.averageAnswerRank === null
                  ? "unavailable"
                  : rank(item.averageAnswerRank)}{" "}
                across {item.inlineAnswers} answers with a matching URL.
              </li>
            ))}
          </ul>
        </details>
      ) : null}
    </div>
  );
}
