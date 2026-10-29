import { Link } from "@tanstack/react-router";
import { SafeExternalLink } from "@/client/components/SafeExternalLink";
import { Button } from "@/client/components/ui/button";
import type { CiteFixReport } from "./citeFixApi";

type Topics = NonNullable<CiteFixReport["topics"]>;
type Page = Topics["clusters"][number]["pages"][number];

function PageList({ label, pages }: { label: string; pages: Page[] }) {
  if (pages.length === 0) return null;
  return (
    <div>
      <p className="font-medium">{label}</p>
      <ul className="ml-4 list-disc text-muted-foreground">
        {pages.map((page) => (
          <li key={page.url}>
            <SafeExternalLink
              url={page.url}
              label={page.title || page.url}
              className="hover:underline"
            />
          </li>
        ))}
      </ul>
    </div>
  );
}

/** Read-only topic estimates from the latest audit; opening a draft is free. */
export function TopicsSection({
  projectId,
  topics,
}: {
  projectId: string;
  topics: Topics;
}) {
  return (
    <section
      aria-labelledby="your-topics"
      className="flex flex-col gap-4 rounded-xl border bg-card p-5"
    >
      <div>
        <h2 id="your-topics" className="font-serif text-2xl">
          Your topics
        </h2>
        <p className="text-sm text-muted-foreground">
          {topics.score === null
            ? "No readable pages to score yet."
            : `Topic coverage: ${topics.score}/100 across ${topics.pagesAnalyzed} pages.`}{" "}
          {topics.limited
            ? "This sample includes the first 500 readable pages."
            : null}{" "}
          {topics.clustersLimited ? "Showing up to 20 topics." : null} This is
          an estimate from the latest scan, not a search ranking.
        </p>
      </div>
      {topics.recommendations.length ? (
        <ul className="ml-4 list-disc text-sm">
          {topics.recommendations.map((recommendation) => (
            <li key={recommendation}>{recommendation}</li>
          ))}
        </ul>
      ) : null}
      <div className="flex flex-col divide-y">
        {topics.clusters.map((cluster) => (
          <div key={cluster.topic} className="flex flex-col gap-2 py-3 text-sm">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <h3 className="font-medium">{cluster.topic}</h3>
              {!cluster.pillar ? (
                <Button
                  variant="outline"
                  size="sm"
                  nativeButton={false}
                  render={
                    <Link
                      to="/p/$projectId/content-editor"
                      params={{ projectId }}
                      search={{ keyword: cluster.topic.slice(0, 200) }}
                    />
                  }
                >
                  Start pillar draft
                </Button>
              ) : null}
            </div>
            <p className="text-muted-foreground">
              {cluster.pageCount} {cluster.pageCount === 1 ? "page" : "pages"} ·{" "}
              {Math.round(cluster.interlinkRatio * 100)}% link to a related page
              · Pillar: {cluster.pillar ? "yes" : "no"}
            </p>
            <details>
              <summary className="cursor-pointer">
                View pages and issues
              </summary>
              <div className="mt-2 flex flex-col gap-2">
                <PageList label="Pages" pages={cluster.pages} />
                <PageList
                  label="No inbound links in this sample"
                  pages={cluster.orphans}
                />
                <PageList label="Thin pages" pages={cluster.thin} />
                <PageList label="Duplicate pages" pages={cluster.duplicates} />
              </div>
            </details>
          </div>
        ))}
      </div>
    </section>
  );
}
