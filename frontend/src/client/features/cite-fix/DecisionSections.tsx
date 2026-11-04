import { useState, type ReactNode } from "react";
import { cn } from "cn";
import { SafeExternalLink } from "@/client/components/SafeExternalLink";
import { engineLabel } from "@/client/features/ai-search/platformLabels";
import type { CiteFixProof, CiteFixReport } from "./citeFixApi";
import {
  allCells,
  headline,
  type Cell,
  type ListPage,
  type Matrix,
  type RivalRank,
  type Step,
  type StepKind,
} from "./citeFixInsights";

const dateFormat = new Intl.DateTimeFormat(undefined, {
  day: "numeric",
  month: "short",
  year: "numeric",
});

function formatDate(value: string): string {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? value : dateFormat.format(date);
}

/** The top of Cite Fix: how often AI names the site now, and what changed. */
export function CiteFixHeadline({
  proof,
  matrix,
}: {
  proof: CiteFixProof;
  matrix: Matrix | null;
}) {
  const facts = matrix ? headline(matrix) : null;
  const runs = proof.visibility?.runs ?? 0;
  const lostTimes = facts?.lost.reduce((sum, c) => sum + c.citedAnswers, 0);
  const lostEngines = [
    ...new Set(facts?.lost.map((cell) => engineLabel(cell.engine))),
  ];
  return (
    <section
      aria-label="What is happening"
      className="grid gap-6 rounded-2xl border bg-card p-5 md:grid-cols-2 md:p-6"
    >
      <div className="space-y-2">
        <p className="label-mono text-muted-foreground">
          Latest check · {formatDate(proof.checkedAt)}
        </p>
        <p className="font-serif text-7xl leading-none tabular-nums">
          {proof.citingYou}
          <span className="text-3xl text-muted-foreground">
            {" "}
            of {proof.answers}
          </span>
        </p>
        <p className="text-lg">AI answers name {proof.domain}.</p>
        {facts && facts.cited.length > 0 ? (
          <p className="text-sm text-muted-foreground">
            {facts.cited
              .slice(0, 3)
              .map(
                (cell) =>
                  `${engineLabel(cell.engine)} #${cell.latestRank} for “${cell.prompt}”`,
              )
              .join(" · ")}
          </p>
        ) : null}
      </div>
      {facts ? (
        <ul className="space-y-3 text-sm md:text-base">
          {facts.lost.length > 0 ? (
            <Fact tone="bad" label="Slipping">
              {lostEngines.join(", ")} named you {lostTimes} time
              {lostTimes === 1 ? "" : "s"} in earlier checks for{" "}
              {facts.lost.length === 1
                ? `“${facts.lost[0].prompt}”`
                : `${facts.lost.length} question and AI pairs`}
              , but not in the latest one.
            </Fact>
          ) : null}
          {facts.neverEngines.length > 0 ? (
            <Fact tone="neutral" label="Never">
              {facts.neverEngines.map(engineLabel).join(", ")}{" "}
              {facts.neverEngines.length === 1 ? "has" : "have"} not named you
              in {runs} check{runs === 1 ? "" : "s"}, for any question.
            </Fact>
          ) : null}
          {facts.strongest ? (
            <Fact tone="good" label="Working">
              {engineLabel(facts.strongest.engine)} names you in{" "}
              {facts.strongest.citedAnswers} of {facts.strongest.answers} checks
              for “{facts.strongest.prompt}”.
            </Fact>
          ) : null}
        </ul>
      ) : null}
    </section>
  );
}

const toneChip = {
  good: "border-success/40 bg-success/10 text-success",
  bad: "border-destructive/40 bg-destructive/10 text-destructive",
  neutral: "text-muted-foreground",
} as const;

function Fact({
  tone,
  label,
  children,
}: {
  tone: keyof typeof toneChip;
  label: string;
  children: ReactNode;
}) {
  return (
    <li className="flex items-start gap-3">
      <span
        className={cn(
          "mt-0.5 shrink-0 rounded-full border px-2 py-0.5 font-mono text-xs",
          toneChip[tone],
        )}
      >
        {label}
      </span>
      <span>{children}</span>
    </li>
  );
}

const cellStyle = {
  cited: "border-success/50 bg-success/10 text-success",
  lost: "border-destructive/50 bg-destructive/10 text-destructive",
  never: "text-muted-foreground",
} as const;

