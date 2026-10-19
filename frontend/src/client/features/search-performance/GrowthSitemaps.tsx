import { useMutation } from "@tanstack/react-query";
import { Button } from "@/client/components/ui/button";
import { QueryError } from "@/client/components/QueryState";
import { safeHttpUrl } from "@/shared/safe-url";
import { getGrowthSitemaps } from "./growthApi";

export function GrowthSitemaps({ projectId }: { projectId: string }) {
  const sitemaps = useMutation({
    mutationFn: () => getGrowthSitemaps(projectId),
    retry: false,
  });
  return (
    <section className="min-w-0 space-y-3">
      <h4 className="font-medium">Can Google read your submitted sitemaps?</h4>
      <Button
        className="h-auto min-h-9 max-w-full whitespace-normal py-2 text-left"
        size="sm"
        variant="outline"
        disabled={sitemaps.isPending}
        onClick={() => sitemaps.mutate()}
      >
        {sitemaps.isPending
          ? "Reading sitemap status…"
          : "Check submitted sitemap status"}
      </Button>
      {sitemaps.isError ? (
        <QueryError
          error={sitemaps.error}
          fallback="Could not read sitemap status."
        />
      ) : null}
      {sitemaps.data ? (
        <div className="space-y-2 text-xs">
          <p className="text-muted-foreground">
            Sitemap submission does not confirm that a page is indexed.
          </p>
          <details>
            <summary className="cursor-pointer font-medium">
              What this status means
            </summary>
            <p className="mt-2 leading-relaxed">{sitemaps.data.message}</p>
          </details>
          {sitemaps.data.truncated ? (
            <p>Only the first 200 sitemap entries are shown.</p>
          ) : null}
          {sitemaps.data.sitemaps.length ? (
            sitemaps.data.sitemaps.map((s) => {
              const url = safeHttpUrl(s.path);
              return (
                <article
                  key={s.path}
                  className="min-w-0 space-y-3 rounded-xl border border-border p-3"
                >
                  <p className="break-all">
                    {url ? (
                      <a
                        href={url}
                        target="_blank"
                        rel="noreferrer noopener"
                        className="underline"
                      >
                        {s.path}
                      </a>
                    ) : (
                      s.path
                    )}
                    {s.isIndex ? " · Sitemap index" : ""}
                  </p>
                  <p className="text-muted-foreground">
                    {s.pending
                      ? "Google processing pending"
                      : "Not marked pending"}
                  </p>
                  <div className="grid grid-cols-2 gap-2 py-2">
                    <p className="rounded-lg bg-muted/50 p-2">
                      Errors:{" "}
                      <strong className="tabular-nums">{s.errors}</strong>
                    </p>
                    <p className="rounded-lg bg-muted/50 p-2">
                      Warnings:{" "}
                      <strong className="tabular-nums">{s.warnings}</strong>
                    </p>
                  </div>
                  <details>
                    <summary className="cursor-pointer font-medium">
                      Submission and download details
                    </summary>
                    <div className="mt-2 space-y-2 leading-relaxed">
                      <p>
                        {s.contents?.length === 0
                          ? "Submitted counts unavailable"
                          : `${s.submitted} submitted entries; these are not unique indexed pages.`}
                      </p>
                      <p>
                        {s.contents
                          ?.map(
                            (content) =>
                              `${content.type || "Unspecified type"}: ${content.submitted}`,
                          )
                          .join(" · ")}
                      </p>
                      <p>Last submitted: {s.lastSubmitted || "Unknown"}.</p>
                      <p>Last downloaded: {s.lastDownloaded || "Unknown"}.</p>
                    </div>
                  </details>
                  {s.errors > 0 ? (
                    <p>
                      Open the Sitemaps report in Search Console for the
                      specific parsing or fetch reason. Fix that issue, then
                      confirm Google can read the file.
                    </p>
                  ) : null}
                  {s.warnings > 0 ? (
                    <p>
                      Review Google's warning details; warnings do not
                      necessarily prevent all URLs from being processed.
                    </p>
                  ) : null}
                </article>
              );
            })
          ) : (
            <p>
              No submitted sitemap entries were returned for this property. This
              does not prove Google cannot discover or index your pages.
            </p>
          )}
          <a
            className="underline"
            target="_blank"
            rel="noreferrer noopener"
            href={`https://search.google.com/search-console/sitemaps?resource_id=${encodeURIComponent(sitemaps.data.siteUrl)}`}
          >
            Review sitemap details in Google Search Console
          </a>
        </div>
      ) : null}
    </section>
  );
}
