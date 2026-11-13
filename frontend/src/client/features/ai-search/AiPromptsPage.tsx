import { useState } from "react";
import { Link } from "@tanstack/react-router";
import { X } from "lucide-react";
import { sortBy } from "remeda";
import { cn } from "cn";
import { Button } from "@/client/components/ui/button";
import { useLatestAnswers } from "@/client/features/insights/useLatestAnswers";
import { MarkdownAnswer } from "./components/MarkdownAnswer";
import { engineLabel } from "./platformLabels";
import {
  promptMatrix,
  type CellState,
  type PromptRow,
} from "./aiVisibilityMetrics";
import type { TrackingAnswer } from "./trackingApi";
import { EngineName } from "./EngineName";
import { EditPageButton } from "@/client/features/content-editor/EditPageButton";
import { EngineLogo } from "@/client/components/BrandLogo";

type Show = "all" | "gaps" | "named";

const FILTERS: { id: Show; label: string }[] = [
  { id: "all", label: "All questions" },
  { id: "gaps", label: "AI leaves you out" },
  { id: "named", label: "AI names you" },
];

const CELL: Record<CellState, { label: string; className: string }> = {
  named: { label: "Named", className: "bg-success/15 text-success" },
  cited: { label: "Linked only", className: "bg-chart-2/15 text-foreground" },
  absent: { label: "Not named", className: "bg-muted text-muted-foreground" },
  failed: { label: "No answer", className: "text-muted-foreground" },
  pending: { label: "Waiting", className: "text-muted-foreground" },
};

/**
 * Every tracked question against every engine, with the saved answer behind
 * each cell one click away. Reads saved answers only.
 */
