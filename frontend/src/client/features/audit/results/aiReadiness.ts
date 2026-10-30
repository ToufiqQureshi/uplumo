import { sortBy } from "remeda";

/**
 * AI readiness (AEO / GEO) score from issues the audit already found. Each
 * check fails when its issue type is present; the score is 100 minus the
 * weights of failed checks. Free: no extra request.
 */

// Audits started before the crawler ran these checks cannot be scored: a
// missing issue there means "not checked", not "passed".
const AI_CHECKS_SINCE = "2026-10-26T12:50:00Z";

type Group = "read" | "know" | "quote";

const CHECKS = [
  {
    issue: "ai-crawlers-blocked",
    group: "read",
    label: "AI bots allowed in robots.txt",
    weight: 30,
  },
  {
    issue: "javascript-rendering-suspected",
    group: "read",
    label: "Content in HTML, not only JavaScript",
    weight: 10,
  },
  {
    issue: "missing-llms-txt",
    group: "read",
    label: "llms.txt file (optional)",
    weight: 5,
  },
  {
    issue: "missing-business-schema",
    group: "know",
    label: "Business details AI can read (schema)",
    weight: 25,
  },
  {
    issue: "several-phone-numbers",
    group: "know",
    label: "One phone number everywhere",
    weight: 10,
  },
  {
    issue: "no-question-answers",
    group: "quote",
    label: "Pages that answer customer questions",
    weight: 20,
  },
] as const satisfies readonly {
  issue: string;
  group: Group;
  label: string;
  weight: number;
}[];

const GROUPS = ["read", "know", "quote"] as const;

const AI_GROUPS: Record<Group, string> = {
  read: "Can AI read it?",
  know: "Does AI know who you are?",
  quote: "Can AI quote your answers?",
};

export function aiReadiness(issueTypes: readonly string[], startedAt: string) {
  if (Date.parse(startedAt) < Date.parse(AI_CHECKS_SINCE)) return null;
  const found = new Set(issueTypes);
  const checks = CHECKS.map((check) => ({
    ...check,
    passed: !found.has(check.issue),
  }));
  const failed = checks.filter((check) => !check.passed);
  const groups = GROUPS.map((group) => {
    const inGroup = checks.filter((check) => check.group === group);
    const fails = inGroup.filter((check) => !check.passed).length;
    return {
      group,
      title: AI_GROUPS[group],
      verdict:
        fails === 0 ? "Pass" : fails === inGroup.length ? "Weak" : "Partly",
    };
  });
  return {
    score: 100 - failed.reduce((sum, check) => sum + check.weight, 0),
    checks,
    groups,
    // Biggest wins first.
    fixes: sortBy(failed, [(check) => check.weight, "desc"]),
  };
}
