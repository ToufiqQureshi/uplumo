export const words = (value: string) =>
  value.match(/[\p{L}\p{N}]+(?:['’-][\p{L}\p{N}]+)*/gu) ?? [];
export const normalizeText = (value: string) =>
  value.toLocaleLowerCase().normalize("NFKC");
export const countWords = (value: string) => words(value).length;
export const headings = (value: string, maxLevel = 6) =>
  value.split("\n").flatMap((line) => {
    const match = line.trim().match(/^(#{1,6})\s+(.+)$/u);
    return match && match[1].length <= maxLevel ? [match[2]] : [];
  });
export const headingsAtLevel = (value: string, level: number) =>
  value.split("\n").flatMap((line) => {
    const match = line.trim().match(/^(#{1,6})\s+(.+)$/u);
    return match && match[1].length === level ? [match[2]] : [];
  });
export function phraseCount(text: string, phrase: string) {
  const parts = words(normalizeText(phrase)).map((part) =>
    part.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"),
  );
  if (!parts.length) return 0;
  const pattern = new RegExp(
    "(^|[^\\p{L}\\p{N}])" +
      parts.join("[\\s\\p{P}]+") +
      "(?=$|[^\\p{L}\\p{N}])",
    "giu",
  );
  return [...normalizeText(text).matchAll(pattern)].length;
}
export const containsPhrase = (text: string, phrase: string) =>
  phraseCount(text, phrase) > 0;

const STOP_WORDS = new Set(
  "a an and are as at be by for from how in into is it of on or the to what why with your".split(
    " ",
  ),
);
/** The words of a topic that carry meaning, for close-variant matches. */
export const topicWords = (topic: string) =>
  words(normalizeText(topic)).filter((word) => !STOP_WORDS.has(word));

/**
 * True when text names the topic: the exact phrase, or every meaningful
 * word of it in any order ("ethnic wear in teal green"). Writers should not
 * have to repeat an exact keyword to earn placement points; Google matches
 * meaning, and forcing the exact phrase pushes keyword stuffing.
 */
export function mentionsTopic(text: string, topic: string) {
  if (!topic.trim()) return false;
  if (containsPhrase(text, topic)) return true;
  const wanted = topicWords(topic);
  if (wanted.length < 2) return false;
  const present = new Set(words(normalizeText(text)));
  return wanted.every((word) => present.has(word));
}

/** Link targets in Markdown, never image sources (`![alt](src)`). */
export const linkTargets = (body: string) =>
  [...body.matchAll(/(?<!!)\[[^\]]+\]\(([^)\s]+)[^)]*\)/gu)].map(
    (match) => match[1] ?? "",
  );
export const paragraphs = (text: string) =>
  text
    .split(/\n\s*\n/u)
    .map((part) => part.trim())
    .filter(
      (part) =>
        part &&
        !/^#{1,6}\s/u.test(part) &&
        !/^\s*(?:[-*+] |\d+[.)] )/u.test(part),
    );
