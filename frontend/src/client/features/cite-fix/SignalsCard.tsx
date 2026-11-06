import type { ReactNode } from "react";
import type { CiteFixReport } from "./citeFixApi";
import { formatDate } from "./DecisionSections";

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
