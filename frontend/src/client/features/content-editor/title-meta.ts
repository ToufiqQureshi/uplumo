type TitleMetaOption = { title: string; description: string };

const clean = (value: string) =>
  value
    .replace(/[*_`]/gu, "")
    .replace(/^["“]|["”]$/gu, "")
    .trim();

/**
 * Reads "Title: …" / "Description: …" pairs from the AI's title and meta
 * suggestion, so the writer can apply one with a click instead of copying.
 * Markdown bold, numbering and "Meta description" labels are tolerated.
 */
export function parseTitleMetaOptions(text: string): TitleMetaOption[] {
  const options: TitleMetaOption[] = [];
  let title = "";
  for (const line of text.split("\n")) {
    const match = line.match(
      /^\s*(?:\d+[.)]\s*)?[-*]?\s*\**\s*(title|(?:meta |search )?description)\s*:?\**\s*:?\s*(.+)$/iu,
    );
    if (!match) continue;
    const value = clean(match[2] ?? "");
    if (!value) continue;
    if (/^title$/iu.test(match[1] ?? "")) {
      title = value;
    } else if (title) {
      options.push({ title, description: value });
      title = "";
    }
  }
  return options.slice(0, 5);
}
