# Content Editor flow

## Open and edit

1. `ContentEditorPage` loads the project's draft list, project domain, and selected draft from `/api/v1/projects/{projectId}/content/drafts/...`.
2. Typing updates the local draft. After three seconds without an edit, one save request writes the current version; **Save draft** sends it immediately. These database requests do not use provider credits.
3. Preview, score tabs, outline, metadata controls, and formatting run locally. Opening the page or changing tabs does not call a paid provider.

## Paid actions

- Each explicit AI action click sends one `POST /content/assistant/generate` with an idempotency key. The backend spend guard settles the provider cost, capped at 15 credits per action; the response reports credits used for the toast. The suggestion changes the draft only after the writer chooses Insert or Replace.
- A cached brief is free. A cache miss or confirmed Refresh sends one `POST /content/brief`, which makes one depth-30 DataForSEO SERP call and parses up to 16 candidate pages to find 8 readable pages. The displayed maximum and provider hold use the same billing catalog entry; extra parse attempts are covered by Uplumo. The shared brief cache lasts 30 days.
- Neither action is triggered by page load, tab change, focus, or rerender. Failed paid requests are not retried automatically.

## WordPress

The Publish tab reads connection and publication status. After a WordPress connection exists and the draft is saved, the writer explicitly chooses **Send as draft** or **Publish now**. `sitepublish.Service.Publish` copies each unique article image when needed, renders the article, saves the post, then adds supported schema. A matching leading body H1 is omitted from preview and publish output; the saved Markdown remains unchanged.
