import { useState } from "react";
import { toast } from "sonner";
import { Button } from "@/client/components/ui/button";
import { Spinner } from "@/client/components/ui/spinner";
import {
  type Overview,
  type Review,
  type ReviewsResult,
  reviewsResultSchema,
  runResult,
  type Theme,
  useDraftReply,
  useStartRun,
} from "./localVisibilityApi";
import { formatWhen, MissingNote, PaidButton, RunPanel } from "./RunPanel";

export function ReviewsTab({
  projectId,
  overview,
}: {
  projectId: string;
  overview: Overview;
}) {
  const start = useStartRun(projectId);
  const run = overview.runs.reviews;
  const result = runResult(run, reviewsResultSchema);
  const rivals = overview.watch.competitors.length;
  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-sm text-muted-foreground">
          Your last 100 reviews
          {rivals > 0
            ? ` and the last 50 of each of the top ${rivals} from your profile check`
            : ""}
          {overview.aiReady
            ? ", grouped into what customers praise and complain about"
            : ""}
          .
        </p>
        <PaidButton
          label="Analyse reviews"
          credits={overview.prices.reviews}
          pending={start.isPending || run?.status === "running"}
          onClick={() => start.mutate({ kind: "reviews" })}
        />
      </div>
      <RunPanel
        projectId={projectId}
        run={run}
        source="Google reviews"
        error={start.error}
      >
        {result ? (
          <ReviewsView
            projectId={projectId}
            result={result}
            replyCredits={overview.prices.reply}
            aiReady={overview.aiReady}
          />
        ) : null}
      </RunPanel>
    </div>
  );
}

function ReviewsView({
  projectId,
  result,
  replyCredits,
  aiReady,
}: {
  projectId: string;
  result: ReviewsResult;
  replyCredits: number;
  aiReady: boolean;
}) {
  return (
    <div className="space-y-4">
      <p className="font-medium">{result.summary}</p>
      <MissingNote missing={result.missing} />
      <div className="grid gap-4 md:grid-cols-2">
        {result.themes.length > 0 ? (
          <ThemeList title="Themes in your reviews" themes={result.themes} />
        ) : null}
        {result.rivals
          .filter((r) => r.themes.length > 0)
          .map((r) => (
            <ThemeList
              key={r.cid}
              title={`${r.title}${r.replyRate !== null ? ` · replies to ${r.replyRate}%` : ""}`}
              themes={r.themes}
            />
          ))}
      </div>
      {result.unanswered.length > 0 ? (
        <section className="space-y-2">
          <h3 className="font-medium">
            Reviews without a reply ({result.unanswered.length})
          </h3>
          <ul className="space-y-3">
            {result.unanswered.map((review) => (
              <ReviewItem
                key={review.id}
                projectId={projectId}
                review={review}
                replyCredits={replyCredits}
                aiReady={aiReady}
              />
            ))}
          </ul>
        </section>
      ) : null}
    </div>
  );
}

function ThemeList({ title, themes }: { title: string; themes: Theme[] }) {
  return (
    <section className="rounded-lg border p-3">
      <h3 className="mb-2 text-sm font-medium">{title}</h3>
      <ul className="space-y-1 text-sm">
        {themes.map((t) => (
          <li key={t.name} className="flex justify-between gap-2">
            <span>{t.name}</span>
            <span className="text-muted-foreground">
              {t.positive > 0 ? `+${t.positive}` : ""}
              {t.positive > 0 && t.negative > 0 ? " · " : ""}
              {t.negative > 0 ? `−${t.negative}` : ""}
            </span>
          </li>
        ))}
      </ul>
      <p className="mt-2 text-xs text-muted-foreground">
        Reviews mentioning the theme: positive (+), negative (−).
      </p>
    </section>
  );
}

function ReviewItem({
  projectId,
  review,
  replyCredits,
  aiReady,
}: {
  projectId: string;
  review: Review;
  replyCredits: number;
  aiReady: boolean;
}) {
  const draft = useDraftReply(projectId, review.id);
  const [reply, setReply] = useState(review.replyDraft ?? "");
  const write = (rewrite: boolean) =>
    draft.mutate(
      { reviewId: review.id, rewrite },
      { onSuccess: (data) => setReply(data.reply) },
    );
  return (
    <li className="space-y-2 rounded-lg border p-3 text-sm">
      <p className="text-muted-foreground">
        {"★".repeat(review.rating)} · {review.author} · {formatWhen(review.at)}
      </p>
      {review.text ? <p>{review.text}</p> : null}
      {reply ? (
        <p className="rounded-md bg-muted p-2 whitespace-pre-wrap">{reply}</p>
      ) : null}
      {draft.error ? (
        <p role="alert" className="text-destructive">
          {draft.error.message}
        </p>
      ) : null}
      <div className="flex flex-wrap gap-2">
        {aiReady ? (
          <Button
            size="sm"
            variant="outline"
            disabled={draft.isPending}
            onClick={() => write(Boolean(reply))}
          >
            {draft.isPending ? <Spinner /> : null}
            {reply ? "Rewrite" : "Draft a reply"} · up to {replyCredits} credit
            {replyCredits === 1 ? "" : "s"}
          </Button>
        ) : null}
        {reply ? (
          <Button
            size="sm"
            variant="ghost"
            onClick={() =>
              void navigator.clipboard
                .writeText(reply)
                .then(() => toast.success("Reply copied"))
            }
          >
            Copy reply
          </Button>
        ) : null}
        {review.url ? (
          <Button
            size="sm"
            variant="ghost"
            render={<a href={review.url} target="_blank" rel="noreferrer" />}
          >
            Open on Google ↗
          </Button>
        ) : null}
      </div>
    </li>
  );
}
