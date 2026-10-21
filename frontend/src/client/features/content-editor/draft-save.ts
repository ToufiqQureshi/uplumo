import type { ContentDraftInput } from "./scoring";

/**
 * True when two drafts are identical. The editor compares the draft it sent
 * in an autosave with the one on screen when the reply arrives: if they
 * differ, the writer kept typing and the reply must not overwrite it.
 */
// Saving never locks the editor's fields (autosave once dropped typed keys),
// so the page compares drafts instead.
export const sameDraft = (a: ContentDraftInput, b: ContentDraftInput) =>
  JSON.stringify(a) === JSON.stringify(b);
