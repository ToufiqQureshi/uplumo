import { recommendedNextStep } from "./indexingGuidance";
import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { ExternalLink, Search } from "lucide-react";
import { Badge } from "@/client/components/ui/badge";
import { Button } from "@/client/components/ui/button";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/client/components/ui/card";
import { Textarea } from "@/client/components/ui/textarea";
import { getStandardErrorMessage } from "@/client/lib/error-messages";
import { safeHttpUrl } from "@/shared/safe-url";
import { inspectSearchConsoleUrls } from "@/client/features/search-performance/searchPerformanceApi";
import { buildContentRefreshQueue } from "@/client/features/search-performance/contentRefreshQueue";

function humanize(value: string | undefined): string {
  return value?.replaceAll("_", " ").toLowerCase() || "Not available";
}

function formatCrawlTime(value: string | undefined): string {
  if (!value) return "Not available";
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

export function SearchPerformanceIndexing({
  projectId,
  strikingDistance,
}: {
  projectId: string;
  strikingDistance: {
    query: string;
    page: string;
    clicks: number;
    impressions: number;
    position: number;
  }[];
}) {
  const [input, setInput] = useState("");
  const mutation = useMutation({
    mutationFn: (urls: string[]) =>
      inspectSearchConsoleUrls({ projectId, urls }),
  });
  const urls = [
    ...new Set(
      input
        .split(/\r?\n/)
        .map((url) => url.trim())
        .filter(Boolean),
    ),
  ];
  const canSubmit = urls.length > 0 && urls.length <= 10 && !mutation.isPending;
  const suggestedUrls = buildContentRefreshQueue(strikingDistance)
    .slice(0, 10)
    .map((candidate) => candidate.page);

  return (
    <Card className="gap-0">
      <CardHeader className="border-b py-3">
        <CardTitle>Index status and crawl checks</CardTitle>
        <p className="text-sm text-muted-foreground">
          Check up to 10 URLs against Google&apos;s stored index data. Paste
          URLs or load priority pages from the refresh queue.
        </p>
      </CardHeader>
      <CardContent className="space-y-4 pt-4">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-start">
          <Textarea
            aria-label="URLs to inspect"
            value={input}
            onChange={(event) => setInput(event.target.value)}
            placeholder="https://example.com/page"
            rows={3}
            className="min-h-20 resize-y"
          />
          <Button
            type="button"
            className="shrink-0"
            disabled={!canSubmit}
            onClick={() => mutation.mutate(urls)}
          >
            <Search data-icon="inline-start" />
            {mutation.isPending ? "Checking…" : "Check URLs"}
          </Button>
        </div>
        {suggestedUrls.length ? (
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => setInput(suggestedUrls.join("\n"))}
          >
            Load {suggestedUrls.length} priority pages from the refresh queue
          </Button>
        ) : null}
        {urls.length > 10 ? (
          <p className="text-sm text-destructive" role="alert">
            Check up to 10 URLs at a time.
          </p>
        ) : null}
        {mutation.isError ? (
          <p className="text-sm text-destructive" role="alert">
            {getStandardErrorMessage(mutation.error, "URL inspection failed")}
          </p>
        ) : null}
        {mutation.data?.results.length ? (
          <ul className="divide-y divide-border rounded-lg border">
            {mutation.data.results.map(({ url, result, error }) => {
              const index = result?.indexStatusResult;
              const inspectionLink = safeHttpUrl(result?.inspectionResultLink);
              return (
                <li key={url} className="space-y-3 p-4">
                  <div className="flex flex-wrap items-start justify-between gap-2">
                    <span className="min-w-0 break-all text-sm font-medium">
                      {url}
                    </span>
                    {inspectionLink ? (
                      <a
                        href={inspectionLink}
                        target="_blank"
                        rel="noreferrer"
                        className="inline-flex shrink-0 items-center gap-1 text-xs text-primary underline-offset-4 hover:underline"
                      >
                        Open in Search Console{" "}
                        <ExternalLink className="size-3" />
                      </a>
                    ) : null}
                  </div>
                  {error ? (
                    <p className="text-sm text-destructive">{error}</p>
                  ) : index ? (
                    <div className="grid gap-3 text-sm sm:grid-cols-2 lg:grid-cols-3">
                      <IndexValue
                        label="Index coverage"
                        value={index.coverageState}
                      />
                      <IndexValue label="Verdict" value={index.verdict} />
                      <IndexValue
                        label="Last crawl"
                        value={formatCrawlTime(index.lastCrawlTime)}
                      />
                      <IndexValue
                        label="Crawl access"
                        value={humanize(index.robotsTxtState)}
                      />
                      <IndexValue
                        label="Indexing"
                        value={humanize(index.indexingState)}
                      />
                      <IndexValue
                        label="Google fetch"
                        value={humanize(index.pageFetchState)}
                      />
                      <div className="sm:col-span-2 lg:col-span-3">
                        <IndexValue
                          label="Suggested next step"
                          value={recommendedNextStep(index)}
                        />
                      </div>
                      {index.googleCanonical ? (
                        <IndexValue
                          label="Google canonical"
                          value={index.googleCanonical}
                        />
                      ) : null}
                      {index.userCanonical ? (
                        <IndexValue
                          label="Declared canonical"
                          value={index.userCanonical}
                        />
                      ) : null}
                    </div>
                  ) : (
                    <Badge variant="secondary">No index status returned</Badge>
                  )}
                </li>
              );
            })}
          </ul>
        ) : null}
        <p className="text-xs text-muted-foreground">
          This shows Google&apos;s stored index data; it does not request
          indexing or test a live page.
        </p>
      </CardContent>
    </Card>
  );
}

function IndexValue({ label, value }: { label: string; value?: string }) {
  return (
    <div className="min-w-0 space-y-1">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="break-words font-medium">{value || "Not available"}</div>
    </div>
  );
}
