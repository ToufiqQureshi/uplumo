import { SectionHeader } from "@/client/components/PageHeader";
import { IndexNowCard } from "@/client/features/indexnow/IndexNowCard";
import { SearchPerformanceIndexing } from "./SearchPerformanceIndexing";
import type { RefreshQuery } from "./contentRefreshQueue";

export function SearchPerformanceTools({
  projectId,
  rows,
}: {
  projectId: string;
  rows?: RefreshQuery[];
}) {
  return (
    <>
      {rows ? (
        <details className="rounded-xl border border-border bg-card">
          <summary className="cursor-pointer px-4 py-4 font-medium">
            Index status and crawl tools
          </summary>
          <div className="px-4 pb-4">
            <SearchPerformanceIndexing
              projectId={projectId}
              strikingDistance={rows}
            />
          </div>
        </details>
      ) : null}
      <section id="indexnow" className="max-w-2xl space-y-3 pt-4">
        <SectionHeader
          title="Request indexing on Bing and other engines (IndexNow)"
          hint="Works on any website, WordPress or not. Google does not use IndexNow: for Google, use URL Inspection in Search Console."
        />
        <IndexNowCard projectId={projectId} />
      </section>
    </>
  );
}
