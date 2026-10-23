import { MarkdownAnswer } from "@/client/features/ai-search/components/MarkdownAnswer";
import { formatUrlForDisplay } from "@/client/components/table/url";
import type { BrandLookupResult } from "@/types/schemas/ai-search";

type TopQuery = BrandLookupResult["topQueries"][number];

function safeExternalHref(raw: string): string | null {
  try {
    const url = new URL(raw);
    return (url.protocol === "http:" || url.protocol === "https:") &&
      !url.username &&
      !url.password
      ? url.toString()
      : null;
  } catch {
    return null;
  }
}

function SourceLink({ url, title }: { url: string; title: string | null }) {
  const href = safeExternalHref(url);
  const label = title || (href ? formatUrlForDisplay(href) : url);
  if (!href) return <span className="break-words font-medium">{label}</span>;
  return (
    <a
      href={href}
      target="_blank"
      rel="noreferrer noopener"
      className="break-words font-medium underline underline-offset-2"
    >
      {label}
    </a>
  );
}

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
  const cited = query.citedSources ?? [];
  const brands = query.brandsMentioned ?? [];
  if (
    !answer &&
    pages.length === 0 &&
    searches.length === 0 &&
    cited.length === 0 &&
    brands.length === 0
  )
    return null;

  return (
    <details className="mt-2 rounded-md border bg-muted/30 p-2 text-sm">
      <summary className="cursor-pointer text-xs font-medium text-muted-foreground">
        See the AI answer
        {query.model ? ` (${query.model})` : ""}
        {pages.length > 0 ? ` · ${pages.length} pages it looked at` : ""}
        {brands.length > 0 ? ` · ${brands.length} brands it named` : ""}
      </summary>
      <div className="mt-2 space-y-3">
        {answer ? <MarkdownAnswer text={answer} /> : null}
        {brands.length > 0 ? (
          <div>
            <p className="text-xs font-medium">Brands the AI named alongside</p>
            <ul className="mt-1 list-disc pl-5 text-xs text-muted-foreground">
              {brands.slice(0, 10).map((brand) => (
                <li key={brand}>{brand}</li>
              ))}
            </ul>
          </div>
        ) : null}
        {cited.length > 0 ? (
          <div>
            <p className="text-xs font-medium">Sources the AI cited</p>
            <ol className="mt-1 list-decimal space-y-1 pl-5 text-xs">
              {cited.map((source) => (
                <li key={source.url}>
                  <SourceLink url={source.url} title={source.title} />
                  <span className="block text-muted-foreground">
                    {source.domain}
                  </span>
                </li>
              ))}
            </ol>
          </div>
        ) : null}
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
                  <SourceLink url={page.url} title={page.title} />
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
