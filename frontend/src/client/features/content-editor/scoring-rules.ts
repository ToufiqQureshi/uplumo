import { sortBy } from "remeda";
import { makeScoreCheck as check, type ScoreCheck } from "./scoring-utils";

import {
  containsPhrase as has,
  countWords as count,
  headings as heads,
  headingsAtLevel,
  linkTargets,
  mentionsTopic,
  normalizeText as norm,
  paragraphs as paras,
  phraseCount,
  words,
} from "./scoring-text";
import type { ContentBrief, ContentDraftInput } from "./scoring";
import {
  definesTopic,
  experienceCheck,
  freshnessCheck,
  realFaq,
  searchDescriptionCheck,
} from "./scoring-signals";

export { freshnessCheck, freshWithinTwelveMonths } from "./scoring-signals";

const LIST_OR_TABLE = /^\s*(?:[-*+] |\d+[.)] )|\|.+\|/mu;

// Topic placement looks where a reader and Google look first. The title
// field is the page's H1, so a second H1 in the body is not required.
function topicPlacement(d: ContentDraftInput, body: string): ScoreCheck {
  const topic = d.targetKeyword.trim();
  if (!topic) {
    return check(
      "topic-placement",
      "Topic placement",
      "Add a target topic to check placement.",
      15,
      0,
    );
  }
  const subheadings = [
    ...headingsAtLevel(body, 2),
    ...headingsAtLevel(body, 3),
  ].join("\n");
  const places: Array<[string, string]> = [
    ["title", d.title || (headingsAtLevel(body, 1)[0] ?? "")],
    ["opening", words(body).slice(0, 100).join(" ")],
    ["a subheading", subheadings],
    ["search description", d.metaDescription],
  ];
  const missing = places
    .filter(([, text]) => !mentionsTopic(text, topic))
    .map(([name]) => name);
  const placed = places.length - missing.length;
  return check(
    "topic-placement",
    "Topic placement",
    placed +
      "/4: title, opening, a subheading and search description" +
      (missing.length ? ". Missing in: " + missing.join(", ") + "." : "."),
    15,
    (15 * placed) / 4,
  );
}

export function staticSeo(
  d: ContentDraftInput,
  body: string,
  siteDomain?: string,
): ScoreCheck[] {
  const links = linkTargets(body),
    inside = links.filter((url) => isInternalLink(url, siteDomain)).length,
    outside = links.length - inside;
  const images = [...body.matchAll(/!\[([^\]]*)\]\(([^)]+)\)/gu)],
    alt = images.filter((m) => (m[1] ?? "").trim().length >= 8).length;
  const allWords = words(body);
  const hindiWords = new Set(
    "hai hain ka ki ke se mein me aur ko par yeh ye kya kaise nahi nahin hum aap iska uska hota hoti hote karne karega chahiye".split(
      " ",
    ),
  );
  const hindiShare =
    allWords.filter((word) => hindiWords.has(norm(word))).length /
    Math.max(1, allWords.length);
  const english =
    allWords.length > 0 && hindiShare < 0.05 && !/[ऀ-ॿ]/u.test(body);
  const pc = paras(body).map(count).filter(Boolean),
    avg = pc.length ? pc.reduce((a, b) => a + b, 0) / pc.length : 0;
  return [
    topicPlacement(d, body),
    check(
      "headings",
      "Clear sections",
      heads(body).length + " headings found.",
      3,
      heads(body).length ? 3 : 0,
    ),
    check(
      "list-table",
      "Scannable format",
      "Use a list or table where useful.",
      5,
      LIST_OR_TABLE.test(body) ? 5 : 0,
    ),
    check(
      "sources-internal",
      "Sources and internal links",
      outside +
        " link(s) to other sites, " +
        inside +
        " to your own pages. Images do not count.",
      8,
      Math.min(8, (outside ? 5 : 0) + (inside ? 3 : 0)),
    ),
    check(
      "readability",
      "Readable paragraphs",
      english
        ? "English paragraphs average " + Math.round(avg) + " words."
        : "No English readability penalty applied for this language.",
      7,
      allWords.length > 0 && (!english || (avg > 0 && avg <= 120)) ? 7 : 0,
    ),
    check(
      "alt-text",
      "Descriptive image alt text",
      images.length
        ? alt + "/" + images.length + " images have descriptive alt text."
        : "Not applicable: no images in this draft.",
      images.length ? 5 : 0,
      images.length ? (5 * alt) / images.length : 0,
    ),
    searchDescriptionCheck(d),
    experienceCheck(body),
  ];
}
function isInternalLink(url: string, siteDomain?: string) {
  if (url.startsWith("/") || url.startsWith("#")) return true;
  if (!siteDomain) return false;
  try {
    const linkedHost = new URL(url).hostname
      .toLocaleLowerCase()
      .replace(/^www\./u, "");
    const ownHost = siteDomain.toLocaleLowerCase().replace(/^www\./u, "");
    return linkedHost === ownHost;
  } catch {
    return false;
  }
}
export function briefSeo(
  d: ContentDraftInput,
  body: string,
  brief: ContentBrief,
): ScoreCheck[] {
  const terms = sortBy(
      brief.terms,
      [(term) => term.domains, "desc"],
      [(term) => term.term, "asc"],
    ).slice(0, 10),
    denom = terms.reduce((s, t) => s + Math.max(1, t.domains), 0),
    covered = terms.reduce((s, t) => {
      const n = phraseCount(body, t.term);
      const coverage =
        t.max <= 0
          ? n
            ? 0
            : 1
          : n < t.min
            ? Math.max(0, n / Math.max(1, t.min))
            : 1;
      return s + Math.max(1, t.domains) * coverage;
    }, 0),
    over = terms.filter((t) => phraseCount(body, t.term) > t.max).length;
  const h2 = headingsAtLevel(body, 2),
    sh = brief.subtopics.filter((t) => h2.some((h) => has(h, t))).length;
  const lengths = sortBy(
      brief.competitorTexts.map(count).filter((n) => n > 0),
      (value) => value,
    ),
    lo = lengths.length
      ? lengths[Math.floor((lengths.length - 1) * 0.25)]
      : 200,
    hi = lengths.length
      ? lengths[Math.ceil((lengths.length - 1) * 0.75)]
      : 1800,
    n = count(body),
    depth =
      n >= lo && n <= hi
        ? 10
        : n < lo
          ? (10 * n) / Math.max(1, lo)
          : (10 * hi) / n;
  return [
    check(
      "term-coverage",
      "Important term coverage",
      terms.length + " brief terms weighted by competitor prevalence.",
      30,
      denom ? (30 * covered) / denom : 0,
    ),
    check(
      "term-ranges",
      "Terms within suggested ranges",
      over + " of " + terms.length + " terms exceed their range.",
      10,
      terms.length ? (10 * (terms.length - over)) / terms.length : 10,
    ),
    topicPlacement(d, body),
    check(
      "depth",
      "Draft depth",
      n +
        " words; top pages use " +
        lo +
        "–" +
        hi +
        " words (" +
        lengths.length +
        " pages). Cover the topic fully; length alone is not a goal.",
      10,
      depth,
    ),
    check(
      "subtopics",
      "Brief subtopics in H2s",
      sh + "/" + brief.subtopics.length + " subtopics covered.",
      10,
      brief.subtopics.length ? (10 * sh) / brief.subtopics.length : 10,
    ),
  ];
}

