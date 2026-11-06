import type { ReactNode } from "react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { MARKDOWN_COMPONENTS } from "@/client/components/Markdown";

// The shared Markdown styles are sized for chat answers. An article preview
// needs headings that read like the published page, so only those change.
const ARTICLE_COMPONENTS = {
  ...MARKDOWN_COMPONENTS,
  h1: ({ children }: { children?: ReactNode }) => (
    <h1 className="mt-6 mb-3 text-3xl font-semibold first:mt-0">{children}</h1>
  ),
  h2: ({ children }: { children?: ReactNode }) => (
    <h2 className="mt-6 mb-2 text-2xl font-semibold first:mt-0">{children}</h2>
  ),
  h3: ({ children }: { children?: ReactNode }) => (
    <h3 className="mt-5 mb-2 text-xl font-semibold first:mt-0">{children}</h3>
  ),
  img: ({ src, alt }: { src?: string | Blob; alt?: string }) =>
    typeof src === "string" ? (
      <img src={src} alt={alt ?? ""} className="my-4 rounded-md" />
    ) : null,
};

const ITALIC_ASTERISK = /(^|[^\p{L}\p{N}_])\*(\S(.*?\S)?)\*/gu;
const ITALIC_UNDERSCORE = /(^|[^\p{L}\p{N}_])_(\S(.*?\S)?)_/gu;

function withoutDuplicateTitleHeading(content: string, title: string) {
  if (!title.trim()) return content;
  const lines = content.split("\n");
  const index = lines.findIndex((line) => {
    const value = line.trim();
    return value !== "" && !value.startsWith("![");
  });
  if (index < 0) return content;
  const heading = lines[index]?.trim().match(/^#\s+(.+?)\s*#*\s*$/u)?.[1];
  if (
    !heading ||
    heading
      .replace(/(?:\*\*|__|~~|`)/gu, "")
      .replace(ITALIC_ASTERISK, "$1$2")
      .replace(ITALIC_UNDERSCORE, "$1$2")
      .replace(/\s+/gu, " ")
      .trim()
      .toLowerCase() !== title.trim().replace(/\s+/gu, " ").toLowerCase()
  )
    return content;

  lines.splice(index, 1);
  while (index < lines.length && !lines[index]?.trim()) lines.splice(index, 1);
  return lines.join("\n");
}

/** Renders a draft's Markdown the way a reader will see the article. */
export function ArticlePreview({
  content,
  title,
}: {
  content: string;
  title?: string;
}) {
  return (
    <ReactMarkdown remarkPlugins={[remarkGfm]} components={ARTICLE_COMPONENTS}>
      {withoutDuplicateTitleHeading(content, title ?? "")}
    </ReactMarkdown>
  );
}
