# Uplumo blog style

Use [SEOWriting.ai’s keyword placement guide](https://seowriting.ai/blog/where-to-place-seo-keywords) as a reference for its answer-first opening, short sections, step-by-step examples and images placed beside the action they explain. Write original copy and make original illustrations or use Uplumo screenshots with demo data. The reference supplies the structure.

## Build each article around one reader task

1. Start with a real scenario or question. Give the practical answer in the opening paragraphs and say what the reader will be able to check or change.
2. Add **In this guide** for a long article. Link directly to the major H2 sections. Use short H2 and H3 headings, brief paragraphs and numbered instructions where order matters.
3. Show the work. Use an exact menu path, before-and-after example, troubleshooting table or worked calculation. Label illustrative examples. When numbers come from different systems, state the source, scope and denominator.
4. Bring in Uplumo at the relevant step. Explain what the feature does, what evidence the reader can see and what decision comes next. Verify plan access, data coverage and cadence in the product and current pricing before publishing.
5. End with a clear first action. Put FAQs under one `## Frequently asked questions` heading, one `###` question each, and close that section with another `##` heading before any closing call to action. The renderer turns that section into FAQPage structured data, so anything left under the last question becomes part of its answer. Link to relevant published Uplumo guides and cite official sources near platform-specific claims.

## Use visuals where they teach something

- Use a cover and at least two helpful in-body visuals in a substantial guide. Put each visual by the paragraph, action or result it explains.
- Use real, sanitised Uplumo screenshots for UI walkthroughs when available. Use an original diagram or before-and-after illustration for processes and comparisons. Label sample data and diagrams so readers can distinguish them from live results.
- Export optimised WebP files to `backend/internal/site/static/blog/`. Use shared product screenshots under `/site/product/`. Give every in-body image meaningful alt text and a short caption when its takeaway needs context.
- Check the images at mobile width. Avoid dense text, irrelevant decoration and personally identifiable data.

## Publish through the existing blog renderer

- Store articles in `backend/internal/site/blog/<slug>.md`. Keep an existing slug and publication date when editing. Set `updated` for substantial revisions.
- Use `title` (up to 70 characters), `description` (50–160 characters), `date`, `author` and `image` in front matter. The template supplies the H1. Keep Markdown headings at H2 and below. The `image` value is for sharing metadata. Include the cover in the article body as well.
- Section links use the Markdown renderer’s generated heading IDs. Check the rendered anchors. Verify the page, image requests, canonical URL and sitemap entry with `go test -race ./internal/site` from `backend/`.
- Distinguish measured traffic, sampled AI answers and estimates. Give comparisons a verification date when features may change. Avoid ranking or citation guarantees and unsupported claims of popularity.
