import { MarkdownAnswer } from "@/client/features/ai-search/components/MarkdownAnswer";
import { formatUrlForDisplay } from "@/client/components/table/url";
import type { BrandLookupResult } from "@/types/schemas/ai-search";

type TopQuery = BrandLookupResult["topQueries"][number];

/**
 * What the AI said for one prompt, the pages it looked at and the searches it
 * ran. DataForSEO returns all of it with the mention that is already paid for,
 * so it is shown here instead of being dropped. Renders nothing when the
 * provider sent none of it (results cached before it was kept).
 */
export function QueryAnswerDetails({ query }: { query: TopQuery }) {
  const answer = query.answer?.trim();
  const pages = query.retrievedPages ?? [];
  const searches = query.fanOutQueries ?? [];
  if (!answer && pages.length === 0 && searches.length === 0) return null;

  return (
    <details className="mt-2 rounded-md border bg-muted/30 p-2 text-sm">
      <summary className="cursor-pointer text-xs font-medium text-muted-foreground">
        See the AI answer
        {query.model ? ` (${query.model})` : ""}
        {pages.length > 0 ? ` · ${pages.length} pages it looked at` : ""}
      </summary>
      <div className="mt-2 space-y-3">
        {answer ? <MarkdownAnswer text={answer} /> : null}
        {searches.length > 0 ? (
          <div>
            <p className="text-xs font-medium">Searches the AI ran</p>
            <ul className="mt-1 list-disc pl-5 text-xs text-muted-foreground">
              {searches.map((search) => (
                <li key={search}>{search}</li>
              ))}
            </ul>
          </div>
        ) : null}
        {pages.length > 0 ? (
          <div>
            <p className="text-xs font-medium">
              Pages the AI looked at (cited or not)
            </p>
            <ol className="mt-1 list-decimal space-y-1.5 pl-5 text-xs">
              {pages.map((page) => (
                <li key={page.url}>
                  <a
                    href={page.url}
                    target="_blank"
                    rel="noreferrer noopener"
                    className="break-words font-medium underline underline-offset-2"
                  >
                    {page.title || formatUrlForDisplay(page.url)}
                  </a>
                  <span className="block text-muted-foreground">
                    {page.domain}
                    {page.description ? ` · ${page.description}` : ""}
                  </span>
                </li>
              ))}
            </ol>
          </div>
        ) : null}
      </div>
    </details>
  );
}
