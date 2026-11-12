import { useState } from "react";
import { Link } from "@tanstack/react-router";
import { Button } from "@/client/components/ui/button";
import { useQuery } from "@tanstack/react-query";
import { cn } from "cn";
import { trackingApi, trackingKey, type TrackingAnswer } from "./trackingApi";
import { visibilityAlerts, type VisibilityAlert } from "./visibilityAlerts";

const KIND: Record<
  VisibilityAlert["kind"],
  { label: string; className: string }
> = {
  lost: { label: "Lost", className: "bg-destructive/10 text-destructive" },
  negative: {
    label: "Negative",
    className: "bg-destructive/10 text-destructive",
  },
  new_rival: { label: "New rival", className: "bg-chart-2/15 text-foreground" },
  new_source: { label: "New source", className: "bg-muted text-foreground" },
  won: { label: "Won", className: "bg-success/15 text-success" },
};

/**
 * Changes since the previous comparable check, read from saved answers of
 * both checks. Reading the previous check's answers is free.
 */
export function VisibilityAlerts({
  projectId,
  previousRunId,
  previousDate,
  latest,
}: {
  projectId: string;
  previousRunId: string;
  previousDate: string;
  latest: TrackingAnswer[];
}) {
  const [showAll, setShowAll] = useState(false);
  const previous = useQuery({
    queryKey: [...trackingKey(projectId), "answers", previousRunId],
    queryFn: () => trackingApi.answers(projectId, previousRunId),
    staleTime: 60_000,
  });
  if (!previous.data) return null;
  const alerts = visibilityAlerts(previous.data, latest);

  return (
    <section
      aria-label="Changes since the previous check"
      className="flex flex-col gap-3 rounded-xl border bg-card p-5"
    >
      <div>
        <h2 className="text-base font-semibold">
          What changed since {new Date(previousDate).toLocaleDateString()}
        </h2>
        <p className="text-sm text-muted-foreground">
          {alerts.length
            ? "Compared with the previous check of the same questions, engines and market."
            : "Nothing changed: AI named the same businesses and cited the same sites."}
        </p>
      </div>
      {alerts.length ? (
        <ul className="flex flex-col divide-y">
          {(showAll ? alerts : alerts.slice(0, 8)).map((a) => (
            <li
              key={`${a.kind}:${a.title}`}
              className="flex flex-wrap items-start gap-3 py-3"
            >
              <span
                className={cn(
                  "mt-0.5 shrink-0 rounded-full px-2.5 py-0.5 text-xs font-semibold",
                  KIND[a.kind].className,
                )}
              >
                {KIND[a.kind].label}
              </span>
              <div className="min-w-0 flex-1 basis-64 text-sm">
                <p className="font-medium">{a.title}</p>
                <p className="text-muted-foreground">{a.detail}</p>
              </div>
              {a.prompt && a.engine ? (
                <Link
                  to="/p/$projectId/ai-visibility/prompts"
                  params={{ projectId }}
                  search={{ q: a.prompt, e: a.engine }}
                  className="shrink-0 text-sm font-medium text-primary underline underline-offset-2"
                >
                  Read the answer
                </Link>
              ) : null}
            </li>
          ))}
        </ul>
      ) : null}
      {alerts.length > 8 && !showAll ? (
        <Button
          variant="outline"
          className="self-start"
          onClick={() => setShowAll(true)}
        >
          Show all {alerts.length} changes
        </Button>
      ) : null}
    </section>
  );
}
