import { Link } from "@tanstack/react-router";
import { Button } from "@/client/components/ui/button";
import type { TrackingAnswer } from "../trackingApi";
import { whyNotCited, type GapAction } from "../whyNotCited";

/** Reasons AI did not cite the site, from answers already saved (free). */
export function WhyNotCitedCard({
  projectId,
  answers,
  ownDomain,
}: {
  projectId: string;
  answers: TrackingAnswer[];
  ownDomain: string;
}) {
  const result = whyNotCited(answers, ownDomain);
  if (!result.answered) return null;
  return (
    <section
      aria-labelledby="why-not-cited"
      className="space-y-4 rounded-xl border p-5"
    >
      <div>
        <h2 id="why-not-cited" className="text-lg font-semibold">
          Why AI is not citing you
        </h2>
        <p className="text-sm text-muted-foreground">
          AI linked your site in {result.cited} of {result.answered} answers.
          Built from the answers this check already saved; no credits used.
        </p>
      </div>
      {result.gaps.length ? (
        <ol className="space-y-3">
          {result.gaps.map((gap, index) => (
            <li key={gap.kind} className="rounded-lg border p-4">
              <p className="font-medium">
                {index + 1}. {gap.title}
              </p>
              <p className="mt-1 text-sm">{gap.detail}</p>
              <ul className="mt-2 flex flex-wrap gap-2 text-xs">
                {gap.evidence.map((item) => (
                  <li
                    key={item}
                    className="max-w-full truncate rounded bg-muted px-2 py-1"
                    title={item}
                  >
                    {item}
                  </li>
                ))}
              </ul>
              <div className="mt-3">
                <GapButton projectId={projectId} action={gap.action} />
              </div>
            </li>
          ))}
        </ol>
      ) : (
        <p className="text-sm">
          AI linked your site in every saved answer. Keep these pages fresh and
          track more questions.
        </p>
      )}
    </section>
  );
}

function GapButton({
  projectId,
  action,
}: {
  projectId: string;
  action: GapAction;
}) {
  if (action.kind === "write")
    return (
      <Button
        size="sm"
        nativeButton={false}
        render={
          <Link
            to="/p/$projectId/content-editor"
            params={{ projectId }}
            search={{ keyword: action.keyword }}
          />
        }
      >
        {action.label}
      </Button>
    );
  if (action.kind === "external")
    return (
      <Button
        size="sm"
        variant="outline"
        nativeButton={false}
        render={
          <a href={action.href} target="_blank" rel="noopener noreferrer" />
        }
      >
        {action.label}
      </Button>
    );
  return (
    <Button
      size="sm"
      variant="outline"
      nativeButton={false}
      render={
        <Link
          to="/p/$projectId/prompt-tracking"
          params={{ projectId }}
          hash="tracker-setup"
        />
      }
    >
      {action.label}
    </Button>
  );
}
