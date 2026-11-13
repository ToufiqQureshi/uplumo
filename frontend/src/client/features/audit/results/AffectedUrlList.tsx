import { useMemo } from "react";
import { useParams } from "@tanstack/react-router";
import { EditPageButton } from "@/client/features/content-editor/EditPageButton";
import type { AuditResultsData } from "@/client/features/audit/results/types";

type AuditIssueRow = AuditResultsData["issues"][number];

export const MAX_RENDERED_URLS = 100;

/**
 * The pages an issue was found on, each with its details and, when the page
 * loaded, a button that opens its text in the Content Editor.
 */
export function AffectedUrlList({
  issues,
  pages,
}: {
  issues: AuditIssueRow[];
  pages: AuditResultsData["pages"];
}) {
  // Only pages that loaded can be opened in the editor.
  const { projectId } = useParams({ strict: false });
  const editable = new Set(
    pages
      .filter(
        (p) =>
          p.statusCode !== null && p.statusCode >= 200 && p.statusCode < 300,
      )
      .map((p) => p.url),
  );
  const rendered = issues.slice(0, MAX_RENDERED_URLS);
  const remaining = issues.length - rendered.length;

  return (
    <div className="max-h-[320px] overflow-y-auto rounded border border-border bg-card">
      {rendered.map((issue) => (
        <div
          key={issue.id}
          className="px-3 py-1.5 text-sm flex flex-col gap-0.5 border-b border-border last:border-b-0"
        >
          <div className="flex min-w-0 items-center gap-2">
            <a
              className="min-w-0 flex-1 truncate text-foreground/80 hover:underline"
              href={issue.pageUrl}
              target="_blank"
              rel="noreferrer"
              title={issue.pageUrl}
            >
              {issue.pageUrl}
            </a>
            {projectId && editable.has(issue.pageUrl) ? (
              <EditPageButton
                projectId={projectId}
                url={issue.pageUrl}
                variant="ghost"
                label="Edit"
              />
            ) : null}
          </div>
          <IssueDetails detailsJson={issue.detailsJson} />
        </div>
      ))}
      {remaining > 0 && (
        <div className="px-3 py-2 text-xs text-muted-foreground">
          …and {remaining} more — export the issues CSV for the full list.
        </div>
      )}
    </div>
  );
}

// AuditRepository is the only writer and always stores JSON.stringify of a
// details object, so the parse cannot throw.
function parseDetails(detailsJson: string): Array<[string, unknown]> | null {
  const parsed: unknown = JSON.parse(detailsJson);
  if (typeof parsed === "object" && parsed !== null && !Array.isArray(parsed)) {
    return Object.entries(parsed);
  }
  return null;
}

function IssueDetails({ detailsJson }: { detailsJson: string | null }) {
  const details = useMemo(
    () => (detailsJson ? parseDetails(detailsJson) : null),
    [detailsJson],
  );

  if (!details) return null;

  const entries = details.filter(
    ([, value]) => value !== null && value !== undefined,
  );
  if (entries.length === 0) return null;

  return (
    <span className="text-xs text-muted-foreground truncate">
      {entries
        .map(([key, value]) => {
          // Redirect hops are a path; other lists (blocked AI crawlers) are not.
          const rendered = Array.isArray(value)
            ? value.join(key === "hops" ? " → " : ", ")
            : String(value);
          return `${key}: ${rendered}`;
        })
        .join(" · ")}
    </span>
  );
}
