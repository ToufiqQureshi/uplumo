# Customer-site publishing: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when implementation behavior or invariants change.

## Responsibility

Connects WordPress and publishes content-editor drafts after explicit authorization and user action.

## Decisions to preserve

- Publish is an external side effect; preview and draft save must never publish. _Why:_ Passive viewing and editing should not create paid work or external side effects without the user asking for them.
- Do not retry uncertain create-post calls blindly; use persisted result/idempotency safeguards. _Why:_ Retries, replays and parallel requests are normal; idempotency prevents duplicate charges, records and side effects.
- Keep credentials encrypted/redacted and sanitize content/schema before sending. _Why:_ These values or checks protect account access; exposing them or weakening validation can enable account or provider abuse.
- **A live publish is offered to IndexNow, and never fails because of it.** _Decision:_ after `status: publish` succeeds, `handler.submitToIndexNow` calls `PageSubmitter.SubmitPublished` (35 seconds) and adds `indexNow: {status, message}` to the answer; drafts are never sent. _Why:_ the owner wants new articles discovered quickly without a second step; a WordPress draft is not public. _Preserve:_ the post is already published, so an IndexNow error, timeout or refusal only changes the answer's `indexNow` field; a nil submitter stays a nil interface (`httpapi.go`). _Trace:_ `handler.go`, `TestPublishSubmitsLivePagesToIndexNow`; rules in `backend/internal/indexnow/decision.md`.

## Dependencies

WordPress REST API; Postgres site connection/publication/draft rows; plan catalog; `indexnow.Service` after a live publish.

## Verification

Use the package tests next to the source files in `README.md`. For changes crossing an API boundary, also check the mounted route/contract and the caller feature’s `flow.md`.
