import { getIssueDescriptor } from "@/shared/audit-issues";
import { aiReadiness } from "./aiReadiness";

/** Site Audit's AI readiness (AEO / GEO) summary: score, three areas, fixes. */
export function AiReadinessCard({
  issueTypes,
  startedAt,
}: {
  issueTypes: readonly string[];
  startedAt: string;
}) {
  const result = aiReadiness(issueTypes, startedAt);
  if (!result) {
    return (
      <section className="rounded-xl border bg-card p-5 text-sm">
        <p className="font-medium">AI readiness (AEO / GEO)</p>
        <p className="mt-1 text-muted-foreground">
          Run a new audit to check whether ChatGPT, Gemini and Google AI can
          read, understand and quote your site.
        </p>
      </section>
    );
  }
  return (
    <section
      aria-labelledby="ai-readiness"
      className="space-y-4 rounded-xl border bg-card p-5"
    >
      <div>
        <p className="label-mono text-muted-foreground">
          AI readiness (AEO / GEO)
        </p>
        <h2 id="ai-readiness" className="text-lg font-semibold">
          Can ChatGPT, Gemini and Google AI cite your site?
        </h2>
      </div>
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <div className="rounded-lg bg-foreground p-4 text-background">
          <p className="text-xs text-background/70">AI readiness</p>
          <p className="text-3xl font-semibold tabular-nums">
            {result.score}/100
          </p>
          <p className="text-xs text-background/70">
            {result.fixes.length === 0
              ? "Nothing to fix"
              : `${result.fixes.length} fixes below`}
          </p>
        </div>
        {result.groups.map((group) => (
          <div key={group.group} className="rounded-lg border p-4">
            <p className="text-xs text-muted-foreground">{group.title}</p>
            <p
              className={`text-xl font-semibold ${
                group.verdict === "Pass"
                  ? "text-emerald-700 dark:text-emerald-400"
                  : group.verdict === "Weak"
                    ? "text-red-700 dark:text-red-400"
                    : "text-amber-700 dark:text-amber-400"
              }`}
            >
              {group.verdict}
            </p>
          </div>
        ))}
      </div>
      {result.fixes.length > 0 ? (
        <ol className="divide-y rounded-lg border">
          {result.fixes.map((fix) => {
            const descriptor = getIssueDescriptor(fix.issue);
            return (
              <li key={fix.issue} className="p-4">
                <details>
                  <summary className="cursor-pointer font-medium">
                    {descriptor?.title ?? fix.label}
                    <span className="ml-2 text-xs font-normal text-muted-foreground">
                      +{fix.weight} points
                    </span>
                  </summary>
                  {descriptor ? (
                    <div className="mt-2 space-y-2 text-sm">
                      <p className="text-muted-foreground">
                        {descriptor.explanation}
                      </p>
                      <p>{descriptor.howToFix}</p>
                    </div>
                  ) : null}
                </details>
              </li>
            );
          })}
        </ol>
      ) : null}
      <ul className="grid gap-1 text-sm sm:grid-cols-2">
        {result.checks.map((check) => (
          <li key={check.issue}>
            <span aria-hidden="true">{check.passed ? "✓" : "✗"}</span>{" "}
            <span className="sr-only">
              {check.passed ? "Passed:" : "Failed:"}
            </span>
            {check.label}
          </li>
        ))}
      </ul>
    </section>
  );
}
