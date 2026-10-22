# Content editing and AI writing

Owns project drafts, content briefs, assistant generation, image upload and draft storage.

This is an internal developer/AI-agent guide, not customer-facing product copy. Read [flow.md](flow.md) for the request path and [decision.md](decision.md) for cost, data and security invariants.

## Source map

- `handler.go`: drafts, image upload and assistant routes.
- `brief_handler.go`, `brief.go`, `brief_terms.go`, `brief_cache.go`: search
  briefs and their shared 30-day cache.
- `contenteditor.go`: draft storage.
- `writer.go`: the 13 AI writing actions (`WriterAction`), each capped at
  `MaxWriterCreditsPerAction` (15 credits).
- `images.go`: image upload.

## API/trigger

POST /api/v1/projects/{projectId}/content/{brief|assistant/generate|drafts/...|images/upload}