export function aeo(
  d: ContentDraftInput,
  body: string,
  brief?: ContentBrief,
): ScoreCheck[] {
  const first = paras(body)[0] ?? "",
    opening = count(first),
    topic = d.targetKeyword.trim(),
    // Google: no ideal length and no need to chunk content for AI. The
    // check only asks that the article opens with a real answer.
    direct =
      opening >= 25 &&
      opening <= 120 &&
      (!topic || mentionsTopic(first, topic)),
    qheads = heads(body).filter((h) => h.includes("?") || h.includes("？")),
    questions = brief?.questions ?? [];
  const answered = questions.filter((q) => {
    const h = qheads.find((x) => has(x, q.replace(/[?？]$/u, "")));
    if (!h) return false;
    const answer =
      body.slice(body.indexOf(h) + h.length).split(/\n#{1,6}\s/u)[0] ?? "";
    return count(answer) >= 15;
  }).length;
  const external = linkTargets(body).some(
      (url) => !isInternalLink(url, brief?.siteDomain),
    ),
    entities = new Set(
      body
        .split(/(?<=[.!?])\s+|\n+/u)
        .map((sentence) =>
          sentence.replace(/^(\s*)[A-Z][\p{L}\p{N}&.-]*/u, (initial) =>
            initial.toLocaleLowerCase(),
          ),
        )
        .join(" ")
        .match(/\b[A-Z][\p{L}\p{N}&.-]*(?:\s+[A-Z][\p{L}\p{N}&.-]*){0,3}/gu) ??
        [],
    ).size,
    freshness = freshnessCheck(d),
    author = Boolean(d.authorId?.trim());
  return [
    check(
      "direct-answer",
      "Direct answer",
      "Opening paragraph has " +
        opening +
        " words and " +
        (topic
          ? (mentionsTopic(first, topic) ? "names" : "does not name") +
            " the topic. Answer the reader's question first."
          : "is present."),
      15,
      direct ? 15 : 0,
    ),
    check(
      "definition",
      "Explain the topic simply",
      "Say plainly what the topic is, e.g. “Teal is a deep blue-green shade.”",
      10,
      topic && definesTopic(body, topic) ? 10 : 0,
    ),
    questions.length
      ? check(
          "question-coverage",
          "Answer the questions people ask",
          answered +
            "/" +
            questions.length +
            " questions from the brief have a matching heading and answer.",
          20,
          (20 * answered) / questions.length,
        )
      : check(
          "question-coverage",
          "Answer the questions people ask",
          "Not applicable yet: get a search brief to see the questions people ask.",
          0,
          0,
        ),
    check(
      "list-table",
      "Scannable format",
      "Use a list, steps or comparison table where useful.",
      5,
      LIST_OR_TABLE.test(body) ? 5 : 0,
    ),
    check(
      "sources",
      "Support factual claims",
      external
        ? "Links to sources readers can check. Images do not count."
        : "Link a reliable source for facts readers may want to check.",
      15,
      external ? 15 : 0,
    ),
    check(
      "faq",
      "FAQ section",
      "An FAQ heading with real questions and answers (not the template text).",
      5,
      realFaq(body) ? 5 : 0,
    ),
    check(
      "entities",
      "Name things clearly",
      entities + " names of people, places, brands or products found.",
      5,
      entities >= 2 ? 5 : entities === 1 ? 2 : 0,
    ),
    experienceCheck(body),
    check(
      "author",
      "Show who wrote it",
      author
        ? "An author is selected for the byline."
        : "Publish tab → choose an author so readers know who wrote it.",
      10,
      author ? 10 : 0,
    ),
    check(
      "freshness",
      "Show freshness",
      freshness.message,
      freshness.weight,
      freshness.passed ? freshness.weight : 0,
    ),
  ];
}