function cellLabel(cell: Cell): { main: string; sub: string } {
  if (cell.state === "cited")
    return {
      main: `#${cell.latestRank}`,
      sub: `${cell.citedAnswers}/${cell.answers}`,
    };
  if (cell.state === "lost")
    return { main: "lost", sub: `was ${cell.citedAnswers}×` };
  return { main: "–", sub: "" };
}

function cellStory(cell: Cell): string {
  const engine = engineLabel(cell.engine);
  if (cell.state === "cited")
    return `${engine} named you at #${cell.latestRank} in the latest check, and in ${cell.citedAnswers} of ${cell.answers} checks.`;
  if (cell.state === "lost")
    return `${engine} named you in ${cell.citedAnswers} earlier check${cell.citedAnswers === 1 ? "" : "s"}, but not in the latest one.`;
  return `${engine} has not named you for this question in ${cell.answers} check${cell.answers === 1 ? "" : "s"}.`;
}

/** Each tracked question against each AI engine; a cell opens its story. */
export function CitationMatrix({ matrix }: { matrix: Matrix }) {
  const first =
    allCells(matrix).find((cell) => cell.state === "lost") ??
    allCells(matrix)[0];
  const [selected, setSelected] = useState(
    first ? `${first.prompt} ${first.engine}` : "",
  );
  const pick = allCells(matrix).find(
    (cell) => `${cell.prompt} ${cell.engine}` === selected,
  );
  const columns = {
    gridTemplateColumns: `minmax(0, 2fr) repeat(${matrix.engines.length}, minmax(3rem, 1fr))`,
  };
  return (
    <section
      aria-labelledby="cite-fix-matrix"
      className="space-y-4 rounded-2xl border bg-card p-5"
    >
      <div className="flex flex-wrap items-end justify-between gap-3">
        <h2 id="cite-fix-matrix" className="font-serif text-2xl">
          Each question, each AI
        </h2>
        <div className="flex flex-wrap gap-2 text-xs">
          <span
            className={cn("rounded-full border px-2 py-0.5", cellStyle.cited)}
          >
            #5 named now
          </span>
          <span
            className={cn("rounded-full border px-2 py-0.5", cellStyle.lost)}
          >
            named before, not now
          </span>
          <span className="rounded-full border px-2 py-0.5 text-muted-foreground">
            – never named
          </span>
        </div>
      </div>
      <div className="overflow-x-auto">
        <div className="min-w-[36rem] space-y-2">
          <div
            className="grid gap-2 font-mono text-xs text-muted-foreground"
            style={columns}
          >
            <span>Question</span>
            {matrix.engines.map((engine) => (
              <span key={engine} className="text-center">
                {engineLabel(engine)}
              </span>
            ))}
          </div>
          {matrix.prompts.map((prompt) => (
            <div
              key={prompt}
              className="grid items-center gap-2"
              style={columns}
            >
              <span className="text-sm">“{prompt}”</span>
              {matrix.engines.map((engine) => {
                const cell = matrix.cells.get(prompt)?.get(engine);
                if (!cell)
                  return (
                    <span
                      key={engine}
                      className="text-center text-xs text-muted-foreground"
                    >
                      not asked
                    </span>
                  );
                const label = cellLabel(cell);
                const key = `${prompt} ${engine}`;
                return (
                  <button
                    key={engine}
                    type="button"
                    aria-pressed={selected === key}
                    aria-label={`${engineLabel(engine)}, ${prompt}: ${label.main}`}
                    onClick={() => setSelected(key)}
                    className={cn(
                      "flex min-h-12 flex-col items-center justify-center rounded-lg border font-mono text-sm",
                      cellStyle[cell.state],
                      selected === key &&
                        "ring-2 ring-foreground ring-offset-2 ring-offset-background",
                    )}
                  >
                    {label.main}
                    {label.sub ? (
                      <span className="text-[11px] opacity-80">
                        {label.sub}
                      </span>
                    ) : null}
                  </button>
                );
              })}
            </div>
          ))}
        </div>
      </div>
      {pick ? (
        <div className="space-y-2 rounded-xl border p-4" aria-live="polite">
          <p className="font-medium">
            {engineLabel(pick.engine)} · “{pick.prompt}”
          </p>
          <p className="text-sm text-muted-foreground">{cellStory(pick)}</p>
          {pick.citedInstead.length > 0 ? (
            <p className="text-sm">
              <span className="text-muted-foreground">Cited instead: </span>
              {pick.citedInstead.join(", ")}
            </p>
          ) : null}
        </div>
      ) : null}
    </section>
  );
}

