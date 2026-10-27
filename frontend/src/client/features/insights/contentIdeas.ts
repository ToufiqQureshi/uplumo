import { sortBy } from "remeda";

type CompetitorKeyword = {
  keyword: string;
  position: number | null;
  searchVolume: number | null;
  traffic: number | null;
  url: string | null;
  relativeUrl: string | null;
};
type OwnKeyword = { keyword: string; position: number | null };

export type IdeaPage = {
  url: string;
  path: string;
  traffic: number;
  /** "gap": you rank for none of its keywords in the top 30; "improve": you rank 11-30 for at least one. */
  kind: "gap" | "improve";
  keywords: {
    keyword: string;
    volume: number;
    theirs: number | null;
    yours: number | null;
  }[];
};

/**
 * Groups a competitor's ranking keywords by page and compares each keyword
 * with the customer's own rankings. Pages where the customer already ranks in
 * the top 10 for the page's main keyword are dropped: there is nothing to win.
 */
export function contentIdeas(
  competitor: CompetitorKeyword[],
  own: OwnKeyword[],
): IdeaPage[] {
  const mine = new Map(
    own.map((k) => [k.keyword.trim().toLowerCase(), k.position] as const),
  );
  const pages = new Map<string, IdeaPage>();
  for (const k of competitor) {
    if (!k.url) continue;
    const page = pages.get(k.url) ?? {
      url: k.url,
      path: k.relativeUrl ?? k.url,
      traffic: 0,
      kind: "gap" as const,
      keywords: [],
    };
    page.traffic += k.traffic ?? 0;
    page.keywords.push({
      keyword: k.keyword,
      volume: k.searchVolume ?? 0,
      theirs: k.position,
      yours: mine.get(k.keyword.trim().toLowerCase()) ?? null,
    });
    pages.set(k.url, page);
  }
  const out: IdeaPage[] = [];
  for (const page of pages.values()) {
    page.keywords = sortBy(page.keywords, [(k) => k.volume, "desc"]);
    const ranked = page.keywords.map((k) => k.yours).filter((p) => p !== null);
    if (page.keywords[0].yours !== null && page.keywords[0].yours <= 10)
      continue;
    page.kind = ranked.some((p) => p > 10 && p <= 30) ? "improve" : "gap";
    out.push(page);
  }
  return sortBy(out, [(p) => p.traffic, "desc"]);
}
