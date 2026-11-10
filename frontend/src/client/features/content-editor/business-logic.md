# Content Editor business logic

- Draft reads, writes, deletes, search briefs, authors, and publications are scoped to the selected project and its organization.
- The backend validates title, topic, slug, description, content size, review date, and author. A project can store up to `MaxDraftsPerProject` drafts.
- A user action is required for each paid AI generation and for a new or refreshed brief. AI actions are bounded to 15 credits; a brief's pre-click quote and provider hold use the same billing catalog entry. Cache hits do not buy new search data.
- AI suggestions do not modify the article until the writer chooses Insert or Replace. Publish uses saved content and is blocked while edits are unsaved or keyword stuffing remains.
- Images need alt text; article images are limited by `MAX_ARTICLE_IMAGES`. Uploads are compressed and validated before storage. A public URL insertion does not upload a file.
- The preview and WordPress output omit a leading H1 only when its text matches the separate page title. The stored draft remains intact.
- A blog theme is CSS only, at most 100 KB (`MaxThemeCSS`; the database checks it too), one per project. A `<` is rejected because the CSS sits inside a `<style>` tag in the preview frame and in exported files, where it could close the tag. The preview frame runs no scripts, so the CSS never styles or controls the app.
- Exported .html escapes the title, description, CSS and JSON-LD so none of them can break out of their tags. Structured data is included only when the draft has no unsaved edits, so it always matches the exported text.
- SEO and AEO scores do not award points for first-person wording because text alone cannot verify that an experience is real.
