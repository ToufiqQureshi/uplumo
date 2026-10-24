# Content editor: flow

> Internal guide for developers and coding agents. Confirm current behavior in the linked implementation and tests before changing it.

## User and request flow

1. The editor loads a selected project and draft, then keeps editing state in the editor components.
2. Brief, AI assistant, authors, link suggestions, image upload and draft save each use their own API action; link suggestions run only when requested.
3. Go validates project access and inputs, stores drafts/authors, optionally calls configured providers, and returns a structured response.
4. A publish preview is separate from publishing; publication occurs only through the explicit site-publishing action.

## Implementation map

- `ContentEditorPage.tsx`
- `contentEditorApi.ts`
- `WriterAssistant.tsx`
- `linkSuggestions.ts`
- `backend/internal/contenteditor`
- `backend/internal/contentauthors`
- `backend/internal/linksuggest`
- `backend/internal/sitepublish`

## API and service dependencies

Go packages: `contenteditor`, `contentauthors`, `linksuggest` and `sitepublish`; costs depend on the specific explicit action.

## Cost, security and failure boundaries

- AI writing/brief calls may spend credits or provider money; keep plan, credit and output-size guards server-side.
- Do not auto-save or publish unreviewed generated text to a customer site.
- Uploaded images and generated HTML must be constrained to allowed storage/content types.
- Link suggestions use project-owned GSC/audit data and must never cross project boundaries.

## Focused issue handoff

The writer selects one of the current unfinished score checks → sidebar stores
only its ID/mode and draft/topic scope → AI help resolves fresh check evidence →
an explicit action sends bounded task instructions with the selected passage.
Preparing a task never generates or inserts content. Published articles link to
the existing explicit paid-tracking flows.

AI rewrite actions require an explicit visual-editor selection within one text
block and at most 5,000 characters, with no images or other non-text nodes.
`writer-target.ts`
captures that range and the document at request time: accepting replaces only
the selected passage, and refuses a different draft or edits made meanwhile.
Section/FAQ/gap actions insert at the saved cursor; outlines, title/meta options
and fact-check notes are review-only. Markdown mode does not expose a hidden
visual selection to the assistant. Never restore whole-article replacement as a
fallback when the visual editor is unavailable.

## Search Performance handoff

A user explicitly chooses Start draft on a Search Console refresh candidate -> TanStack route passes the strongest query as `keyword` -> the editor seeds the target topic on a blank draft. The route never starts a brief or AI call; editing, saving, and publishing retain their existing explicit workflows.

Growth handoff: explicit Improve this page -> `saveSearchContext` validates the
source URL, bounded queries/reasons and stores a 30-minute project-scoped tab
reference -> `SearchContextPanel` shows the existing page and evidence above
the unchanged editor workflow. An invalid/expired reference is omitted.

Saved draft identity: save/open → update the validated `draft` URL parameter
with the opaque ID using replace navigation → reload fetches that ID through
the existing project-scoped draft API/TanStack cache → restore article and
author/review metadata. The request uses Go `lastReviewedDate` (YYYY-MM-DD);
the response uses `lastReviewedAt` (RFC3339 midnight). Missing/forbidden IDs show an error without a writable
blank replacement. No brief/AI/publish call occurs. New/delete clear the link.
The router's blocker protects unsaved SPA exits and before-unload; internal ID
synchronization does not discard content and does not prompt or remount.
