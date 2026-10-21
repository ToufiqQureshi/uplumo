import { makeScoreCheck as check, type ScoreCheck } from "./scoring-utils";
import {
  containsPhrase as has,
  countWords as count,
  normalizeText as norm,
  topicWords,
  words,
} from "./scoring-text";
import type { ContentDraftInput } from "./scoring";

// Signals from Google's helpful-content and E-E-A-T guidance: freshness,
// first-hand experience, a written search description, a plain definition
// and a real FAQ.

// freshnessRule: the "Show freshness" check passes only when the draft has
// a reviewed date less than 12 months old. The source of truth is the
// draft's reviewed date, not a guessed date in the text.

export function freshnessCheck(
  d: ContentDraftInput,
  now: Date = new Date(),
): { passed: boolean; message: string; weight: number } {
  const weight = 5;
  if (!d.lastReviewedAt?.trim()) {
    return {
      passed: false,
      message:
        'Publish tab → "Mark as reviewed today" after you check the facts.',
      weight,
    };
  }
  if (!freshWithinTwelveMonths(d.lastReviewedAt, now)) {
    return {
      passed: false,
      message:
        "The last review is more than 12 months old; mark it reviewed after checking the facts.",
      weight,
    };
  }
  return {
    passed: true,
    message: "Reviewed within the last 12 months.",
    weight,
  };
}

// freshWithinTwelveMonths compares calendar months: the review stays fresh
// through the same day of the twelfth month, and a day after expires.
export function freshWithinTwelveMonths(reviewed: string, now: Date): boolean {
  const parsed = new Date(`${reviewed}T00:00:00Z`);
  if (Number.isNaN(parsed.getTime())) return false;
  const ageMonths =
    (now.getUTCFullYear() - parsed.getUTCFullYear()) * 12 +
    (now.getUTCMonth() - parsed.getUTCMonth());
  if (ageMonths < 12) return true;
  if (ageMonths > 12) return false;
  // Exactly 12 months: fresh until the same calendar day passes.
  return parsed.getUTCDate() >= now.getUTCDate();
}
// First-hand experience is what Google's helpful-content guidance asks for
// most: something the writer saw, tried, measured or was asked.
const EXPERIENCE = [
  /\b(?:i|we|my|our)\b[^.!?\n]{0,60}\b(?:tried|tested|wore|used|visited|bought|found|saw|asked|made|cooked|built|measured|compared|noticed|learned|worked|helped|ran|checked)\b/iu,
  /\b(?:in my|in our|from my|from our)\s+(?:experience|tests?|store|shop|clinic|practice|work)\b/iu,
  /\b(?:maine|humne|mera|meri|hamare|hamari)\b[^.!?\n]{0,60}\b(?:dekha|kiya|try|pehna|use|test|khareeda|poocha)\b/iu,
];
export function experienceCheck(body: string): ScoreCheck {
  const shown = EXPERIENCE.some((pattern) => pattern.test(body));
  return check(
    "experience",
    "Your own experience",
    shown
      ? "The article shares something you saw, tried or measured."
      : "Add one real example: what you or your customers tried, saw or measured.",
    10,
    shown ? 10 : 0,
  );
}

export function searchDescriptionCheck(d: ContentDraftInput): ScoreCheck {
  const length = d.metaDescription.trim().length;
  return check(
    "search-description",
    "Search description",
    length
      ? length + " characters. Google may still choose its own snippet."
      : "Open “How it looks on Google” and write one or two sentences.",
    5,
    length ? 5 : 0,
  );
}

// A definition sentence: the topic or one of its words, then "is", "means"
// or a Hindi/Hinglish equivalent, near the start of the sentence.
const DEFINES =
  /\b(?:is|are|means|refers to|stands for)\b|\b(?:hota hai|hoti hai|hote hain|matlab|yaani)\b|(?:होता है|होती है|मतलब)/iu;
export function definesTopic(body: string, topic: string) {
  const wanted = topicWords(topic);
  if (!wanted.length) return false;
  return (
    body
      .split(/(?<=[.!?।])\s+|\n+/u)
      // A heading or a question ("What is teal?") asks; it does not define.
      .filter(
        (sentence) => !/^\s*#/u.test(sentence) && !/[?？]\s*$/u.test(sentence),
      )
      .some((sentence) => {
        // The sentence must start with the topic: "Teal is…", not
        // "Crawlability and indexing are … technical SEO tasks".
        const opening = words(norm(sentence)).slice(0, wanted.length + 1);
        return (
          (has(opening.join(" "), topic) ||
            wanted.some((word) => opening.slice(0, 2).includes(word))) &&
          DEFINES.test(sentence)
        );
      })
  );
}

// The template text the FAQ block inserts; it must not earn FAQ points.
const FAQ_TEMPLATE = /^question people ask\??$/iu;
export function realFaq(body: string) {
  const start = body.search(
    /^#{2,3}\s+(?:faq|frequently asked questions)\b/imu,
  );
  if (start < 0) return false;
  const sections = body
    .slice(start)
    .split(/\n(?=#{2,4}\s)/u)
    .slice(1);
  return sections.some((section) => {
    const [heading = "", ...rest] = section.split("\n");
    const question = heading.replace(/^#+\s*/u, "").trim();
    return (
      /[?？]$/u.test(question) &&
      !FAQ_TEMPLATE.test(question) &&
      count(rest.join(" ")) >= 8
    );
  });
}