/** The sites AI names most, with their average position. */
export function RivalRanksCard({ rivals }: { rivals: RivalRank[] }) {
  if (rivals.length === 0) return null;
  const worst = Math.max(...rivals.map((row) => row.averageRank ?? 0), 1);
  return (
    <section
      aria-labelledby="cite-fix-rivals"
      className="space-y-3 rounded-2xl border bg-card p-5"
    >
      <div>
        <h2 id="cite-fix-rivals" className="font-serif text-2xl">
          Who AI picks instead
        </h2>
        <p className="text-sm text-muted-foreground">
          Most named in the latest check, with the average spot across checks.
          Lower is better.
        </p>
      </div>
      <ul className="space-y-3">
        {rivals.map((row) => (
          <li key={row.domain} className="space-y-1">
            <div className="flex justify-between gap-3 text-sm">
              <span
                className={cn(
                  "truncate",
                  row.you && "font-semibold text-primary",
                )}
              >
                {row.domain}
                {row.you ? " (you)" : ""}
              </span>
              <span className="shrink-0 font-mono text-muted-foreground">
                {row.averageRank === null
                  ? ""
                  : `#${row.averageRank.toFixed(1)}`}
                {row.latestCount > 0 ? ` · ${row.latestCount} answers` : ""}
              </span>
            </div>
            {row.averageRank === null ? null : (
              <div className="h-1.5 rounded bg-muted" aria-hidden>
                <div
                  className={cn(
                    "h-1.5 rounded",
                    row.you ? "bg-primary" : "bg-muted-foreground/50",
                  )}
                  style={{
                    width: `${Math.max(8, 100 - ((row.averageRank - 1) / worst) * 90)}%`,
                  }}
                />
              </div>
            )}
          </li>
        ))}
      </ul>
    </section>
  );
}

export const LIST_PAGES_ID = "cite-fix-list-pages";

/** AI-cited pages that list rivals but not the customer. */
export function ListPagesCard({ pages }: { pages: ListPage[] }) {
  if (pages.length === 0) return null;
  return (
    <section
      id={LIST_PAGES_ID}
      aria-labelledby="cite-fix-lists"
      className="scroll-mt-4 space-y-3 rounded-2xl border bg-card p-5"
    >
      <div>
        <h2 id="cite-fix-lists" className="font-serif text-2xl">
          Pages that list your rivals, not you
        </h2>
        <p className="text-sm text-muted-foreground">
          AI quoted these pages. Each links to rivals AI named and not to your
          site. Ask the owner to add you.
        </p>
      </div>
      <ul className="divide-y">
        {pages.map((page) => (
          <li key={page.url} className="space-y-1 py-3">
            <SafeExternalLink
              url={page.url}
              label={page.title || page.url}
              className="inline-flex max-w-full items-center gap-1 truncate font-medium underline"
            />
            <p className="text-xs text-muted-foreground">
              {page.domain} · quoted by{" "}
              {page.engines.map(engineLabel).join(", ")}
            </p>
            <p className="text-sm">
              <span className="text-muted-foreground">Lists: </span>
              {page.listsRivals.join(", ")}
            </p>
          </li>
        ))}
      </ul>
    </section>
  );
}

type Comparison = CiteFixReport["rivals"][number];
type Signals = Comparison["rival"]["signals"];

const SIGNALS: {
  label: string;
  why: string;
  show: (signals: Signals) => string;
}[] = [
  {
    label: "Answers in the first lines",
    why: "AI lifts the opening sentence",
    show: (s) => (s.answerFirstWords ? `${s.answerFirstWords} words` : "No"),
  },
  {
    label: "FAQ",
    why: "Matches how people ask AI",
    show: (s) => (s.faq ? "Yes" : "No"),
  },
  {
    label: "Structured data",
    why: "Tells AI what the page is about",
    show: (s) => (s.schema.length ? `${s.schema.length} types` : "None"),
  },
  {
    label: "Numbers and facts",
    why: "Specific claims get quoted",
    show: (s) => (s.statistics ? String(s.statistics) : "None"),
  },
  {
    label: "Author",
    why: "Shows who stands behind it",
    show: (s) => (s.author ? "Yes" : "No"),
  },
  {
    label: "Last updated",
    why: "Fresh pages are preferred",
    show: (s) => (s.date ? formatDate(s.date) : "Not shown"),
  },
  {
    label: "Words",
    why: "Depth of the page",
    show: (s) => s.wordCount.toLocaleString(),
  },
];

