import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { withoutDuplicateTitleHeading } from "./ArticlePreview";

// Exported files use plain HTML so the reader's own blog CSS styles them.
// Without a saved theme they get this small readable default instead.
const DEFAULT_CSS =
  "body{font:17px/1.7 system-ui,sans-serif;max-width:42rem;margin:0 auto;padding:2rem 1rem;color:#1d2420}img{max-width:100%;height:auto}h1{line-height:1.2}table{border-collapse:collapse}td,th{border:1px solid #ccd5cf;padding:.4rem .6rem}";

/** The article body as plain semantic HTML elements, with no app classes. */
export function PlainArticle({
  content,
  title,
}: {
  content: string;
  title: string;
}) {
  return (
    <ReactMarkdown remarkPlugins={[remarkGfm]}>
      {withoutDuplicateTitleHeading(content, title)}
    </ReactMarkdown>
  );
}

/** Body HTML without the title, for pasting into a CMS that has a title field. */
export async function articleBodyHTML(content: string, title: string) {
  // Loaded on click so the editor does not ship the renderer up front.
  const { renderToStaticMarkup } = await import("react-dom/server");
  return renderToStaticMarkup(<PlainArticle content={content} title={title} />);
}

function escapeHTML(value: string) {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

/**
 * A complete page for the article. body is the inside of <article>; css is
 * the project's blog theme, if any. "<" is escaped in CSS and JSON-LD so
 * neither can close its tag.
 */
export function articleDocument({
  title,
  description,
  body,
  css,
  schemas = [],
}: {
  title: string;
  description: string;
  body: string;
  css?: string;
  schemas?: unknown[];
}) {
  const head = [
    '<meta charset="utf-8">',
    '<meta name="viewport" content="width=device-width, initial-scale=1">',
    `<title>${escapeHTML(title.trim() || "Untitled article")}</title>`,
    description.trim()
      ? `<meta name="description" content="${escapeHTML(description.trim())}">`
      : "",
    ...schemas.map(
      (schema) =>
        `<script type="application/ld+json">${JSON.stringify(schema).replaceAll("<", "\\u003c")}</script>`,
    ),
    `<style>${(css?.trim() || DEFAULT_CSS).replaceAll("<", "\\3c ")}</style>`,
  ].filter(Boolean);
  return `<!doctype html>\n<html>\n<head>\n${head.join("\n")}\n</head>\n<body>\n<article>\n${body}\n</article>\n</body>\n</html>\n`;
}

/** The full .html file: title heading plus body. */
export async function articleHTMLFile(input: {
  title: string;
  description: string;
  content: string;
  css?: string;
  schemas?: unknown[];
}) {
  const body = await articleBodyHTML(input.content, input.title);
  const heading = `<h1>${escapeHTML(input.title.trim() || "Untitled article")}</h1>`;
  return articleDocument({ ...input, body: `${heading}\n${body}` });
}

/** Markdown with front matter that Hugo, Jekyll, Astro and Ghost read. */
export function articleMarkdownFile({
  title,
  description,
  slug,
  content,
}: {
  title: string;
  description: string;
  slug: string;
  content: string;
}) {
  // JSON strings are valid YAML double-quoted scalars.
  const front = [
    `title: ${JSON.stringify(title.trim())}`,
    description.trim()
      ? `description: ${JSON.stringify(description.trim())}`
      : "",
    slug.trim() ? `slug: ${JSON.stringify(slug.trim())}` : "",
  ].filter(Boolean);
  return `---\n${front.join("\n")}\n---\n\n${content.trim()}\n`;
}

/** A safe file name from the slug, else the title, else "article". */
export function exportFileName(slug: string, title: string, extension: string) {
  const base = (slug.trim() || title)
    .toLowerCase()
    // \p{M} keeps the vowel signs of scripts such as Devanagari.
    .replace(/[^\p{L}\p{M}\p{N}]+/gu, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 80)
    .replace(/-+$/, "");
  return `${base || "article"}.${extension}`;
}
