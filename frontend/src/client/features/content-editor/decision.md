# Content editor: decisions

> Internal guide for developers and coding agents. Update this file in the same PR when a behavior or invariant changes.

## Feature purpose

Creates and edits SEO drafts, content briefs, author profiles, internal-link suggestions, images and publish previews.

## Decisions to preserve

- AI writing/brief calls may spend credits or provider money; keep plan, credit and output-size guards server-side. _Why:_ This work consumes paid quota or shared capacity, so authorization and bounds must be checked before dispatch.
- Do not auto-save or publish unreviewed generated text to a customer site. _Why:_ These actions affect a customer account, site or external service and must follow an explicit user action with a clear result.
- Uploaded images and generated HTML must be constrained to allowed storage/content types. _Why:_ User and provider content or URLs can be hostile; validation and escaping prevent unauthorized exposure or executable content.
- Link suggestions use project-owned GSC/audit data and must never cross project boundaries. _Why:_ Explicit bounds prevent a large or abusive request from consuming shared memory, latency or service capacity.
- AI rewrites require one selected plain-text block within the 5,000-character context sent to the model. _Why:_ Truncating the prompt while replacing a longer passage, or replacing a selection containing an image/table, could silently lose customer content.

- Accept a Search Performance `keyword` handoff only as a bounded topic seed on a blank, clean draft. _Why:_ It carries evidence to the editor without overwriting customer work or starting a paid action.

## Verification pointers

- `ArticlePreview.test.ts`
- `brief-view.test.ts`

- Search Growth evidence is a read-only reference, not imported HTML or an
  instruction to overwrite the draft. Validate URL scheme, lengths and expiry
  before displaying the tab-local context.

## Search Growth evidence reference

- **Decision:** carry a bounded source URL, query themes, dates and findings as a
  validated project-scoped tab-local reference, preserving existing draft text.
  **Why:** the writer needs the reason for a suggested edit; navigating from an
  opportunity is not permission to import arbitrary HTML or overwrite work.
- **Decision:** retain the existing selected-text AI review/apply and explicit
  publish workflow. **Why:** an analyst recommendation is evidence for the
  writer's decision, not an automatic content or paid-generation instruction.
  **Trace:** `searchContext.ts`, `ContentEditorPage.tsx`, Search Growth handoff
  tests and backend SAM's execution-boundary decision.

Keep this file updated when the reference schema, draft-preservation behavior or
review/apply boundary changes.
