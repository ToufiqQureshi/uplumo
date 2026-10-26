# DataForSEO audit (26 October 2026)

Question from the owner: "Is DataForSEO billing me for calls we did not need?"
This audit reads production usage (read-only aggregate `SELECT`s on
`go_dataforseo_usage`, `go_ai_tracking_*`), the paid browser hooks and the
tracker worker. It fixes one bug (F1) in the same PR.

## Verdict

- **No runaway or duplicate buying found.** Production total since 14 Oct is
  $4.17 across 3 test workspaces. Calls within 10 seconds of each other on
  the same endpoint are explained by designed fan-out (8 page reads per
  brief, rank-tracking chunks, the old six-task Brand Lookup before 22 Oct,
  Local SEO map grids before its removal), not by retries.
- **One real "paid for nothing" bug, now fixed (F1).**

## Evidence

| Check                                | Result                                                                                                                                                   |
| ------------------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Paid browser queries                 | Domain, Keywords, SERP, Backlinks, Brand Lookup, Prompt Explorer, rank suggestions: all `retry: false`, no refetch on focus or reconnect.                |
| Paid work only after a user choice   | Each query is `enabled` only once a domain, target, keyword or prompt is chosen; Backlinks tables wait for an overview with links (no paid empty table). |
| Automatic server retry of paid POSTs | None (`retrySafe=false` on every paid POST; see the [credits audit](2026-10-25-credits-audit.md)).                                                       |
| Dedupe of the same work              | Server caches: Backlinks 24 h, Domain and SERP 12 h; equal cold misses in one process share one call (`docs/integrations/DATAFORSEO.md`).                |
| Usage vs. spend                      | `go_dataforseo_usage` $4.17 equals the DataForSEO part of `go_provider_spend`.                                                                           |

## Findings

**F1. Google AI Overview tracker answers were bought and never shown (fixed).**
The daily tracker on a test workspace bought 3 AI Overview SERP tasks per run
(24 and 25 Oct), polled for 2 hours and ended `collection_timeout`. Cause:
`parseTrackingAnswer` failed on `related_searches` items, which are plain
keywords, and a failed answer still counted as pending. Fix and tests:
`backend/internal/aisearch/decision.md`. Cost so far is small ($0.0036/run),
but every customer with AI Overview in a tracker paid for it.

**F2. A test workspace runs a daily paid tracker (owner decision).** 3
prompts × 5 engines every day: about $0.19/day, ~$5.7/month of real
DataForSEO spend for a test account. Pause it or move it to weekly.

**F3. Paging, sorting and filters each buy a new table (by design).**
Backlinks and Domain tables are a paid call per page, sort or filter change.
Each is a user click and cached, so it follows the rules, but it is the main
way a curious customer spends credits fast.

**F4. Opening a saved link can buy again after the cache ends.** The search
lives in the URL, so reopening a Backlinks or Domain link after 24 h / 12 h
buys a fresh report without a new click. Low cost; listed so the owner can
decide whether a "Refresh" button should be required.

## Not done

- No live DataForSEO call; F1 is verified by tests only. After this is
  deployed (merging does not deploy), the next daily run on the test
  workspace (about 11:35 UTC) is the live check: AI Overview answers should
  complete.
- Provider invoice comparison needs the owner's DataForSEO dashboard.
- Calls across several server instances can still repeat cold work; there
  is one instance today.
