# Content editing and AI writing: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Owns project drafts, content briefs, assistant generation, image upload and draft storage.

## Decisions to preserve

- Do not call paid providers on page load or keystrokes; cache and explicit user actions control spend. _Why:_ Passive viewing and editing should not create paid work or external side effects without the user asking for them.
- Apply output/input/image bounds and do not publish generated content automatically. _Why:_ These actions affect a customer account, site or external service and must follow an explicit user action with a clear result.
- Project authorization and draft IDs must be checked together; do not trust the caller’s project field alone. _Why:_ This keeps the behavior explicit so future changes do not silently alter customer-visible results.

- A brief with fewer than 3 readable competitor pages charges the customer nothing (`dataforseo.ErrNothingDelivered`); the provider cost stays recorded. _Why:_ The editor promises "no brief, no charge", and a customer must not pay for an error screen.
- A brief tries up to 16 results to read 8 pages, with the customer charge capped at the quoted 8-page ceiling. _Why:_ Shop and marketplace results parse to no text, which made briefs fail for whole niches; the extra parses cost Uplumo at most $0.0012.

## Dependencies

Postgres drafts/brief cache/authors; configured AI/DataForSEO providers; object storage for images; billing.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.

## Keep provider cost separate from writer refunds

- **Decision:** Use the existing provider/customer settlement contract for writing, including partial failures and empty responses.
- **Why:** Settling zero or a capped amount as the provider cost hid real expense from the emergency spend guard.
- **Preserve:** One model call, no automatic retry, zero customer charge for unusable answers and the 15-credit ceiling. Unknown usage remains an estimate rather than claimed invoice evidence.
- **Trace:** `writer.go`, `TestWriterRecordsProviderSpendEvenWhenCustomerIsRefundedOrCapped`; provider spend and customer charge are asserted separately.
