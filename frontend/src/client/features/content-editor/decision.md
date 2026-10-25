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

## Preserve saved draft identity and editorial metadata

- **Decision:** Save/open preserve `authorId` and normalize the returned review timestamp to its YYYY-MM-DD editor date. Record the selected opaque draft ID in a validated TanStack Router search parameter and reopen it through the existing project-scoped API on reload. Missing/forbidden drafts show an error before a writable editor. A new empty draft is labeled New draft, not Saved.
- **Why:** A real Go/Postgres/browser pass saved the article but reload opened a blank new editor; source review and the browser regression also exposed author/review fields being dropped by the response-to-editor mapping. Users should not lose their byline/freshness state or mistake a blank editor for lost work.
- **Preserve:** No article text in browser storage or URLs, no paid lookup on save/reload, no automatic publication. Backend authorization remains authoritative for every draft ID. Cached linked data must not overwrite later unsaved edits. New/delete clear the selected link; save/open update it without remounting the editor or adding autosave history entries.
- **Decision:** Use the installed TanStack Router `useBlocker` for unsaved SPA navigation and before-unload protection. Internal ID synchronization bypasses the blocker only after existing save/open/new/delete handling; explicit discard choices retain their confirmations.
- **Trace:** `ContentEditorPage.tsx`, `useLinkedDraft.ts` (installed query/router hooks), `EditorPanels.tsx` (existing toolbar/loading state extracted to keep the page bounded), `routes/_app/p/$projectId/content-editor.tsx`, `frontend/tests/content-editor.spec.ts`. The author-selection assertion failed before the mapping fix; tests exercise reload, safe missing-draft state, cancelled navigation and absence of paid calls. Existing Go storage/API contracts remain unchanged.

The slow-save browser check preserves the existing input lock until the saved snapshot is applied, then verifies later edits remain unsaved. It does not pretend users can type into disabled fields.

- **Contract correction:** The editor's date-only `lastReviewedAt` is serialized as Go's `lastReviewedDate`; responses remain RFC3339 `lastReviewedAt`. Real Go rejected the previous request with 400, while the initial permissive browser fixture incorrectly accepted it. The API payload unit regression fails without this mapping, and browser fixtures now reject the invalid response-only request field. Author and review-date persistence must be verified through real Go/database reloads, not solely permissive fixtures.

- **Autosave failure boundary:** A refused save must not resubmit the unchanged draft every three seconds. The mutation carries the immutable submitted input; autosave pauses when that exact input failed. A later edit may save again, and Save draft remains an explicit retry. Real backend 400 logs exposed the loop, and the browser regression observed a second unchanged submission before the guard. Preserve unsaved content and the existing pending input lock; no paid generation or publication is added.

Browser fixtures live in `frontend/tests/content-editor.fixtures.ts` so request/response contract handling stays in one bounded file shared by the browser scenarios. Fixtures reject response-only date fields rather than silently accepting every payload.

## Verify current writing and credit UX in the browser

- **Decision:** Update browser selectors for the current AI menu and cost disclosure; replace the obsolete save-lock assumption with an assertion that writing during a delayed save survives its response and can be saved again.
- **Why:** The editor intentionally preserves newer in-memory edits with its sent-snapshot comparison. Stale tests failed before reaching the business assertions.
- **Preserve:** AI double-click sends one keyed request; typing/autosave buys nothing; delayed responses never erase newer writing; failed autosave does not loop.
- **Trace:** `tests/content-editor.spec.ts`, existing save-snapshot unit checks and browser suite. Runtime editor behavior is unchanged.

## One module for pure draft utilities

- **Decision:** Keep title/preview, save-snapshot comparison, UTF-8 size validation and target-topic helpers in `draft.ts`, with their existing assertions together in `draft.test.ts`. Reuse the preview-title fallback and installed Remeda `uniqueBy` for topics.
- **Why:** Four small modules fragmented operations on the same draft; the title fallback and case-insensitive deduplication repeated existing logic.
- **Preserve:** Newer typing survives delayed saves; the size limit counts UTF-8 bytes; topic order/case and the five-topic cap remain. Typing and parsing start no API, AI or publish action.
- **Trace:** `draft.ts`, `draft.test.ts`, editor imports and the full frontend suite. These local checks do not establish live save/provider acceptance.
