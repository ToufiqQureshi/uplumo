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

## Opening live pages

- Only the project's own domain and subdomains, so nobody copies another site's content into Uplumo; redirects off the site are refused.
- Free: the server reads the page itself, through the same SSRF guard as the audit crawler (no private or internal addresses), with a 10 s timeout and a 2 MB limit.
- One draft per page (`source_url` is unique per project): a second click never overwrites edits or creates copies, and counts toward the 100-draft limit only once.
- Images are not copied, to avoid rights and size problems; text keeps its links.

## Evidence and score boundaries

- Keyword metrics come from the selected market; missing metrics stay unknown, including a distinction between zero volume and missing volume. Research failure does not fabricate estimates.
- Term coverage rewards presence once, not repeated keyword use. Competitor frequency is descriptive guidance. Subtopic checks require prose below a matching heading, but do not prove semantic completeness. Missing research checks contribute no available points.
- Competitor depth uses original page word counts, not truncated excerpts. Own-site pages are excluded before deriving terms and headings.
- SEO and AEO both check overlap against sampled competitor excerpts. This is not an internet-wide plagiarism or AI detector. Future and impossible dates do not satisfy freshness.
- Editorial review supplies priorities, supporting passages, suggested edits and uncertainty. It cannot guarantee rankings or AI citations. Publishing still requires human fact checking.
