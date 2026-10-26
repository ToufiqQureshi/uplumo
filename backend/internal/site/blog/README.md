# Blog posts

Each `.md` file in this folder (except this one) is one post at
`https://uplumo.com/blog/<file-name>`. Posts are rendered when the server
starts, listed on `/blog` newest first, added to `/sitemap.xml` and given
`BlogPosting` structured data. A post with a mistake stops the server from
starting, and the tests fail first, so a broken post never reaches the site.

## Add a post

1. Name the file after the URL you want: lower-case words joined by hyphens,
   for example `what-is-aeo.md` → `/blog/what-is-aeo`. Never rename a
   published file; the old URL would break.
2. Start it with this front matter, then write Markdown below it:

   ```
   ---
   title: What is AEO? Answer engine optimisation explained
   description: How to get your pages quoted by ChatGPT, Perplexity and Google AI Overviews, in five steps any small business can follow.
   date: 2026-10-20
   ---

   Your post starts here. Use ## for sections and ### below them.
   ```

   | Field         | Required | Rule                                                                   |
   | ------------- | -------- | ---------------------------------------------------------------------- |
   | `title`       | yes      | Up to 70 characters, the parser's editorial limit. Describe the task accurately. |
   | `description` | yes      | 50 to 160 characters, the parser's editorial range. Google may choose a different snippet. |
   | `date`        | yes      | `YYYY-MM-DD`, the publish date.                                        |
   | `updated`     | no       | `YYYY-MM-DD`, set it when you change a post a lot.                     |
   | `author`      | no       | Defaults to "Uplumo team".                                             |
   | `draft`       | no       | `true` keeps the post off the site.                                    |

3. Images: put them in `backend/internal/site/static/blog/` and write
   `![What the image shows](/site/blog/file-name.svg)` or use `.webp`. Only images on
   uplumo.com load (the pages' security policy blocks other sites).
4. Raw HTML in a post is dropped. Use Markdown only.
5. Commit on a branch, open a PR, merge, then deploy (`docs/DEPLOY.md`).
   Editing on github.com works: open this folder, "Add file" → "Create new
   file".

## Write a complete answer to a real task

- **Define the intent.** Identify the reader, situation and decision. A
  troubleshooting guide, buying checklist and product page have different jobs.
  Use natural descriptive titles; no keyword-density or heading quota.
- **Answer early, then demonstrate.** Give the conclusion and its scope before
  the background. Include the topic-specific asset the task needs: annotated
  code, a reproducible diagnostic, a decision table or a copyable worksheet.
  Explain expected output, exceptions and how to verify the next step.
- **Verify facts.** Read current primary documentation for platform rules and
  reporting behavior. Link the source near volatile or consequential claims.
  Distinguish live tests from indexed state, observations from causes, and
  mentions/impressions from visits, qualified leads and revenue.
- **Use honest evidence.** Label invented numbers and scenarios as hypothetical.
  Never invent customer results, first-hand tests, screenshots, authors or
  credentials. Product claims must match the current code and plan catalog;
  link `/pricing` rather than duplicating changing plan prices and limits.
- **Remove repetition.** No shared filler paragraphs or generic safety sections
  pasted into unrelated guides. Keep warnings specific to the actual action.
  Length follows the task; do not pad to a word count or a publishing cadence.
- **Maintain useful links.** Open the intended destination, check that the label
  describes it, avoid self-links and prefer canonical URLs. A valid HTTP status
  alone does not prove a source supports the claim. Cite documentation directly
  rather than generic forum search results as evidence of platform behavior.
- **Review the whole post.** Check title, description, opening, examples, image
  labels, FAQ and sources for contradictions. Preserve published filenames and
  original dates; use `updated:` for a substantive review, not cosmetic freshness.
- **Verify rendering.** Run the site package tests for parsing, metadata, sitemap,
  assets and link destinations. Review rendered tables and fenced examples too.
  These checks cannot certify editorial accuracy or future rankings.
- **Measure after release.** Inspect important URLs in Search Console after
  deployment; request recrawl where useful. The site serves `/sitemap.xml`, but
  processing, indexing, rankings and AI citations are not guaranteed. Never claim
  sitemap submission or Search Console status without actually checking it.

Primary guidance: [Google's people-first content questions](https://developers.google.com/search/docs/fundamentals/creating-helpful-content),
[SEO Starter Guide](https://developers.google.com/search/docs/fundamentals/seo-starter-guide)
and [generative AI Search guidance](https://developers.google.com/search/docs/fundamentals/ai-optimization-guide).

