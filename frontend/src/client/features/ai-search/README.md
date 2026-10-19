# AI Visibility and prompt research

Brand Lookup compares brand mentions and cited pages across supported answer engines; Prompt Explorer runs a prompt against selected models.

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
