import type { AgentMode, AgentOffer, CiteFixReport } from "./citeFixApi";

/** Pages at or above this score are "already good" and not preselected. */
const GOOD_SCORE = 80;

/** The weakest pages that still need work, up to the offer's default. */
export function defaultSelection(
  pages: CiteFixReport["pages"],
  offer: AgentOffer,
): string[] {
  return pages
    .filter((page) => page.score < GOOD_SCORE)
    .slice(0, offer.defaultPages)
    .map((page) => page.url);
}

/** The most a batch can cost: every page at the mode's cap. */
export function batchMaxCredits(
  pages: number,
  mode: AgentMode,
  offer: AgentOffer,
): number {
  return (
    pages * (mode === "deep" ? offer.deepMaxCredits : offer.quickMaxCredits)
  );
}