export function AiPromptsPage({
  projectId,
  show,
  initialOpen,
  onShowChange,
}: {
  projectId: string;
  show: Show;
  initialOpen: { prompt: string; engine: string } | null;
  onShowChange: (next: Show) => void;
}) {
  const ai = useLatestAnswers(projectId);
  const [open, setOpen] = useState(initialOpen);

  if (ai.isPending) {
    return (
      <p role="status" className="text-sm text-muted-foreground">
        Reading saved answers…
      </p>
    );
  }
  if (ai.isError) {
    return (
      <p role="alert" className="text-sm text-destructive">
        Could not load the answers. Refresh the page to try again.
      </p>
    );
  }
  if (!ai.run) {
    return (
      <section className="flex flex-col items-start gap-3 rounded-xl border bg-card p-6">
        <h2 className="text-lg font-semibold">No answers saved yet</h2>
        <p className="text-sm text-muted-foreground">
          Run a check from Setup & history, then every answer shows up here.
        </p>
        <Button
          nativeButton={false}
          render={
            <Link
              to="/p/$projectId/ai-visibility/setup"
              params={{ projectId }}
            />
          }
        >
          Go to setup
        </Button>
      </section>
    );
  }

  const rows = promptMatrix(ai.answers);
  const engines = sortBy(
    [...new Set(ai.answers.map((a) => a.engine))],
    engineLabel,
  );
  const visible = rows.filter((r) =>
    show === "gaps"
      ? r.gap
      : show === "named"
        ? Object.values(r.cells).some((c) => c.state === "named")
        : true,
  );
  const selected = open
    ? rows.find((r) => r.prompt === open.prompt)?.cells[open.engine]?.answer
    : undefined;

  return (
    <div className="flex flex-col gap-4">
      <div
        className="flex flex-wrap items-center gap-2"
        role="group"
        aria-label="Filter questions"
      >
        {FILTERS.map((f) => {
          const count =
            f.id === "all"
              ? rows.length
              : f.id === "gaps"
                ? rows.filter((r) => r.gap).length
                : rows.filter((r) =>
                    Object.values(r.cells).some((c) => c.state === "named"),
                  ).length;
          return (
            <Button
              key={f.id}
              variant={show === f.id ? "default" : "outline"}
              aria-pressed={show === f.id}
              onClick={() => onShowChange(f.id)}
            >
              {f.label} · {count}
            </Button>
          );
        })}
        <span className="ml-auto text-xs text-muted-foreground">
          Check of {new Date(ai.run.createdAt).toLocaleDateString()} · click a
          cell to read the answer
        </span>
      </div>

      <div className="overflow-x-auto rounded-xl border bg-card">
        <table className="w-full min-w-[720px] text-sm">
          <thead>
            <tr className="border-b text-left text-xs uppercase tracking-wide text-muted-foreground">
              <th scope="col" className="px-4 py-3 font-medium">
                Question
              </th>
              {engines.map((e) => (
                <th key={e} scope="col" className="px-2 py-3 font-medium">
                  <EngineName engine={e} />
                </th>
              ))}
              <th scope="col" className="px-4 py-3 font-medium">
                Named instead
              </th>
            </tr>
          </thead>
          <tbody>
            {visible.map((row) => (
              <tr key={row.prompt} className="border-b last:border-0">
                <th
                  scope="row"
                  className="max-w-xs px-4 py-3 text-left font-medium"
                >
                  {row.prompt}
                  {row.topic && row.topic !== "General" ? (
                    <span className="block text-xs font-normal text-muted-foreground">
                      {row.topic}
                    </span>
                  ) : null}
                </th>
                {engines.map((e) => (
                  <td key={e} className="px-2 py-3">
                    <Cell
                      row={row}
                      engine={e}
                      active={open?.prompt === row.prompt && open.engine === e}
                      onOpen={() => setOpen({ prompt: row.prompt, engine: e })}
                    />
                  </td>
                ))}
                <td className="px-4 py-3 text-muted-foreground">
                  {row.namedInstead.length ? row.namedInstead.join(", ") : "—"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!visible.length ? (
          <p className="p-6 text-sm text-muted-foreground">
            {show === "gaps"
              ? "AI names you somewhere for every question. Add harder questions in Setup."
              : "No question matches this filter."}
          </p>
        ) : null}
      </div>

      {selected ? (
        <AnswerProof
          projectId={projectId}
          answer={selected}
          onClose={() => setOpen(null)}
        />
      ) : null}
    </div>
  );
}

function Cell({
  row,
  engine,
  active,
  onOpen,
}: {
  row: PromptRow;
  engine: string;
  active: boolean;
  onOpen: () => void;
}) {
  const cell = row.cells[engine];
  if (!cell) return <span className="text-muted-foreground">—</span>;
  const look = CELL[cell.state];
  return (
    <button
      type="button"
      onClick={onOpen}
      aria-pressed={active}
      aria-label={`${engineLabel(engine)} on "${row.prompt}": ${look.label}. Read the answer.`}
      className={cn(
        "inline-flex min-h-9 items-center rounded-full px-3 text-xs font-semibold outline-offset-2 hover:ring-1 hover:ring-border aria-pressed:ring-2 aria-pressed:ring-primary",
        look.className,
      )}
    >
      {look.label}
    </button>
  );
}

/** The saved answer behind a cell: what AI said, who it named, what it cited. */
function AnswerProof({
  projectId,
  answer,
  onClose,
}: {
  projectId: string;
  answer: TrackingAnswer;
  onClose: () => void;
}) {
  const own = answer.brands[0];
  const rivals = answer.brands.slice(1).filter((b) => b.mentioned);
  return (
    <section
      aria-label="Saved answer"
      className="flex flex-col gap-4 rounded-xl border border-primary/40 bg-card p-5"
    >
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="font-mono text-xs uppercase tracking-wider text-muted-foreground">
            <EngineLogo engine={answer.engine} size={14} /> What{" "}
            {engineLabel(answer.engine)} answered
            {answer.collectedAt
              ? ` · ${new Date(answer.collectedAt).toLocaleString()}`
              : ""}
          </p>
          <h2 className="mt-1 text-lg font-semibold">“{answer.prompt}”</h2>
        </div>
        <Button
          variant="ghost"
          size="icon"
          aria-label="Close answer"
          onClick={onClose}
        >
          <X />
        </Button>
      </div>

      <dl className="grid gap-3 text-sm sm:grid-cols-3">
        <div className="rounded-lg bg-muted px-3 py-2">
          <dt className="text-xs text-muted-foreground">
            {own?.name ?? "You"}
          </dt>
          <dd
            className={cn(
              "font-semibold",
              own?.mentioned ? "text-success" : "text-destructive",
            )}
          >
            {own?.mentioned ? "Named" : "Not named"} ·{" "}
            {own?.cited ? "site linked" : "site not linked"}
          </dd>
        </div>
        <div className="rounded-lg bg-muted px-3 py-2">
          <dt className="text-xs text-muted-foreground">Recommended</dt>
          <dd className="font-semibold">
            {answer.recommended.length
              ? answer.recommended
                  .slice(0, 4)
                  .map((r) => r.name)
                  .join(", ")
              : rivals.length
                ? rivals.map((r) => r.name).join(", ")
                : "No business named"}
          </dd>
        </div>
        <div className="rounded-lg bg-muted px-3 py-2">
          <dt className="text-xs text-muted-foreground">Sources cited</dt>
          <dd className="font-semibold">
            {answer.citations.length
              ? `${answer.citations.length} source${answer.citations.length === 1 ? "" : "s"}`
              : "None"}
          </dd>
        </div>
      </dl>

      <div className="rounded-lg bg-muted/50 p-4 text-sm">
        {answer.answer ? (
          <MarkdownAnswer text={answer.answer} />
        ) : (
          <p className="text-muted-foreground">
            {answer.status === "completed"
              ? "This engine returned no answer for the question."
              : answer.errorCode
                ? `Not collected: ${answer.errorCode.replaceAll("_", " ")}.`
                : "Not collected yet."}
          </p>
        )}
      </div>

      {answer.citations.length ? (
        <div className="flex flex-col gap-2">
          <h3 className="text-sm font-semibold">Sources it trusted</h3>
          <ul className="flex flex-col gap-1 text-sm">
            {answer.citations.map((c) => (
              <li
                key={c.url}
                className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1"
              >
                <span className="min-w-0 truncate">
                  <a
                    className="text-primary underline underline-offset-2"
                    href={c.url}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    {c.title ?? c.url}
                  </a>
                  <span className="text-muted-foreground">
                    {" "}
                    · {c.domain ?? c.source ?? ""}
                    {c.matchedBrand ? " · your site" : ""}
                  </span>
                </span>
                {c.matchedBrand ? (
                  <EditPageButton
                    projectId={projectId}
                    url={c.url}
                    variant="ghost"
                    label="Improve this page"
                  />
                ) : null}
              </li>
            ))}
          </ul>
        </div>
      ) : null}
    </section>
  );
}
