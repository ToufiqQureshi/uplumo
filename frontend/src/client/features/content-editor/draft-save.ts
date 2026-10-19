import type { ContentDraftInput } from "./scoring";

export const sameDraft = (a: ContentDraftInput, b: ContentDraftInput) =>
  JSON.stringify(a) === JSON.stringify(b);

/**
 * The draft to show after a save returns. The writer keeps typing while an
 * autosave is in flight, so the server's copy replaces the editor only when
 * nothing changed since the save was sent; newer edits stay and the next
 * autosave sends them.
 */
export function draftAfterSave(
  current: ContentDraftInput,
  sent: ContentDraftInput,
  saved: ContentDraftInput,
): ContentDraftInput {
  return sameDraft(current, sent) ? saved : current;
}
