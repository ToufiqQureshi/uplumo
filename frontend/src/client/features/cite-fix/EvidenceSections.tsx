import { Link } from "@tanstack/react-router";
import { Button } from "@/client/components/ui/button";
import { SafeExternalLink } from "@/client/components/SafeExternalLink";
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

type RivalComparison = CiteFixReport["rivals"][number];
type RivalSignals = RivalComparison["rival"]["signals"];

export function RivalComparisonsSection({
  proof,
  comparisons,
}: {
  proof: CiteFixReport["proof"];
  comparisons: CiteFixReport["rivals"];
}) {
  if (!proof?.misses.length) return null;
  return (
    <section
      aria-labelledby="cite-fix-rival-title"
      className="flex flex-col gap-3"
    >
      <div>
        <h2 id="cite-fix-rival-title" className="font-serif text-2xl">
          Why they got cited, you didn&apos;t
        </h2>
        <p className="text-sm text-muted-foreground">
          Free page signals from the top cited rival URLs for missed questions.
          We respect robots.txt; app-shell pages are reported without paid
          rendering.
        </p>
      </div>
      <div className="flex flex-col gap-3">
        {proof.misses.map((miss, index) => {
          const matching = comparisons.filter(
            (comparison) => comparison.prompt === miss.prompt,
          );
          return (
            <article
              key={`${miss.prompt}:${miss.engine}`}
              id={`cite-fix-rival-${index}`}
              className="rounded-xl border bg-card p-4"
            >
              <h3 className="font-medium">“{miss.prompt}”</h3>
              <p className="mb-3 text-xs text-muted-foreground">
                {miss.engine} cited {miss.cited.join(", ") || "other sites"}
              </p>
              {matching.length ? (
                <div className="flex flex-col gap-4">
                  {matching.map((comparison) => (
                    <div
                      key={comparison.rival.url}
                      className="flex flex-col gap-2 border-t pt-3"
                    >
                      <p className="text-xs text-muted-foreground">
                        Evidence from {comparison.engine}
                      </p>
                      <p className="flex flex-wrap items-center gap-2 text-sm">
                        <span className="font-medium">Rival:</span>
                        <SafeExternalLink
                          url={comparison.rival.url}
                          label={comparison.rival.title || comparison.rival.url}
                          className="inline-flex items-center gap-1 text-primary underline"
                        />
                        <span className="text-muted-foreground">
                          ({comparison.rival.domain})
                        </span>
                      </p>
                      {comparison.rival.appShell ? (
                        <p className="rounded-md bg-muted px-3 py-2 text-xs text-muted-foreground">
                          This rival is an app shell. It was not rendered
                          because rendering costs credits.
                        </p>
                      ) : null}
                      <div className="grid gap-3 md:grid-cols-2">
                        <SignalSummary
                          title="Yours"
                          signals={comparison.yours?.signals ?? null}
                        />
                        <SignalSummary
                          title="Theirs"
                          signals={comparison.rival.signals}
                        />
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">
                  No rival page could be compared for this question. The report
                  fetches at most three public pages and skips pages blocked by
                  robots.txt.
                </p>
              )}
            </article>
          );
        })}
      </div>
    </section>
  );
}

function SignalSummary({
  title,
  signals,
}: {
  title: string;
  signals: RivalSignals | null;
}) {
  const rows = signals
    ? [
        ["Answer first", `${signals.answerFirstWords} words`],
        ["Author", signals.author ? "Found" : "Not found"],
        ["Date", signals.date || "Not found"],
        ["Schema", signals.schema.join(", ") || "Not found"],
        ["Statistics", String(signals.statistics)],
        ["FAQ", signals.faq ? "Found" : "Not found"],
        ["Word count", signals.wordCount.toLocaleString()],
      ]
    : [["Page", "No matching page in the latest audit"]];
  return (
    <div className="rounded-lg bg-muted/40 p-3">
      <h4 className="mb-2 text-sm font-medium">{title}</h4>
      <dl className="space-y-1 text-xs">
        {rows.map(([label, value]) => (
          <div key={label} className="flex justify-between gap-3">
            <dt className="text-muted-foreground">{label}</dt>
            <dd className="text-right">{value}</dd>
          </div>
        ))}
      </dl>
    </div>
  );
}
