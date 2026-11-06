# Content Editor

The Content Editor manages project-scoped writing drafts, scores SEO and answer-readiness signals, builds optional search briefs, and can send a saved draft to a connected WordPress site.

## Code map

- `ContentEditorPage.tsx` loads drafts, tracks edits, autosaves after three idle seconds, and arranges the editor and side panel.
- `ArticleComposer.tsx` owns the title, target topics, search metadata, images, and preview.
- `visual-editor/ArticleEditor.tsx` edits Markdown with TipTap; `ArticlePreview.tsx` renders the reader view.
- `ContentEditorSidePanel.tsx` connects the score, brief, outline, AI help, and publish panels.
- `scoring.ts`, `scoring-rules.ts`, and `scoring-signals.ts` calculate writing checks; scores are editorial signals, not ranking predictions.
- `contentEditorApi.ts` defines the browser API calls. `WriterAssistant.tsx` and `BriefPanel.tsx` make the paid actions explicit.
- Backend handlers live in `backend/internal/contenteditor/`; WordPress publishing and schema generation live in `backend/internal/sitepublish/` and `frontend/src/client/features/site-publishing/`.

## Data and routes

- Drafts and authors: `go_content_drafts`, `go_content_authors`.
- Shared search briefs: `go_content_brief_cache`.
- WordPress connections and publications: `go_site_connections`, `go_site_publications`.
- Draft routes: `POST /api/v1/projects/{projectId}/content/drafts/list`, `POST /api/v1/projects/{projectId}/content/drafts/{draftId}/get`, `POST /api/v1/projects/{projectId}/content/drafts/save`, and `POST /api/v1/projects/{projectId}/content/drafts/{draftId}/delete`.
- Brief, AI, and image routes: `POST /api/v1/projects/{projectId}/content/brief`, `POST /api/v1/projects/{projectId}/content/assistant/generate`, and `POST /api/v1/projects/{projectId}/content/images/upload`.
- Publishing routes: `POST /api/v1/projects/{projectId}/content/drafts/{draftId}/schema-preview`, `/publish`, and `/publication/get`.