/** Rival pages and the customer's best page, signal by signal. */
export function SignalsCard({
  rivals,
  scanAction,
}: {
  rivals: Comparison[];
  scanAction: ReactNode;
}) {
  if (rivals.length === 0) return null;
  const yours = rivals.find((comparison) => comparison.yours)?.yours ?? null;
  const columns = {
    gridTemplateColumns: `minmax(0, 1.6fr) repeat(${rivals.length + 1}, minmax(5rem, 1fr))`,
  };
  return (
    <section
      aria-labelledby="cite-fix-signals"
      className="space-y-3 rounded-2xl border bg-card p-5"
    >
      <div>
        <h2 id="cite-fix-signals" className="font-serif text-2xl">
          What their pages have
        </h2>
        <p className="text-sm text-muted-foreground">
          Read for free from the pages AI cited
          {yours ? ", next to your strongest scanned page" : ""}. A gap here is
          a likely reason, not proof: the AI decides.
        </p>
      </div>
      <div className="overflow-x-auto">
        <div className="min-w-[34rem] text-sm">
          <div
            className="grid gap-3 py-2 font-mono text-xs text-muted-foreground"
            style={columns}
          >
            <span>Signal</span>
            {rivals.map((comparison) => (
              <span key={comparison.rival.url} className="truncate">
                {comparison.rival.domain}
              </span>
            ))}
            <span>You</span>
          </div>
          {SIGNALS.map((signal) => (
            <div
              key={signal.label}
              className="grid items-center gap-3 border-t py-2"
              style={columns}
            >
              <span>
                {signal.label}
                <span className="block text-xs text-muted-foreground">
                  {signal.why}
                </span>
              </span>
              {rivals.map((comparison) => (
                <span key={comparison.rival.url} className="font-mono">
                  {signal.show(comparison.rival.signals)}
                </span>
              ))}
              <span className="font-mono">
                {yours ? signal.show(yours.signals) : "–"}
              </span>
            </div>
          ))}
        </div>
      </div>
      {yours ? (
        <p className="text-xs text-muted-foreground">
          Your page: {yours.title || yours.url}
        </p>
      ) : (
        <div className="flex flex-wrap items-center gap-3">
          {scanAction}
          <span className="text-sm text-muted-foreground">
            A free scan fills the “You” column.
          </span>
        </div>
      )}
    </section>
  );
}

/** The ordered moves, each with its reason, its evidence and its action. */
export function NextStepsCard({
  steps,
  actionFor,
}: {
  steps: Step[];
  actionFor: (kind: StepKind) => ReactNode;
}) {
  if (steps.length === 0) return null;
  return (
    <section aria-labelledby="cite-fix-steps" className="space-y-3">
      <h2 id="cite-fix-steps" className="font-serif text-2xl">
        What to do next
      </h2>
      <ol className="space-y-3">
        {steps.map((step, index) => (
          <li
            key={step.kind}
            className="flex flex-wrap items-start gap-4 rounded-2xl border bg-card p-5"
          >
            <span className="font-mono text-xs text-primary">
              STEP {index + 1}
            </span>
            <div className="min-w-0 flex-[999_1_24rem] space-y-1">
              <p className="text-lg font-semibold">{step.title}</p>
              <p className="text-sm text-muted-foreground">{step.why}</p>
              <p className="text-sm">
                <span className="text-muted-foreground">Evidence: </span>
                {step.proof}
              </p>
            </div>
            <div className="flex flex-[1_1_12rem] justify-end">
              {actionFor(step.kind)}
            </div>
          </li>
        ))}
      </ol>
      <p className="text-xs text-muted-foreground">
        No tool can promise a citation: the AI decides. These are the gaps the
        evidence points to.
      </p>
    </section>
  );
}
