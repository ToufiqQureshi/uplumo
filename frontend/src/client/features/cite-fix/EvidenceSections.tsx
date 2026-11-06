import { Link } from "@tanstack/react-router";
import { Button } from "@/client/components/ui/button";
import { BrandLookupResults } from "@/client/features/ai-search/components/BrandLookupResults";
import type { CiteFixReport } from "./citeFixApi";

type SavedMentions = NonNullable<CiteFixReport["mentions"]>;

export function AIMentionsSection({
  projectId,
  mentions,
}: {
  projectId: string;
  mentions: SavedMentions | null;
}) {
  return (
    <section
      aria-labelledby="cite-fix-mentions"
      className="flex flex-col gap-3"
    >
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 id="cite-fix-mentions" className="font-serif text-2xl">
            Who AI mentions for your topics
          </h2>
          {mentions ? (
            <p className="text-sm text-muted-foreground">
              Saved from AI Visibility on{" "}
              {new Date(mentions.fetchedAt).toLocaleDateString()}. Reading this
              snapshot is free. Refresh it on AI Visibility at its listed price.
            </p>
          ) : (
            <p className="text-sm text-muted-foreground">
              Cite Fix reads your saved AI Visibility lookup and never buys a
              mentions refresh here.
            </p>
          )}
        </div>
        <Button
          size="sm"
          variant="outline"
          nativeButton={false}
          render={
            <Link to="/p/$projectId/brand-lookup" params={{ projectId }} />
          }
        >
          {mentions ? "Refresh in AI Visibility" : "Open AI Visibility"}
        </Button>
      </div>
      {mentions ? (
        <BrandLookupResults
          result={mentions.result}
          projectId={projectId}
          countryLabel="the saved lookup's selected market"
        />
      ) : (
        <p className="rounded-lg border border-dashed p-4 text-sm text-muted-foreground">
          Run a Brand Lookup in AI Visibility to see your domain beside the top
          mentioned sites and pages.
        </p>
      )}
    </section>
  );
}
