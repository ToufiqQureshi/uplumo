# AI Visibility and prompt research

Brand Lookup compares brand mentions and cited pages across supported answer engines; Prompt Explorer runs a prompt against selected models.

Brand Lookup runs for a chosen country (`loc` in the URL, default the
project's market; the lookup waits for the market rather than buying the US).
Its language is the country's default (`getLanguageCode`). ChatGPT data is US
English only and is labelled "(US)". `components/BrandLookupInsights.tsx`
turns the lookup into three answers: where AI recommends the business to people
who did not search its name (unbranded prompts), the sites AI reads about it
(`sourceSites`, the brand's own site left out) and the brands it names beside
it, each with what to do; a lookup with no mentions shows three first steps and
a link to the tracker. All of it comes with the lookup; nothing extra is bought.

Paid Brand Lookup and Prompt Explorer queries disable automatic retries,
window-focus refetch and reconnect refetch. Explicit Retry remains a new lookup;
returning to a browser window must not repeat a failed paid action.

Read [flow.md](flow.md) for the request path and [decision.md](decision.md) for the rules that protect user data, provider spend and customer trust.

## Code map

- `BrandLookupPage.tsx`
- `PromptExplorerPage.tsx`
- `aiSearchApi.ts`
- `backend/internal/aisearch/handler.go`
- `backend/internal/aisearch/service.go`
- `backend/internal/aisearch/provider.go`

- `components/BrandVisibilityHistoryCard.tsx` (saved monthly trend and separate analytics referrals)

## Ongoing question tracking

`PromptTrackingPage.tsx`, `TrackingForm.tsx` and `trackingApi.ts` expose the saved
tracker in project navigation. React Query caches history; only active checks
poll every 15 seconds. Typing changes local draft state. Cost review is a free,
explicit action; changes invalidate its approval. Saving or running collection
requires the reviewed ceiling. A page refresh/history export cannot collect
paid answers. Saved answers render as text and safe source links.

Mention/citation summaries exclude failed and empty answers. Trend warnings
compare complete identical scopes; market/question changes must not imply a
visibility decline. The comparison is a sampled answer rate, not AI traffic,
search demand, guaranteed ranking or revenue attribution.

Source titles may be null; link labels fall back to the safe URL. Vitest discovers both `.test.ts` and `.test.tsx`, including the empty-history rendering regression.

`whyNotCited.ts` and `components/WhyNotCitedCard.tsx` explain why AI did not cite the site, from a check's saved answers only (no credits): listing sites, competitor pages, named-not-linked, recommended rivals and community sites, each with one action.
