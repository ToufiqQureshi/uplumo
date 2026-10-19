import { useState } from "react";
import { Button } from "@/client/components/ui/button";
import { readSearchContext } from "./searchContext";

export function SearchContextPanel({ projectId }: { projectId: string }) {
  const [context, setContext] = useState(() => readSearchContext(projectId));
  if (!context) return null;
  return (
    <aside
      className="space-y-2 rounded-xl border bg-muted/30 p-4 text-sm"
      aria-label="Existing page improvement context"
    >
      <div className="flex items-start justify-between gap-3">
        <div>
          <h2 className="font-medium">Improve an existing page</h2>
          <a
            className="break-all text-xs underline"
            href={context.page}
            target="_blank"
            rel="noreferrer noopener"
          >
            {context.page}
          </a>
        </div>
        <Button size="sm" variant="ghost" onClick={() => setContext(null)}>
          Dismiss context
        </Button>
      </div>
      <p className="text-xs text-muted-foreground">
        Search evidence from {context.startDate} to {context.endDate}. Use your
        existing page as the starting point; this handoff does not import,
        overwrite or publish content.
      </p>
      <p>
        Related queries:{" "}
        {context.queries.join(" · ") || "No query rows returned"}
      </p>
      <ul className="list-disc space-y-1 pl-4 text-xs">
        {context.reasons.map((reason) => (
          <li key={reason}>{reason}</li>
        ))}
      </ul>
      <p className="text-xs text-muted-foreground">
        Paste the relevant existing content into a new draft, or open its saved
        draft. Review suggested edits with the editor's selection-based AI tools
        before applying them.
      </p>
    </aside>
  );
}
