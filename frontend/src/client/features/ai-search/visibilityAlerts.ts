import { sortBy } from "remeda";
import type { TrackingAnswer } from "./trackingApi";
import { engineLabel } from "./platformLabels";

/**
 * What changed between two tracker checks with the same questions, engines
 * and market. Built from answers already saved: no provider call.
 *
 * Only pairs where both checks got a real answer are compared, so a failed
 * collection never reads as "AI stopped naming you".
 */
export type VisibilityAlert = {
  kind: "lost" | "won" | "new_rival" | "negative" | "new_source";
  /** Higher first: lost and negative before wins. */
  severity: number;
  title: string;
  detail: string;
  /** The answer to open as proof, when there is one. */
  prompt?: string;
  engine?: string;
};

const measured = (a: TrackingAnswer | undefined): a is TrackingAnswer =>
  !!a && a.status === "completed" && a.answer.trim() !== "";

const key = (a: TrackingAnswer) => `${a.engine}\u0000${a.prompt}`;

const bare = (d: string) => d.toLowerCase().replace(/^www\./, "");

function citedDomains(a: TrackingAnswer): string[] {
  return a.citations.flatMap((c) => {
    if (c.domain) return [bare(c.domain)];
    try {
      return [bare(new URL(c.url).hostname)];
    } catch {
      return [];
    }
  });
}

/** Businesses AI named in an answer: tracked rivals it mentioned and its picks. */
function namedRivals(a: TrackingAnswer): string[] {
  const own = a.brands[0]?.name.toLowerCase();
  return [
    ...a.brands
      .slice(1)
      .filter((b) => b.mentioned)
      .map((b) => b.name),
    ...a.recommended.map((r) => r.name),
  ].filter((n) => n.toLowerCase() !== own);
}

export function visibilityAlerts(
  previous: TrackingAnswer[],
  latest: TrackingAnswer[],
): VisibilityAlert[] {
  const before = new Map(previous.filter(measured).map((a) => [key(a), a]));
  if (before.size === 0) return [];
  const alerts: VisibilityAlert[] = [];

  const rivalsBefore = new Set(
    [...before.values()].flatMap(namedRivals).map((n) => n.toLowerCase()),
  );
  const sourcesBefore = new Set([...before.values()].flatMap(citedDomains));
  // Businesses AI recommends that the customer does not track yet.
  const newRivals = new Map<string, { name: string; answer: TrackingAnswer }>();
  const newSources = new Map<string, number>();

  for (const now of latest.filter(measured)) {
    const was = before.get(key(now));
    const engine = engineLabel(now.engine);
    if (was) {
      const ownBefore = !!was.brands[0]?.mentioned;
      const ownNow = !!now.brands[0]?.mentioned;
      if (ownBefore && !ownNow) {
        const instead = namedRivals(now).slice(0, 2);
        alerts.push({
          kind: "lost",
          severity: 3,
          title: `${engine} stopped naming you for “${now.prompt}”`,
          detail: instead.length
            ? `It now recommends ${instead.join(" and ")}.`
            : "It no longer names your brand in this answer.",
          prompt: now.prompt,
          engine: now.engine,
        });
      } else if (!ownBefore && ownNow) {
        alerts.push({
          kind: "won",
          severity: 1,
          title: `${engine} now names you for “${now.prompt}”`,
          detail: "It did not in the previous check.",
          prompt: now.prompt,
          engine: now.engine,
        });
      }
    }
    if (now.sentiment === "negative" && was?.sentiment !== "negative") {
      alerts.push({
        kind: "negative",
        severity: 3,
        title: `${engine} describes you negatively for “${now.prompt}”`,
        detail: now.sentimentQuote
          ? `“${now.sentimentQuote}”`
          : "Read the answer to see what it says.",
        prompt: now.prompt,
        engine: now.engine,
      });
    }
    const tracked = new Set(now.brands.map((b) => b.name.toLowerCase()));
    for (const name of now.recommended.map((r) => r.name)) {
      const id = name.toLowerCase();
      if (!tracked.has(id) && !rivalsBefore.has(id) && !newRivals.has(id)) {
        newRivals.set(id, { name, answer: now });
      }
    }
    if (!now.brands[0]?.cited) {
      for (const domain of new Set(citedDomains(now))) {
        if (!sourcesBefore.has(domain)) {
          newSources.set(domain, (newSources.get(domain) ?? 0) + 1);
        }
      }
    }
  }

  for (const { name, answer: a } of newRivals.values()) {
    alerts.push({
      kind: "new_rival",
      severity: 2,
      title: `New business recommended: ${name}`,
      detail: `${engineLabel(a.engine)} named it for “${a.prompt}”. It was not named in the previous check; add it as a competitor to track it.`,
      prompt: a.prompt,
      engine: a.engine,
    });
  }
  for (const [domain, answers] of sortBy(
    [...newSources.entries()],
    [([, n]) => n, "desc"],
  ).slice(0, 3)) {
    alerts.push({
      kind: "new_source",
      severity: 2,
      title: `AI started citing ${domain}`,
      detail: `Cited in ${answers} answer${answers === 1 ? "" : "s"} that do not link your site. Getting listed there can bring you into those answers.`,
    });
  }

  return sortBy(alerts, [(a) => a.severity, "desc"]);
}
