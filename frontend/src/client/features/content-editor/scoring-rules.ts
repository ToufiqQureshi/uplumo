import { sortBy } from "remeda";
import { makeScoreCheck as check, type ScoreCheck } from "./scoring-utils";

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
  topicWords,
  words,
} from "./scoring-text";
import type { ContentBrief, ContentDraftInput } from "./scoring";

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

// First-hand experience is what Google's helpful-content guidance asks for
// most: something the writer saw, tried, measured or was asked.
const EXPERIENCE = [
  /\b(?:i|we|my|our)\b[^.!?\n]{0,60}\b(?:tried|tested|wore|used|visited|bought|found|saw|asked|made|cooked|built|measured|compared|noticed|learned|worked|helped|ran|checked)\b/iu,
  /\b(?:in my|in our|from my|from our)\s+(?:experience|tests?|store|shop|clinic|practice|work)\b/iu,
  /\b(?:maine|humne|mera|meri|hamare|hamari)\b[^.!?\n]{0,60}\b(?:dekha|kiya|try|pehna|use|test|khareeda|poocha)\b/iu,
];
function experienceCheck(body: string): ScoreCheck {
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

function searchDescriptionCheck(d: ContentDraftInput): ScoreCheck {
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

// A definition sentence: the topic or one of its words, then "is", "means"
// or a Hindi/Hinglish equivalent, near the start of the sentence.
const DEFINES =
  /\b(?:is|are|means|refers to|stands for)\b|\b(?:hota hai|hoti hai|hote hain|matlab|yaani)\b|(?:होता है|होती है|मतलब)/iu;
function definesTopic(body: string, topic: string) {
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
function realFaq(body: string) {
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
