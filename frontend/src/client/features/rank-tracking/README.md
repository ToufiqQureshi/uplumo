# Rank tracking

Configures keywords, devices/locations and manual or scheduled rank checks with history.

Weekly is the default. The setup shows the estimated monthly credits before
save, counts desktop plus mobile as two checks, and leaves discovered keyword
suggestions unselected so saving a domain cannot silently add paid work.

Read [flow.md](flow.md) for the request path and [decision.md](decision.md) for the rules that protect user data, provider spend and customer trust.

## Code map

- `RankTrackingConfigModal.tsx`
- `KeywordSuggestionStep.tsx`
- `rankTrackingApi.ts`
- `useRankCheckTrigger.ts`
- `useRankRunPolling.ts`
- `backend/internal/ranktracking`

Optional tracker suggestions: configuration save → optional choice/credit hint
(no lookup) → explicit Find existing keywords sends `userInitiated: true` →
existing authorization/paid gate and validation → scoped coalescing/cache → one
ranked-keywords task on a miss. Skip and manual entry buy no suggestion task.
No automatic paid retry; a failure keeps the manual path. See the feature's
`decision.md` for compatibility and one-replica constraints.
