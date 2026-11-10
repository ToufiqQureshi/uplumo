import { Badge } from "@/client/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/client/components/ui/table";
import {
  type CheckResult,
  type Contender,
  checkResultSchema,
  type Overview,
  runResult,
  useStartRun,
} from "./localVisibilityApi";
import { MissingNote, PaidButton, RunPanel } from "./RunPanel";

const STATUS = {
  missing: { label: "Missing", variant: "destructive" },
  behind: { label: "Behind", variant: "warning" },
  fine: { label: "Fine", variant: "success" },
} as const;

export function CheckTab({
  projectId,
  overview,
}: {
  projectId: string;
  overview: Overview;
}) {
  const start = useStartRun(projectId);
  const run = overview.runs.check;
  const result = runResult(run, checkResultSchema);
  const term = overview.business?.searchTerm ?? "";
  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <p className="text-sm text-muted-foreground">
          One Google Maps search for “{term}” at your address, then your profile
          and the top 3 profiles side by side.
        </p>
        <PaidButton
          label="Run profile check"
          credits={overview.prices.check}
          pending={start.isPending || run?.status === "running"}
          onClick={() => start.mutate({ kind: "check" })}
        />
      </div>
      <RunPanel
        projectId={projectId}
        run={run}
        source={`Google Maps results for “${term}” at your address, and the public profiles of the top 3`}
        error={start.error}
      >
        {result ? <CheckView result={result} /> : null}
      </RunPanel>
    </div>
  );
}

function CheckView({ result }: { result: CheckResult }) {
  return (
    <div className="space-y-4">
      <p className="font-medium">{result.summary}</p>
      <MissingNote missing={result.missing} />
      {result.fixes.length > 0 ? (
        <ul className="divide-y rounded-lg border">
          {result.fixes.map((fix) => (
            <li
              key={fix.title}
              className="flex flex-col gap-1 p-3 sm:flex-row sm:items-start sm:gap-3"
            >
              <Badge variant={STATUS[fix.status].variant}>
                {STATUS[fix.status].label}
              </Badge>
              <div className="min-w-0 flex-1">
                <p className="font-medium">{fix.title}</p>
                <p className="text-sm text-muted-foreground">{fix.detail}</p>
              </div>
              {fix.others ? (
                <p className="text-xs whitespace-nowrap text-muted-foreground">
                  you {fix.you} · others {fix.others}
                </p>
              ) : null}
            </li>
          ))}
        </ul>
      ) : null}
      <SideBySide result={result} />
    </div>
  );
}

function SideBySide({ result }: { result: CheckResult }) {
  const rows = [...result.rivals, result.you].filter((c) => c.profile);
  if (rows.length < 2) return null;
  return (
    <div className="overflow-x-auto rounded-lg border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Business</TableHead>
            <TableHead className="text-right">Map rank</TableHead>
            <TableHead className="text-right">Rating</TableHead>
            <TableHead className="text-right">Reviews</TableHead>
            <TableHead className="text-right">Photos</TableHead>
            <TableHead className="text-right">Posts / 30 d</TableHead>
            <TableHead className="text-right">Categories</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {rows.map((row) => (
            <Row key={row.place.cid} row={row} you={row === result.you} />
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

function Row({ row, you }: { row: Contender; you: boolean }) {
  const profile = row.profile!;
  return (
    <TableRow className={you ? "bg-primary/5 font-medium" : undefined}>
      <TableCell>
        {row.place.title || profile.title}
        {you ? " (you)" : ""}
      </TableCell>
      <TableCell className="text-right">
        {row.place.rank > 0 ? row.place.rank : "20+"}
      </TableCell>
      <TableCell className="text-right">{profile.rating ?? "–"}</TableCell>
      <TableCell className="text-right">{profile.reviews}</TableCell>
      <TableCell className="text-right">{profile.photos}</TableCell>
      <TableCell className="text-right">{row.postsLast30 ?? "–"}</TableCell>
      <TableCell className="text-right">{profile.categories.length}</TableCell>
    </TableRow>
  );
}
