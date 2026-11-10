import { Badge } from "@/client/components/ui/badge";
import { Button } from "@/client/components/ui/button";
import { Spinner } from "@/client/components/ui/spinner";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/client/components/ui/table";
import {
  type Overview,
  runResult,
  type WatchResult,
  useSetWatch,
  watchResultSchema,
} from "./localVisibilityApi";
import { formatWhen, RunPanel } from "./RunPanel";

export function WatchTab({
  projectId,
  overview,
}: {
  projectId: string;
  overview: Overview;
}) {
  const setWatch = useSetWatch(projectId);
  const { watch } = overview;
  const run = overview.runs.watch;
  const result = runResult(run, watchResultSchema);
  if (watch.competitors.length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        Run the profile check first. Its top 3 are the competitors watched every
        week.
      </p>
    );
  }
  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-3 rounded-lg border p-3 sm:flex-row sm:items-center">
        <div className="min-w-0 flex-1 text-sm">
          <p className="font-medium">
            Weekly watch: {watch.competitors.map((c) => c.title).join(", ")}
          </p>
          <p className="text-muted-foreground">
            {watch.enabled && watch.nextRunAt
              ? `On · next check ${formatWhen(watch.nextRunAt)} · up to ${watch.creditsPerWeek} credits a week · turn off any time`
              : `Off · no automatic spend. Turning it on checks them now and every week after, up to ${watch.creditsPerWeek} credits a week.`}
          </p>
        </div>
        <Button
          variant={watch.enabled ? "outline" : "default"}
          disabled={setWatch.isPending}
          onClick={() => setWatch.mutate(!watch.enabled)}
        >
          {setWatch.isPending ? <Spinner /> : null}
          {watch.enabled
            ? "Turn off"
            : `Turn on · up to ${watch.creditsPerWeek} credits now`}
        </Button>
      </div>
      {setWatch.error ? (
        <p role="alert" className="text-sm text-destructive">
          {setWatch.error.message}
        </p>
      ) : null}
      <RunPanel
        projectId={projectId}
        run={run}
        source="weekly profile snapshots of your competitors"
        error={null}
      >
        {result ? <Changes result={result} /> : null}
      </RunPanel>
    </div>
  );
}

function signed(value: number | null, digits = 0) {
  if (value === null) return "–";
  if (value === 0) return "=";
  return `${value > 0 ? "+" : ""}${value.toFixed(digits)}`;
}

function Changes({ result }: { result: WatchResult }) {
  return (
    <div className="space-y-3">
      <p className="font-medium">{result.summary}</p>
      {result.since ? (
        <p className="text-xs text-muted-foreground">
          Changes since {formatWhen(result.since)}.
        </p>
      ) : null}
      <div className="overflow-x-auto rounded-lg border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Competitor</TableHead>
              <TableHead className="text-right">Rating</TableHead>
              <TableHead className="text-right">New reviews</TableHead>
              <TableHead className="text-right">Posts, 7 days</TableHead>
              <TableHead>Profile changes</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {result.changes.map((c) => (
              <TableRow key={c.cid}>
                <TableCell>{c.title}</TableCell>
                <TableCell className="text-right">
                  {c.rating ?? "–"}{" "}
                  <span className="text-xs text-muted-foreground">
                    {c.ratingDelta !== null ? signed(c.ratingDelta, 1) : ""}
                  </span>
                </TableCell>
                <TableCell className="text-right">
                  {signed(c.newReviews)}
                </TableCell>
                <TableCell className="text-right">{c.newPosts}</TableCell>
                <TableCell className="space-x-1">
                  {c.addedCategories.map((cat) => (
                    <Badge key={cat} variant="info">
                      + {cat}
                    </Badge>
                  ))}
                  {c.removedCategories.map((cat) => (
                    <Badge key={cat} variant="outline">
                      − {cat}
                    </Badge>
                  ))}
                  {c.hoursChanged ? (
                    <Badge variant="outline">hours changed</Badge>
                  ) : null}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
