import { SectionHeader } from "@/client/components/PageHeader";
import { IndexNowCard } from "@/client/features/indexnow/IndexNowCard";
import { SearchPerformanceIndexing } from "./SearchPerformanceIndexing";
import type { Report } from "./SearchPerformanceColumns";

export function SearchIndexingTools({
  projectId,
  rows,
}: {
  projectId: string;
  rows?: Report["strikingDistance"];
}) {
  return (
    <>
      {rows ? (
        <details className="rounded-xl border border-border bg-card p-4">
          <summary className="cursor-pointer font-medium">
            Inspect Google index status for specific URLs
          </summary>
          <div className="mt-4">
            <SearchPerformanceIndexing
              projectId={projectId}
              strikingDistance={rows}
            />
          </div>
        </details>
      ) : null}
      <details
        id="indexnow"
        className="rounded-xl border border-border bg-card p-4"
      >
        <summary className="cursor-pointer font-medium">
          Submit updated URLs to Bing and other engines
        </summary>
        <div className="mt-4 max-w-2xl space-y-3">
          <SectionHeader
            title="Request indexing on Bing and other engines (IndexNow)"
            hint="Works on any website, WordPress or not. Google does not use IndexNow: for Google, use URL Inspection in Search Console."
          />
          <IndexNowCard projectId={projectId} />
        </div>
      </details>
    </>
  );
}
