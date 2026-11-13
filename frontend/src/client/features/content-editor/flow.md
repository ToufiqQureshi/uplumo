# Content Editor flow

## Open and edit

1. `ContentEditorPage` loads the project's draft list, project domain, and selected draft from `/api/v1/projects/{projectId}/content/drafts/...`.
2. Typing updates the local draft. After three seconds without an edit, one save request writes the current version; **Save draft** sends it immediately. These database requests do not use provider credits.
3. Preview, score tabs, outline, metadata controls, and formatting run locally. Opening the page or changing tabs does not call a paid provider.

## Paid actions

- Each explicit AI action click sends one `POST /content/assistant/generate` with an idempotency key. The backend spend guard settles the provider cost, capped at 15 credits per action; the response reports credits used for the toast. The suggestion changes the draft only after the writer chooses Insert or Replace.
- Search briefs require an eligible paid plan; a matching cached brief uses no extra credits. A cache miss or confirmed Refresh sends one `POST /content/brief`, which makes one depth-30 DataForSEO SERP call and parses up to 16 candidate pages to find 8 readable pages. The displayed maximum and provider hold use the same billing catalog entry; all 16 parse attempts are included in the ceiling. For supported Labs markets, one keyword-suggestions call (20 suggestions plus seed) supplies volume, difficulty and intent together; there is no separate intent call. The shared brief cache lasts 30 days. Concurrent refreshes and refreshes within one minute reuse the same result. Maximum external calls per fresh brief: 1 SERP + 16 page parses + 1 keyword lookup = 18; parsing stops at 8 usable pages. Unsupported markets skip the keyword lookup.
- Neither action is triggered by page load, tab change, focus, or rerender. Failed paid requests are not retried automatically.

## Export and blog theme

All of this is free: no provider call, no credits.

- **Copy HTML**, **Download .html** and **Download .md** in the Publish tab run in the browser from the current editor text (`article-export.tsx`). The .html file adds the title, meta description and, when the draft has no unsaved edits, the JSON-LD from the existing schema preview. Each draft save refreshes that preview.
- **My blog theme** reads the project's CSS once with `POST /content/theme/get` (cached for the session) and saves it with `POST /content/theme/save`; an empty CSS removes it. One click is one save; failed saves are not retried.
- The preview dialog shows the article in that CSS inside a sandboxed frame without scripts. The frame gets the same plain HTML the export writes.

## WordPress

The Publish tab reads connection and publication status. After a WordPress connection exists and the draft is saved, the writer explicitly chooses **Send as draft** or **Publish now**. `sitepublish.Service.Publish` copies each unique article image when needed, renders the article, saves the post, then adds supported schema. A matching leading body H1 is omitted from preview and publish output; the saved Markdown remains unchanged.

## Full-draft review

Analyze full draft sends the full document and up to 6,000 characters of brief evidence. Oversized inputs are refused before the provider call. Reports are read-only and must be checked by the writer. Identical workspace, document, research and instructions reuse a private seven-day PostgreSQL report; concurrent identical requests on one server share a single model call and credit hold, and wait without holding a database connection. A report that was built and billed is returned even if caching it fails. Different workspaces never share reports. Switching tools or completing the first autosave preserves the displayed result. Opening another draft clears it.
