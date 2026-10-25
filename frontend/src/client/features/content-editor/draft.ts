import { uniqueBy } from "remeda";
import type { ContentDraftInput } from "./scoring";

export const titleForDraft = (draft: ContentDraftInput) =>
  searchPreviewTitle(draft) || "Untitled draft";

export const searchPreviewTitle = (draft: ContentDraftInput) =>
  draft.title.trim() || draft.content.match(/^#\s+(.+)$/mu)?.[1]?.trim() || "";

export const searchPreviewUrl = (domain: string, slug: string) =>
  domain
    ? `https://${domain}/${slug || "your-page"}`
    : "Connect a website to preview its URL";

/** The server limits article Markdown by its UTF-8 byte length. */
const MAX_ARTICLE_CONTENT_BYTES = 150_000;

const encoder = new TextEncoder();

export function isArticleContentWithinLimit(content: string): boolean {
  return encoder.encode(content).byteLength <= MAX_ARTICLE_CONTENT_BYTES;
}

// A delayed autosave must not overwrite text typed after its sent snapshot.
export const sameDraft = (a: ContentDraftInput, b: ContentDraftInput) =>
  JSON.stringify(a) === JSON.stringify(b);

export const MAX_TARGET_TOPICS = 5;

export function parseTargetTopics(value: string): string[] {
  return uniqueBy(
    value
      .split(/[\n,]+/u)
      .map((part) => part.trim())
      .filter(Boolean),
    (part) => part.toLocaleLowerCase(),
  ).slice(0, MAX_TARGET_TOPICS);
}

export const primaryTargetTopic = (value: string) =>
  parseTargetTopics(value)[0] ?? "";
